package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/pkg/security"
)

type TokenParser interface {
	Parse(token string) (security.Claims, error)
}

// AuthMiddleware exige "Authorization: Bearer <jwt>" e injeta userID e role no contexto (RN-05.1).
func AuthMiddleware(tokens TokenParser) gin.HandlerFunc {
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

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func abort(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"status": status, "data": gin.H{"error": message}})
}
