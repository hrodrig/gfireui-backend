package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/domain"
	"github.com/hrodrig/gfireui-backend/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ensureUsersSchema(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT true,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	return err
}

func TestCreateUserAndGetUserByEmail(t *testing.T) {
	dsn := os.Getenv("GFIREUI_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_TEST_DSN unset")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ensureUsersSchema(ctx, dsn); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}

	store, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(store.Close)

	email := fmt.Sprintf("user-%s@example.com", uuid.NewString())
	user := &domain.User{
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        email,
		Role:         domain.RoleAdministrator,
		Enabled:      true,
		PasswordHash: "hash",
	}

	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == uuid.Nil {
		t.Fatal("expected CreateUser to assign an ID")
	}

	got, err := store.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if got.Email != email {
		t.Fatalf("email = %q, want %q", got.Email, email)
	}
	if got.ID != user.ID {
		t.Fatalf("id = %s, want %s", got.ID, user.ID)
	}
	if got.FirstName != user.FirstName || got.LastName != user.LastName || got.Role != user.Role || got.Enabled != user.Enabled {
		t.Fatalf("got user = %#v, want %#v", got, user)
	}
}
