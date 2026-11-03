# Deploy da API no Railway

Projeto Railway: **nexus-os** (`58da0777-4fc0-4891-98bd-554ca078557e`), ambiente `production`. A API **ainda não foi publicada**.

## Criar (uma vez)

1. Railway → projeto **nexus-os** → **New → Database → PostgreSQL**.
2. **New → GitHub repo → `eliseurs/neonflow-api`**. Nome do serviço: `neonflow-api`.
3. Variáveis do serviço:

| Variável | Valor |
|---|---|
| `APP_ENV` | `production` |
| `DATABASE_URL` | referência: `${{Postgres.DATABASE_URL}}` |
| `JWT_SECRET` | `openssl rand -base64 48` — nunca commitar |
| `JWT_EXPIRY_HOURS` | `12` |
| `ALLOWED_ORIGINS` | domínio do frontend (ex.: `https://neonflow-web-production.up.railway.app`) |

4. **Settings → Networking → Generate Domain**.
5. Deploy. O `railway.json` faz: build pelo `Dockerfile`, `./migrate up` no pre-deploy, healthcheck em `/api/health`.
6. Primeiro usuário:

```bash
railway run --service neonflow-api sh -c 'ADMIN_EMAIL=... ADMIN_NAME=... ADMIN_PASSWORD=... ./create-admin'
```

Sem `JWT_SECRET` ou `DATABASE_URL`, a API recusa subir em produção (`config.Validate`).

## Verificar

- `GET /api/health` → `200` com `"database":"ok"`.
- `POST /api/auth/login` com o usuário criado → `200` com `token`.
- `railway logs --service neonflow-api`: logs em JSON, sem dados pessoais.

## Rollback

**Deployments → deploy anterior → Redeploy.** Migrations não são revertidas automaticamente: se precisar, `railway run --service neonflow-api ./migrate down` (desfaz uma por vez).

## Prazo

`railway.json` (config as code) foi descontinuado pelo Railway e para de funcionar em **01/12/2026**. Migrar com `railway config migrate`.
