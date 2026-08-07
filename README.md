# GFireUI Backend — BFF for the GFire ops console

<a id="readme-top"></a>

**🔐** _Users, roles, audit, and a thin door into GFire._

[![Version](https://img.shields.io/badge/version-0.1.0-blue)](./VERSION)
[![Go](https://img.shields.io/badge/Go-service-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)
[![Status](https://img.shields.io/badge/status-design-yellow)](#current-status)
[![Companion](https://img.shields.io/badge/UI-gfireui-FF3E00)](https://github.com/hrodrig/gfireui)

**Repo:** [github.com/hrodrig/gfireui-backend](https://github.com/hrodrig/gfireui-backend) · **UI:** [gfireui](https://github.com/hrodrig/gfireui) · **Engine:** [gfire](https://github.com/hrodrig/gfire) · **Design:** [platform design](./docs/superpowers/specs/2026-08-06-gfireui-platform-design.md) · **Site:** [gfire.net](https://gfire.net)

<p align="center">
  <img src="docs/assets/gfireui-backend-hero.png" alt="GFireUI Backend — BFF, auth, RBAC, audit" width="100%" />
</p>

```
                 ┌─────────────┐
                 │  GFireUI    │  SvelteKit SPA
                 └──────┬──────┘
                        │ JWT (human)
                        ▼
              ┌─────────────────────┐
              │  gfireui-backend    │  ◄── you are here
              │  auth · RBAC · audit│
              │  thin GFire proxy   │
              └─────────┬───────────┘
                   │    │
          PG gfireui    │ service Bearer
                        ▼
                 ┌─────────────┐
                 │    gfire    │  headless
                 └─────────────┘
```

This is the **Backend-for-Frontend** for [GFireUI](https://github.com/hrodrig/gfireui). Humans authenticate here. Permissions are decided here. Every peek at jobs, queues, recurring definitions, and servers goes **through** this service to [GFire](https://github.com/hrodrig/gfire). The browser never sees GFire’s URL or service token.

> **Status: design approved, implementation starting.** v0.1.0 — platform design locked; no runnable binary yet.

**Related tools (same maintainer):**
- **[gfire](https://github.com/hrodrig/gfire)** — standalone background job service ([gfire.net](https://gfire.net))
- **[gfireui](https://github.com/hrodrig/gfireui)** — ops console (SvelteKit)
- **[pgwd](https://github.com/hrodrig/pgwd)** — PostgreSQL connection watchdog
- **[gghstats](https://github.com/hrodrig/gghstats)** — GitHub traffic beyond 14 days
- **[kzero](https://github.com/hrodrig/kzero)** — bastion-first declarative workload reset
- **[groot](https://github.com/hrodrig/groot)** — Kubernetes diagnostics archive

## Table of contents

- [What a BFF is (here)](#what-a-bff-is-here)
- [Responsibilities](#responsibilities)
- [API surface (v0.1 target)](#api-surface-v01-target)
- [Roles](#roles)
- [Data model highlights](#data-model-highlights)
- [Bootstrap](#bootstrap)
- [Current status](#current-status)
- [Repository layout](#repository-layout)
- [Development](#development)
- [Docs](#docs)
- [License](#license)

[↑ Back to top](#readme-top)

## What a BFF is (here)

**BFF** = *Backend For Frontend*: an API shaped for the console, not a second job engine.

| Concern | Owner |
| ------- | ----- |
| Job execution, storage backends, handlers | **gfire** |
| Login, JWT, users, RBAC, audit | **gfireui-backend** |
| Screens and charts | **gfireui** |

Thin proxy: `/api/gfire/*` ≈ GFire `/v1/*`, with RBAC on every hop.

[↑ Back to top](#readme-top)

## Responsibilities

- **Auth** — email/password, argon2id, JWT (browser-first; API clients later). **OAuth2 / OIDC** planned post-v0.1 (IdP login; backend still mints GFireUI JWTs + roles)
- **Users** — UUIDv7, first/last name, email, role, enable/disable
- **RBAC** — Administrator / Operator / Auditor / Guest
- **Audit log** — append-only events (login, user changes, proxied mutations)
- **GFire client** — service Bearer from config; never exposed to the browser
- **Ops summary** — aggregate endpoint(s) for dashboard charts (UI polls 2–5s)
- **Optional** — serve the built SPA static files in production

[↑ Back to top](#readme-top)

## API surface (v0.1 target)

| Prefix | Purpose |
| ------ | ------- |
| `/api/auth/*` | Login, session/token helpers |
| `/api/users/*` | User admin (Administrator) |
| `/api/audit/*` | Audit feed (Administrator, Auditor) |
| `/api/gfire/*` | Thin proxy to GFire REST |
| `/api/ops/summary` | Chart-friendly snapshot |

Exact routes land in `SPECIFICATIONS.md` during implementation.

[↑ Back to top](#readme-top)

## Roles

| Role | Ops data | Mutations | Users | Audit |
| ---- | -------- | --------- | ----- | ----- |
| Administrator | ✓ | ✓ | ✓ | ✓ |
| Operator | ✓ | ✓ | ✗ | ✗ |
| Auditor | ✓ | ✗ | ✗ | ✓ |
| Guest | ✗ | ✗ | ✗ | ✗ |

[↑ Back to top](#readme-top)

## Data model highlights

**PostgreSQL** database `gfireui` (own DSN — may share a cluster with GFire, never job tables).

- `users` — uuidv7, names, email, role, enabled, password_hash, timestamps  
- `audit_events` — uuidv7, actor, action, resource, payload JSONB, created_at  

Signing key + GFire Bearer: **env/config only** in v0.1.

[↑ Back to top](#readme-top)

## Bootstrap

1. **First boot (env):** `GFIREUI_ADMIN_USER` / `GFIREUI_ADMIN_PASSWORD` (+ name fields as specified) creates an Administrator when the DB is empty.  
2. **CLI:** `gfireui-backend user create --role Administrator` for ongoing ops.

[↑ Back to top](#readme-top)

## Current status

| Item | State |
| ---- | ----- |
| Platform design | ✅ Approved 2026-08-06 |
| Go module / `cmd` | ⬜ |
| Migrations + auth | ⬜ |
| GFire proxy + RBAC | ⬜ |
| Docker Compose | ⬜ |
| First runnable release | ⬜ |

[↑ Back to top](#readme-top)

## Repository layout

```
gfireui-backend/
├── README.md
├── VERSION
├── LICENSE
└── docs/
    ├── assets/
    │   └── gfireui-backend-hero.png
    └── superpowers/specs/
        └── 2026-08-06-gfireui-platform-design.md
```

`cmd/`, `internal/`, migrations, and Makefile appear with scaffolding.

[↑ Back to top](#readme-top)

## Development

Not runnable yet. After scaffold (expected shape):

```sh
make build
make run            # or: go run ./cmd/gfireui-backend
# GFIREUI_* and GFire base URL / token via config + env
```

Requires PostgreSQL (`gfireui` database) and a reachable **[gfire](https://github.com/hrodrig/gfire)** API.

[↑ Back to top](#readme-top)

## Docs

| Document | Role |
| -------- | ---- |
| [Platform design](./docs/superpowers/specs/2026-08-06-gfireui-platform-design.md) | Approved architecture & API intent |
| [gfireui](https://github.com/hrodrig/gfireui) | Console frontend |
| [GFire SPEC](https://github.com/hrodrig/gfire/blob/main/SPECIFICATIONS.md) | Engine behavior |

`SPECIFICATIONS.md` / `ROADMAP.md` land with the first implementation band.

[↑ Back to top](#readme-top)

## License

[MIT](./LICENSE) © Hermes Rodriguez

[↑ Back to top](#readme-top)
