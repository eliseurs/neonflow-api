package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"neonflow-api/internal/pkg/authz"
)

// Require libera a rota só para os papéis que têm a permissão (RN-05.10). Use depois do AuthMiddleware.
func Require(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authz.Can(c.GetString("role"), permission) {
			abort(c, http.StatusForbidden, "acesso negado para este perfil")
			return
		}
		c.Next()
	}
}
