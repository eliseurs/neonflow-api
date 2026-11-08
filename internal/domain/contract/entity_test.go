package contract

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/authentication"
)

var (
	admin    = Actor{UserID: "a", Role: authentication.RoleAdmin}
	reviewer = Actor{UserID: "r", Role: authentication.RoleLegalReviewer}
	central  = Actor{UserID: "c", Role: authentication.RoleCentral}
)

func TestNext(t *testing.T) {
	sha := Hash("texto")
	cases := []struct {
		name    string
		req     EventRequest
		actor   Actor
		current string
		want    string
		err     error
	}{
		{"admin envia a minuta", EventRequest{Action: ActionEnviar}, admin, StatusMinuta, StatusEnviado, nil},
		{"revisora não envia", EventRequest{Action: ActionEnviar}, reviewer, StatusMinuta, "", ErrForbiddenAction},
		{"não reenvia versão com ajustes", EventRequest{Action: ActionEnviar}, admin, StatusAjustes, "", ErrWrongStatus},
		{"revisora pede ajustes", EventRequest{Action: ActionAjustes}, reviewer, StatusEnviado, StatusAjustes, nil},
		{"revisora aprova o hash lido", EventRequest{Action: ActionAprovar, SHA256: sha}, reviewer, StatusEnviado, StatusAprovado, nil},
		{"admin aprova até a advogada ter conta", EventRequest{Action: ActionAprovar, SHA256: sha}, admin, StatusEnviado, StatusAprovado, nil},
		{"admin pede ajustes", EventRequest{Action: ActionAjustes}, admin, StatusEnviado, StatusAjustes, nil},
		{"admin revoga", EventRequest{Action: ActionRevogar}, admin, StatusAprovado, StatusEnviado, nil},
		{"revisora não libera", EventRequest{Action: ActionLiberar}, reviewer, StatusAprovado, "", ErrForbiddenAction},
		{"admin com hash diferente não aprova", EventRequest{Action: ActionAprovar, SHA256: Hash("outro")}, admin, StatusEnviado, "", ErrHashMismatch},
		{"central não aprova", EventRequest{Action: ActionAprovar, SHA256: sha}, central, StatusEnviado, "", ErrForbiddenAction},
		{"hash diferente não aprova", EventRequest{Action: ActionAprovar, SHA256: Hash("outro")}, reviewer, StatusEnviado, "", ErrHashMismatch},
		{"minuta não enviada não aprova", EventRequest{Action: ActionAprovar, SHA256: sha}, reviewer, StatusMinuta, "", ErrWrongStatus},
		{"revoga aprovação", EventRequest{Action: ActionRevogar}, reviewer, StatusAprovado, StatusEnviado, nil},
		{"revoga depois da liberação", EventRequest{Action: ActionRevogar}, reviewer, StatusLiberado, StatusEnviado, nil},
		{"não revoga publicada", EventRequest{Action: ActionRevogar}, reviewer, StatusPublicado, "", ErrWrongStatus},
		{"admin libera aprovada", EventRequest{Action: ActionLiberar}, admin, StatusAprovado, StatusLiberado, nil},
		{"não libera sem aprovação", EventRequest{Action: ActionLiberar}, admin, StatusEnviado, "", ErrWrongStatus},
		{"admin publica liberada", EventRequest{Action: ActionPublicar}, admin, StatusLiberado, StatusPublicado, nil},
		{"aprovação não publica direto", EventRequest{Action: ActionPublicar}, admin, StatusAprovado, "", ErrWrongStatus},
		{"ação desconhecida", EventRequest{Action: "apagar"}, admin, StatusMinuta, "", ErrInvalidAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Next(tc.req, tc.actor, tc.current, sha)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEventRequestValidate(t *testing.T) {
	r, err := EventRequest{Action: " aprovar ", SHA256: " ABC ", Confirm: true}.Validate()
	require.NoError(t, err)
	assert.Equal(t, ActionAprovar, r.Action)
	assert.Equal(t, "abc", r.SHA256)

	_, err = EventRequest{Action: "x"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidAction)
	_, err = EventRequest{Action: ActionAprovar}.Validate()
	assert.ErrorIs(t, err, ErrConfirmRequired)
	_, err = EventRequest{Action: ActionAjustes, Note: "  "}.Validate()
	assert.ErrorIs(t, err, ErrNoteRequired)
	_, err = EventRequest{Action: ActionRevogar}.Validate()
	assert.ErrorIs(t, err, ErrNoteRequired)
	_, err = EventRequest{Action: ActionEnviar, Note: strings.Repeat("a", maxNote+1)}.Validate()
	assert.ErrorIs(t, err, ErrNoteTooLong)
}

func TestVersionRequestValidate(t *testing.T) {
	r, err := VersionRequest{Label: " v0.4 ", Body: "  texto  "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "v0.4", r.Label)
	assert.Equal(t, "texto", r.Body)

	_, err = VersionRequest{Body: " "}.Validate()
	assert.ErrorIs(t, err, ErrBodyRequired)
	_, err = VersionRequest{Body: strings.Repeat("a", maxBody+1)}.Validate()
	assert.ErrorIs(t, err, ErrBodyTooLong)
	_, err = VersionRequest{Body: "x", Label: strings.Repeat("a", maxLabel+1)}.Validate()
	assert.ErrorIs(t, err, ErrLabelTooLong)
}

func TestCommentRequestValidate(t *testing.T) {
	r, err := CommentRequest{Clause: " 4.2 ", Kind: KindCorrecao, Body: " ajustar ", ProposedText: " nova redação "}.Validate()
	require.NoError(t, err)
	assert.Equal(t, "4.2", r.Clause)
	assert.Equal(t, "nova redação", r.ProposedText)

	_, err = CommentRequest{Kind: KindObservacao, Body: "x"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidClause)
	_, err = CommentRequest{Clause: "1", Kind: "outro", Body: "x"}.Validate()
	assert.ErrorIs(t, err, ErrInvalidKind)
	_, err = CommentRequest{Clause: "1", Kind: KindObservacao}.Validate()
	assert.ErrorIs(t, err, ErrInvalidComment)
	_, err = CommentRequest{Clause: "1", Kind: KindObservacao, Body: "x", ProposedText: strings.Repeat("a", maxProposedText+1)}.Validate()
	assert.ErrorIs(t, err, ErrInvalidComment)
}

func TestCanCommentAndHash(t *testing.T) {
	assert.True(t, CanComment(authentication.RoleAdmin))
	assert.True(t, CanComment(authentication.RoleLegalReviewer))
	assert.False(t, CanComment(authentication.RoleCentral))
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", Hash("hello"))
}
