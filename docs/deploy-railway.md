# Deploy da API no Railway

Projeto Railway: **neonflow** (`58da0777-4fc0-4891-98bd-554ca078557e`), ambiente `hml` (renomeado de `production` em 05/11/2026; os domínios passaram para `*-hml.up.railway.app`, menos o do protótipo).

Serviço `neonflow-api`, ligado a `eliseurs/neonflow-api` (`main`): https://neonflow-api-hml.up.railway.app. Cada push na `main` publica. Banco: plugin `Postgres` no mesmo projeto.

## Como foi criado (03/11/2026)

1. Railway → projeto **nexus-os** → **New → Database → PostgreSQL**.
2. **New → GitHub repo → `eliseurs/neonflow-api`**. Nome do serviço: `neonflow-api`.
3. Variáveis do serviço:

| Variável | Valor |
|---|---|
| `APP_ENV` | `hml` (qualquer valor diferente de `local` exige `JWT_SECRET` e `DATABASE_URL`) |
| `DATABASE_URL` | referência: `${{Postgres.DATABASE_URL}}` |
| `JWT_SECRET` | `openssl rand -base64 48` — nunca commitar |
| `JWT_EXPIRY_HOURS` | `12` |
| `ALLOWED_ORIGINS` | `https://neonflow-app-hml.up.railway.app` |
| `STORAGE_ENDPOINT`, `STORAGE_REGION`, `STORAGE_BUCKET`, `STORAGE_ACCESS_KEY_ID`, `STORAGE_SECRET_ACCESS_KEY` | bucket de arquivos (ADR-0014). Em hml: bucket `neonflow-arquivos` do Railway (06/11/2026), com as credenciais de `railway bucket credentials --bucket neonflow-arquivos --json` gravadas por `railway variables --set-from-stdin`, sem passar pelo terminal. Para o Cloudflare R2, trocar as cinco. Sem elas, o envio de fotos responde 503 |

4. **Settings → Networking → Generate Domain**.
5. Deploy. O build usa o `Dockerfile`, e o container roda `./migrate up && ./api` ao iniciar.

> **Serviços Railway criados depois da descontinuação do config-as-code ignoram o `railway.json`** (`fileServiceManifest` vazio). Por isso as migrations rodam no `CMD` da imagem, e não no `preDeployCommand`. O healthcheck (`/api/health`) precisa ser configurado no painel (Settings → Deploy → Healthcheck Path) ou via `.railway/railway.ts`.
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
