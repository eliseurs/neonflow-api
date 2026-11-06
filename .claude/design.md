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
        user/                   # gestão de usuários pelo admin e troca da própria senha (RN-05.7)
        partner/                # parceiros: proprietário, motorista, indicador; código de indicação (RN-07)
    payout/                 # comissão do indicador e repasse ao dono, gerados no encerramento (RN-03.12, RN-03.13)
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
| POST | `/api/vehicles` | admin, central | Cadastra (`plate`, `name`, `category`, `owner`, `seats`, `gear`, `daily_rate_cents`, `in_maintenance` e, opcionais, `deposit_cents`, `km_limit_per_day` com `extra_km_cents`) |
| PUT | `/api/vehicles/:id` | admin, central | Substitui todos os campos (inclusive manutenção) |
| GET | `/api/availability?start_at=&end_at=` | admin, central | Veículos fora de manutenção e sem reserva ativa no intervalo (`start_date`/`end_date` só com a data ainda são aceitos e viram 09:00) |
| GET | `/api/bookings?vehicle_id=` | admin, central | Lista reservas |
| GET | `/api/bookings/:id` | admin, central | Uma reserva |
| GET | `/api/bookings/:id/events` | admin, central | Histórico de etapas |
| POST | `/api/bookings` | admin, central | Pré-reserva (`vehicle_id`, `client_name`, `start_at`, `end_at`, `place`, `km_plan` livre|limitado): bloqueia o intervalo. Data e hora `AAAA-MM-DDTHH:MM` no horário de Gramado ou RFC 3339 |
| PATCH | `/api/bookings/:id/checks` | admin, central | Marca conferências da etapa atual (`{"cnh": true}`) |
| POST | `/api/bookings/:id/advance` | admin, central | Próxima etapa, se as conferências estão feitas. Corpo opcional `{"at": "AAAA-MM-DDTHH:MM", "km": 12345}`: hora real e odômetro da entrega ou da devolução (hora padrão: agora; km obrigatório no plano limitado). Na devolução, calcula o atraso (RN-02.19) |
| POST | `/api/bookings/:id/cancel` | admin, central | Cancela (preparação ou confirmada) e libera o período |
| GET | `/api/checklist-items` | admin, central | Itens do checklist de devolução, em ordem (ativos e inativos) |
| POST | `/api/checklist-items` | admin, central | Novo item (`label`, `position`, `active`); a chave sai do nome e não muda |
| PUT | `/api/checklist-items/:id` | admin, central | Troca `label`, `position` e `active` (ausente mantém) |
| GET | `/api/users` | admin | Usuários do painel (ativos primeiro) |
| POST | `/api/users` | admin | Cria (`name`, `email`, `password` com 10+ caracteres, `role`) |
| PUT | `/api/users/:id` | admin | Troca `name`, `role` e `active` (ausente mantém). 409 se o admin tentar se desativar ou deixar de ser admin |
| POST | `/api/users/:id/password` | admin | Redefine a senha de outro usuário (`password`) |
| POST | `/api/auth/password` | todos | Troca a própria senha (`current_password`, `new_password`) |
| GET | `/api/partners?kind=` | admin, central | Parceiros (ativos primeiro); `kind` = proprietario, motorista ou indicador |
| GET | `/api/partners/:id` | admin, central | Um parceiro |
| GET | `/api/partners/code/:code` | admin, central | Parceiro pelo código de indicação |
| POST | `/api/partners` | admin, central | Cria (`name`, `kinds[]`, contato, percentuais 0–100 ou nulo); gera `referral_code` |
| PUT | `/api/partners/:id` | admin, central | Substitui os dados (o código não muda); `active` ausente mantém |
| GET | `/api/payouts?kind=&status=&partner_id=` | admin, central | Comissões (`comissao`) e repasses (`repasse`); `status` pendente ou paga |
| POST | `/api/payouts/:id/pay` | admin, central | Marca como pago (data e quem marcou). 409 se já pago ou sem percentual |

Erros de reserva: 400 validação, 404 não encontrado, 409 conflito (período ocupado, manutenção, conferência pendente, etapa final ou já alterada, veículo aguardando o checklist de outra reserva ou ainda em locação em outra); 400 também para hora no futuro e devolução antes da entrega.

Em "devolvida", as conferências exigidas são `issues` mais as chaves dos itens ativos do checklist. Uma reserva não vai para "em locação" enquanto o mesmo veículo tiver outra em "devolvida" (RN-01.6).

---

## Modelo de dados

```
users
  id UUID PK DEFAULT gen_random_uuid()
  name, email, password_hash TEXT NOT NULL
  role TEXT CHECK (role IN ('admin','central'))
  active BOOL DEFAULT true              -- V8: desativado não entra
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
      deposit_cents, km_limit_per_day, extra_km_cents   -- V7, opcionais; V10: limite pode vir sem valor (excedente a definir), valor exige limite
  owner_partner_id FK partners NULL     -- V9: parceiro proprietário (RN-07.4)

partner_payouts (V11)                   -- RN-03.12, RN-03.13; gerados ao encerrar a reserva
  id, booking_id FK, partner_id FK, kind CHECK (comissao|repasse), UNIQUE (booking_id, kind)
  base_cents                            -- comissao: diárias; repasse: diárias + atraso + km
  percent_bp, amount_cents NULL         -- cópia do parceiro no encerramento; nulo = a definir (copiado no pagamento)
  paid_at, paid_by FK users, created_at

partners (V9)                           -- RN-07
  id, name, is_owner, is_driver, is_referrer BOOL (ao menos um)
  whatsapp (dígitos), email, city, notes TEXT
  referral_code TEXT UNIQUE ^[A-Z0-9]{6,12}$ (não muda)
  commission_bp, owner_share_bp, driver_share_bp INT 0..10000 NULL   -- pontos-base, por parceiro
  active BOOL, created_at, updated_at


bookings
  id, vehicle_id FK, client_name, start_at TIMESTAMPTZ, end_at TIMESTAMPTZ, place   -- V5 (RN-02.23)
  stage CHECK (preparacao|confirmada|em_locacao|devolvida|encerrada|cancelada)
  days = blocos de 24h, para cima       -- RN-02.24
  daily_rate_cents, total_cents         -- cópia da diária na criação (RN-02.6)
  checks JSONB                          -- conferências: cnh, terms, deposit, payment, delivery, return, issues
    delivered_at, returned_at TIMESTAMPTZ  -- hora real da entrega e da devolução (V6)
  deposit_cents, km_limit_per_day, extra_km_cents   -- cópia do veículo na criação (V7); limite nulo = km livre
  delivered_km, returned_km, excess_km, excess_km_cents   -- odômetro e km excedente (RN-02.18)
            extra_minutes, extra_cents            -- atraso por minuto, diária ÷ 1440 (RN-02.19, V10); total_cents não muda
  referred_by_partner_id FK partners NULL   -- V9: quem indicou (RN-07.5)

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
