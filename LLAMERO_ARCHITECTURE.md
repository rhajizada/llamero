# Llamero Architecture

Llamero is a **secure, distributed load balancer and control plane** for Ollama clusters.  
It provides centralized user management, token-based authentication, intelligent request routing, 
and background orchestration for managing Ollama nodes. Llamero is designed for organizations 
deploying multiple Ollama instances and requiring unified authentication, role-based access control, 
and fine-grained permissioned API access.

---

## 1. System Overview

Llamero acts as a **reverse proxy and control plane** sitting in front of a cluster of Ollama nodes.

It serves two purposes:

1. **API Gateway** — Presents LLM (OpenAI API-compatible) endpoints to clients (`/v1/chat/completions`, `/v1/embeddings`, etc.)
   and proxies requests to the correct Ollama node, enforcing role- and scope-based authorization.
2. **Cluster Manager** — Periodically pings and syncs Ollama nodes to detect health, model availability, 
   and version changes. All node metadata is cached in Redis, refreshed via background Asynq jobs.

---

## 2. Components

| Component | Description |
|------------|-------------|
| **Server** | Main HTTP API and reverse proxy using `net/http` |
| **Worker** | Asynq worker processing node pings, syncs, and cleanup tasks |
| **Scheduler** | Asynq scheduler registering recurring maintenance tasks |
| **Redis** | Control-plane registry and Asynq backend |
| **PostgreSQL** | Persistent storage for PATs, users, audit logs |
| **UI** | Optional React-based dashboard for admin management |
| **Config File** | Defines static nodes, OAuth provider details, and runtime options |

All services are containerized and can run independently.

---

## 3. Authentication and Authorization

Llamero uses **OAuth2 + JWT** for all authentication.  
There are no sessions or cookies — every client authenticates via JWT in the `Authorization` header.

### 3.1 Token Types

| Token Type | Source | Persistence | Description |
|-------------|---------|-------------|--------------|
| **Session JWT** | Issued after OAuth2 login | Stateless | Used by human users |
| **PAT (Personal Access Token)** | Created manually by user | Stored + revocable | Used by automation, CI, or scripts |

---

### 3.2 OAuth2 Login Flow

1. User authenticates through the configured external OAuth2 provider (e.g., Authentik, Keycloak, Google).
2. Llamero receives OAuth `code` at `/auth/callback`.
3. Exchanges it for an `id_token` and `access_token`.
4. Validates token signature via provider JWKS endpoint (`/.well-known/openid-configuration`).
5. Extracts user info: `sub`, `email`, `roles`, and `groups`.
6. Validates role membership using environment-defined role mappings.
7. Exchanges external token for **internal Llamero JWT**, which encodes roles and scopes.

---

### 3.3 OAuth Configuration

Environment-driven configuration:

```yaml
OAUTH_CLIENT_ID: llamero-client
OAUTH_CLIENT_SECRET: <secret>
OAUTH_PROVIDER_URL: https://auth.company.com/application/o/llamero/
OAUTH_REDIRECT_URI: https://llamero.company.com/auth/callback
OAUTH_PROVIDER_NAME: authentik
OAUTH_USER_ROLE: llamero-users
OAUTH_ADMIN_ROLE: llamero-admins
OAUTH_VIEWER_ROLE: llamero-viewer
```

This ensures that only users belonging to approved groups in the OAuth provider can access Llamero.

---

### 3.4 Internal JWT Structure

Llamero issues internal tokens for both human and automation users.  

Example payload:

```json
{
  "iss": "llamero",
  "sub": "user-uuid",
  "email": "dev@company.com",
  "role": "admin",
  "scopes": ["generate", "models:pull", "models:push"],
  "type": "session",
  "jti": "uuid-token-id",
  "iat": 1736250000,
  "exp": 1736264400,
  "aud": "ollama-clients"
}
```

- Signed using Ed25519 or RS256
- Keys stored securely as:
  - `/secrets/jwt_private.pem`
  - `/secrets/jwt_public.pem`
- Public key available at `/auth/jwks.json`

---

### 3.5 Roles and Scopes

Roles map directly to permission scopes.  
Scopes represent fine-grained access rights for API endpoints.

