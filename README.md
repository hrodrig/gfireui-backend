# GFireUI Backend — BFF for the GFire ops console

<a id="readme-top"></a>

**🔐** _Users, roles, audit, and a thin door into GFire._

[![Version](https://img.shields.io/badge/version-0.1.0-blue)](./VERSION)
[![Go](https://img.shields.io/badge/Go-service-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)
[![Status](https://img.shields.io/badge/status-v0.1.0-brightgreen)](#current-status)
[![GHCR](https://img.shields.io/badge/image-ghcr.io%2Fhrodrig%2Fgfireui--backend-2496ED?logo=github)](https://github.com/hrodrig/gfireui-backend/pkgs/container/gfireui-backend)
[![Companion](https://img.shields.io/badge/UI-gfireui-FF3E00)](https://github.com/hrodrig/gfireui)

**Repo:** [github.com/hrodrig/gfireui-backend](https://github.com/hrodrig/gfireui-backend) · **UI:** [gfireui](https://github.com/hrodrig/gfireui) · **Engine:** [gfire](https://github.com/hrodrig/gfire) · **Design:** [platform design](./docs/superpowers/specs/2026-08-06-gfireui-platform-design.md) · **Security:** [SECURITY.md](./SECURITY.md) · **Site:** [gfire.net](https://gfire.net)

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

> **Status: v0.1.0 packaging ready.** Auth, users, audit, thin GFire proxy, ops summary. CI fail-closed (fmt/vet/gocyclo/test/cover≥80%). Production image target: `ghcr.io/hrodrig/gfireui-backend` via GoReleaser on tag `v*`.

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

1. **First boot (env):** when `users` is empty, set `GFIREUI_BACKEND_BOOTSTRAP_ADMIN_EMAIL`, `GFIREUI_BACKEND_BOOTSTRAP_ADMIN_PASSWORD`, `GFIREUI_BACKEND_BOOTSTRAP_ADMIN_FIRST_NAME`, `GFIREUI_BACKEND_BOOTSTRAP_ADMIN_LAST_NAME`.  
2. **CLI:** `gfireui-backend user create --email … --password … --role Administrator --first-name … --last-name …`

[↑ Back to top](#readme-top)

## Current status

| Item | State |
| ---- | ----- |
| Platform design | ✅ Approved 2026-08-06 |
| Go module / `cmd` | ✅ |
| Migrations + auth + users + audit | ✅ |
| GFire proxy + RBAC + ops summary | ✅ |
| Docker Compose (app stack) | ✅ |
| CI (fmt/vet/gocyclo/test/cover≥80%/docker) | ✅ |
| GoReleaser + release.yml (GHCR multi-arch, SBOM, cosign) | ✅ |
| SECURITY.md | ✅ |
| Kubernetes / selfhosted sibling | ⬜ post-v0.1 (separate repo) |
| Merge to `main` / first tag `v0.1.0` | ✅ |

[↑ Back to top](#readme-top)

## Repository layout

```
gfireui-backend/
├── README.md
├── VERSION
├── LICENSE
├── SPECIFICATIONS.md
├── ROADMAP.md
├── Dockerfile              # local/CI multi-stage build
├── Dockerfile.release      # GoReleaser runtime image (binary pre-built)
├── docker-compose.yml
├── cmd/gfireui-backend/
├── internal/
├── migrations/
└── docs/
    ├── assets/
    │   └── gfireui-backend-hero.png
    └── superpowers/
```

[↑ Back to top](#readme-top)

## Development

**Quality gate (gghstats-style Makefile):**

```sh
make lint          # gofmt -s + go vet
make gocyclo       # complexity ≤ 15
make test
make cover         # Postgres via compose; fail if total < COVER_MIN_PERCENT (default 80)
make docker-smoke  # local image; curl /healthz
make release-check # full pre-tag gate (must be green before any release)
```

CI runs the same quality bar on every PR and push to `develop`/`main`. **No release if cover (or fmt/gocyclo/test) is red.**

### Release quality (fail-closed)

- Tag `v*` only from `main` after merging `develop`.
- Local bar: `make release-check` (lint, test, cover ≥80%, govulncheck, gocyclo, grype, docker-scan, `goreleaser check`).
- Tag workflow re-runs `make release-check` **before** GoReleaser publishes binaries/GHCR (SBOM + cosign).
- Red gate = no image, no GitHub Release assets.
- Contract detail: [OCI / CI / release design](./docs/superpowers/specs/2026-08-08-gfireui-backend-oci-ci-release-design.md).

**Compose (recommended):**

```sh
# Cooks image gfireui-backend:<VERSION>, then starts postgres + migrate + backend
make compose-up
curl -sS http://127.0.0.1:8090/healthz

# With upstream GFire (optional):
export GFIREUI_BACKEND_GFIRE_BASE_URL=http://host.docker.internal:8080
export GFIREUI_BACKEND_GFIRE_TOKEN=your-gfire-bearer   # omit if GFire auth disabled
make compose-up
```

`make compose-up` = `make docker-build` (VERSION/COMMIT/BUILDDATE ldflags) then `docker compose up` using image `gfireui-backend:<VERSION>` (no in-compose Go rebuild).

**Local binary:**

```sh
make test
make build
export GFIREUI_BACKEND_DATABASE_DSN='postgres://gfireui:gfireui@127.0.0.1:5433/gfireui?sslmode=disable'
export GFIREUI_BACKEND_AUTH_JWT_SECRET=dev-only
make migrate-up   # requires golang-migrate CLI
./gfireui-backend serve
```

PostgreSQL (`gfireui` database) required for auth/users/audit. Reachable **[gfire](https://github.com/hrodrig/gfire)** required only for proxy/ops routes.

**Images:** `Dockerfile` = compile-in-Docker for local/CI. `Dockerfile.release` = distroless runtime for GoReleaser (binary already built). Same split as sibling Go repos.

```sh
# After tag v0.1.0 publishes:
docker pull ghcr.io/hrodrig/gfireui-backend:v0.1.0
```

Ops packaging: [gfire-selfhosted](https://github.com/hrodrig/gfire-selfhosted) (`GFIREUI_BACKEND_VERSION`).

**Note:** Helm/k8s packaging is intentionally out of this repo — owned by the selfhosted sibling.

[↑ Back to top](#readme-top)

## Docs

| Document | Role |
| -------- | ---- |
| [Platform design](./docs/superpowers/specs/2026-08-06-gfireui-platform-design.md) | Approved architecture & API intent |
| [OCI / CI / release](./docs/superpowers/specs/2026-08-08-gfireui-backend-oci-ci-release-design.md) | GoReleaser, GHCR, fail-closed gates |
| [SPECIFICATIONS.md](./SPECIFICATIONS.md) | Behavior + image/quality contract |
| [ROADMAP.md](./ROADMAP.md) | Band status (B-040…) |
| [SECURITY.md](./SECURITY.md) | Vulnerability reporting (private advisories) |
| [CHANGELOG.md](./CHANGELOG.md) | Shipped notes |
| [gfireui](https://github.com/hrodrig/gfireui) | Console frontend |
| [GFire SPEC](https://github.com/hrodrig/gfire/blob/main/SPECIFICATIONS.md) | Engine behavior |

[↑ Back to top](#readme-top)

## License

[MIT](./LICENSE) © Hermes Rodriguez

[↑ Back to top](#readme-top)
