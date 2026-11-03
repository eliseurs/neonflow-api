# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api          ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate      ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/create-admin ./cmd/create-admin

# ── Stage 2: runtime ──────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/api          .
COPY --from=builder /out/migrate      .
COPY --from=builder /out/create-admin .
COPY --from=builder /app/sql          ./sql

USER app

ENV APP_ENV=production PORT=8080 TZ=America/Sao_Paulo
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:${PORT}/api/health || exit 1

# Aplica migrations pendentes antes de subir (idempotente, uma transação por migration).
# Não depende do preDeployCommand: serviços Railway novos ignoram railway.json.
CMD ["sh", "-c", "./migrate up && exec ./api"]
