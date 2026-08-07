package app

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/hrodrig/gfireui-backend/internal/config"
	"github.com/hrodrig/gfireui-backend/internal/version"
)

// WriteStartupBanner prints a single-line summary to w (always shown, gfire-style).
func WriteStartupBanner(w io.Writer, cfg *config.Config) {
	if cfg == nil {
		return
	}
	fmt.Fprintf(w,
		"gfireui-backend %s | build %s | commit %s | platform %s/%s | listen %s | db %s | gfire %s | jwt %s\n",
		version.Version,
		version.BuildDate,
		version.Commit,
		runtime.GOOS, runtime.GOARCH,
		cfg.Server.Addr,
		dbLabel(cfg.Database.DSN),
		gfireLabel(cfg.GFire.BaseURL, cfg.GFire.Token),
		jwtLabel(cfg.Auth.JWTSecret),
	)
}

// LogStartup writes the banner and a structured summary before the HTTP server starts.
func LogStartup(cfg *config.Config) {
	WriteStartupBanner(os.Stderr, cfg)
	base := BaseURL(cfg)
	slog.Info("gfireui-backend starting",
		"version", version.Version,
		"commit", version.Commit,
		"listen", cfg.Server.Addr,
		"api", base+"/api",
		"health", base+"/healthz",
		"database", dbLabel(cfg.Database.DSN),
		"gfire", gfireLabel(cfg.GFire.BaseURL, cfg.GFire.Token),
		"jwt", jwtLabel(cfg.Auth.JWTSecret),
	)
}

// LogListening emits the structured "listening" line after bind (gfire-style).
func LogListening(cfg *config.Config) {
	base := BaseURL(cfg)
	slog.Info("listening",
		"url", base,
		"api", base+"/api",
		"health", base+"/healthz",
		"bind", cfg.Server.Addr,
	)
}

// BaseURL returns a curl-friendly HTTP origin (localhost when bind-all).
func BaseURL(cfg *config.Config) string {
	host, port, err := net.SplitHostPort(strings.TrimSpace(cfg.Server.Addr))
	if err != nil {
		// Addr may be ":8090"
		if strings.HasPrefix(cfg.Server.Addr, ":") {
			host = "127.0.0.1"
			port = strings.TrimPrefix(cfg.Server.Addr, ":")
			return fmt.Sprintf("http://%s:%s", host, port)
		}
		return "http://127.0.0.1:8090"
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(host, port))
}

func dbLabel(dsn string) string {
	if strings.TrimSpace(dsn) == "" {
		return "off"
	}
	return "postgres"
}

func gfireLabel(baseURL, token string) string {
	if strings.TrimSpace(baseURL) == "" {
		return "off"
	}
	if strings.TrimSpace(token) == "" {
		return "on(no-token)"
	}
	return "on"
}

func jwtLabel(secret string) string {
	if strings.TrimSpace(secret) == "" {
		return "unset"
	}
	return "set"
}
