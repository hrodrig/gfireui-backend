package config_test

import (
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/config"
)

func TestLoad_EnvOverridesAddr(t *testing.T) {
	t.Setenv("GFIREUI_BACKEND_SERVER_ADDR", "127.0.0.1:9090")
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Addr != "127.0.0.1:9090" {
		t.Fatalf("got %q", cfg.Server.Addr)
	}
}

func TestLoad_EnvCORSAllowedOrigins(t *testing.T) {
	t.Setenv("GFIREUI_BACKEND_SERVER_CORS_ALLOWED_ORIGINS", "http://127.0.0.1:5173, http://localhost:5173")
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.Server.CORSOrigins()
	if len(got) != 2 || got[0] != "http://127.0.0.1:5173" || got[1] != "http://localhost:5173" {
		t.Fatalf("got %#v", got)
	}
}
