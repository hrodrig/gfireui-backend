.PHONY: test build run migrate-up migrate-down

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
	go build -o bin/gfireui-backend ./cmd/gfireui-backend
run: build
	./bin/gfireui-backend

migrate-up:
	@test -n "$(MIGRATE_DSN)" || (echo "Set DATABASE_URL or GFIREUI_DATABASE_DSN" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" up

migrate-down:
	@test -n "$(MIGRATE_DSN)" || (echo "Set DATABASE_URL or GFIREUI_DATABASE_DSN" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(MIGRATE_DSN)" down 1
