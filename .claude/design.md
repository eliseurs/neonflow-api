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
    fleet/                  # veículos (RN-01)
    booking/                # reservas, disponibilidade e histórico (RN-02)
    checklist/              # itens do checklist de devolução, configuráveis (RN-01.6)
  infra/
    api/                    # App (Gin), Routes, Respond, healthcheck, graceful shutdown
    db/                     # Open (pool + ping)
    migrator/               # runner V{N}_*.sql (transação por migration)
  config/                   # settings (env + defaults locais)
  pkg/
    middleware/             # JWT auth, role guard, CORS
    timezone/               # fuso da operação (America/Sao_Paulo)
    security/               # TokenProvider (JWT), BcryptHasher
    logger/                 # slog (JSON em produção)
  testutil/                 # banco de teste (só build tag integration)
sql/migrations/
devops/initdb/              # cria neonflow_test no docker compose
docs/                       # ADRs, pendências, produto, deploy
```

Plano do ciclo de reserva: `docs/plano-reserva-persistente.md`.

---

## Endpoints

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| GET | `/health`, `/api/health` | — | 200 com banco ok, 503 sem banco |
| POST | `/api/auth/login` | — | `{email, password}` → `{token, expires_at, user}` |
| GET | `/api/auth/me` | JWT | Usuário do token |
| GET | `/api/vehicles` | admin, central | Lista a frota, com `status` (`manutencao`, `em_locacao`, `aguardando_checklist`, `reservado`, `livre`) e `next_booking` (RN-01.3) |
| GET | `/api/vehicles/:id` | admin, central | Um veículo, com `status` e `next_booking` |
| POST | `/api/vehicles` | admin, central | Cadastra (`plate`, `name`, `category`, `owner`, `seats`, `gear`, `daily_rate_cents`, `in_maintenance`) |
| PUT | `/api/vehicles/:id` | admin, central | Substitui todos os campos (inclusive manutenção) |
| GET | `/api/availability?start_at=&end_at=` | admin, central | Veículos fora de manutenção e sem reserva ativa no intervalo (`start_date`/`end_date` só com a data ainda são aceitos e viram 09:00) |
| GET | `/api/bookings?vehicle_id=` | admin, central | Lista reservas |
| GET | `/api/bookings/:id` | admin, central | Uma reserva |
| GET | `/api/bookings/:id/events` | admin, central | Histórico de etapas |
| POST | `/api/bookings` | admin, central | Pré-reserva (`vehicle_id`, `client_name`, `start_at`, `end_at`, `place`): bloqueia o intervalo. Data e hora `AAAA-MM-DDTHH:MM` no horário de Gramado ou RFC 3339 |
| PATCH | `/api/bookings/:id/checks` | admin, central | Marca conferências da etapa atual (`{"cnh": true}`) |
| POST | `/api/bookings/:id/advance` | admin, central | Próxima etapa, se as conferências estão feitas |
| POST | `/api/bookings/:id/cancel` | admin, central | Cancela (preparação ou confirmada) e libera o período |
| GET | `/api/checklist-items` | admin, central | Itens do checklist de devolução, em ordem (ativos e inativos) |
| POST | `/api/checklist-items` | admin, central | Novo item (`label`, `position`, `active`); a chave sai do nome e não muda |
| PUT | `/api/checklist-items/:id` | admin, central | Troca `label`, `position` e `active` (ausente mantém) |

Erros de reserva: 400 validação, 404 não encontrado, 409 conflito (período ocupado, manutenção, conferência pendente, etapa final ou já alterada, veículo aguardando o checklist de outra reserva).

Em "devolvida", as conferências exigidas são `issues` mais as chaves dos itens ativos do checklist. Uma reserva não vai para "em locação" enquanto o mesmo veículo tiver outra em "devolvida" (RN-01.6).

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

Frota e reservas (V2, V3) — regras em `.claude/business-rules/01-frota.md` e `02-reservas.md`:

```
vehicles
  id, plate (UNIQUE, sem hífen), name, owner TEXT
  category CHECK (economy|premium), gear CHECK (manual|automatico), seats INT
  daily_rate_cents BIGINT > 0, in_maintenance BOOL, created_at, updated_at

bookings
  id, vehicle_id FK, client_name, start_at TIMESTAMPTZ, end_at TIMESTAMPTZ, place   -- V5 (RN-02.23)
  stage CHECK (preparacao|confirmada|em_locacao|devolvida|encerrada|cancelada)
  days = blocos de 24h, para cima       -- RN-02.24
  daily_rate_cents, total_cents         -- cópia da diária na criação (RN-02.6)
  checks JSONB                          -- conferências: cnh, terms, deposit, payment, delivery, return, issues
  expires_at = criação + 24h            -- pré-reserva não paga é cancelada (RN-02.11)
  created_by FK users, created_at, updated_at
  EXCLUDE USING gist (vehicle_id WITH =, tstzrange(start_at, end_at) WITH &&)
    WHERE (stage NOT IN ('encerrada','cancelada'))   -- btree_gist (RN-02.1/02.2)

booking_events (booking_id FK, from_stage, to_stage, user_id NULL = automático, created_at)

checklist_items (V4)                    -- checklist de devolução (RN-01.6)
  id, key TEXT UNIQUE CHECK ^[a-z0-9_]{1,40}$ (não muda), label, position INT, active BOOL
  -- lista inicial do fundador: combustível, estética, óleo, água, pastilhas, pneus,
  -- limpeza interna, documentos, fotos, quilometragem
```

O estado do veículo é calculado na leitura da frota, sem coluna: manutenção > em locação > aguardando checklist (reserva em "devolvida") > reservado (confirmada ou pré-reserva no prazo que ainda não terminou) > livre. "Hoje" é o dia em `America/Sao_Paulo`.

A expiração roda dentro de cada operação de reserva (mesma conexão ou transação), antes de ler ou gravar. Não há job em segundo plano: uma pré-reserva vencida fica `preparacao` no banco até a próxima chamada, mas nunca é vista assim pela API.

`tstzrange(start, end)` é semiaberto `[start, end)`: devolver às 10h e retirar às 10h não se sobrepõem (RN-02.25).

A API devolve `start_at` e `end_at` no fuso de Gramado e, por compatibilidade, `start_date` e `end_date` (o dia em Gramado). O fuso fica em `internal/pkg/timezone`, com a base de fusos embutida no binário (a imagem Alpine não tem `tzdata`).

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
| `neonflow-api` | `eliseurs/neonflow-api` | Dockerfile | `/api/health` — https://neonflow-api-hml.up.railway.app |
| `Postgres` | plugin | — | interno (`${{Postgres.DATABASE_URL}}`) |

Ambiente Railway `hml`. Variáveis: `APP_ENV=hml`, `DATABASE_URL` (referência ao plugin), `JWT_SECRET`, `ALLOWED_ORIGINS` (URL do frontend). Runbook: `docs/deploy-railway.md`.
