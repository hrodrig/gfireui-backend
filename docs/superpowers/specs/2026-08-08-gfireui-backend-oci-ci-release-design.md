# GFireUI Backend — OCI Image, CI, and Release Design

**Date:** 2026-08-08  
**Status:** Approved  
**Repo:** [hrodrig/gfireui-backend](https://github.com/hrodrig/gfireui-backend)  
**Consumers:** [gfire-selfhosted](https://github.com/hrodrig/gfire-selfhosted) console stack (`gfireui-backend` → container **:8090**); [gfireui](https://github.com/hrodrig/gfireui) SPA via `PUBLIC_GFIREUI_API_BASE`

## 1. Goal

Ship a **production-quality** BFF image and CI/release pipeline so operators can pull **`ghcr.io/hrodrig/gfireui-backend`** with confidence: distroless nonroot, multi-arch, supply-chain (SBOM + cosign), and **hard quality gates that never unlock a release when they fail**.

**Packaging:** GoReleaser (same strategy as [gfire](https://github.com/hrodrig/gfire)) — binaries + `dockers_v2` + SBOM + `docker_signs`.

**Image arches (first tag):** **linux/amd64** and **linux/arm64**.

**Learned from gfireui:** design → plan → implement; SECURITY.md early; fail-closed cover; tag only after `main` is green; pin `GFIREUI_BACKEND_VERSION=v0.1.0` in selfhosted once GHCR exists.

## 2. Non-goals

- Changing auth/RBAC/API contracts
- Serving the SPA from this binary
- Traefik / TLS / observability sidecars
- Raising the coverage floor above **80%** in this band
- Helm charts (owned by gfire-selfhosted)

## 3. Quality gates — early, constant, fail-closed

### 3.1 Principle (non-negotiable)

Quality is **not** a release-day checklist. Gates run from **the first CI landing** and on **every** push/PR to `develop` and `main`.

| Rule | Meaning |
|------|---------|
| Fail closed | Red `fmt` / `vet` / `gocyclo` / `test` / `cover` → merge blocked; **release must not run** (or must abort before publish) |
| Same bar local + CI | `make lint` / `make gocyclo` / `make test` / `make cover` / `make release-check` match what CI enforces |
| Cover is a gate | Statement coverage **&lt; `COVER_MIN_PERCENT` (default 80)** fails the job — no exceptions for “we’ll fix after tag” |
| Release re-runs gates | Tag workflow **must** re-execute the full quality suite **before** GoReleaser publishes; GoReleaser never runs on a red tree |

If coverage (or fmt/gocyclo/tests) is not green, **there is no release** — do not tag, do not push GHCR, do not attach assets.

### 3.2 What already exists (reuse)

| Piece | Status |
|-------|--------|
| `Dockerfile` / `Dockerfile.release` | Distroless `static-debian13:nonroot`, listen **8090**, `CMD ["serve"]` |
| `make cover` | Postgres-backed; `COVER_MIN_PERCENT ?= 80` |
| `make lint` / `gocyclo` / `govulncheck` / `grype` / `security` | Present |
| `make release-check` | lint → test → cover → security → docker-scan (+ goreleaser check when config exists) |
| Unit/integration tests | Across `internal/*`; ~81.8% locally when DSN available |

### 3.3 CI jobs (every PR + push to `develop`/`main`)

Workflow: `.github/workflows/ci.yml`

1. **lint** — `gofmt -s -l .` empty; `make lint` (`go vet`); `make gocyclo`  
2. **test** — `go test -race ./...` (unit path)  
3. **cover** — Postgres service (same DSN pattern as Makefile); `make cover` or equivalent; **fail if &lt; 80%**  
4. **docker** — `docker build` of `Dockerfile` (verify image builds; no push)

Jobs may be parallel where independent; **cover must not be skipped** on CI. Optional: upload `coverage.out` as artifact for debugging.

### 3.4 Local discipline

```text
make lint          # gofmt check + go vet (or fmt-check + vet — match Makefile)
make gocyclo
make test
make cover         # requires Postgres (compose)
make release-check # full pre-tag gate
```

Maintainers keep cover ≥ 80% **while landing features**, not in a rush before `v*`. Dropping below the floor is a bug to fix on `develop` before any release work.

## 4. OCI / GoReleaser

### 4.1 Runtime

| Choice | Value |
|--------|--------|
| Release Dockerfile | `Dockerfile.release` (binaries from GoReleaser context) |
| Base | `gcr.io/distroless/static-debian13:nonroot` |
| User | `nonroot` |
| Listen | **8090** |
| Health | `GET /healthz` → `{"status":"ok"}` |
| Migrations | Copied into image (`/app/migrations`) as today |

### 4.2 `.goreleaser.yaml` (mirror gfire)

- Build Go binary `gfireui-backend` for release platforms (linux/darwin/windows as family convention; **linux amd64+arm64 required** for images)  
- `dockers_v2` → `ghcr.io/hrodrig/gfireui-backend` tags `v{{ .Version }}` and `latest` (non-snapshot)  
- Multi-arch **amd64 + arm64** via prebuilt linux binaries + `Dockerfile.release`  
- `sboms` on relevant artifacts; image SBOM via `dockers_v2.sbom: true`  
- `docker_signs` with cosign keyless (`COSIGN_EXPERIMENTAL=1`, sign `${artifact}@${digest}`)  
- OCI labels: title, description, source, version, revision, created  

Local multi-stage: keep `make docker-build` on `Dockerfile` (not GoReleaser).

### 4.3 Smoke

- `make docker-smoke` (or document in release-check): run container, `curl -sf http://127.0.0.1:8090/healthz`  
- CI optional smoke after local build; release path relies on gates + GoReleaser success

## 5. Release workflow

Workflow: `.github/workflows/release.yml`  
Trigger: annotated tag `v*` on commit that is on **`main`** (git-flow: develop → main → tag).

Order (strict):

1. Checkout + setup Go + install GoReleaser / bc / QEMU / buildx / cosign / syft (same toolkit as gfire)  
2. **`make release-check`** (or equivalent CI steps that include **fmt, vet, gocyclo, test, cover ≥ 80%, security**) — **abort on failure; do not publish**  
3. Login GHCR  
4. `goreleaser release --clean`  

Permissions: `contents: write`, `packages: write`, `id-token: write`.

**No bypass flags.** No “publish then fix cover.” If cover fails on the tag commit, delete/retag after fix on `develop` → `main`.

## 6. Docs and housekeeping

| Artifact | Action |
|----------|--------|
| `SECURITY.md` | Private GitHub advisories; scope = BFF + image; point SPA/engine/selfhosted to sibling repos |
| `CHANGELOG.md` | Keep a Changelog; `[0.1.0]` when first tag ships; `[Unreleased]` for WIP |
| `ROADMAP.md` | Add band IDs below; mark Done when shipped |
| `SPECIFICATIONS.md` / `README.md` | Image contract, GHCR pull, quality/release rules, healthz |
| `.gitignore` | Already whitelists `SECURITY.md`, `CHANGELOG.md`, `.goreleaser.yaml`, `.github/` |

## 7. Roadmap IDs

| ID | Item |
|----|------|
| B-040 | `.goreleaser.yaml` — binaries + multi-arch dockers_v2 + SBOM + cosign signs |
| B-041 | `.github/workflows/ci.yml` — fmt, vet, gocyclo, test, cover≥80% (Postgres), docker build |
| B-042 | `.github/workflows/release.yml` — gates first, then GoReleaser → GHCR |
| B-043 | `SECURITY.md` + `CHANGELOG.md` + README/SPEC/ROADMAP sync |
| B-044 | Docker smoke `/healthz`; document `make release-check` before tag |
| B-045 | Create `main` if missing; first annotated tag `v0.1.0` **only when CI green** |

## 8. Relation to gfire-selfhosted

Console compose already pins:

`ghcr.io/hrodrig/gfireui-backend:${GFIREUI_BACKEND_VERSION:-v0.1.0}`

Dogfood after `v0.1.0` image exists (pair with published `gfireui:v0.1.0`).

## 9. Decisions log

| Decision | Choice |
|----------|--------|
| Packaging | **A — GoReleaser** (mirror gfire) |
| Image arches | **amd64 + arm64** from first release |
| Quality timing | **Early and constant** — CI on every PR/`develop`/`main`; release re-runs gates; no publish if cover/fmt/gocyclo/test red |
| Coverage floor | **80%** statements (`COVER_MIN_PERCENT`); fail-closed |
| Runtime base | Distroless static debian13 nonroot (existing) |
| First version | `0.1.0` / tag `v0.1.0` when gates green on `main` |
