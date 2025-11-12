# Llamero Server (MVP)

This repository currently exposes the minimal server that issues JWTs after a successful OAuth flow. The steps below walk through the local setup so you can generate keys, define roles, create an environment file, and start the server.

## 1. Generate Signing Keys

```bash
# Ed25519 (preferred)
openssl genpkey -algorithm ed25519 -out secrets/jwt_private.pem
openssl pkey -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem

# or RS256 if your environment doesn’t support EdDSA
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 -out secrets/jwt_private.pem
openssl rsa -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem
```

Place the private/public keys in a `secrets/` directory or any secure path you reference in the `.env`.

## 2. Define Roles and Scopes

`config/roles.yaml` ships with the default `admin` and `user` roles. Update the scopes if needed, but keep the canonical role names—environment variables use them when mapping IdP groups to permissions.

```yaml
default_role: user
roles:
  - name: admin
    scopes:
      - backends:list
      - backends:listModels
      - backends:ps
      - backends:createModel
      - backends:pullModel
      - backends:pushModel
      - backends:deleteModel
      - llm:chat
      - llm:embeddings
      - profile:get
  - name: user
    scopes:
      - llm:chat
      - llm:embeddings
      - profile:get
```

## 3. Create `.env`

Copy the snippet below into `.env` (or export the variables another way). Adjust URLs and IDs to match your OAuth provider.

```bash
LLAMERO_SERVER_ADDRESS=:8080
LLAMERO_SERVER_EXTERNAL_URL=http://localhost:8080

LLAMERO_OAUTH_PROVIDER_NAME=authentik
LLAMERO_OAUTH_CLIENT_ID=example-client
LLAMERO_OAUTH_CLIENT_SECRET=example-secret
LLAMERO_OAUTH_AUTHORIZE_URL=https://idp.example.com/application/o/authorize/
LLAMERO_OAUTH_TOKEN_URL=https://idp.example.com/application/o/token/
LLAMERO_OAUTH_USERINFO_URL=https://idp.example.com/application/o/userinfo/
LLAMERO_OAUTH_REDIRECT_URL=http://localhost:8080/auth/callback
LLAMERO_OAUTH_SCOPES=openid,email,profile

LLAMERO_JWT_ISSUER=llamero
LLAMERO_JWT_AUDIENCE=ollama-clients
LLAMERO_JWT_SIGNING_METHOD=EdDSA        # or RS256
LLAMERO_JWT_PRIVATE_KEY_PATH=secrets/jwt_private.pem
LLAMERO_JWT_PUBLIC_KEY_PATH=secrets/jwt_public.pem
LLAMERO_JWT_TTL=1h

LLAMERO_POSTGRES_HOST=localhost
LLAMERO_POSTGRES_PORT=5432
LLAMERO_POSTGRES_USER=llamero
LLAMERO_POSTGRES_PASSWORD=changeme
LLAMERO_POSTGRES_DBNAME=llamero
LLAMERO_POSTGRES_SSLMODE=disable
LLAMERO_MIGRATIONS_DIR=data/sql/migrations

LLAMERO_REDIS_ADDR=redis:6379
LLAMERO_REDIS_USERNAME=
LLAMERO_REDIS_PASSWORD=
LLAMERO_REDIS_DB=0

LLAMERO_BACKENDS_FILE=config/backends.yaml

# Map IdP groups to internal roles defined in config/roles.yaml
# admin=AdminsGroup;user=EveryoneGroup
LLAMERO_ROLE_GROUPS=admin=admins;user=users
```

Source the file before running anything:

```bash
source .env

# Run migrations (optional if you trust the server to migrate on boot)
export LLAMERO_DATABASE_URL="postgres://$LLAMERO_POSTGRES_USER:$LLAMERO_POSTGRES_PASSWORD@$LLAMERO_POSTGRES_HOST:$LLAMERO_POSTGRES_PORT/$LLAMERO_POSTGRES_DBNAME?sslmode=$LLAMERO_POSTGRES_SSLMODE"
go run github.com/pressly/goose/v3/cmd/goose -dir data/sql/migrations postgres "$LLAMERO_DATABASE_URL" up
```

## 4. Start the Server

With Go 1.21+ installed:

```bash
go run ./cmd/server
```

The server listens on `LLAMERO_SERVER_ADDRESS` (default `:8080`). Visit `http://localhost:8080/healthz` to confirm it’s up, hit `/auth/login` to begin the OAuth flow, and inspect `/auth/me` (requires a valid token with at least the `profile:get` scope).
