# GFireUI Backend — Specifications (v0.1)

Behavior contract for `gfireui-backend`. Design source: [platform design](./docs/superpowers/specs/2026-08-06-gfireui-platform-design.md).

## Role

Go BFF for [GFireUI](https://github.com/hrodrig/gfireui). Owns human auth, RBAC, audit, and a thin proxy to [GFire](https://github.com/hrodrig/gfire). The browser never receives the GFire service Bearer.

## HTTP surface

| Method | Path | Auth | Roles | Notes |
| ------ | ---- | ---- | ----- | ----- |
| GET | `/healthz` | no | — | `{"status":"ok"}` |
| POST | `/api/auth/login` | no | — | `{email,password}` → `{token,user}` |
| GET | `/api/auth/me` | JWT | any enabled | current user |
| GET/POST | `/api/users` | JWT | Administrator | list / create |
| GET/PATCH | `/api/users/{id}` | JWT | Administrator | get / update |
| POST | `/api/users/{id}/password` | JWT | Administrator | set password |
| GET | `/api/audit` | JWT | Administrator, Auditor | paginated audit |
| * | `/api/gfire/{path...}` | JWT | see RBAC | thin proxy to GFire |
| GET | `/api/ops/summary` | JWT | Admin, Operator, Auditor | chart snapshot |

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

YAML + env prefix `GFIREUI_` (dots → underscores). See `gfireui-backend.example.yaml`.

## Non-goals (v0.1)

- Embedding in `gfire` binary  
- Job secret vault  
- SSE/WebSocket  
- Kubernetes/Helm charts (future selfhosted sibling)  
- Serving the SPA (optional later)
