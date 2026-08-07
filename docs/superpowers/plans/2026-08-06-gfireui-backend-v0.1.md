# GFireUI Backend v0.1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a runnable Go BFF that authenticates humans (JWT), enforces RBAC, writes an audit log, thinly proxies GFire, and exposes an ops summary for charts.

**Architecture:** Single binary `gfireui-backend`. PostgreSQL database `gfireui` owns users + audit. Browser never talks to GFire; service Bearer lives only in backend config. Thin proxy preserves GFire JSON shapes under `/api/gfire/*`.

**Tech Stack:** Go 1.26.5, `net/http`, `pgx`/`golang-migrate`, `golang-jwt/jwt/v5`, `argon2id` (via `golang.org/x/crypto/argon2` or `alexedwards/argon2id`), `google/uuid` (v7), YAML+env config (viper or hand-rolled matching gfire habits), testify.

**Spec:** [2026-08-06-gfireui-platform-design.md](../specs/2026-08-06-gfireui-platform-design.md)  
**Companion UI plan:** [gfireui plan](https://github.com/hrodrig/gfireui/blob/develop/docs/superpowers/plans/2026-08-06-gfireui-v0.1.md) (implement after Task 8+ of this plan is usable).

## Global Constraints

- English only in code, comments, commits, docs.
- Work on `develop`; no commits to `main` until release flow.
- Whitelist `.gitignore` — add new root files to whitelist when introduced.
- No CBPI / employer product names in this repo.
- v0.1 auth = local email/password + JWT only (OAuth2/OIDC is post-v0.1 per design §5).
- Roles exactly: `Administrator` | `Operator` | `Auditor` | `Guest`.
- User `id` = UUIDv7; required fields: first_name, last_name, email, role, enabled.
- Passwords: argon2id; never log plaintext or hashes.
- Audit: append-only; Admin+Auditor read; no secrets in `payload`.
- Browser never receives GFire service Bearer.
- TDD: failing test → implement → pass → commit per task.
- Module path: `github.com/hrodrig/gfireui-backend`.

## File map (target)

```
cmd/gfireui-backend/main.go
internal/config/config.go
internal/domain/user.go
internal/domain/role.go
internal/domain/audit.go
internal/auth/password.go
internal/auth/jwt.go
internal/auth/context.go
internal/store/postgres/postgres.go
internal/store/postgres/users.go
internal/store/postgres/audit.go
internal/api/server.go
internal/api/middleware_auth.go
internal/api/middleware_rbac.go
internal/api/handlers_auth.go
internal/api/handlers_users.go
internal/api/handlers_audit.go
internal/api/handlers_ops.go
internal/api/proxy_gfire.go
internal/gfire/client.go
internal/bootstrap/admin.go
migrations/000001_init.up.sql
migrations/000001_init.down.sql
gfireui-backend.example.yaml
Makefile
docker-compose.yml
Dockerfile
SPECIFICATIONS.md
ROADMAP.md
AGENTS.md
```

---

### Task 1: Go module scaffold + healthz

**Files:**
- Create: `go.mod`, `Makefile`, `cmd/gfireui-backend/main.go`, `internal/api/server.go`, `internal/api/server_test.go`, `AGENTS.md`
- Modify: `.gitignore` (ensure `go.mod` / `go.sum` / `cmd/` already whitelisted)
- Modify: `README.md` Current status row for scaffold when done

**Interfaces:**
- Produces: `api.NewServer() http.Handler` with `GET /healthz` → `{"status":"ok"}`

- [ ] **Step 1: Init module**

```bash
cd /Volumes/Data/addlink/github/gfireui-backend
go mod init github.com/hrodrig/gfireui-backend
```

Expected: `go.mod` with `go 1.26.5`.

- [ ] **Step 2: Write failing health test**

```go
// internal/api/server_test.go
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/api"
)

func TestHealthz(t *testing.T) {
	srv := api.NewServer(api.Deps{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body=%v", body)
	}
}
```

- [ ] **Step 3: Run test — expect fail**

```bash
go test ./internal/api/ -count=1
```

Expected: FAIL — package or `NewServer` missing.

- [ ] **Step 4: Minimal server + main**

```go
// internal/api/server.go
package api

import (
	"encoding/json"
	"net/http"
)

type Deps struct{}

type Server struct {
	mux *http.ServeMux
}

func NewServer(_ Deps) http.Handler {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	return s.mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

```go
// cmd/gfireui-backend/main.go
package main

import (
	"log"
	"net/http"

	"github.com/hrodrig/gfireui-backend/internal/api"
)

func main() {
	addr := ":8090"
	log.Printf("gfireui-backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, api.NewServer(api.Deps{})))
}
```

```makefile
# Makefile (minimal)
.PHONY: test build run
test:
	go test ./... -count=1
build:
	go build -o bin/gfireui-backend ./cmd/gfireui-backend
run: build
	./bin/gfireui-backend
```

- [ ] **Step 5: Run tests — expect pass**

```bash
make test
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum Makefile cmd/ internal/api/ AGENTS.md README.md
git commit -m "$(cat <<'EOF'
feat: scaffold Go server with /healthz

EOF
)"
```

(Show commit message to user first if commit-message-review rule applies.)

---

### Task 2: Config load (YAML + env)

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`, `gfireui-backend.example.yaml`
- Modify: `cmd/gfireui-backend/main.go` to load config and bind addr
- Modify: `.gitignore` — already has `!gfireui-backend.example.yaml`

**Interfaces:**
- Produces: `config.Load(path string) (*Config, error)`
- `Config` fields (minimum): `Server.Addr`, `Database.DSN`, `Auth.JWTSecret`, `Auth.TokenTTL`, `GFire.BaseURL`, `GFire.Token`, `Bootstrap.AdminEmail`, `Bootstrap.AdminPassword`, `Bootstrap.AdminFirstName`, `Bootstrap.AdminLastName`

- [ ] **Step 1: Failing test — defaults / env override**

```go
func TestLoad_EnvOverridesAddr(t *testing.T) {
	t.Setenv("GFIREUI_BACKEND_SERVER_ADDR", "127.0.0.1:9090")
	cfg, err := config.Load("") // empty path = defaults only + env
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Addr != "127.0.0.1:9090" {
		t.Fatalf("got %q", cfg.Server.Addr)
	}
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test ./internal/config/ -count=1
```

- [ ] **Step 3: Implement Load** using viper (`GFIREUI_BACKEND` env prefix, `.` → `_`) or equivalent; default addr `:8090`.

- [ ] **Step 4: Example YAML** documenting every key used in v0.1.

- [ ] **Step 5: Tests pass + commit**

```bash
git commit -m "feat: load YAML/env configuration"
```

---

### Task 3: Domain types + migrations

**Files:**
- Create: `internal/domain/role.go`, `internal/domain/user.go`, `internal/domain/audit.go`
- Create: `migrations/000001_init.up.sql`, `migrations/000001_init.down.sql`
- Create: `internal/domain/role_test.go`

**Interfaces:**
- Produces:
  - `type Role string` with constants + `func (r Role) Valid() bool`
  - `type User struct { ID uuid.UUID; FirstName, LastName, Email string; Role Role; Enabled bool; PasswordHash string; CreatedAt, UpdatedAt time.Time }`
  - `type AuditEvent struct { ... }` per design §6

- [ ] **Step 1: Failing test for role validation**

```go
func TestRoleValid(t *testing.T) {
	if !domain.RoleAdministrator.Valid() {
		t.Fatal("Administrator should be valid")
	}
	if domain.Role("SecretsManager").Valid() {
		t.Fatal("unknown role must be invalid")
	}
}
```

- [ ] **Step 2: Implement roles + structs**

- [ ] **Step 3: SQL migration**

```sql
-- migrations/000001_init.up.sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  email TEXT NOT NULL UNIQUE,
  role TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
  id UUID PRIMARY KEY,
  actor_user_id UUID NULL REFERENCES users(id),
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NULL,
  ip TEXT NULL,
  user_agent TEXT NULL,
  payload JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_events_created_at_idx ON audit_events (created_at DESC);
```

- [ ] **Step 4: Makefile targets `migrate-up` / `migrate-down`** (golang-migrate CLI or embed).

- [ ] **Step 5: Commit**

```bash
git commit -m "feat: domain types and initial Postgres migrations"
```

---

### Task 4: Postgres store — users CRUD

**Files:**
- Create: `internal/store/postgres/postgres.go`, `users.go`, `users_test.go`
- Test: integration with `GFIREUI_BACKEND_TEST_DSN` or dockertest; skip if unset

**Interfaces:**
- Produces:
  - `func Open(ctx, dsn) (*Store, error)`
  - `func (s *Store) CreateUser(ctx, u *domain.User) error`
  - `func (s *Store) GetUserByEmail(ctx, email) (*domain.User, error)`
  - `func (s *Store) GetUserByID(ctx, id) (*domain.User, error)`
  - `func (s *Store) ListUsers(ctx) ([]domain.User, error)`
  - `func (s *Store) UpdateUser(ctx, u *domain.User) error`
- Sentinel: `store.ErrNotFound`

- [ ] **Step 1: Failing integration test** CreateUser + GetByEmail (UUIDv7 via `uuid.NewV7()`).

- [ ] **Step 2: Implement with pgx.**

- [ ] **Step 3: `make test` (unit) + optional integration.**

- [ ] **Step 4: Commit** `feat: postgres user store`

---

### Task 5: Password hashing + JWT

**Files:**
- Create: `internal/auth/password.go`, `password_test.go`, `jwt.go`, `jwt_test.go`, `context.go`

**Interfaces:**
- Produces:
  - `func HashPassword(plain string) (string, error)`
  - `func CheckPassword(hash, plain string) bool`
  - `type Claims struct { Sub uuid.UUID; Email string; Role domain.Role; jwt.RegisteredClaims }`
  - `func IssueToken(secret []byte, ttl time.Duration, u *domain.User) (string, error)`
  - `func ParseToken(secret []byte, token string) (*Claims, error)`
  - `func WithUser(ctx, *domain.User) context.Context` / `UserFromContext(ctx) (*domain.User, bool)`

- [ ] **Step 1: Failing tests** — hash round-trip; issue+parse claims.

- [ ] **Step 2: Implement argon2id + HS256 JWT.**

- [ ] **Step 3: Commit** `feat: argon2id passwords and JWT helpers`

---

### Task 6: Auth HTTP — login + me + middleware

**Files:**
- Create: `internal/api/handlers_auth.go`, `handlers_auth_test.go`, `middleware_auth.go`
- Modify: `internal/api/server.go` wire routes + Deps (`Store`, `JWTSecret`, `TokenTTL`, `Audit`)

**Interfaces:**
- Produces:
  - `POST /api/auth/login` body `{email,password}` → `{token, user}` (user without password_hash)
  - `GET /api/auth/me` requires Bearer → user JSON
  - Middleware: missing/invalid token → 401; disabled user → 403

- [ ] **Step 1: Table-driven handler tests** with fake store or pg test DSN.

- [ ] **Step 2: Implement handlers + `Authorization: Bearer` middleware.**

- [ ] **Step 3: On login success/fail write audit** (can stub AuditWriter interface until Task 7).

- [ ] **Step 4: Commit** `feat: login and JWT auth middleware`

---

### Task 7: Audit store + list API

**Files:**
- Create: `internal/store/postgres/audit.go`, `internal/api/handlers_audit.go`, tests
- Modify: login/user mutation paths to call `Audit.Write`

**Interfaces:**
- Produces:
  - `func (s *Store) WriteAudit(ctx, e *domain.AuditEvent) error`
  - `func (s *Store) ListAudit(ctx, limit, offset int) ([]domain.AuditEvent, error)`
  - `GET /api/audit` — Administrator + Auditor only

- [ ] **Step 1: Failing tests** for write+list and RBAC denial for Operator.

- [ ] **Step 2: Implement.**

- [ ] **Step 3: Commit** `feat: append-only audit log API`

---

### Task 8: RBAC middleware + users admin API

**Files:**
- Create: `internal/api/middleware_rbac.go`, `handlers_users.go`, tests
- Modify: `server.go`

**Interfaces:**
- Produces:
  - `func RequireRoles(roles ...domain.Role) func(http.Handler) http.Handler`
  - `GET/POST /api/users`, `GET/PATCH /api/users/{id}` — Administrator only
  - PATCH can update names, role, enabled; password change optional separate endpoint `POST /api/users/{id}/password`

**RBAC matrix (enforce):**

| Route class | Roles |
| ----------- | ----- |
| `/api/users*` | Administrator |
| `/api/audit*` | Administrator, Auditor |
| GFire mutating proxy | Administrator, Operator |
| GFire read proxy | Administrator, Operator, Auditor |
| Guest | auth/me only |

- [ ] **Step 1: Failing RBAC tests** per matrix row.

- [ ] **Step 2: Implement.**

- [ ] **Step 3: Commit** `feat: user admin API and RBAC middleware`

---

### Task 9: GFire HTTP client + thin proxy

**Files:**
- Create: `internal/gfire/client.go`, `client_test.go` (httptest.Server fake GFire)
- Create: `internal/api/proxy_gfire.go`, `proxy_gfire_test.go`
- Modify: config already has BaseURL + Token

**Interfaces:**
- Produces:
  - `gfire.Client` with `Do(ctx, method, path string, body io.Reader) (*http.Response, error)` attaching `Authorization: Bearer <service>`
  - Proxy: `/api/gfire/{rest...}` → `{BaseURL}/{rest...}` forwarding method, query, body; copy response status + JSON body
  - Strip hop-by-hop headers; never forward browser `Authorization` to GFire (replace with service token)
  - RBAC: GET/HEAD → Admin|Operator|Auditor; POST/PUT/PATCH/DELETE → Admin|Operator
  - Audit mutating proxy calls (`action=gfire.proxy`, resource from path)

- [ ] **Step 1: Fake GFire server tests** — assert service Bearer used, not user JWT.

- [ ] **Step 2: Implement client + reverse proxy handler.**

- [ ] **Step 3: Commit** `feat: thin GFire reverse proxy with RBAC`

---

### Task 10: Ops summary for charts

**Files:**
- Create: `internal/api/handlers_ops.go`, tests
- Modify: `gfire.Client` helpers to list queues / count jobs if needed

**Interfaces:**
- Produces: `GET /api/ops/summary` → JSON snapshot e.g.

```json
{
  "jobs_by_state": {"pending": 0, "processing": 0, "succeeded": 0, "failed": 0, "dead": 0},
  "queues": [{"name": "default", "depth": 0}],
  "generated_at": "2026-08-06T00:00:00Z"
}
```

- Roles: Administrator, Operator, Auditor
- Implementation may call GFire `/v1/queues` and `/v1/jobs?state=` (best-effort aggregation; document limits in SPEC)

- [ ] **Step 1: Test against fake GFire.**

- [ ] **Step 2: Implement.**

- [ ] **Step 3: Commit** `feat: ops summary endpoint for dashboard charts`

---

### Task 11: Bootstrap admin (env + CLI)

**Files:**
- Create: `internal/bootstrap/admin.go`, `cmd/gfireui-backend/main.go` subcommands OR `cobra` root
- Prefer simple: `gfireui-backend serve` and `gfireui-backend user create --email --password --role --first-name --last-name`

**Interfaces:**
- On `serve` start: if users table empty AND bootstrap env set → create Administrator
- CLI create always available for ops

- [ ] **Step 1: Test bootstrap creates only when empty.**

- [ ] **Step 2: Implement.**

- [ ] **Step 3: Commit** `feat: admin bootstrap via env and CLI`

---

### Task 12: Docker Compose + Dockerfile + docs sync

**Files:**
- Create: `Dockerfile`, `docker-compose.yml`, `SPECIFICATIONS.md`, `ROADMAP.md`
- Modify: `README.md` status table → scaffold/auth/proxy rows as done

**Compose services:** `postgres` (db `gfireui`), `gfireui-backend` (depends on migrate). Document external GFire URL.

- [ ] **Step 1: `docker compose up` smoke** — healthz + login with bootstrap admin.

- [ ] **Step 2: Write SPEC from design (behavior contract).**

- [ ] **Step 3: ROADMAP bands for v0.1 leftover + post-v0.1 OAuth2 + theme packs note (UI-owned).**

- [ ] **Step 4: Commit** `docs: SPEC ROADMAP and compose for v0.1 backend`

---

## Spec coverage checklist (self-review)

| Design section | Tasks |
| -------------- | ----- |
| §2 Architecture BFF | 1, 9 |
| §3 Roles matrix | 8, 9 |
| §4 User fields UUIDv7 | 3, 4 |
| §5 JWT + bootstrap; OAuth2 deferred | 5, 6, 11 (OAuth2 not in this plan) |
| §6 Audit | 7 |
| §7 Postgres gfireui | 3, 4 |
| §8 Thin proxy | 9 |
| §11 Ops summary / poll source | 10 |
| §13 Go service | 1–12 |
| §14 Non-goals respected | no vault, no SSE, no OAuth2 impl |

## Out of this plan

- SvelteKit UI (separate plan in `gfireui`)
- OAuth2/OIDC
- Theme packs
- Serving SPA static from Go (optional stretch after UI build exists)

---

## Execution handoff

Plan saved to `docs/superpowers/plans/2026-08-06-gfireui-backend-v0.1.md`.

**UI plan:** write/execute `gfireui` companion plan after Task 8 (users+RBAC) so login can be developed against a real API; proxy pages need Task 9+.
