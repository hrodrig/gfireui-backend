# gfireui-backend — build, quality and release workflow
# Shape mirrors https://github.com/hrodrig/gghstats Makefile.

BINARY      := gfireui-backend
MODULE      := github.com/hrodrig/gfireui-backend
DIST        := dist
# Single source of truth: VERSION file at repo root (no silent fallback — avoids wrong image/tarball names).
VERSION_RAW ?= $(shell cat VERSION 2>/dev/null | tr -d '\n\r')
VERSION     := $(patsubst v%,%,$(VERSION_RAW))
TAG         := v$(VERSION)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BRANCH      := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null | tr -d '\n\r' || echo unknown)
BUILDDATE   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
# Fails early when Docker is required but not running.
check-docker = @docker info >/dev/null 2>&1 || { echo "Error: Docker is not running. Start Docker and try again."; exit 1; }
GRYPE_FAIL_ON  ?= high
# Empty = native arch. Set linux/amd64 when building on Apple Silicon for a typical VPS.
DOCKER_PLATFORM ?=
LDFLAGS     := -s -w \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.Branch=$(BRANCH)' \
	-X '$(MODULE)/internal/version.BuildDate=$(BUILDDATE)'

MIGRATE ?= migrate
MIGRATIONS_DIR ?= migrations
ifneq ($(DATABASE_URL),)
MIGRATE_DSN := $(DATABASE_URL)
else
MIGRATE_DSN := $(GFIREUI_BACKEND_DATABASE_DSN)
endif

.DEFAULT_GOAL := help

# Project-wide statement coverage floor (make cover / release-check).
COVER_MIN_PERCENT ?= 80
# Integration store tests need Postgres (compose service on host port 5433).
COVER_DSN ?= postgres://gfireui:gfireui@127.0.0.1:5433/gfireui?sslmode=disable

