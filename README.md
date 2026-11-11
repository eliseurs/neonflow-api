# Neonflow API

Backend do **Neon Flow**, a vertical de aluguel de veículos, frota, reservas e transfer do Nexus OS (Serra Gaúcha).

Go 1.25 · Gin · PostgreSQL 16 · JWT · Railway. Frontend: [`eliseurs/neonflow-app`](https://github.com/eliseurs/neonflow-app).

## Rodar local

```bash
make up                 # PostgreSQL + API (aplica migrations) em http://localhost:8080
make create-admin ADMIN_EMAIL=central@neonflow.test ADMIN_NAME="Central" ADMIN_PASSWORD='uma-senha-longa'
curl localhost:8080/api/health
```

Sem Docker para a API: `make db-up && make migrate-up && make dev`.

## Endpoints

| Método | Rota | Auth |
|---|---|---|
| GET | `/api/health` | — |
| POST | `/api/auth/login` | — |
| GET | `/api/auth/me` | JWT |

## Testes

```bash
make lint test          # gofmt, go vet e unitários
make cover              # cobertura de internal/ (meta 80%)
TEST_DATABASE_URL=postgres://neonflow:neonflow@localhost:5432/neonflow_test?sslmode=disable make test-integration
```

## Estado

Frota, reservas, transfer, clientes, parceiros, repasses e contratos implementados e operados pela central; o estado detalhado está em [.claude/CLAUDE.md](.claude/CLAUDE.md). O que falta decidir e fazer: [docs/decisoes.md](docs/decisoes.md) e [docs/pendencias.md](docs/pendencias.md).

## Documentação

- Para agentes e devs: [`CLAUDE.md`](CLAUDE.md) e [`.claude/`](.claude/CLAUDE.md) (regras, design, estilo, changelog)
- Regras de negócio: [`.claude/business-rules/`](.claude/business-rules/README.md)
- ADRs, produto, deploy: [`docs/`](docs/README.md)
