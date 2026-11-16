package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/pkg/security"
)

// ErrAccountNotFound é o usuário do token que não existe mais ou foi desativado.
var ErrAccountNotFound = errors.New("account not found or inactive")

type TokenParser interface {
	Parse(token string) (security.Claims, error)
}

// Accounts confere o usuário do token no banco: devolve o papel atual de quem está ativo.
type Accounts interface {
	ActiveRole(ctx context.Context, userID string) (string, error)
}

// AuthMiddleware exige "Authorization: Bearer <jwt>" e injeta userID e role no contexto (RN-05.1).
// Com accounts, o papel vem do banco a cada requisição: quem foi desativado perde o acesso na hora
// e a troca de papel vale sem novo login (RN-05.8).
func AuthMiddleware(tokens TokenParser, accounts Accounts) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || scheme != "Bearer" || strings.TrimSpace(token) == "" {
			abort(c, http.StatusUnauthorized, "token de acesso ausente ou mal formatado")
			return
		}

		claims, err := tokens.Parse(strings.TrimSpace(token))
		if err != nil {
			abort(c, http.StatusUnauthorized, "token inválido ou expirado")
			return
		}

		role := claims.Role
		if accounts != nil {
			role, err = accounts.ActiveRole(c.Request.Context(), claims.UserID)
			if errors.Is(err, ErrAccountNotFound) {
				abort(c, http.StatusUnauthorized, "sessão encerrada; entre de novo")
				return
			}
			if err != nil {
				abort(c, http.StatusInternalServerError, "erro interno")
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Set("role", role)
		c.Next()
	}
}

func abort(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"status": status, "data": gin.H{"error": message}})
}
