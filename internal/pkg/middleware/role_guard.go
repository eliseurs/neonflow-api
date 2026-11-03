package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
)

// RoleGuard permite a rota só para os papéis informados (RN-05.3). Use depois do AuthMiddleware.
func RoleGuard(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !slices.Contains(roles, c.GetString("role")) {
			abort(c, http.StatusForbidden, "acesso negado para este perfil")
			return
		}
		c.Next()
	}
}
