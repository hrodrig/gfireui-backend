package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunServeWithDatabaseAndBootstrapSkip(t *testing.T) {
	dsn := os.Getenv("GFIREUI_BACKEND_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_BACKEND_TEST_DSN unset")
	}

	clearGFireUIEnv(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	t.Setenv("GFIREUI_BACKEND_SERVER_ADDR", addr)
	t.Setenv("GFIREUI_BACKEND_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GFIREUI_BACKEND_DATABASE_DSN", dsn)
	// Leave bootstrap empty so validate passes and bootstrap is disabled.
	t.Setenv("GFIREUI_BACKEND_GFIRE_BASE_URL", "http://127.0.0.1:9")
	t.Setenv("GFIREUI_BACKEND_GFIRE_TOKEN", "tok")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, "", []string{"serve"})
	}()

	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("server did not start: %v", dialErr)
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not shut down")
	}
}

func ensureUsersTable(t *testing.T, dsn string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
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
	if err != nil {
		t.Fatalf("ensure users table: %v", err)
	}
}

func TestRunUserCreatePersists(t *testing.T) {
	dsn := os.Getenv("GFIREUI_BACKEND_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_BACKEND_TEST_DSN unset")
	}

	ensureUsersTable(t, dsn)

	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_BACKEND_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GFIREUI_BACKEND_DATABASE_DSN", dsn)

	email := fmt.Sprintf("cli-%s@example.com", uuid.NewString())
	err := run(context.Background(), "", []string{
		"user", "create",
		"--email", email,
		"--password", "secret-password",
		"--role", "Operator",
		"--first-name", "CLI",
		"--last-name", "User",
	})
	if err != nil {
		t.Fatalf("user create: %v", err)
	}

	// unexpected trailing args
	err = run(context.Background(), "", []string{
		"user", "create",
		"--email", "x@example.com",
		"--password", "secret-password",
		"--role", "Operator",
		"--first-name", "X",
		"--last-name", "Y",
		"extra",
	})
	if err == nil || !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("error = %v", err)
	}
}
