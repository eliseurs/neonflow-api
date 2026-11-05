# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> After every change, update `.claude/specs.md` with what was modified.

## Reference files

| File | When to read |
|---|---|
| `.claude/rules.md` | **Always** — technical rules and constraints |
| `.claude/design.md` | When creating domains, routes, tables, or new features |
| `.claude/style.md` | When writing any code (Go, SQL) |
| `.claude/specs.md` | To understand recent changes and in-progress plans |
| `.claude/business-rules/` | Before touching frota, reservas, financeiro, integrações or auth |
| `docs/pendencias.md` | Before implementing any business rule — 65 items, 15 blocking |
| `docs/adr/` | Architecture decisions (MADR) |

Frontend: repositório irmão `../neonflow-app` (React). Protótipo congelado: `../neonflow-app/prototipo`.

---

## Commands

```bash
make help                          # lista todos os comandos

# Infraestrutura local
make db-up                         # sobe só o PostgreSQL (docker compose)
make up                            # sobe PostgreSQL + API (aplica migrations ao subir)
make down                          # derruba tudo

# Desenvolvimento
make dev                           # roda a API local (defaults de settings.go)
make build                         # compila api, migrate e create-admin em bin/
make lint                          # gofmt + go vet (unit e integration)

# Testes
make test                          # unitários
make test-integration              # + integração (TEST_DATABASE_URL=postgres://neonflow:neonflow@localhost:5432/neonflow_test?sslmode=disable)
make cover                         # cobertura de internal/ (meta 80%)

# Database
make migrate-up                    # aplica migrations
make migrate-down                  # desfaz a última
make migrate-status                # lista status
make migrate-create name=foo       # cria V{N+1}_foo.up/down.sql
make create-admin ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... [ADMIN_ROLE=central]

# Railway (projeto neonflow)
railway run --service neonflow-api ./migrate status
railway logs --service neonflow-api
```

---

## Architecture

**Neonflow API** — backend da vertical de aluguel de veículos, frota, reservas e transfer do Nexus OS (Serra Gaúcha).

- **Backend**: Go 1.25, Gin, PostgreSQL 16, pgx v5, JWT (golang-jwt v5), bcrypt
- **Frontend**: React + TypeScript + Vite + Tailwind + Zustand (repo separado em `../neonflow-app`)
- **Infra**: Railway, projeto `neonflow`, ambiente `hml` (serviço `neonflow-api` + plugin `Postgres`), https://neonflow-api-production.up.railway.app (o domínio manteve o nome antigo). Local via `docker-compose.yml`
- **Migrations**: runner Go puro — `sql/migrations/V{N}_*.up/down.sql`. Rodam ao iniciar o container (`CMD` do Dockerfile)

### Estado atual

Scaffold: `GET /health`, `GET /api/health`, `POST /api/auth/login`, `GET /api/auth/me`. **Frota e reservas ainda não existem** — plano em `docs/plano-reserva-persistente.md`, bloqueado pelas pendências PEN-01 a PEN-06.

### Ambientes

| `APP_ENV` | Arquivo | Comportamento |
|---|---|---|
| `local` | nenhum | defaults de `internal/config/settings.go`, banco do docker compose |
| `hml` | nenhum (env vars injetadas) | Railway, ambiente `hml` (o único hoje) — sem `JWT_SECRET` e `DATABASE_URL` a API não sobe |
| `production` ou outro | nenhum (env vars injetadas) | mesmas exigências: todo valor diferente de `local` é ambiente publicado (`IsDeployed`) |

Defaults locais ficam em `internal/config/settings.go` — não criar `.env.local`.

### Domain-Driven Design

Cada domínio em `internal/domain/` segue:

```
{domain}/
  entity.go       # structs, request/response types, validação
  contracts.go    # interfaces Repository, Service, Logger
  errors.go       # var ErrFoo = errors.New("...")
  repository/     # SQL com pgx
  service/        # lógica de negócio
  handler/        # handlers Gin
  route/          # composição: repo → service → handler + registro de rotas
```

Todos os domínios são registrados em `cmd/api/main.go`.

### Request flow

```
Request → Gin Router → Recovery → Request logger → CORS → [JWT Auth] → [Role Guard] → Handler → Service → Repository → PostgreSQL
```

JWT middleware injeta `c.Set("userID", ...)` e `c.Set("role", ...)`.

### Resposta da API

Sempre `{ "status": <int>, "data": <T> }`. Erros: `{ "status": <int>, "data": { "error": "<mensagem em português>" } }`.
