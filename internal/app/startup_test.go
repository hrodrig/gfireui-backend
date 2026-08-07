package app_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/app"
	"github.com/hrodrig/gfireui-backend/internal/config"
)

func TestBaseURLVariants(t *testing.T) {
	t.Parallel()

	cases := []struct {
		addr string
		want string
	}{
		{addr: ":8090", want: "http://127.0.0.1:8090"},
		{addr: "0.0.0.0:8090", want: "http://127.0.0.1:8090"},
		{addr: "127.0.0.1:9090", want: "http://127.0.0.1:9090"},
		{addr: "[::]:8090", want: "http://127.0.0.1:8090"},
		{addr: "not-a-host", want: "http://127.0.0.1:8090"},
	}
	for _, tt := range cases {
		cfg := &config.Config{Server: config.ServerConfig{Addr: tt.addr}}
		if got := app.BaseURL(cfg); got != tt.want {
			t.Fatalf("BaseURL(%q) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}

func TestWriteStartupBannerLabels(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Server:   config.ServerConfig{Addr: ":8090"},
		Database: config.DatabaseConfig{DSN: ""},
		GFire:    config.GFireConfig{BaseURL: "http://gfire:8080", Token: ""},
		Auth:     config.AuthConfig{JWTSecret: ""},
	}
	var b strings.Builder
	app.WriteStartupBanner(&b, cfg)
	got := b.String()
	for _, want := range []string{"gfireui-backend", "db off", "gfire on(no-token)", "jwt unset"} {
		if !strings.Contains(got, want) {
			t.Fatalf("banner missing %q: %s", want, got)
		}
	}

	app.WriteStartupBanner(&b, nil) // no-op
}

func TestLogStartupAndListening(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	cfg := &config.Config{
		Server:   config.ServerConfig{Addr: "127.0.0.1:8090"},
		Database: config.DatabaseConfig{DSN: "postgres://x"},
		GFire:    config.GFireConfig{},
		Auth:     config.AuthConfig{JWTSecret: "secret"},
	}
	app.LogStartup(cfg)
	app.LogListening(cfg)

	out := buf.String()
	if !strings.Contains(out, "gfireui-backend starting") {
		t.Fatalf("missing starting log: %s", out)
	}
	if !strings.Contains(out, "listening") {
		t.Fatalf("missing listening log: %s", out)
	}
}