```yaml
roles:
  admin:
    scopes:
      - admin
      - chat
      - generate
      - embed
      - models:*
      - models:pull
      - models:push
      - models:create
      - models:delete
      - models:copy
  maintainer:
    scopes:
      - chat
      - generate
      - embed
      - models:read
      - models:pull
  user:
    scopes:
      - chat
      - generate
      - embed
      - models:read
```

Each endpoint requires a minimum scope for access.  
Scopes can be wildcarded (`models:*` grants all model operations).

---

### 3.6 PAT Lifecycle

- PATs are long-lived internal JWTs stored in PostgreSQL.
- Users create them via `/api/tokens`.
- Each PAT includes explicit scopes and expiration.

Example request:

```json
{
  "name": "ci-runner",
  "scopes": ["generate", "models:read"],
  "expires_in": 2592000
}
```

Stored metadata:

```sql
CREATE TABLE tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_sub TEXT NOT NULL,
  name TEXT,
  scopes TEXT[],
  jti TEXT UNIQUE NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT now()
);
```

Only PATs are persisted; session JWTs are ephemeral.

---

## 4. Authorization Middleware

Requests flow through:

```
Authenticate → Verify JWT → Check PAT (if applicable) → Enforce Scopes → Handler
```

- Session JWTs validated via public key signature.
- PATs verified via signature + revocation lookup by `jti`.
- Routes declare required scopes, e.g.:
  ```go
  router.Handle("/v1/models/pull", RequireScopes("models:pull"), pullHandler)
  ```

---

## 5. Proxy and Routing

Llamero proxies LLM (OpenAI API-compatible) endpoints directly to healthy Ollama nodes.

### Flow

1. Parse incoming request, extract model name.
2. Query Redis registry for nodes serving that model.
3. Select a healthy node (round-robin or random).
4. Forward request using `net/http` client.
5. Stream response to client.

Non-LLM (non-OpenAI) endpoints (e.g., `/api/pull`, `/api/push`) are **not** automatically forwarded;  
they require explicit admin permissions and a `node` parameter to target a backend directly.

---

### Node Selection Strategy

Llamero uses a **registry-based routing model**.

Rules:
- Prefer nodes that already have the requested model loaded.
- Skip unhealthy or circuit-open nodes.
- Fallback to the next available node.
- Cache selected node locally for short duration (~10s).

---

## 6. Redis Registry Schema

Redis is the single source of truth for node runtime state.  

### Key Schema

| Key | Type | Purpose |
|------|------|----------|
| `llm:nodes` | `SET` | all node names |
| `llm:nodes:<name>` | `HASH` | node metadata (status, version, last_heartbeat) |
| `llm:nodes:<name>:models` | `SET` | models available on node |
| `llm:models:<model>` | `SET` | list of nodes serving model |
| `llm:tasks:last_ping:<node>` | `STRING` | timestamp of last ping |
| `llm:tasks:last_sync:<node>` | `STRING` | timestamp of last model sync |

### Example
```bash
> HGETALL llm:nodes:ollama-1
address             http://10.0.0.11:11434
status              healthy
version             0.5.4
last_heartbeat      1736342800
models_last_sync    1736342600
```

### TTL and Expiry

| Key | TTL | Behavior |
|------|-----|-----------|
| `llm:nodes:<node>` | 180s | expires if node stops reporting |
| `llm:nodes:<node>:models` | 300s | re-synced by worker |
| `llm:tasks:last_ping:<node>` | 60s | transient metrics |

Registry is self-healing — workers automatically rebuild it.

---

## 7. Worker Architecture (Asynq)

Llamero uses **Asynq** for all control-plane background work.

### Core Jobs

| Job | Frequency | Description |
|------|------------|-------------|
| `ping_nodes` | every 30s | pings `/api/version` for health |
| `sync_models` | every 5m | queries `/api/ps` for model list |
| `purge_stale_nodes` | every 10m | removes expired or stale nodes |
| `rebuild_model_index` | every 10m | recomputes model→node sets |

### Example Payload

```json
{
  "node": "ollama-2",
  "address": "http://10.0.0.12:11434"
}
```

### Ping Handler

```go
func HandlePing(ctx context.Context, task *asynq.Task) error {
  resp, err := http.Get(node.Address + "/api/version")
  if err != nil || resp.StatusCode != 200 {
    redis.MarkUnhealthy(node)
    return nil
  }
  redis.MarkHealthy(node)
  return nil
}
```

