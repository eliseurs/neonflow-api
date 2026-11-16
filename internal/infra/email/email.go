// Package email envia os e-mails da API pelo Resend (ADR-0016): confirmação do cadastro do cliente,
// convite do parceiro e nova senha. Sem RESEND_API_KEY e EMAIL_FROM, o envio fica desligado.
package email

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/resend/resend-go/v3"

	"neonflow-api/internal/config"
)

// ErrNotConfigured: sem a chave e o remetente, não há como enviar.
var ErrNotConfigured = errors.New("envio de e-mail não configurado")

// Message é um e-mail em texto simples para um destinatário.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Sender é o que os domínios usam; os testes usam um falso.
type Sender interface {
	Enabled() bool
	Send(ctx context.Context, m Message) error
}

// New devolve o Resend configurado ou um Sender desligado.
func New(cfg config.Email) Sender {
	if !cfg.Enabled() {
		return Disabled{}
	}
	return &Resend{client: resend.NewClient(cfg.APIKey), from: cfg.From}
}

// Disabled responde ErrNotConfigured a todo envio.
type Disabled struct{}

func (Disabled) Enabled() bool                       { return false }
func (Disabled) Send(context.Context, Message) error { return ErrNotConfigured }

// Resend envia pela API do Resend.
type Resend struct {
	client *resend.Client
	from   string
}

func (r *Resend) Enabled() bool { return true }

// Send não devolve o destinatário no erro: e-mail é dado pessoal e o erro vai para o log.
func (r *Resend) Send(ctx context.Context, m Message) error {
	_, err := r.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    r.from,
		To:      []string{m.To},
		Subject: m.Subject,
		Text:    m.Text,
	})
	if err != nil {
		return fmt.Errorf("resend: %w", err)
	}
	return nil
}

// setBaseURL aponta o cliente para outro endereço (testes com servidor local).
func (r *Resend) setBaseURL(raw string) error {
	u, err := url.Parse(raw + "/")
	if err != nil {
		return err
	}
	r.client.BaseURL = u
	return nil
}
