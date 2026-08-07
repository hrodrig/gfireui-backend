.PHONY: test build run migrate-up migrate-down

APP_NAME := gfireui-backend
BIN_DIR := bin
VERSION := $(shell cat VERSION 2>/dev/null || echo dev)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.Version=$(VERSION)' \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.Commit=$(GIT_COMMIT)' \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.Branch=$(GIT_BRANCH)' \
	-X 'github.com/hrodrig/gfireui-backend/internal/version.BuildDate=$(BUILD_DATE)'

MIGRATE ?= migrate
MIGRATIONS_DIR ?= migrations

# Set DATABASE_URL or GFIREUI_DATABASE_DSN (from config env) for migrate targets.
ifneq ($(DATABASE_URL),)
MIGRATE_DSN := $(DATABASE_URL)
else
MIGRATE_DSN := $(GFIREUI_DATABASE_DSN)
endif

test:
	go test ./... -count=1
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
run: build
	./$(BIN_DIR)/$(APP_NAME) serve

migrate-up:
	@test -n "$(MIGRATE_DSN)" || (echo "Set DATABASE_URL or GFIREUI_DATABASE_DSN" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" up

migrate-down:
	@test -n "$(MIGRATE_DSN)" || (echo "Set DATABASE_URL or GFIREUI_DATABASE_DSN" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" down 1