Workers continuously maintain consistency between actual nodes and the Redis registry.

---

## 8. Circuit Breaker

Each Llamero gateway tracks node runtime health locally.

### State Machine

| State | Meaning |
|--------|----------|
| **Closed** | Node healthy |
| **Open** | Node temporarily blocked |
| **Half-open** | Node being tested for recovery |

### Parameters
```
MaxFailures = 3
HalfOpenAfter = 15s
ResetTimeout = 30s
```

Circuit breakers are local to each process and reset automatically.  
When open, the proxy skips that node for 30 seconds.  
Redis-based pings eventually reconcile cluster-wide health.

### Optional Broadcast

Degradation can be broadcast via Redis Pub/Sub:

```
PUBLISH llm:events:node_degraded '{"node":"ollama-3"}'
```

All gateways subscribe and open their local circuits for that node.

---

## 9. Scheduler

A dedicated `scheduler` service registers recurring Asynq jobs:

| Task | Cron | Purpose |
|------|------|----------|
| `ping_nodes` | @every 30s | node health |
| `sync_models` | @every 5m | model state |
| `purge_stale_nodes` | @every 10m | cleanup |
| `rebuild_model_index` | @every 10m | refresh model registry |

---

## 10. Deployment Model

| Service | Function |
|----------|-----------|
| `llamero-server` | HTTP server and proxy |
| `llamero-worker` | Asynq worker process |
| `llamero-scheduler` | Recurring task scheduler |
| `postgres` | Persistent storage |
| `redis` | Task queue + node registry |

**Environment Variables**

```bash
DATABASE_URL=postgres://user:pass@pg:5432/llamero
REDIS_ADDR=redis:6379
JWT_PRIVATE_KEY_PATH=/secrets/private.pem
JWT_PUBLIC_KEY_PATH=/secrets/public.pem
OAUTH_PROVIDER_URL=https://auth.company.com/application/o/...
OAUTH_CLIENT_ID=llamero-client
OAUTH_CLIENT_SECRET=<secret>
OAUTH_ALLOWED_ROLES=llamero-users,llamero-admins
OAUTH_ADMIN_ROLES=llamero-admins
NODES_FILE=/etc/llamero/nodes.yaml
```

---

## 11. Reliability and Self-Healing

| Mechanism | Scope | Owner |
|------------|--------|-------|
| Circuit breaker | Per gateway | Local memory |
| Redis TTL expiry | Cluster | Redis workers |
| Asynq retries | Job | Worker |
| Periodic rebuilds | Global | Scheduler |

If Redis is flushed, the system auto-rebuilds state from node configuration and background pings.

---

## 12. Security Model

- **Zero trust per request**: JWT verified for every request.
- **Scoped access control**: Scopes enforce API-level permissions.
- **No persistent sessions**: Fully stateless.
- **OAuth2-based identity**: Delegated authentication.
- **PAT storage**: Only PAT metadata persisted (never private keys).
- **TLS enforced**: All communication HTTPS only.

---

## 13. Observability

| Metric | Description |
|--------|--------------|
| `llamero_node_failures_total` | Count of failed node proxy attempts |
| `llamero_node_latency_seconds` | Proxy response time |
| `llamero_requests_total` | Gateway request count |
| `llamero_tasks_processed` | Background tasks completed |

All metrics are exportable to Prometheus.

---

## 14. Future Extensions

| Area | Improvement |
|-------|--------------|
| **Usage quotas** | Track token-based usage per user |
| **Billing integration** | Chargeback for compute usage |
| **Node autoscaling** | Integrate with Kubernetes HPA |
| **Model replication** | Automatically distribute models |
| **Request deduplication** | Avoid redundant model pulls |
| **Advanced routing** | Latency-aware selection, sticky sessions |
| **Admin UI** | Manage users, nodes, and PATs visually |

---

## 15. Summary

Llamero provides:
- OAuth2 → JWT-based authentication
- Scoped role-based access control
- LLM (OpenAI API-compatible) reverse proxying
- Health and model registry via Redis
- Background orchestration via Asynq
- Circuit breaking and failure isolation
- Modular microservice deployment

Llamero is the control plane for secure, multi-tenant Ollama clusters.
