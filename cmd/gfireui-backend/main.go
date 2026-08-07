package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hrodrig/gfireui-backend/internal/api"
	"github.com/hrodrig/gfireui-backend/internal/config"
	"github.com/hrodrig/gfireui-backend/internal/gfire"
	"github.com/hrodrig/gfireui-backend/internal/store/postgres"
)

func main() {
	cfgPath := os.Getenv("GFIREUI_CONFIG")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := api.Deps{
		JWTSecret: []byte(cfg.Auth.JWTSecret),
		TokenTTL:  cfg.Auth.TokenTTL,
	}

	if cfg.Database.DSN != "" {
		store, err := postgres.Open(ctx, cfg.Database.DSN)
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		defer store.Close()
		deps.Store = store
		deps.AuditStore = store
		deps.Audit = store
	} else {
		log.Printf("warning: database.dsn empty — /api/auth/* will fail until configured")
	}

	if cfg.GFire.BaseURL == "" && cfg.GFire.Token == "" {
		log.Printf("warning: gfire.base_url and gfire.token empty — /api/gfire/* will fail until configured")
	} else {
		client, err := gfire.NewClient(cfg.GFire.BaseURL, cfg.GFire.Token, nil)
		if err != nil {
			log.Fatalf("configure gfire client: %v", err)
		}
		deps.GFire = client
	}

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           api.NewServer(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("gfireui-backend listening on %s", cfg.Server.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
