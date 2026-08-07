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
