# RN-05 — Autenticação e Autorização

## Regras de Negócio

| ID | Regra | Status | Onde é aplicada | Pendências |
|---|---|---|---|---|
| RN-05.1 | Toda rota `/api/*`, exceto login e health, exige JWT válido (`Authorization: Bearer`) | implementada | `internal/pkg/middleware/auth.go` | — |
| RN-05.2 | Não há cadastro aberto: usuários da central são criados pela CLI `create-admin` | implementada | `cmd/create-admin` | PEN-14 |
| RN-05.3 | Papéis do token: `admin` e `central`. Perfis de cliente, motorista, captador e proprietário ainda não existem | implementada (só admin/central) | `internal/pkg/middleware/role_guard.go` | PEN-14, PEN-49 |
| RN-05.4 | Senhas guardadas com bcrypt. E-mail é único e comparado sem diferenciar maiúsculas | implementada | `internal/pkg/security/hasher.go`, `sql/migrations/V1_initial_schema.up.sql` | — |
| RN-05.5 | Falha de login responde a mesma mensagem para e-mail inexistente e senha errada | implementada | `internal/domain/authentication/service` | — |
| RN-05.6 | No início, **só o fundador** (como admin e atendente da central) usa o painel. Parceiros (hotéis, proprietários de veículos, indicadores) e motoristas ganham acesso depois | aprovada (06/11/2026, fundador, q78) | papéis `admin` e `central` já existem | PEN-14 |

**Fontes:** padrão dos projetos cognia/finder; decisão do scaffold em 03/11/2026; questionário do fundador em 06/11/2026 (RN-05.6). A matriz de perfis dos parceiros e do cliente (RN-04.6) depende de PEN-14.

## Fluxo de login

```mermaid
sequenceDiagram
    participant SPA as neonflow-app
    participant API as neonflow-api
    participant DB as PostgreSQL
    SPA->>API: POST /api/auth/login {email, password}
    API->>DB: SELECT user by lower(email)
    DB-->>API: user + password_hash
    API->>API: bcrypt.Compare
    API-->>SPA: 200 {status, data: {token, user}}
    SPA->>API: GET /api/auth/me (Bearer token)
    API-->>SPA: 200 {status, data: user}
```
