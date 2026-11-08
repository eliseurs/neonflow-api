// Package contract guarda os seis contratos do Neon Flow em versões com hash e a revisão jurídica
// (pacote V15 de 07/11/2026, 06_GUIA_AREA_JURIDICA). O admin cria versões e decide a publicação;
// só a revisora jurídica aprova, e a aprovação vale para o hash exato da versão. Nada é editado:
// cada mudança é uma versão nova, e cada decisão é um evento novo.
package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode/utf8"

	"neonflow-api/internal/domain/authentication"
)

const (
	StatusMinuta    = "minuta"
	StatusEnviado   = "enviado_para_revisao"
	StatusAjustes   = "ajustes_solicitados"
	StatusAprovado  = "aprovado_pela_revisora"
	StatusLiberado  = "liberacao_administrativa"
	StatusPublicado = "publicado"
	ActionEnviar    = "enviar"
	ActionAjustes   = "solicitar_ajustes"
	ActionAprovar   = "aprovar"
	ActionRevogar   = "revogar"
	ActionLiberar   = "liberar"
	ActionPublicar  = "publicar"
	KindObservacao  = "observacao"
	KindCorrecao    = "correcao"
	maxBody         = 200_000
	maxLabel        = 40
	maxClause       = 40
	maxComment      = 10_000
	maxNote         = 2_000
	maxProposedText = 20_000
)

// transition é quem pode fazer a ação e de que situação para qual.
type transition struct {
	roles []string
	from  []string
	to    string
}

var (
	admins = []string{authentication.RoleAdmin}
	// reviewers: até a advogada ter conta, o admin também revisa (decisão de 08/11/2026, PEN-68).
	// O evento grava o papel de quem agiu, então a aprovação do admin fica visível como tal.
	reviewers = []string{authentication.RoleLegalReviewer, authentication.RoleAdmin}
)

var transitions = map[string]transition{
	ActionEnviar:  {admins, []string{StatusMinuta}, StatusEnviado},
	ActionAjustes: {reviewers, []string{StatusEnviado}, StatusAjustes},
	ActionAprovar: {reviewers, []string{StatusEnviado}, StatusAprovado},
	ActionRevogar: {reviewers, []string{StatusAprovado, StatusLiberado}, StatusEnviado},
	ActionLiberar: {admins, []string{StatusAprovado}, StatusLiberado},
	// Aprovação jurídica não publica sozinha: publicar é decisão posterior do admin.
	ActionPublicar: {admins, []string{StatusLiberado}, StatusPublicado},
}

type Document struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	Title  string   `json:"title"`
	Latest *Summary `json:"latest"`
}

// Summary é a versão sem o texto, para listas.
type Summary struct {
	ID        string    `json:"id"`
	Version   int       `json:"version"`
	Label     string    `json:"label"`
	SHA256    string    `json:"sha256"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Version struct {
	Summary
	DocumentCode  string    `json:"document_code"`
	DocumentTitle string    `json:"document_title"`
	Body          string    `json:"body"`
	Events        []Event   `json:"events"`
	Comments      []Comment `json:"comments"`
}

type Event struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Note      string    `json:"note"`
	SHA256    string    `json:"sha256"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Comment struct {
	ID           string    `json:"id"`
	Clause       string    `json:"clause"`
	Kind         string    `json:"kind"`
	Body         string    `json:"body"`
	ProposedText string    `json:"proposed_text"`
	UserID       string    `json:"user_id"`
	UserName     string    `json:"user_name"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Actor é quem está logado (do JWT); o autor nunca vem do corpo da requisição.
type Actor struct {
	UserID string
	Role   string
}

type VersionRequest struct {
	Label string `json:"label"`
	Body  string `json:"body"`
}

func (r VersionRequest) Validate() (VersionRequest, error) {
	r.Label = strings.TrimSpace(r.Label)
	r.Body = strings.TrimSpace(r.Body)
	switch {
	case r.Body == "":
		return r, ErrBodyRequired
	case len(r.Body) > maxBody:
		return r, ErrBodyTooLong
	case utf8.RuneCountInString(r.Label) > maxLabel:
		return r, ErrLabelTooLong
	}
	return r, nil
}

type CommentRequest struct {
	Clause       string `json:"clause"`
	Kind         string `json:"kind"`
	Body         string `json:"body"`
	ProposedText string `json:"proposed_text"`
}

func (r CommentRequest) Validate() (CommentRequest, error) {
	r.Clause = strings.TrimSpace(r.Clause)
	r.Kind = strings.TrimSpace(r.Kind)
	r.Body = strings.TrimSpace(r.Body)
	r.ProposedText = strings.TrimSpace(r.ProposedText)
	switch {
	case r.Clause == "" || utf8.RuneCountInString(r.Clause) > maxClause:
		return r, ErrInvalidClause
	case r.Kind != KindObservacao && r.Kind != KindCorrecao:
		return r, ErrInvalidKind
	case r.Body == "" || utf8.RuneCountInString(r.Body) > maxComment:
		return r, ErrInvalidComment
	case utf8.RuneCountInString(r.ProposedText) > maxProposedText:
		return r, ErrInvalidComment
	}
	return r, nil
}

// EventRequest: na aprovação, `sha256` é o hash do texto que a revisora leu e `confirm` a declaração expressa.
type EventRequest struct {
	Action  string `json:"action"`
	Note    string `json:"note"`
	SHA256  string `json:"sha256"`
	Confirm bool   `json:"confirm"`
}

func (r EventRequest) Validate() (EventRequest, error) {
	r.Action = strings.TrimSpace(r.Action)
	r.Note = strings.TrimSpace(r.Note)
	r.SHA256 = strings.ToLower(strings.TrimSpace(r.SHA256))
	if _, ok := transitions[r.Action]; !ok {
		return r, ErrInvalidAction
	}
	switch {
	case utf8.RuneCountInString(r.Note) > maxNote:
		return r, ErrNoteTooLong
	case (r.Action == ActionAjustes || r.Action == ActionRevogar) && r.Note == "":
		return r, ErrNoteRequired
	case r.Action == ActionAprovar && !r.Confirm:
		return r, ErrConfirmRequired
	}
	return r, nil
}

// Next diz para qual situação a ação leva, conferindo o papel de quem age, a situação atual e,
// na aprovação, se o hash lido é o da versão.
func Next(req EventRequest, actor Actor, current, versionSHA string) (string, error) {
	t, ok := transitions[req.Action]
	if !ok {
		return "", ErrInvalidAction
	}
	if !contains(t.roles, actor.Role) {
		return "", ErrForbiddenAction
	}
	if !contains(t.from, current) {
		return "", ErrWrongStatus
	}
	if req.Action == ActionAprovar && req.SHA256 != versionSHA {
		return "", ErrHashMismatch
	}
	return t.to, nil
}

// CanComment: admin e revisora comentam; a central não participa da revisão.
func CanComment(role string) bool {
	return role == authentication.RoleAdmin || role == authentication.RoleLegalReviewer
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func Hash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}
