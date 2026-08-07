# GFireUI Backend — Agent Guide

Brief guide for AI agents working on this Go BFF.

## Project

- **Go module:** `github.com/hrodrig/gfireui-backend`
- **Language:** Go 1.26.5
- **Branch:** `develop` only (no feature branches)
- **License:** MIT

## Layout

- Entry point: `cmd/gfireui-backend/main.go`
- HTTP API: `internal/api/`
- All application code lives under `internal/` (service, not library)

## Repository rules

- Whitelist `.gitignore` — new root files must be added to the whitelist
- English only in code, comments, commits, and docs
- TDD per task: failing test → implement → pass → commit

## Docs

| Document | Role |
| -------- | ---- |
| [Platform design](./docs/superpowers/specs/2026-08-06-gfireui-platform-design.md) | Approved architecture & API intent |
| [v0.1 implementation plan](./docs/superpowers/plans/2026-08-06-gfireui-backend-v0.1.md) | Task-by-task build plan |

## Build & test

```sh
make test
make build
make run
```
