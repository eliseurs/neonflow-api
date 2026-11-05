# Rules — Neonflow API

> Leia este arquivo antes de qualquer alteração de código.
> Após cada alteração, atualize `specs.md` com o que foi mudado.

---

## Regras Gerais

1. **Nunca salte hooks de git** (`--no-verify`) sem autorização explícita.
2. **Nunca faça `git push --force` para main/master** sem confirmação do usuário.
3. **Commits somente quando solicitado** — nunca comitar espontaneamente.
4. **Não adicione dependências externas** sem aprovação — verifique `go.mod` antes.
5. **Não refatore código não relacionado** à tarefa em andamento.
6. **Não adicione comentários, docstrings ou type annotations** em código que não foi alterado.

---

## Regras de Negócio

1. **Não invente** tarifa, percentual, prazo, multa, comissão ou termo. Sem fonte, registre em `docs/pendencias.md`.
2. Toda regra nova vai para `.claude/business-rules/NN-tema.md` como `RN-NN.x`, com status (`aprovada` / `implementada` / `protótipo` / `proposta`), fonte e "onde é aplicada".
3. Ao implementar uma regra, preencha "Onde é aplicada" com `arquivo:linha` e mude o status para `implementada`.
4. Números históricos (R$400, 15%, 20%/15%, SLA 60s, R$1.500/dia) **não são vigentes** (PEN-43).
5. Valores comerciais (tarifa, caução) são **dados cadastrados pela central**, nunca constantes no código.
6. Decisão estrutural → ADR em `docs/adr/` (template em `docs/adr/template.md`).
7. Antes de criar serviço para identidade, agenda, documentos ou pagamentos, verifique se o Nexus já tem um dono (ADR-0004, PEN-35).
8. Ao resolver uma pendência, atualize o status dela em `docs/pendencias.md`.

---

## Estrutura de domínio

- Todo domínio novo segue: `entity.go` / `contracts.go` / `errors.go` / `repository/` / `service/` / `handler/` / `route/`
- Interfaces em `contracts.go` — repository, service e logger sempre como interfaces
- Erros de domínio em `errors.go` com `var Err... = errors.New(...)`, mensagem em português
- Registrar `route.Register` em `cmd/api/main.go`

## Banco de dados

- **Nunca use ORM** — SQL puro via `database/sql` + `pgx/v5/stdlib`
- Parâmetros posicionais (`$1`, `$2`), **nunca** interpolação de string
- `NULL`: `sql.NullString` / `sql.NullTime`
- `RETURNING` com todos os campos que o Go precisa
- Valores monetários em **centavos** (`BIGINT`, sufixo `_cents`), nunca `float`
- Datas de reserva como `DATE`; instantes como `TIMESTAMPTZ`
- Violação de unicidade (`23505`) vira erro de domínio no repository

## UUIDs

- **Nunca use `github.com/google/uuid`** — gere no banco (`gen_random_uuid()`)

## Context

- Handlers: `c.Request.Context()`
- `context.Background()` só para trabalho que precisa sobreviver ao cancelamento da request

## Concorrência

- Disponibilidade (RN-02.1, RN-02.2) é garantida **no banco** (constraint `EXCLUDE` ou lock), nunca só no Go (PEN-24)

---

## Regras de Migração

- **Formato**: `V{N}_descricao.up.sql` / `V{N}_descricao.down.sql` (descrição em `snake_case` minúsculo)
- **N** sequencial, nunca pule números — o migrator recusa sequência quebrada
- Ups: `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`
- Downs: `DROP TABLE IF EXISTS ... CASCADE`
- Nunca altere uma migration já aplicada — crie uma nova
- Criar: `make migrate-create name=descricao`

---

## Regras de Segurança e LGPD

- Nunca logar senhas, tokens, query string, corpo de request ou dados pessoais (nome, e-mail, CNH, telefone)
- `JWT_SECRET` só via env var; fora do ambiente `local` (hml, production) a API não sobe sem ele
- Login responde a mesma mensagem para e-mail inexistente e senha errada (RN-05.5)
- Nunca commitar `.env` — só `.env.example`
- Dados de exemplo e de teste são **sempre fictícios** (`@neonflow.test`)
- CNH e documentos só entram depois de PEN-48 (base legal, retenção)

---

## Regras de Teste

- Toda feature, alteração ou código novo tem teste, ou ajuste no teste existente
- Meta: **80% de cobertura** em `internal/` (`make cover`)
- Fakes manuais das interfaces de `contracts.go` (sem framework de mock)
- Repository e migrator: testes com build tag `integration` + `TEST_DATABASE_URL` apontando para banco **descartável** (os testes recriam o schema)
