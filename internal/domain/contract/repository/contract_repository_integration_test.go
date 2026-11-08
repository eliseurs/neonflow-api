//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/contract"
	"neonflow-api/internal/testutil"
)

func TestContractRepository_Integration(t *testing.T) {
	database := testutil.MigratedDB(t, "../../../../sql/migrations")
	repo := NewContractRepository(database)
	ctx := context.Background()

	var adminID, reviewerID string
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Admin', 'admin@neonflow.test', 'h', 'admin') RETURNING id`).Scan(&adminID))
	require.NoError(t, database.QueryRowContext(ctx,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Revisora', 'revisora@neonflow.test', 'h', 'legal_reviewer') RETURNING id`).Scan(&reviewerID))
	admin := contract.Actor{UserID: adminID, Role: authentication.RoleAdmin}
	reviewer := contract.Actor{UserID: reviewerID, Role: authentication.RoleLegalReviewer}

	// Os seis documentos vêm da migration, sem versão.
	docs, err := repo.Documents(ctx)
	require.NoError(t, err)
	require.Len(t, docs, 6)
	assert.Equal(t, "01", docs[0].Code)
	assert.Nil(t, docs[0].Latest)

	_, err = repo.Versions(ctx, "99")
	assert.ErrorIs(t, err, contract.ErrDocumentNotFound)
	_, err = repo.CreateVersion(ctx, "99", contract.VersionRequest{Body: "x"}, contract.Hash("x"), admin)
	assert.ErrorIs(t, err, contract.ErrDocumentNotFound)

	v1, err := repo.CreateVersion(ctx, "04", contract.VersionRequest{Label: "v0.4", Body: "Texto v1"}, contract.Hash("Texto v1"), admin)
	require.NoError(t, err)
	assert.Equal(t, 1, v1.Version)
	assert.Equal(t, contract.StatusMinuta, v1.Status)

	// Fluxo: envia → revisora comenta e pede ajustes → nova versão → envia → aprova o hash → libera → publica.
	approve := func(current, sha string) (string, error) {
		return contract.Next(contract.EventRequest{Action: contract.ActionAprovar, SHA256: contract.Hash("Texto v1"), Confirm: true}, reviewer, current, sha)
	}
	_, err = repo.AddEvent(ctx, v1.ID, contract.ActionAprovar, "", reviewer, approve)
	assert.ErrorIs(t, err, contract.ErrWrongStatus, "minuta ainda não enviada")

	step := func(versionID, action, note string, actor contract.Actor, sha string) contract.Event {
		t.Helper()
		e, err := repo.AddEvent(ctx, versionID, action, note, actor, func(current, s string) (string, error) {
			return contract.Next(contract.EventRequest{Action: action, SHA256: sha}, actor, current, s)
		})
		require.NoError(t, err, action)
		return e
	}
	e := step(v1.ID, contract.ActionEnviar, "", admin, "")
	assert.Equal(t, contract.StatusEnviado, e.Status)
	assert.Equal(t, contract.Hash("Texto v1"), e.SHA256)
	assert.Equal(t, "Admin", e.UserName)

	c, err := repo.AddComment(ctx, v1.ID, contract.CommentRequest{Clause: "4.2", Kind: contract.KindCorrecao, Body: "Prazo de liberação", ProposedText: "Nova redação"}, reviewer)
	require.NoError(t, err)
	assert.Equal(t, "Revisora", c.UserName)
	_, err = repo.AddComment(ctx, "00000000-0000-0000-0000-000000000000", contract.CommentRequest{Clause: "1", Kind: contract.KindObservacao, Body: "x"}, reviewer)
	assert.ErrorIs(t, err, contract.ErrVersionNotFound)
	_, err = repo.AddComment(ctx, "nao-e-uuid", contract.CommentRequest{Clause: "1", Kind: contract.KindObservacao, Body: "x"}, reviewer)
	assert.ErrorIs(t, err, contract.ErrVersionNotFound)

	step(v1.ID, contract.ActionAjustes, "ajustar 4.2", reviewer, "")

	v2, err := repo.CreateVersion(ctx, "04", contract.VersionRequest{Label: "v0.5", Body: "Texto v2"}, contract.Hash("Texto v2"), admin)
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)
	step(v2.ID, contract.ActionEnviar, "", admin, "")

	// Aprovar com o hash da versão antiga é recusado.
	_, err = repo.AddEvent(ctx, v2.ID, contract.ActionAprovar, "", reviewer, approve)
	assert.ErrorIs(t, err, contract.ErrHashMismatch)
	step(v2.ID, contract.ActionAprovar, "ok", reviewer, contract.Hash("Texto v2"))
	step(v2.ID, contract.ActionLiberar, "", admin, "")
	step(v2.ID, contract.ActionPublicar, "", admin, "")

	got, err := repo.Version(ctx, v2.ID)
	require.NoError(t, err)
	assert.Equal(t, contract.StatusPublicado, got.Status)
	assert.Equal(t, "04", got.DocumentCode)
	assert.Equal(t, "Texto v2", got.Body)
	require.Len(t, got.Events, 4)
	assert.Equal(t, contract.ActionAprovar, got.Events[1].Action)
	assert.Equal(t, reviewerID, got.Events[1].UserID)

	old, err := repo.Version(ctx, v1.ID)
	require.NoError(t, err)
	assert.Equal(t, contract.StatusAjustes, old.Status, "a versão anterior guarda a própria história")
	require.Len(t, old.Comments, 1)
	assert.Equal(t, "4.2", old.Comments[0].Clause)

	versions, err := repo.Versions(ctx, "04")
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, 2, versions[0].Version, "mais nova primeiro")
	docs, err = repo.Documents(ctx)
	require.NoError(t, err)
	require.NotNil(t, docs[3].Latest)
	assert.Equal(t, contract.StatusPublicado, docs[3].Latest.Status)

	_, err = repo.Version(ctx, "nao-e-uuid")
	assert.ErrorIs(t, err, contract.ErrVersionNotFound)
	_, err = repo.AddEvent(ctx, "00000000-0000-0000-0000-000000000000", contract.ActionEnviar, "", admin, approve)
	assert.ErrorIs(t, err, contract.ErrVersionNotFound)

	// Append-only: o banco recusa alterar ou apagar versão, evento e comentário.
	for _, q := range []string{
		`UPDATE contract_versions SET body = 'adulterado' WHERE id = $1`,
		`DELETE FROM contract_versions WHERE id = $1`,
		`UPDATE contract_events SET note = 'x' WHERE version_id = $1`,
		`DELETE FROM contract_comments WHERE version_id = $1`,
	} {
		id := v2.ID
		if q == `DELETE FROM contract_comments WHERE version_id = $1` {
			id = v1.ID
		}
		_, err := database.ExecContext(ctx, q, id)
		assert.Error(t, err, q)
	}

	require.NoError(t, database.Close())
	_, err = repo.Documents(ctx)
	assert.Error(t, err)
	_, err = repo.Versions(ctx, "04")
	assert.Error(t, err)
	_, err = repo.CreateVersion(ctx, "04", contract.VersionRequest{Body: "x"}, contract.Hash("x"), admin)
	assert.Error(t, err)
	_, err = repo.Version(ctx, v2.ID)
	assert.Error(t, err)
	_, err = repo.AddComment(ctx, v2.ID, contract.CommentRequest{Clause: "1", Kind: contract.KindObservacao, Body: "x"}, admin)
	assert.Error(t, err)
	_, err = repo.AddEvent(ctx, v2.ID, contract.ActionEnviar, "", admin, approve)
	assert.Error(t, err)
}
