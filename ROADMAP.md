# GFireUI Backend Roadmap

**Start:** August 2026  
**Branch policy:** work on `develop` until end-to-end functional with GFireUI; then merge to `main` and tag.

## v0.1 — Application BFF

| ID | Item | Status |
| -- | ---- | ------ |
| B-001 | Go scaffold + `/healthz` | ✅ |
| B-002 | Config YAML/env | ✅ |
| B-003 | Domain + Postgres migrations | ✅ |
| B-004 | User store | ✅ |
| B-005 | argon2id + JWT helpers | ✅ |
| B-006 | Login + auth middleware | ✅ |
| B-007 | Audit log API | ✅ |
| B-008 | Users admin + RBAC | ✅ |
| B-009 | Thin GFire proxy | ✅ |
| B-010 | Ops summary | ✅ |
| B-011 | Admin bootstrap + CLI | ✅ |
| B-012 | Dockerfile + Dockerfile.release + compose + SPEC | ✅ |

**v0.1 done when:** compose brings API up, bootstrap Admin can login, proxy reaches a running GFire.

## Release packaging (OCI / CI)

Spec: [docs/superpowers/specs/2026-08-08-gfireui-backend-oci-ci-release-design.md](./docs/superpowers/specs/2026-08-08-gfireui-backend-oci-ci-release-design.md)  
Plan: [docs/superpowers/plans/2026-08-08-gfireui-backend-oci-ci-release.md](./docs/superpowers/plans/2026-08-08-gfireui-backend-oci-ci-release.md)

| ID | Item | Status |
| -- | ---- | ------ |
| B-040 | GoReleaser multi-arch GHCR + SBOM + cosign | ✅ |
| B-041 | CI fmt/vet/gocyclo/test/cover≥80%/docker | ✅ |
| B-042 | release.yml gates-then-GoReleaser | ✅ |
| B-043 | SECURITY + CHANGELOG + docs sync | ✅ |
| B-044 | docker-smoke `/healthz` | ✅ |
| B-045 | `main` + tag `v0.1.0` when CI green | ✅ |

**Release rule:** no tag / no GHCR publish if fmt, vet, gocyclo, test, or cover (&lt; 80%) is red.

## Post-v0.1

| ID | Item | Notes |
| -- | ---- | ----- |
| B-020 | OAuth2 / OIDC login | Design §5; BFF still mints GFireUI JWTs |
| B-021 | Refresh tokens / session hardening | |
| B-022 | Serve built SPA from binary (optional) | |
| B-030 | `*-selfhosted` sibling | Helm/k8s/manifests — **not** in this repo |

## Explicit non-goals here

Kubernetes charts, operators, and multi-cluster packaging wait for a selfhosted companion repository. This repo stays focused on the application.
