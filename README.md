# 🦙 Llamero

Llamero is a **secure load balancer and control plane** for Ollama clusters.

**Stack**: Go services (server, worker, scheduler), Next.js UI, Postgres, Redis, Ollama, Nginx.

**Related**:

- 🤖 Ollama – [ollama.ai](https://ollama.ai/) • [github.com/ollama/ollama](https://github.com/ollama/ollama)
- 📜 API docs (generated) – `docs/swagger.json` and `ui/lib/api`

## 🚀 Quick start (Docker Compose)

1. 🔑 Generate signing keys

```bash
mkdir -p secrets

# Ed25519 (preferred)
openssl genpkey -algorithm ed25519 -out secrets/jwt_private.pem
openssl pkey -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem

# or RS256
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 -out secrets/jwt_private.pem
openssl rsa -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem
```

2. 📝 Create `.env` with service settings

```bash
# Server (HTTP + OAuth/JWT)
LLAMERO_SERVER_ADDRESS=:8080
LLAMERO_SERVER_EXTERNAL_URL=http://localhost:8080

LLAMERO_OAUTH_PROVIDER_NAME=authentik             # your IdP name
LLAMERO_OAUTH_CLIENT_ID=change-me
LLAMERO_OAUTH_CLIENT_SECRET=change-me
LLAMERO_OAUTH_AUTHORIZE_URL=https://idp.example.com/application/o/authorize/
LLAMERO_OAUTH_TOKEN_URL=https://idp.example.com/application/o/token/
LLAMERO_OAUTH_USERINFO_URL=https://idp.example.com/application/o/userinfo/
LLAMERO_OAUTH_REDIRECT_URL=http://localhost:8080/auth/callback
LLAMERO_OAUTH_SCOPES=openid,email,profile

LLAMERO_JWT_ISSUER=llamero
LLAMERO_JWT_AUDIENCE=ollama-clients
LLAMERO_JWT_SIGNING_METHOD=EdDSA                 # or RS256
LLAMERO_JWT_PRIVATE_KEY_PATH=secrets/jwt_private.pem
LLAMERO_JWT_PUBLIC_KEY_PATH=secrets/jwt_public.pem
LLAMERO_JWT_TTL=1h

LLAMERO_ROLE_GROUPS=admin=admins;user=users       # maps IdP groups -> roles

# Database (server + worker)
LLAMERO_POSTGRES_HOST=postgres
LLAMERO_POSTGRES_PORT=5432
LLAMERO_POSTGRES_USER=llamero
LLAMERO_POSTGRES_PASSWORD=llamero
LLAMERO_POSTGRES_DBNAME=llamero
LLAMERO_POSTGRES_SSLMODE=disable
LLAMERO_MIGRATIONS_DIR=data/sql/migrations

# Redis + background jobs (server + worker + scheduler)
LLAMERO_REDIS_ADDR=redis:6379
LLAMERO_REDIS_USERNAME=
LLAMERO_REDIS_PASSWORD=
LLAMERO_REDIS_DB=0
LLAMERO_WORKER_CONCURRENCY=5       # worker only
LLAMERO_SCHEDULER_PING_SPEC=@every 5m # scheduler only

# Static backends (server)
LLAMERO_BACKENDS_FILE=config/backends.yaml
```

Worker and scheduler ignore the OAuth/JWT values above—they only need the Postgres/Redis/job settings. Defaults in `docker-compose.yml` wire up Postgres, Redis, Ollama, Nginx. Adjust `config/backends.yaml` (LLM endpoints) and `config/roles.yaml` (scope sets) if needed.

3. 🚀 Launch the stack

```bash
docker compose up --build
```

Visit `http://localhost:8080` (Nginx). Health: `/healthz`. OAuth start: `/auth/login`. Tokened info: `/auth/me`.

## 🧑‍💻 Local dev (without Docker)

Prereqs: Go 1.25+, Postgres, Redis available per your `.env`.

```bash
source .env
go run ./cmd/server
```

Migrations auto-run on startup from `data/sql/migrations`.
