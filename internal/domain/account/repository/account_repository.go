package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"neonflow-api/internal/domain/account"
	"neonflow-api/internal/domain/authentication"
)

const (
	uniqueViolation           = "23505"
	invalidTextRepresentation = "22P02"

	queryCreateCustomerAccount = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, 'cliente')
		RETURNING id, name, email, role`
	// Mesmo filtro do login: conta ativa, e o parceiro dela também.
	queryFindAccount = `
		SELECT u.id, u.name, u.email, u.role, u.email_verified_at IS NOT NULL
		FROM users u
		LEFT JOIN partners p ON p.id = u.partner_id
		WHERE lower(u.email) = $1 AND u.active AND (u.partner_id IS NULL OR p.active)`
	querySaveToken = `
		INSERT INTO account_tokens (purpose, token_hash, user_id, partner_id, expires_at)
		VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5)`
	// Uso único: o link vale uma vez, dentro do prazo.
	queryUseToken = `
		UPDATE account_tokens SET used_at = now()
		WHERE token_hash = $1 AND purpose = $2 AND used_at IS NULL AND expires_at > now()
		RETURNING coalesce(user_id::text, ''), coalesce(partner_id::text, '')`
	queryConfirm = `
		UPDATE users SET email_verified_at = coalesce(email_verified_at, now()), updated_at = now()
		WHERE id = $1 AND active AND role = 'cliente'
		RETURNING name, lower(email), coalesce(customer_id::text, '')`
	// Cadastro feito pela central com o mesmo e-mail e ainda sem conta: a conta passa a ver o histórico dele.
	queryCustomersByEmail = `
		SELECT c.id FROM customers c
		WHERE lower(c.email) = $1 AND c.anonymized_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.customer_id = c.id)
		LIMIT 2`
	queryCreateCustomer = `INSERT INTO customers (name, email) VALUES ($1, $2) RETURNING id`
	queryLinkCustomer   = `UPDATE users SET customer_id = $2, updated_at = now() WHERE id = $1`
	querySetPassword    = `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1 AND active`
	// Uma senha nova invalida os outros links de nova senha ainda abertos.
	queryCloseResetTokens = `
		UPDATE account_tokens SET used_at = now()
		WHERE user_id = $1 AND purpose = 'nova_senha' AND used_at IS NULL`
	queryPartnerForInvite = `
		SELECT p.name, p.email, p.active,
		       EXISTS (SELECT 1 FROM users u WHERE u.partner_id = p.id),
		       EXISTS (SELECT 1 FROM users u WHERE lower(u.email) = lower(trim(p.email)) AND trim(p.email) <> '')
		FROM partners p
		WHERE p.id = $1`
	// O lock no parceiro faz dois aceites do mesmo convite esperarem um pelo outro.
	queryLockPartner = `
		SELECT name, lower(trim(email)), active, EXISTS (SELECT 1 FROM users u WHERE u.partner_id = partners.id)
		FROM partners WHERE id = $1 FOR UPDATE`
	queryCreatePartnerAc = `
		INSERT INTO users (name, email, password_hash, role, partner_id, email_verified_at)
		VALUES ($1, $2, $3, 'parceiro', $4, now())`
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) CreateCustomerAccount(ctx context.Context, n account.NewCustomerAccount) (account.Account, error) {
	var a account.Account
	err := r.db.QueryRowContext(ctx, queryCreateCustomerAccount, n.Name, n.Email, n.PasswordHash).Scan(&a.ID, &a.Name, &a.Email, &a.Role)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return a, authentication.ErrEmailTaken
	}
	if err != nil {
		return a, fmt.Errorf("create customer account: %w", err)
	}
	return a, nil
}

func (r *AccountRepository) FindAccount(ctx context.Context, email string) (account.Account, error) {
	var a account.Account
	err := r.db.QueryRowContext(ctx, queryFindAccount, authentication.NormalizeEmail(email)).Scan(&a.ID, &a.Name, &a.Email, &a.Role, &a.Verified)
	if errors.Is(err, sql.ErrNoRows) {
		return a, authentication.ErrUserNotFound
	}
	if err != nil {
		return a, fmt.Errorf("find account: %w", err)
	}
	return a, nil
}

func (r *AccountRepository) SaveToken(ctx context.Context, t account.Token) error {
	if _, err := r.db.ExecContext(ctx, querySaveToken, t.Purpose, t.Hash, t.UserID, t.PartnerID, t.ExpiresAt); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	return nil
}

func (r *AccountRepository) ConfirmEmail(ctx context.Context, tokenHash string) error {
	return r.inTx(ctx, "confirm email", func(tx *sql.Tx) error {
		userID, _, err := useToken(ctx, tx, tokenHash, account.PurposeConfirmEmail)
		if err != nil {
			return err
		}
		var name, email, customerID string
		err = tx.QueryRowContext(ctx, queryConfirm, userID).Scan(&name, &email, &customerID)
		if errors.Is(err, sql.ErrNoRows) {
			return account.ErrInvalidToken
		}
		if err != nil {
			return err
		}
		if customerID != "" {
			return nil
		}
		customerID, err = customerFor(ctx, tx, name, email)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, queryLinkCustomer, userID, customerID)
		return err
	})
}

// customerFor liga a conta ao único cadastro livre com o mesmo e-mail; com nenhum ou mais de um, cria um
// cadastro novo, e a central junta os repetidos se precisar.
func customerFor(ctx context.Context, tx *sql.Tx, name, email string) (string, error) {
	rows, err := tx.QueryContext(ctx, queryCustomersByEmail, email)
	if err != nil {
		return "", err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return "", err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return "", err
	}
	if len(ids) == 1 {
		return ids[0], nil
	}
	var id string
	err = tx.QueryRowContext(ctx, queryCreateCustomer, name, email).Scan(&id)
	return id, err
}

func (r *AccountRepository) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	return r.inTx(ctx, "reset password", func(tx *sql.Tx) error {
		userID, _, err := useToken(ctx, tx, tokenHash, account.PurposeResetPassword)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, querySetPassword, userID, passwordHash)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return account.ErrInvalidToken
		}
		_, err = tx.ExecContext(ctx, queryCloseResetTokens, userID)
		return err
	})
}

func (r *AccountRepository) PartnerForInvite(ctx context.Context, partnerID string) (account.PartnerContact, error) {
	var p account.PartnerContact
	err := r.db.QueryRowContext(ctx, queryPartnerForInvite, partnerID).Scan(&p.Name, &p.Email, &p.Active, &p.HasAccount, &p.EmailTaken)
	if errors.Is(err, sql.ErrNoRows) {
		return p, account.ErrPartnerNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == invalidTextRepresentation {
		return p, account.ErrPartnerNotFound
	}
	if err != nil {
		return p, fmt.Errorf("partner for invite: %w", err)
	}
	return p, nil
}

func (r *AccountRepository) AcceptInvite(ctx context.Context, tokenHash, passwordHash string) error {
	return r.inTx(ctx, "accept invite", func(tx *sql.Tx) error {
		_, partnerID, err := useToken(ctx, tx, tokenHash, account.PurposeInvite)
		if err != nil {
			return err
		}
		var name, email string
		var active, hasAccount bool
		if err := tx.QueryRowContext(ctx, queryLockPartner, partnerID).Scan(&name, &email, &active, &hasAccount); err != nil {
			return err
		}
		// Parceiro desativado ou sem e-mail depois do convite: o link não vale mais.
		if !active || email == "" {
			return account.ErrInvalidToken
		}
		if hasAccount {
			return account.ErrPartnerHasAccount
		}
		_, err = tx.ExecContext(ctx, queryCreatePartnerAc, name, email, passwordHash, partnerID)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return authentication.ErrEmailTaken
		}
		return err
	})
}

// useToken marca o link como usado e devolve a conta ou o parceiro dele.
func useToken(ctx context.Context, tx *sql.Tx, tokenHash, purpose string) (userID, partnerID string, err error) {
	err = tx.QueryRowContext(ctx, queryUseToken, tokenHash, purpose).Scan(&userID, &partnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", account.ErrInvalidToken
	}
	return userID, partnerID, err
}

// inTx roda fn numa transação; os erros de fn voltam como estão, para os de domínio chegarem ao handler.
func (r *AccountRepository) inTx(ctx context.Context, op string, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
