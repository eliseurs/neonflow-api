# ADR-0009 — Stack e repositórios iguais aos projetos cognia/finder

- **Status:** aceita
- **Data:** 03/11/2026
- **Decisores:** time (pedido explícito: "mesma stack tecnológica que já utilizo nos meus outros projetos")
- **Relacionados:** PEN-40, [ADR-0003](ADR-0003-prototipo-estatico-e-evidencia.md), [ADR-0004](ADR-0004-reutilizar-capacidades-nexus.md)

## Contexto e problema

O Neon Flow só tinha um protótipo estático. Para começar o produto era preciso escolher a stack. O time já mantém o cognia (cognia-api + cognia-app, repositórios separados) e o finder-b2b-app com uma stack consistente.

## Decisão

Dois repositórios irmãos, como o cognia-project, com o código espelhando o cognia:

| Camada | Escolha | Referência |
|---|---|---|
| Backend | Go 1.25, Gin, `database/sql` + pgx v5, JWT v5, bcrypt, testify | cognia-api |
| Estrutura backend | DDD por domínio: `entity/contracts/errors/repository/service/handler/route` | cognia-api |
| Migrations | Runner Go puro, `sql/migrations/V{N}_*.up/down.sql` | cognia-api |
| Banco | PostgreSQL 16 | cognia, finder |
| Frontend | React 18, TypeScript, Vite 5, Tailwind 3, Zustand 5, React Query 5, axios, React Router 6, Vitest | cognia-app, finder-app |
| Servir o frontend | Caddy (SPA fallback, `$PORT`) | finder-app |
| Deploy | Railway, um serviço por pasta, `railway.json` com Dockerfile | finder, cognia |
| Local | `docker-compose` + `Makefile` por repositório | cognia-api, finder |

| Repositório | Pasta local | Conteúdo |
|---|---|---|
| `eliseurs/neonflow-api` | `nepo/neonflow-api` | Backend Go, regras de negócio, ADRs, pendências |
| `eliseurs/neonflow-app` | `nepo/neonflow-app` | Frontend React e `prototipo/` (demo congelada) |

## Consequências

- Boas: o mesmo modelo mental, os mesmos comandos e as mesmas regras do `.claude/` valem entre os projetos.
- Ruins: a decisão foi tomada antes do levantamento do Nexus/Replit (PEN-35, PEN-36). Se surgir um dono existente para identidade, agenda ou pagamentos, o Neon Flow integra com ele em vez de reimplementar ([ADR-0004](ADR-0004-reutilizar-capacidades-nexus.md)).
- O serviço Railway `neonflow-app` hoje serve o protótipo a partir da raiz do repo do frontend: o Root Directory precisa mudar para `/prototipo` (ver `docs/deploy-railway.md` no `neonflow-app`).
- O histórico git dos documentos movidos para este repositório fica no `neonflow-app` (commits até `6422f5f`).
