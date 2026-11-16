package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditEntry é uma linha da auditoria: quem, com qual papel, o que chamou e o resultado.
// Não leva corpo nem query string (LGPD).
type AuditEntry struct {
	UserID string
	Role   string
	Method string
	Route  string
	Path   string
	Status int
}

type AuditRecorder interface {
	Record(ctx context.Context, e AuditEntry) error
}

// Audit grava, depois da resposta, toda chamada de usuário identificado que altera dados (inclusive
// o login) e todo acesso negado por perfil (RN-05.9). Leituras permitidas não são gravadas.
// Falha ao gravar não derruba a resposta: vai para onError.
func Audit(rec AuditRecorder, onError func(error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		userID, status := c.GetString("userID"), c.Writer.Status()
		read := c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead
		if userID == "" || (read && status != http.StatusForbidden) {
			return
		}

		// A gravação não pode morrer junto com a requisição do cliente.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 2*time.Second)
		defer cancel()
		err := rec.Record(ctx, AuditEntry{
			UserID: userID,
			Role:   c.GetString("role"),
			Method: c.Request.Method,
			Route:  c.FullPath(),
			Path:   c.Request.URL.Path,
			Status: status,
		})
		if err != nil {
			onError(err)
		}
	}
}
