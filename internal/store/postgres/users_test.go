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

	byID, err := store.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if byID.Email != email {
		t.Fatalf("by id email = %q", byID.Email)
	}

	listed, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	found := false
	for _, item := range listed {
		if item.ID == user.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("created user missing from ListUsers")
	}

	user.FirstName = "Augusta"
	user.Enabled = false
	if err := store.UpdateUser(ctx, user); err != nil {
		t.Fatalf("update user: %v", err)
	}
	updated, err := store.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload updated user: %v", err)
	}
	if updated.FirstName != "Augusta" || updated.Enabled {
		t.Fatalf("updated = %#v", updated)
	}

	if _, err := store.GetUserByEmail(ctx, "missing-"+email); err == nil {
		t.Fatal("expected not found by email")
	}
	if _, err := store.GetUserByID(ctx, uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f99")); err == nil {
		t.Fatal("expected not found by id")
	}
	missing := &domain.User{
		ID:           uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f98"),
		FirstName:    "No",
		LastName:     "Such",
		Email:        "nosuch-" + email,
		Role:         domain.RoleGuest,
		Enabled:      true,
		PasswordHash: "hash",
	}
	if err := store.UpdateUser(ctx, missing); err == nil {
		t.Fatal("expected update not found")
	}
	if err := store.CreateUser(ctx, nil); err == nil {
		t.Fatal("expected nil create error")
	}
	if err := store.UpdateUser(ctx, nil); err == nil {
		t.Fatal("expected nil update error")
	}
	if err := store.WriteAudit(ctx, nil); err == nil {
		t.Fatal("expected nil audit error")
	}
}

func TestOpenRejectsBadDSN(t *testing.T) {
	dsn := os.Getenv("GFIREUI_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_TEST_DSN unset")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := postgres.Open(ctx, "://not-a-dsn"); err == nil {
		t.Fatal("expected open error for invalid DSN")
	}
	if _, err := postgres.Open(ctx, "postgres://gfireui:gfireui@127.0.0.1:1/gfireui?sslmode=disable"); err == nil {
		t.Fatal("expected open error for unreachable postgres")
	}
}
