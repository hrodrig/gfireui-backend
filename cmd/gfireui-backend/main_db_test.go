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
)

func TestRunServeWithDatabaseAndBootstrapSkip(t *testing.T) {
	dsn := os.Getenv("GFIREUI_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_TEST_DSN unset")
	}

	clearGFireUIEnv(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	t.Setenv("GFIREUI_SERVER_ADDR", addr)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GFIREUI_DATABASE_DSN", dsn)
	// Leave bootstrap empty so validate passes and bootstrap is disabled.
	t.Setenv("GFIREUI_GFIRE_BASE_URL", "http://127.0.0.1:9")
	t.Setenv("GFIREUI_GFIRE_TOKEN", "tok")

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

func TestRunUserCreatePersists(t *testing.T) {
	dsn := os.Getenv("GFIREUI_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_TEST_DSN unset")
	}

	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GFIREUI_DATABASE_DSN", dsn)

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
