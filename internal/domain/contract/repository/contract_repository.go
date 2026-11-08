package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/contract"
	"neonflow-api/internal/pkg/timezone"
)

const (
	invalidTextRepresentation = "22P02"

	// statusOf: situação = último evento da versão; sem evento, minuta.
	statusOf = `coalesce((SELECT e.status FROM contract_events e WHERE e.version_id = v.id
		ORDER BY e.created_at DESC, e.id DESC LIMIT 1), 'minuta')`

	queryDocuments = `
		SELECT d.id, d.code, d.title, v.id, v.version, v.label, v.sha256, ` + statusOf + `, v.created_at
		FROM contract_documents d
		LEFT JOIN LATERAL (SELECT * FROM contract_versions cv WHERE cv.document_id = d.id
			ORDER BY cv.version DESC LIMIT 1) v ON true
		ORDER BY d.code`
	queryDocumentID = `SELECT id FROM contract_documents WHERE code = $1`
	queryVersions   = `
		SELECT v.id, v.version, v.label, v.sha256, ` + statusOf + `, v.created_at
		FROM contract_versions v WHERE v.document_id = $1 ORDER BY v.version DESC`
	queryCreateVersion = `
		INSERT INTO contract_versions (document_id, version, label, body, sha256, created_by)
		SELECT $1, coalesce(max(version), 0) + 1, $2, $3, $4, NULLIF($5, '')::uuid
		FROM contract_versions WHERE document_id = $1
		RETURNING id, version, label, sha256, created_at`
	queryVersion = `
		SELECT v.id, v.version, v.label, v.sha256, ` + statusOf + `, v.created_at, d.code, d.title, v.body
		FROM contract_versions v JOIN contract_documents d ON d.id = v.document_id
		WHERE v.id = $1`
	queryEvents = `
		SELECT e.id, e.action, e.status, e.note, e.sha256, e.user_id, u.name, e.role, e.created_at
		FROM contract_events e JOIN users u ON u.id = e.user_id
		WHERE e.version_id = $1 ORDER BY e.created_at, e.id`
	queryComments = `
		SELECT c.id, c.clause, c.kind, c.body, c.proposed_text, c.user_id, u.name, c.role, c.created_at
		FROM contract_comments c JOIN users u ON u.id = c.user_id
		WHERE c.version_id = $1 ORDER BY c.created_at, c.id`
	queryAddComment = `
		INSERT INTO contract_comments (version_id, clause, kind, body, proposed_text, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, (SELECT name FROM users WHERE id = $6)`
	queryLockVersion = `SELECT v.sha256, ` + statusOf + ` FROM contract_versions v WHERE v.id = $1 FOR UPDATE`
	queryAddEvent    = `
		INSERT INTO contract_events (version_id, action, status, note, sha256, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, clock_timestamp())
		RETURNING id, created_at, (SELECT name FROM users WHERE id = $6)`
)

type ContractRepository struct {
	db *sql.DB
}

