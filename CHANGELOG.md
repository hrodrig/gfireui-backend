# Changelog

All notable changes to **gfireui-backend** are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.1] - 2026-08-08

### Added

- `GET /api/ops/summary` expands to canonical GFire states, `servers_count`, `recurring_count`, and `versions[]` (BFF + upstream gfire from `/healthz`).
- `GET /healthz` includes `version` and `commit` (ldflags).

### Changed

- README documents the fail-closed release contract (gates before GoReleaser; links OCI/CI design).

## [0.1.0] - 2026-08-08

### Added

- Go BFF for GFireUI: login/JWT, users + RBAC, audit log, thin GFire proxy, ops summary, admin bootstrap + CLI.
- Postgres store + migrations; config via YAML/env (`GFIREUI_BACKEND_*`).
- Dockerfile (multi-stage) + `Dockerfile.release` (distroless `static-debian13:nonroot`, listen **8090**).
- Local Compose stack (postgres + migrate + backend).
- Quality gates: `make lint`, `gocyclo`, `test`, `cover` (≥80% with Postgres), `security`, `release-check`.
- `make docker-smoke` — curl `/healthz` on local image.
- [SECURITY.md](./SECURITY.md) — vulnerability reporting via GitHub Security Advisories.
- `.goreleaser.yaml` — multi-arch (`amd64`/`arm64`) GHCR image, SBOM, cosign signs (mirror gfire).
- GitHub Actions CI: fmt, vet, gocyclo, race tests, cover ≥80% (Postgres), docker build (fail-closed).
- Release workflow: `make release-check` then GoReleaser → `ghcr.io/hrodrig/gfireui-backend`.

### Notes

- Pair with [gfireui](https://github.com/hrodrig/gfireui) `v0.1.0` and [gfire](https://github.com/hrodrig/gfire). Deploy via [gfire-selfhosted](https://github.com/hrodrig/gfire-selfhosted) console stack (`GFIREUI_BACKEND_VERSION=v0.1.0`).

[Unreleased]: https://github.com/hrodrig/gfireui-backend/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/hrodrig/gfireui-backend/releases/tag/v0.1.0
