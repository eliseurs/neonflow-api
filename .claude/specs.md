# Specs — Changelog de Alterações

---

## Em andamento — Ciclo de reserva persistente

Plano: `docs/plano-reserva-persistente.md`. **Bloqueado** até resolver PEN-01 a PEN-06 (regras de locação), PEN-35/PEN-36 (donos no Nexus/Replit), PEN-48/PEN-49 (LGPD e isolamento) e PEN-52 (autorização).

Já decidido: o período fica bloqueado desde a criação, sem expiração (RN-02.1, ADR-0006). A stack está definida (ADR-0009).

Próximos domínios: `fleet` (veículos, RN-01) e `booking` (reservas, RN-02), com a constraint `EXCLUDE` descrita em `design.md`.

---

## [2026-11-03] — Migrations no start do container

O primeiro deploy subiu sem tabelas: login respondia 500 (`relation "users" does not exist`). Causa: serviços Railway novos ignoram o `railway.json` (config-as-code descontinuado), então o `preDeployCommand: ./migrate up` nunca rodou. Correção: `CMD ["sh","-c","./migrate up && exec ./api"]` no Dockerfile; `preDeployCommand` removido do `railway.json` e o `command` duplicado do `docker-compose.yml`. Healthcheck do Railway fica pendente de configuração no painel.

## [2026-11-03] — Publicação no Railway

- Serviço `neonflow-api` criado a partir de `eliseurs/neonflow-api`, com plugin `Postgres` (`DATABASE_URL` por referência), `APP_ENV=production`, `JWT_SECRET` gerado, `ALLOWED_ORIGINS` = frontend. URL: https://neonflow-api-production.up.railway.app.
- Frontend em https://neonflow-app-production.up.railway.app; demo em https://neonflow-prototipo-production.up.railway.app.

## [2026-11-03] — Scaffold da API e separação em dois repositórios

Repositório criado (`eliseurs/neonflow-api`), separado do frontend (`eliseurs/neonflow-app`), seguindo cognia-api (ADR-0009) e o formato `.claude/` (ADR-0010).

- **Infra**: `internal/config` (defaults locais, validação de produção), `internal/infra/api` (Gin, `Routes`, `Respond`, healthcheck com ping do banco, graceful shutdown), `internal/infra/db`, `internal/infra/migrator` (V{N}, transação por migration, valida sequência).
- **pkg**: `security` (JWT HS256 com issuer, expiração obrigatória e só HS256; bcrypt), `middleware` (auth, role guard, CORS por allowlist), `logger` (slog; não loga query string nem corpo).
- **Domínio `authentication`**: `POST /api/auth/login`, `GET /api/auth/me`; mesma mensagem para e-mail inexistente e senha errada, com comparação bcrypt fictícia para igualar o tempo (RN-05.5).
- **CLIs**: `cmd/migrate` (up|down|status), `cmd/create-admin` (lê `ADMIN_*` do ambiente para a senha não aparecer no `ps`).
- **Migration V1**: `users` com `CHECK` de papel e índice único em `lower(email)`.
- **Deploy**: Dockerfile multi-stage (usuário não-root, healthcheck), `railway.json`.
- **Local**: `docker-compose.yml` (PostgreSQL 16 + API) e `devops/initdb` cria `neonflow_test`.
- **Testes**: unitários em todos os pacotes de `internal/` (cobertura 70,3% só com unitários). Integração para repository, migrator e db com build tag `integration` — **ainda não executados** (sem banco local nesta rodada).
- **Docs**: regras de negócio migradas da wiki (`RN-0001…0009` → `RN-01.1…04.1`) para `.claude/business-rules/`; ADRs, pendências, produto, integrações, LGPD, fontes e plano em `docs/`. ADR-0003 e ADR-0006 aceitos; ADR-0009 e ADR-0010 criados; ADR-0007 substituído. PEN-40 resolvida, PEN-04 parcial.

## [2026-11-03] — Demo no Railway (repositório neonflow-app)

Protótipo publicado em https://neonflow-app-production.up.railway.app (serviço `neonflow-app`, projeto Railway `nexus-os`), com `server.mjs` sem dependências. Basic Auth foi ativado e depois removido a pedido do time: o acesso é público.

## [2026-11-03] — Wiki docs-as-code e pendências

Levantamento de 52 pendências (15 bloqueantes) cruzando o código do protótipo com o dossiê de 09/10. Primeira wiki (ADR-0007), depois migrada para este formato.

## [2026-10-09] — Pacote de consolidação v1 (Drive)

Dossiê A–M, 12 testes de auditoria e manifesto com hashes. Fonte: `docs/fontes.md`.

## [2026-09-29] — Protótipo Sites v2 (`fdf114ef`)

Capítulos 13–16 (revisão estratégica) e área de estratégia.

## [2026-09-27] — Protótipo Sites v1 (`b53b31f0`)

Primeira versão do site e Base funcional v1.1.
