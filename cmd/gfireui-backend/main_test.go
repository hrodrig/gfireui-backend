package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hrodrig/gfireui-backend/internal/config"
)

func TestRunUnknownCommand(t *testing.T) {
	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")

	err := run(context.Background(), "", []string{"nope"})
	if err == nil || !strings.Contains(err.Error(), `unknown command "nope"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunServeRejectsExtraArgs(t *testing.T) {
	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")

	err := run(context.Background(), "", []string{"serve", "extra"})
	if err == nil || !strings.Contains(err.Error(), "serve does not accept arguments") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunUserMissingSubcommand(t *testing.T) {
	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")

	err := run(context.Background(), "", []string{"user"})
	if err == nil || !strings.Contains(err.Error(), "missing user subcommand") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunUserUnknownSubcommand(t *testing.T) {
	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")

	err := run(context.Background(), "", []string{"user", "delete"})
	if err == nil || !strings.Contains(err.Error(), `unknown user subcommand "delete"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunUserCreateRequiresDSN(t *testing.T) {
	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")

	err := run(context.Background(), "", []string{
		"user", "create",
		"--email", "ada@example.com",
		"--password", "secret-password",
		"--role", "Administrator",
		"--first-name", "Ada",
		"--last-name", "Lovelace",
	})
	if err == nil || !strings.Contains(err.Error(), "database.dsn is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunUserCreateFlagParseError(t *testing.T) {
	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")

	err := run(context.Background(), "", []string{"user", "create", "-bogus"})
	if err == nil {
		t.Fatal("expected flag parse error")
	}
}

func TestRunServeStartsAndShutsDown(t *testing.T) {
	clearGFireUIEnv(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	t.Setenv("GFIREUI_SERVER_ADDR", addr)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GFIREUI_DATABASE_DSN", "")
	t.Setenv("GFIREUI_GFIRE_BASE_URL", "")
	t.Setenv("GFIREUI_GFIRE_TOKEN", "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, "", []string{"serve"})
	}()

	deadline := time.Now().Add(2 * time.Second)
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
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not shut down")
	}
}

func TestRunServeWithGFireClientNoToken(t *testing.T) {
	clearGFireUIEnv(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	t.Setenv("GFIREUI_SERVER_ADDR", addr)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GFIREUI_DATABASE_DSN", "")
	t.Setenv("GFIREUI_GFIRE_BASE_URL", "http://127.0.0.1:9")
	t.Setenv("GFIREUI_GFIRE_TOKEN", "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, "", nil)
	}()

	deadline := time.Now().Add(2 * time.Second)
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
			t.Fatalf("run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not shut down")
	}
}

func TestRunServeBadGFireURL(t *testing.T) {
	clearGFireUIEnv(t)
	t.Setenv("GFIREUI_AUTH_JWT_SECRET", "test-secret")
	t.Setenv("GFIREUI_GFIRE_BASE_URL", "://bad")

	err := run(context.Background(), "", []string{"serve"})
	if err == nil || !strings.Contains(err.Error(), "configure gfire client") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunLoadConfigFromFile(t *testing.T) {
	clearGFireUIEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "gfireui-backend.yaml")
	content := `
server:
  addr: "127.0.0.1:0"
auth:
  jwt_secret: "file-secret"
  token_ttl: 1h
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Auth.JWTSecret != "file-secret" {
		t.Fatalf("jwt = %q", cfg.Auth.JWTSecret)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, path, []string{"serve"})
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout")
	}
}

func clearGFireUIEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GFIREUI_SERVER_ADDR",
		"GFIREUI_DATABASE_DSN",
		"GFIREUI_AUTH_JWT_SECRET",
		"GFIREUI_AUTH_TOKEN_TTL",
		"GFIREUI_GFIRE_BASE_URL",
		"GFIREUI_GFIRE_TOKEN",
		"GFIREUI_BOOTSTRAP_ADMIN_EMAIL",
		"GFIREUI_BOOTSTRAP_ADMIN_PASSWORD",
		"GFIREUI_BOOTSTRAP_ADMIN_FIRST_NAME",
		"GFIREUI_BOOTSTRAP_ADMIN_LAST_NAME",
		"GFIREUI_CONFIG",
	} {
		t.Setenv(key, "")
	}
}
