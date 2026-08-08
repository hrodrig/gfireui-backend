# GFireUI Backend — Specifications (v0.1)

Behavior contract for `gfireui-backend`. Design source: [platform design](./docs/superpowers/specs/2026-08-06-gfireui-platform-design.md).

## Role

Go BFF for [GFireUI](https://github.com/hrodrig/gfireui). Owns human auth, RBAC, audit, and a thin proxy to [GFire](https://github.com/hrodrig/gfire). The browser never receives the GFire service Bearer.

## HTTP surface

| Method | Path | Auth | Roles | Notes |
| ------ | ---- | ---- | ----- | ----- |
| GET | `/healthz` | no | — | `{"status":"ok","version":"...","commit":"..."}` (ldflags; may be empty in local builds) |
| POST | `/api/auth/login` | no | — | `{email,password}` → `{token,user}` |
| GET | `/api/auth/me` | JWT | any enabled | current user |
| GET/POST | `/api/users` | JWT | Administrator | list / create |
| GET/PATCH | `/api/users/{id}` | JWT | Administrator | get / update; cannot change own `enabled` |
| POST | `/api/users/{id}/password` | JWT | Administrator | set password |
| GET | `/api/audit` | JWT | Administrator, Auditor | paginated audit |
| * | `/api/gfire/{path...}` | JWT | see RBAC | thin proxy to GFire |
| GET | `/api/ops/summary` | JWT | Admin, Operator, Auditor | chart snapshot: canonical state counts, `servers_count`, `recurring_count`, `queues`, `versions[]` (BFF + upstream gfire), `generated_at` |

### Proxy RBAC

- `GET` / `HEAD` → Administrator, Operator, Auditor  
- Mutating methods → Administrator, Operator  
- Guest → denied (except login / me)

### User JSON

Never includes `password_hash`. Fields: `id` (UUIDv7), `first_name`, `last_name`, `email`, `role`, `enabled`.

### Roles

`Administrator` | `Operator` | `Auditor` | `Guest`

## Auth

- Passwords: argon2id  
- Sessions: JWT HS256 (`auth.jwt_secret`, `auth.token_ttl`)  
- Disabled users: 403  
- Bootstrap: if `users` empty and `bootstrap.admin_*` set → create Administrator on `serve`  
- CLI: `gfireui-backend user create --email --password --role --first-name --last-name`  
- OAuth2/OIDC: **not** in v0.1 (planned post-v0.1)

## Storage

PostgreSQL database `gfireui` (own DSN). Tables: `users`, `audit_events` (see `migrations/`).

## Config

YAML + env prefix `GFIREUI_BACKEND_` (dots → underscores). See `gfireui-backend.example.yaml`.

Family convention: `GFIRE_*` (engine), `GFIREUI_BACKEND_*` (this BFF), `PUBLIC_GFIREUI_*` (SPA browser-public).

- `gfire.base_url` empty → process stays up; `/api/gfire/*` and `/api/ops/summary` fail until set  
- `gfire.token` optional → empty means no `Authorization` header (GFire auth disabled / local)  
- `server.cors_allowed_origins` — comma-separated browser Origins for CORS; empty = no CORS headers; compose defaults `http://127.0.0.1:5173,http://localhost:5173`  
- Startup logs (gfire-style): one-line banner on stderr, then structured `slog` `starting` / `listening`

## Quality

`make cover` starts compose Postgres, runs `go test ./...` with `GFIREUI_BACKEND_TEST_DSN`, and fails if total statement coverage is below `COVER_MIN_PERCENT` (default **80**, same floor as gghstats).

CI (`.github/workflows/ci.yml`) on every PR and push to `develop`/`main` fail-closed on:

1. `gofmt -s`  
2. `go vet`  
3. `gocyclo -over 15`  
4. `go test -race`  
5. cover ≥ **80%** (Postgres service)  
6. `docker build` of `Dockerfile` (no push)

`make release-check` (lint, test, cover, security, docker-scan, `goreleaser check`) must pass before tagging. The release workflow re-runs those gates and **does not publish** if they fail.

## OCI image

| Item | Contract |
|------|----------|
| Local/CI image | `Dockerfile` → distroless `static-debian13:nonroot`, listen **8090**, `CMD ["serve"]` |
| Release image | GoReleaser `dockers_v2` + `Dockerfile.release`; **`ghcr.io/hrodrig/gfireui-backend`** tags `vX.Y.Z` + `latest` |
| Arches | **linux/amd64** and **linux/arm64** |
| Health | `GET /healthz` → `status` + optional `version`/`commit` (no auth) |
| Supply chain | syft SBOM + cosign keyless on release (`id-token: write`) |
| Smoke | `make docker-smoke` curls `/healthz` (process smoke; no Postgres required) |

## Non-goals (v0.1)

- Embedding in `gfire` binary  
- Job secret vault  
- SSE/WebSocket  
- Kubernetes/Helm charts (future selfhosted sibling)  
- Serving the SPA (optional later)
