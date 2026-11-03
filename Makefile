.PHONY: help setup db-up db-down up down logs dev build test test-integration cover lint migrate-up migrate-down migrate-status migrate-create create-admin psql

# GOARCH da máquina: o ambiente global pode ter GOARCH=amd64 (cross-compile).
HOST_ARCH := $(shell go env GOHOSTARCH)
GOTEST    := GOARCH=$(HOST_ARCH) go test
GORUN     := GOARCH=$(HOST_ARCH) go run
COVER_OUT := cover.out

help: ## Mostra este menu de ajuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ─── Setup e infraestrutura local ─────────────────────────────────────────────
setup: ## Baixa dependências Go
	@go mod download

db-up: ## Sobe só o PostgreSQL (docker compose)
	@docker compose up -d db

db-down: ## Derruba o PostgreSQL
	@docker compose stop db

up: ## Sobe PostgreSQL + API via docker compose
	@docker compose up -d --build
	@echo "API: http://localhost:8080/api/health"

down: ## Derruba tudo
	@docker compose down

logs: ## Logs da API
	@docker compose logs -f api

psql: ## Abre psql no container do banco
	@docker compose exec db psql -U neonflow -d neonflow

# ─── Desenvolvimento ──────────────────────────────────────────────────────────
dev: ## Roda a API local (defaults de settings.go, banco do docker compose)
	@$(GORUN) ./cmd/api

build: ## Compila api, migrate e create-admin em bin/
	@go build -trimpath -o bin/ ./cmd/...

lint: ## gofmt + go vet
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; echo "rode: gofmt -w cmd internal"; exit 1)
	@GOARCH=$(HOST_ARCH) go vet ./... && GOARCH=$(HOST_ARCH) go vet -tags integration ./...

# ─── Testes ───────────────────────────────────────────────────────────────────
test: ## Testes unitários
	@$(GOTEST) ./... -count=1

test-integration: ## Unitários + integração (requer TEST_DATABASE_URL, banco descartável)
	@test -n "$$TEST_DATABASE_URL" || (echo "defina TEST_DATABASE_URL (ex.: make db-up e use o banco neonflow_test)"; exit 1)
	@$(GOTEST) -tags integration ./... -count=1 -p 1

cover: ## Cobertura de internal/ (meta: 80%). Com TEST_DATABASE_URL inclui integração
	@$(GOTEST) $$(test -n "$$TEST_DATABASE_URL" && echo "-tags integration -p 1") ./... -count=1 -coverpkg=./internal/... -coverprofile=$(COVER_OUT) >/dev/null
	@go tool cover -func=$(COVER_OUT) | tail -1

# ─── Migrations ───────────────────────────────────────────────────────────────
migrate-up: ## Aplica migrations pendentes
	@$(GORUN) ./cmd/migrate up

migrate-down: ## Desfaz a última migration
	@$(GORUN) ./cmd/migrate down

migrate-status: ## Lista status das migrations
	@$(GORUN) ./cmd/migrate status

migrate-create: ## Cria V{N+1}_<name>.up/down.sql (make migrate-create name=foo)
	@test -n "$(name)" || (echo "uso: make migrate-create name=descricao"; exit 1)
	@n=$$(ls sql/migrations | sed -nE 's/^V([0-9]+)_.*\.up\.sql$$/\1/p' | sort -n | tail -1); n=$$(( $${n:-0} + 1 )); \
	touch sql/migrations/V$${n}_$(name).up.sql sql/migrations/V$${n}_$(name).down.sql; \
	echo "criado sql/migrations/V$${n}_$(name).{up,down}.sql"

create-admin: ## Cria usuário (ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... [ADMIN_ROLE=central])
	@$(GORUN) ./cmd/create-admin
