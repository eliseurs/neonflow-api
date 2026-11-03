# Design & Arquitetura — Neonflow API

## Visão Geral

Backend da vertical de **aluguel de veículos, frota, reservas, transfer e motoristas parceiros** do Nexus OS, na Serra Gaúcha (Gramado/Canela). Nome confirmado pelo fundador: **Neon Flow** (ADR-0002).

Produto, perfis, jornada e roadmap: `docs/produto/`. Pendências: `docs/pendencias.md`.

---

## Stack

| Camada | Tecnologia |
|---|---|
| API | Go 1.25, Gin, `database/sql` + pgx v5 |
| Banco | PostgreSQL 16 |
| Auth | JWT HS256 (golang-jwt v5, issuer `neonflow-api`), bcrypt |
| Testes | testing + testify; integração com build tag `integration` |
| Deploy | Railway — Dockerfile; o container roda `./migrate up && ./api` (serviços novos ignoram `railway.json`) |

Decisão: ADR-0009. Mesma stack do cognia-api.

---

## Estrutura

```
cmd/
  api/main.go               # servidor
  migrate/main.go           # migrate [up|down|status]
  create-admin/main.go      # cria usuário (env ADMIN_*)
internal/
  domain/
    authentication/         # login, me, criação de usuário (CLI)
  infra/
    api/                    # App (Gin), Routes, Respond, healthcheck, graceful shutdown
    db/                     # Open (pool + ping)
    migrator/               # runner V{N}_*.sql (transação por migration)
  config/                   # settings (env + defaults locais)
  pkg/
    middleware/             # JWT auth, role guard, CORS
    security/               # TokenProvider (JWT), BcryptHasher
    logger/                 # slog (JSON em produção)
  testutil/                 # banco de teste (só build tag integration)
sql/migrations/
devops/initdb/              # cria neonflow_test no docker compose
docs/                       # ADRs, pendências, produto, deploy
```

Domínios planejados (`docs/plano-reserva-persistente.md`): `fleet` (veículos) e `booking` (reservas).

---

## Endpoints

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| GET | `/health`, `/api/health` | — | 200 com banco ok, 503 sem banco |
| POST | `/api/auth/login` | — | `{email, password}` → `{token, expires_at, user}` |
| GET | `/api/auth/me` | JWT | Usuário do token |

---

## Modelo de dados

```
users
  id UUID PK DEFAULT gen_random_uuid()
  name, email, password_hash TEXT NOT NULL
  role TEXT CHECK (role IN ('admin','central'))
  created_at, updated_at TIMESTAMPTZ
  UNIQUE INDEX lower(email)

schema_migrations (version TEXT PK, applied_at)
```

Planejado — ver `.claude/business-rules/01-frota.md` e `02-reservas.md`:

```
vehicles  (id, plate, name, category, owner, seats, gear, daily_rate_cents, in_maintenance, ...)
bookings  (id, vehicle_id FK, client_name, start_date, end_date, place, stage,
           daily_rate_cents, total_cents, paid, checks JSONB, ...)
  EXCLUDE USING gist (vehicle_id WITH =, daterange(start_date, end_date) WITH &&)
    WHERE (stage NOT IN ('encerrada','cancelada'))   -- requer btree_gist (RN-02.1/02.2)
```

`daterange(start, end)` é semiaberto `[start, end)`: datas adjacentes não se sobrepõem, como no protótipo.

---

## Ecossistema

| Sistema | Responsabilidade | Relação |
|---|---|---|
| Neon Flow | Oferta, frota, locação, transfer e economia da vertical | Este projeto |
| Nexus Core | Identidade, contexto (CTX), eventos, governança | Capacidades comuns (ADR-0004) |
| Shell | Entrada e continuidade do cliente | Não é painel administrativo |
| NeoAuto / Neo360 / NeoReferral | Venda de veículos / imobiliário / indicações | Verticais separadas (RN-04.1) |

Contratos propostos: `docs/integracoes.md`. C4: `.claude/business-rules/README.md`.

---

## Railway

| Serviço | Repo | Build | Healthcheck |
|---|---|---|---|
| `neonflow-api` | `eliseurs/neonflow-api` | Dockerfile | `/api/health` — https://neonflow-api-production.up.railway.app |
| `Postgres` | plugin | — | interno (`${{Postgres.DATABASE_URL}}`) |

Variáveis: `APP_ENV=production`, `DATABASE_URL` (referência ao plugin), `JWT_SECRET`, `ALLOWED_ORIGINS` (URL do frontend). Runbook: `docs/deploy-railway.md`.
