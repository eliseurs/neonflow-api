# Style Guide — Neonflow API

---

## Go

### Nomenclatura
- Structs de domínio: PascalCase sem sufixo (`User`, `Vehicle`, `Booking`)
- Interfaces de contrato: sem prefixo I (`Repository`, `Service`, `Logger`)
- Erros de domínio: `var ErrFoo = errors.New("mensagem em português")`
- Handlers: `{Domain}Handler` com construtor `New{Domain}Handler(...) *{Domain}Handler`
- Routes: função `Register(app *api.App)` no pacote `route`
- Arquivos: `{domain}_handler.go`, `{domain}_service.go`, `{domain}_repository.go`; testes ao lado

### Handler (Gin)
```go
func (h *AuthHandler) Me(c *gin.Context) {
    userID := c.GetString("userID")
    if userID == "" {
        api.RespondError(c, http.StatusUnauthorized, "não autenticado")
        return
    }
    user, err := h.svc.Me(c.Request.Context(), userID)
    if err != nil {
        if errors.Is(err, authentication.ErrUserNotFound) {
            api.RespondError(c, http.StatusNotFound, authentication.ErrUserNotFound.Error())
            return
        }
        h.log.Error("me failed", "error", err)
        api.RespondError(c, http.StatusInternalServerError, "erro interno")
        return
    }
    api.Respond(c, http.StatusOK, user)
}
```

### Rotas
```go
app.API(api.Routes{}).POST("/auth/login", h.Login)                                  // pública
app.API(api.Routes{Authorization: true}).GET("/auth/me", h.Me)                     // JWT
app.API(api.Routes{Authorization: true, Roles: []string{"admin"}}).POST("/x", h.X) // JWT + papel
```

### Queries SQL
```go
const queryFindByID = `
    SELECT id, name, email, role, created_at, updated_at
    FROM users
    WHERE id = $1
`
```
- Constantes de query no topo do arquivo do repository
- `sql.ErrNoRows` vira erro de domínio (`ErrUserNotFound`)

### Imports
- Ordem: stdlib → interno (`neonflow-api/...`) → externos, separados por linha em branco

### Testes
- testify (`assert`/`require`), table-driven quando houver variações
- Fakes manuais das interfaces de `contracts.go`
- Integração: `//go:build integration` + `testutil.MigratedDB`

---

## SQL

- `snake_case`, tabelas no plural (`users`, `vehicles`, `bookings`)
- PK `id UUID DEFAULT gen_random_uuid()`
- `created_at`/`updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Monetário em centavos (`BIGINT`, sufixo `_cents`)
- `CHECK` para enums pequenos (`role`, `stage`)