func NewContractRepository(db *sql.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

func (r *ContractRepository) Documents(ctx context.Context) ([]contract.Document, error) {
	rows, err := r.db.QueryContext(ctx, queryDocuments)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []contract.Document{}
	for rows.Next() {
		var d contract.Document
		var id, label, sha, status sql.NullString
		var version sql.NullInt64
		var created sql.NullTime
		if err := rows.Scan(&d.ID, &d.Code, &d.Title, &id, &version, &label, &sha, &status, &created); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		if id.Valid {
			d.Latest = &contract.Summary{ID: id.String, Version: int(version.Int64), Label: label.String,
				SHA256: sha.String, Status: status.String, CreatedAt: created.Time.In(timezone.Location)}
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	return list, nil
}

func (r *ContractRepository) Versions(ctx context.Context, code string) ([]contract.Summary, error) {
	docID, err := r.documentID(ctx, r.db, code)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, queryVersions, docID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []contract.Summary{}
	for rows.Next() {
		var s contract.Summary
		if err := rows.Scan(&s.ID, &s.Version, &s.Label, &s.SHA256, &s.Status, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		s.CreatedAt = s.CreatedAt.In(timezone.Location)
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	return list, nil
}

func (r *ContractRepository) CreateVersion(ctx context.Context, code string, req contract.VersionRequest, sha string, actor contract.Actor) (contract.Summary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return contract.Summary{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	docID, err := r.documentID(ctx, tx, code)
	if err != nil {
		return contract.Summary{}, err
	}
	// Trava o documento para duas versões simultâneas não disputarem o mesmo número.
	if _, err := tx.ExecContext(ctx, `SELECT 1 FROM contract_documents WHERE id = $1 FOR UPDATE`, docID); err != nil {
		return contract.Summary{}, fmt.Errorf("lock contract: %w", err)
	}
	s := contract.Summary{Status: contract.StatusMinuta}
	if err := tx.QueryRowContext(ctx, queryCreateVersion, docID, req.Label, req.Body, sha, actor.UserID).
		Scan(&s.ID, &s.Version, &s.Label, &s.SHA256, &s.CreatedAt); err != nil {
		return contract.Summary{}, fmt.Errorf("create version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return contract.Summary{}, fmt.Errorf("commit: %w", err)
	}
	s.CreatedAt = s.CreatedAt.In(timezone.Location)
	return s, nil
}

func (r *ContractRepository) Version(ctx context.Context, id string) (contract.Version, error) {
	var v contract.Version
	err := r.db.QueryRowContext(ctx, queryVersion, id).Scan(&v.ID, &v.Version, &v.Label, &v.SHA256, &v.Status,
		&v.CreatedAt, &v.DocumentCode, &v.DocumentTitle, &v.Body)
	if err != nil {
		return v, mapNotFound(err, contract.ErrVersionNotFound, "find version")
	}
	v.CreatedAt = v.CreatedAt.In(timezone.Location)
	if v.Events, err = r.events(ctx, id); err != nil {
		return v, err
	}
	if v.Comments, err = r.comments(ctx, id); err != nil {
		return v, err
	}
	return v, nil
}

func (r *ContractRepository) AddComment(ctx context.Context, versionID string, req contract.CommentRequest, actor contract.Actor) (contract.Comment, error) {
	c := contract.Comment{Clause: req.Clause, Kind: req.Kind, Body: req.Body, ProposedText: req.ProposedText,
		UserID: actor.UserID, Role: actor.Role}
	err := r.db.QueryRowContext(ctx, queryAddComment, versionID, req.Clause, req.Kind, req.Body, req.ProposedText,
		actor.UserID, actor.Role).Scan(&c.ID, &c.CreatedAt, &c.UserName)
	if err != nil {
		return c, mapWriteErr(err, "add comment")
	}
	c.CreatedAt = c.CreatedAt.In(timezone.Location)
	return c, nil
}

func (r *ContractRepository) AddEvent(ctx context.Context, versionID, action, note string, actor contract.Actor,
	next func(current, sha string) (string, error)) (contract.Event, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return contract.Event{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var sha, current string
	if err := tx.QueryRowContext(ctx, queryLockVersion, versionID).Scan(&sha, &current); err != nil {
		return contract.Event{}, mapNotFound(err, contract.ErrVersionNotFound, "lock version")
	}
	status, err := next(current, sha)
	if err != nil {
		return contract.Event{}, err
	}
	e := contract.Event{Action: action, Status: status, Note: note, SHA256: sha, UserID: actor.UserID, Role: actor.Role}
	if err := tx.QueryRowContext(ctx, queryAddEvent, versionID, action, status, note, sha, actor.UserID, actor.Role).
		Scan(&e.ID, &e.CreatedAt, &e.UserName); err != nil {
		return contract.Event{}, fmt.Errorf("add event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return contract.Event{}, fmt.Errorf("commit: %w", err)
	}
	e.CreatedAt = e.CreatedAt.In(timezone.Location)
	return e, nil
}

func (r *ContractRepository) events(ctx context.Context, versionID string) ([]contract.Event, error) {
	rows, err := r.db.QueryContext(ctx, queryEvents, versionID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []contract.Event{}
	for rows.Next() {
		var e contract.Event
		if err := rows.Scan(&e.ID, &e.Action, &e.Status, &e.Note, &e.SHA256, &e.UserID, &e.UserName, &e.Role, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.CreatedAt = e.CreatedAt.In(timezone.Location)
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return list, nil
}

func (r *ContractRepository) comments(ctx context.Context, versionID string) ([]contract.Comment, error) {
	rows, err := r.db.QueryContext(ctx, queryComments, versionID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	list := []contract.Comment{}
	for rows.Next() {
		var c contract.Comment
		if err := rows.Scan(&c.ID, &c.Clause, &c.Kind, &c.Body, &c.ProposedText, &c.UserID, &c.UserName, &c.Role, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		c.CreatedAt = c.CreatedAt.In(timezone.Location)
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return list, nil
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *ContractRepository) documentID(ctx context.Context, q queryer, code string) (string, error) {
	var id string
	err := q.QueryRowContext(ctx, queryDocumentID, code).Scan(&id)
	return id, mapNotFound(err, contract.ErrDocumentNotFound, "find contract")
}

func mapNotFound(err error, notFound error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return notFound
	}
	return fmt.Errorf("%s: %w", op, err)
}

// mapWriteErr: versão inexistente (FK) ou id inválido viram "não encontrada".
func mapWriteErr(err error, op string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == invalidTextRepresentation) {
		return contract.ErrVersionNotFound
	}
	return fmt.Errorf("%s: %w", op, err)
}
