package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hrodrig/gfireui-backend/internal/config"
)

func TestLoadFromFileAndGFIREUIConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(path, []byte(`
server:
  addr: "127.0.0.1:7070"
auth:
  jwt_secret: "from-file"
  token_ttl: 2h
gfire:
  base_url: "http://gfire:8080"
  token: "svc"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Addr != "127.0.0.1:7070" || cfg.Auth.JWTSecret != "from-file" || cfg.Auth.TokenTTL != 2*time.Hour {
		t.Fatalf("cfg = %#v", cfg)
	}
	if cfg.GFire.BaseURL != "http://gfire:8080" || cfg.GFire.Token != "svc" {
		t.Fatalf("gfire = %#v", cfg.GFire)
	}

	t.Setenv("GFIREUI_CONFIG", path)
	t.Setenv("GFIREUI_SERVER_ADDR", "")
	cfg2, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Auth.JWTSecret != "from-file" {
		t.Fatalf("via GFIREUI_CONFIG: %#v", cfg2)
	}

	if _, err := config.Load(filepath.Join(dir, "missing.yaml")); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()
	d := config.Defaults()
	if d.Server.Addr != ":8090" || d.Auth.TokenTTL != 24*time.Hour {
		t.Fatalf("defaults = %#v", d)
	}
}
