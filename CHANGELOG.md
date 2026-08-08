# Changelog

All notable changes to **gfireui-backend** are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- [SECURITY.md](./SECURITY.md) — vulnerability reporting via GitHub Security Advisories.
- `.goreleaser.yaml` — multi-arch (`amd64`/`arm64`) GHCR image, SBOM, cosign signs (mirror gfire).
- GitHub Actions CI: fmt, vet, gocyclo, race tests, cover ≥80% (Postgres), docker build.
- Release workflow: `make release-check` then GoReleaser → `ghcr.io/hrodrig/gfireui-backend`.
- `make docker-smoke` — curl `/healthz` on local image.

[Unreleased]: https://github.com/hrodrig/gfireui-backend/compare/HEAD...HEAD
