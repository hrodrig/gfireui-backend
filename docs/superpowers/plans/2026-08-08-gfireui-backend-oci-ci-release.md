# GFireUI Backend OCI / CI / Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Commits require user-approved messages (project rule).

**Goal:** Land GoReleaser multi-arch GHCR publish (`ghcr.io/hrodrig/gfireui-backend`) with SBOM + cosign, and CI that fails closed on fmt/vet/gocyclo/test/cover≥80% so a release never ships on a red tree.

**Architecture:** Mirror [gfire](https://github.com/hrodrig/gfire): local `Dockerfile` for day-to-day; `Dockerfile.release` + `.goreleaser.yaml` `dockers_v2` for tagged releases (linux amd64+arm64). CI runs quality on every PR/`develop`/`main`; release workflow re-runs `make release-check` (or equivalent) **before** `goreleaser release`.

**Spec:** [../specs/2026-08-08-gfireui-backend-oci-ci-release-design.md](../specs/2026-08-08-gfireui-backend-oci-ci-release-design.md)

**Tech Stack:** Go 1.26.5, Makefile gates, GitHub Actions, GoReleaser v2, distroless static-debian13, syft, cosign keyless, Postgres service for cover.

## Global Constraints

- English artifacts; work on `develop` only (no feature branches)
- `COVER_MIN_PERCENT ?= 80` — fail-closed; **no release if cover red**
- Quality early and constant: fmt + vet + gocyclo + test + cover on every CI run
- Release: GoReleaser; image arches **linux/amd64 + linux/arm64**; `id-token: write` for cosign
- Runtime listen **8090**; health `GET /healthz`
- No commit without explicit user-approved message
- Do not delete files without approval

## File map

| Path | Role |
|------|------|
| `.goreleaser.yaml` | **Create** — builds, archives, sboms, dockers_v2, docker_signs, checksum signs |
| `.github/workflows/ci.yml` | **Create** — lint / test / cover (Postgres) / docker build |
| `.github/workflows/release.yml` | **Create** — gates then GoReleaser |
| `SECURITY.md` | **Create** — private advisories |
| `CHANGELOG.md` | **Create** — Keep a Changelog |
| `Makefile` | **Modify** — `docker-smoke`; ensure release-check already invokes goreleaser check once yaml exists |
| `ROADMAP.md` | **Modify** — B-040…B-045 |
| `README.md` / `SPECIFICATIONS.md` | **Modify** — GHCR, CI, release contract |
| `Dockerfile` / `Dockerfile.release` | **Reuse** (no redesign unless smoke needs tweak) |

---

### Task 1: B-043 — SECURITY.md + CHANGELOG.md (docs baseline)

**Files:**
- Create: `SECURITY.md`
- Create: `CHANGELOG.md`
- Modify: `ROADMAP.md` (add B-040…B-045 Pending)

**Interfaces:**
- Consumes: sibling SECURITY style from gfireui / gghstats
- Produces: tracked policy + changelog skeleton for later `[0.1.0]` notes

- [x] **Step 1: Write `SECURITY.md`**

Scope: BFF binary + OCI image. Point SPA → gfireui, engine → gfire, deploy → gfire-selfhosted. Preferred: GitHub Security Advisories for `hrodrig/gfireui-backend`. Supported: latest release only.

- [x] **Step 2: Write `CHANGELOG.md`**

Keep a Changelog header; `## [Unreleased]` with note that OCI/CI/release work lands here; prepare empty or stub `## [0.1.0]` only when tagging (prefer fill at release task).

- [x] **Step 3: Update `ROADMAP.md`**

Add “Release packaging” table B-040…B-045 (B-043 in progress).

- [ ] **Step 4: Commit** (message approved by user)

```
docs: add SECURITY.md and CHANGELOG for release band
```

---

### Task 2: B-040 — `.goreleaser.yaml`

**Files:**
- Create: `.goreleaser.yaml`
- Reference: `/Volumes/Data/addlink/github/gfire/.goreleaser.yaml`
- Reference: `Dockerfile.release`, `internal/version/version.go`

**Interfaces:**
- Consumes: `main: ./cmd/gfireui-backend`, binary `gfireui-backend`, ldflags into `github.com/hrodrig/gfireui-backend/internal/version`
- Produces: GHCR image `ghcr.io/hrodrig/gfireui-backend` tags `v{{ .Version }}` + `latest`; multi-arch amd64+arm64

- [ ] **Step 1: Author `.goreleaser.yaml` (version: 2)**

Mirror gfire structure:

- `before.hooks`: `go mod tidy` (optional `go test` — prefer CI/release-check for Postgres cover; keep before-hook light or `go test` without DSN only)
- `builds`: CGO_ENABLED=0; goos linux/darwin/windows; goarch amd64/arm64; ignore windows/arm64
- `ldflags`: Version, Commit, Branch, BuildDate → `internal/version`
- `archives` + `checksum` + `signs` (cosign sign-blob on checksum) + `changelog` filters
- `sboms`: source SPDX + CycloneDX named `gfireui-backend_{{ .Version }}_sbom.*.json`
- `dockers_v2`: `Dockerfile.release`, `sbom: true`, image `ghcr.io/hrodrig/gfireui-backend`, labels
- `docker_signs`: cosign sign `${artifact}@${digest}`

- [ ] **Step 2: Validate config**

Run: `goreleaser check`  
Expected: pass

- [ ] **Step 3: Optional snapshot dry-run** (if Docker/QEMU available)

Run: `make snapshot` or `goreleaser release --snapshot --clean` after `make release-check`  
Expected: local dist artifacts; no GHCR push required

- [ ] **Step 4: Commit**

```
build: add GoReleaser config for multi-arch GHCR
```

---

### Task 3: B-044 — `docker-smoke` + wire into docs/Makefile

**Files:**
- Modify: `Makefile`
- Modify: `README.md` (smoke command)

**Interfaces:**
- Consumes: `make docker-build` → `gfireui-backend:$(VERSION)`
- Produces: `make docker-smoke` curls `/healthz`

- [ ] **Step 1: Add `docker-smoke` target**

Pattern:

1. Ensure image exists (`docker-build` dependency or require prior build)
2. `docker run -d --rm -p 18090:8090` with minimal env (`GFIREUI_BACKEND_SERVER_ADDR=:8090`, JWT secret dummy — confirm serve starts without full Postgres if healthz is process-only; if serve requires DB, document and use compose postgres or skip DB-dependent start)

**Important:** Inspect whether `serve` requires Postgres before listen. If yes, smoke must either:

- hit a binary that serves `/healthz` before DB, or  
- run against `docker compose` stack briefly, or  
- use `docker run` with linked postgres

Prefer: if `/healthz` is available without DB, keep smoke simple. If not, smoke via compose `gfireui-backend` + `curl`.

- [ ] **Step 2: Run smoke locally**

Expected: HTTP 200 + `{"status":"ok"}` (or equivalent)

- [ ] **Step 3: Commit**

```
chore: add docker-smoke healthz check
```

---

### Task 4: B-041 — CI workflow (early constant gates)

**Files:**
- Create: `.github/workflows/ci.yml`
- Reference: gfire `ci.yml` + this repo `Makefile` cover/DSN

**Interfaces:**
- Consumes: `COVER_DSN` / `GFIREUI_BACKEND_TEST_DSN`, `COVER_MIN_PERCENT=80`
- Produces: red CI blocks merge; cover never optional

- [ ] **Step 1: Write `ci.yml`**

Triggers: `push` + `pull_request` to `main`, `develop`  
Concurrency: cancel-in-progress  
Permissions: `contents: read`

Jobs:

1. **lint** — setup-go from `go.mod`; `gofmt -s -l .` fail if non-empty; `make lint` (or vet only if fmt already checked); install/run `make gocyclo`
2. **test** — `go test -race ./... -count=1`
3. **cover** — service `postgres:18` (or match compose image/creds: user/db/password `gfireui`, host port mapping to 5432 in-job); wait `pg_isready`; set `GFIREUI_BACKEND_TEST_DSN`; run cover profile + awk gate ≥80% (same logic as Makefile). Install `bc` only if Makefile uses it — current Makefile uses `awk` only.
4. **docker** — `docker build` with VERSION/COMMIT build-args from git; no push

**Hard rule:** do not mark cover `continue-on-error`. Do not skip cover on PRs.

- [ ] **Step 2: Push to `develop` and confirm Actions green** (after user approves commit)

- [ ] **Step 3: Commit**

```
ci: add fmt vet gocyclo test cover and docker gates
```

---

### Task 5: B-042 — Release workflow

**Files:**
- Create: `.github/workflows/release.yml`
- Reference: gfire `release.yml`

**Interfaces:**
- Consumes: green tree + annotated tag `v*`
- Produces: GitHub Release + GHCR multi-arch + SBOM + cosign

- [ ] **Step 1: Write `release.yml`**

Trigger: `push` tags `v*`  
Permissions: `contents: write`, `packages: write`, `id-token: write`

Steps (order mandatory):

1. checkout fetch-depth 0  
2. setup-go  
3. install goreleaser (install-only)  
4. install bc if needed  
5. **QEMU + buildx** (multi-arch)  
6. **`make release-check`** — **must fail the job before publish if cover/fmt/gocyclo/security fail**  
   - Note: `release-check` needs Docker for postgres cover + docker-scan; ensure runner has Docker  
7. login ghcr.io  
8. cosign-installer + syft download  
9. `goreleaser release --clean` with `GITHUB_TOKEN`

- [ ] **Step 2: Commit**

```
ci: add release workflow with gates before GoReleaser
```

---

### Task 6: Docs sync (README / SPEC / ROADMAP / CHANGELOG)

**Files:**
- Modify: `README.md` — Security link, GHCR badge/pull, CI/release notes, `make docker-smoke`
- Modify: `SPECIFICATIONS.md` — image + supply-chain + coverage CI contract
- Modify: `ROADMAP.md` — mark B-040…B-044 Done when landed; B-045 Pending until tag
- Modify: `CHANGELOG.md` — Unreleased bullets for this band

- [ ] **Step 1: Edit docs**
- [ ] **Step 2: Commit**

```
docs: document GHCR release and CI quality gates
```

---

### Task 7: B-045 — Verify gates, create `main`, tag `v0.1.0`

**Preflight (local):**

- [ ] **Step 1: `make release-check`** — must pass (cover ≥80%, lint, gocyclo, security, docker-scan, goreleaser check)
- [ ] **Step 2: Confirm CI green on `develop`**
- [ ] **Step 3: Fill CHANGELOG `[0.1.0]`** if still under Unreleased; bump docs status; commit if needed
- [ ] **Step 4: Push `develop`**
- [ ] **Step 5: Create `main` from `develop` if missing; push `main`**
- [ ] **Step 6: Annotated tag on `main`:** `git tag -a v0.1.0 -m "Release 0.1.0"` → `git push origin v0.1.0`
- [ ] **Step 7: Watch release workflow — abort/investigate if gates fail (do not force-publish)**
- [ ] **Step 8: Verify** `docker pull ghcr.io/hrodrig/gfireui-backend:v0.1.0` (amd64 + note arm64 manifest)
- [ ] **Step 9: Mark B-045 Done in ROADMAP; pin already `GFIREUI_BACKEND_VERSION=v0.1.0` in selfhosted**

---

## Execution notes

- Prefer implementing Tasks 1→6 on `develop` with CI proving gates **before** Task 7.
- If cover dips below 80% during any task: **stop release work**, add tests, restore gate — do not tag.
- GoReleaser `before.hooks` must not replace CI cover; Postgres-backed cover stays in Makefile/CI/release-check.
- Commit messages: show to user, wait for approval (project rule).

## Done when

- [ ] CI green on `develop` with cover gate enforced  
- [ ] `ghcr.io/hrodrig/gfireui-backend:v0.1.0` multi-arch published with SBOM + cosign  
- [ ] SECURITY.md + CHANGELOG present  
- [ ] Selfhosted can pull BFF image alongside `gfireui:v0.1.0`
