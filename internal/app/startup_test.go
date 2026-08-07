package app_test

import (
	"strings"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/app"
	"github.com/hrodrig/gfireui-backend/internal/config"
)

func TestBaseURLBindAll(t *testing.T) {
	cfg := &config.Config{Server: config.ServerConfig{Addr: ":8090"}}
	if got := app.BaseURL(cfg); got != "http://127.0.0.1:8090" {
		t.Fatalf("BaseURL() = %q", got)
	}
}

func TestWriteStartupBanner(t *testing.T) {
	cfg := &config.Config{
		Server:   config.ServerConfig{Addr: ":8090"},
		Database: config.DatabaseConfig{DSN: "postgres://x"},
		GFire:    config.GFireConfig{BaseURL: "http://gfire:8080"},
		Auth:     config.AuthConfig{JWTSecret: "secret"},
	}
	var b strings.Builder
	app.WriteStartupBanner(&b, cfg)
	got := b.String()
	for _, want := range []string{"gfireui-backend", "listen :8090", "db postgres", "gfire on", "jwt set"} {
		if !strings.Contains(got, want) {
			t.Fatalf("banner missing %q: %s", want, got)
		}
	}
}