GREEN  := \033[0;32m
YELLOW := \033[0;33m
RESET  := \033[0m

.PHONY: build clean compose-down compose-up cover docker-build docker-build-amd64 \
	docker-run docker-scan gocyclo govulncheck grype help install lint lint-fix \
	migrate-down migrate-up release release-check security server snapshot test tools

help:
	@echo "$(GREEN)gfireui-backend$(RESET) — BFF for GFireUI (auth, RBAC, audit, GFire proxy)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "$(YELLOW)Build:$(RESET)"
	@echo "  $(GREEN)build$(RESET)            Build local binary"
	@echo "  $(GREEN)clean$(RESET)            Remove local build artifacts"
	@echo "  $(GREEN)compose-down$(RESET)     Stop stack with docker compose"
	@echo "  $(GREEN)compose-up$(RESET)       docker-build then compose up (uses image $(BINARY):$(VERSION))"
	@echo "  $(GREEN)install$(RESET)          Install binary with ldflags"
	@echo "  $(GREEN)server$(RESET)           Run serve locally (go run)"
	@echo "  $(GREEN)migrate-up$(RESET)       Apply PostgreSQL migrations"
	@echo "  $(GREEN)migrate-down$(RESET)     Roll back one migration"
	@echo ""
	@echo "$(YELLOW)Quality:$(RESET)"
	@echo "  $(GREEN)cover$(RESET)            Postgres via compose + tests; fail if total < $(COVER_MIN_PERCENT)%"
	@echo "  $(GREEN)grype$(RESET)            Grype directory scan (excludes ./dist/**, ./$(BINARY))"
	@echo "  $(GREEN)lint$(RESET)             Check gofmt -s and go vet"
	@echo "  $(GREEN)lint-fix$(RESET)         Apply gofmt -s -w"
	@echo "  $(GREEN)security$(RESET)         Run govulncheck, gocyclo and grype"
	@echo "  $(GREEN)tools$(RESET)            Install govulncheck and gocyclo"
	@echo "  $(GREEN)test$(RESET)             Run unit tests (-race)"
	@echo ""
	@echo "$(YELLOW)Docker:$(RESET)"
	@echo "  $(GREEN)docker-build$(RESET)       Build image $(BINARY):$(VERSION) (optional: DOCKER_PLATFORM=linux/amd64)"
	@echo "  $(GREEN)docker-build-amd64$(RESET) Same, forced linux/amd64"
	@echo "  $(GREEN)docker-run$(RESET)         Run local Docker image on :8090"
	@echo "  $(GREEN)docker-scan$(RESET)        Build and scan image with Grype"
	@echo ""
	@echo "$(YELLOW)Release:$(RESET)"
	@echo "  $(GREEN)release$(RESET)            Publish release (main branch only; needs .goreleaser.yaml)"
	@echo "  $(GREEN)release-check$(RESET)      lint, test, cover (≥$(COVER_MIN_PERCENT)%), security, docker-scan"
	@echo "  $(GREEN)snapshot$(RESET)           Goreleaser snapshot (VERSION → <semver>-next, dist/)"
	@echo ""
	@echo "Current version: $(VERSION) (tag: $(TAG))"

build:
	@test -n "$(VERSION)" || { echo "Error: VERSION file empty or missing"; exit 1; }
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/$(BINARY)

install:
	@test -n "$(VERSION)" || { echo "Error: VERSION file empty or missing"; exit 1; }
	go install -ldflags "$(LDFLAGS)" ./cmd/$(BINARY)

server:
	@test -n "$(VERSION)" || { echo "Error: VERSION file empty or missing"; exit 1; }
	go run -ldflags "$(LDFLAGS)" ./cmd/$(BINARY) serve

compose-up: docker-build
	GFIREUI_BACKEND_IMAGE=$(BINARY):$(VERSION) docker compose up -d

compose-down:
	docker compose down

migrate-up:
	@test -n "$(MIGRATE_DSN)" || (echo "Set DATABASE_URL or GFIREUI_BACKEND_DATABASE_DSN" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" up

migrate-down:
	@test -n "$(MIGRATE_DSN)" || (echo "Set DATABASE_URL or GFIREUI_BACKEND_DATABASE_DSN" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" down 1

test:
	go test -race ./...

cover:
	$(check-docker)
	@docker compose up -d postgres
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 15 18 20; do \
		docker compose exec -T postgres pg_isready -U gfireui -d gfireui >/dev/null 2>&1 && break; \
		sleep 1; \
	done
	GFIREUI_BACKEND_TEST_DSN="$(COVER_DSN)" go test ./... -coverprofile=coverage.out -covermode=atomic -count=1
	@total=$$(go tool cover -func=coverage.out | tail -1); echo "$$total"; \
	pct=$$(echo "$$total" | awk '{print $$NF}' | tr -d '%'); \
	awk -v p="$$pct" -v min="$(COVER_MIN_PERCENT)" 'BEGIN { if ((p+0) < (min+0)) { printf "coverage %s%% must be >= %s%%\n", p, min; exit 1 } }'

lint:
	@echo "Checking gofmt -s..."
	@unformatted=$$(gofmt -s -l .); [ -z "$$unformatted" ] || { echo "Files not formatted (run make lint-fix):"; echo "$$unformatted"; exit 1; }
	@echo "Running go vet..."
	@go vet ./...

lint-fix:
	gofmt -s -w .

clean:
	rm -f $(BINARY) coverage.out
	rm -rf $(DIST) bin

docker-build:
	$(check-docker)
	@test -n "$(VERSION)" || { echo "Error: VERSION file empty or missing"; exit 1; }
	@set -e; \
	opts=""; \
	if [ -n "$(strip $(DOCKER_PLATFORM))" ]; then opts="--platform $(DOCKER_PLATFORM)"; fi; \
	DOCKER_BUILDKIT=1 docker build $$opts \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) \
		--build-arg BRANCH=$(BRANCH) --build-arg BUILDDATE=$(BUILDDATE) \
		-t $(BINARY):$(VERSION) .

docker-build-amd64:
	$(MAKE) docker-build DOCKER_PLATFORM=linux/amd64

docker-scan: docker-build
	@if command -v grype >/dev/null 2>&1; then \
		grype $(BINARY):$(VERSION) --fail-on $(GRYPE_FAIL_ON) ; \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v /var/run/docker.sock:/var/run/docker.sock anchore/grype:latest \
			$(BINARY):$(VERSION) --fail-on $(GRYPE_FAIL_ON) ; \
	fi

docker-run:
	$(check-docker)
	docker run --rm -p 8090:8090 \
		-e GFIREUI_BACKEND_SERVER_ADDR=:8090 \
		-e GFIREUI_BACKEND_AUTH_JWT_SECRET=dev-only-change-me \
		$(BINARY):$(VERSION)

tools:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

gocyclo:
	@command -v gocyclo >/dev/null 2>&1 || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 15 .

# Dir scan: exclude local build outputs — binaries embed buildinfo and skew CVE noise.
GRYPE_DIR_EXCLUDES := --exclude './dist/**' --exclude './$(BINARY)' --exclude './bin/**'

grype:
	@if command -v grype >/dev/null 2>&1; then \
		grype dir:. $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON) ; \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v "$(PWD):/workspace" anchore/grype:latest \
			dir:/workspace $(GRYPE_DIR_EXCLUDES) --fail-on $(GRYPE_FAIL_ON) ; \
	fi

security: govulncheck gocyclo grype

release-check:
	$(check-docker)
	@test -f VERSION || (echo "VERSION file is required"; exit 1)
	@echo "Release version: $(VERSION) (tag: $(TAG))"
	@echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || (echo "VERSION must be semantic version (e.g. 0.1.0)"; exit 1)
	@echo "Running release checks (lint, test, cover ≥$(COVER_MIN_PERCENT)%, security, docker-scan)..."
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) cover
	@$(MAKE) security
	@$(MAKE) docker-scan
	@if [ -f .goreleaser.yaml ]; then \
		command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser is required. Install from https://goreleaser.com/install/"; exit 1; }; \
		goreleaser check; \
	else \
		echo "Note: .goreleaser.yaml not present yet — skipped goreleaser check"; \
	fi
	@echo "All release checks passed."

define export_snapshot_version
	set -e; \
	ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	export GFIREUI_BACKEND_SNAPSHOT_VERSION="$$ver-next"; \
	echo "Snapshot version: $$GFIREUI_BACKEND_SNAPSHOT_VERSION (from VERSION)"
endef

snapshot: release-check
	@test -f .goreleaser.yaml || { echo "Error: .goreleaser.yaml is required for snapshot"; exit 1; }
	@$(export_snapshot_version); \
	goreleaser release --snapshot --clean

release: release-check
	@test -f .goreleaser.yaml || { echo "Error: .goreleaser.yaml is required for release"; exit 1; }
	@branch=$$(git branch --show-current 2>/dev/null); \
	if [ "$$branch" != "main" ]; then \
	  echo "Error: release only from main (current: $$branch)."; \
	  exit 1; \
	fi
	goreleaser release --clean
