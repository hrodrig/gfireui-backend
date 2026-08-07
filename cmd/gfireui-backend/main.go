package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hrodrig/gfireui-backend/internal/api"
	"github.com/hrodrig/gfireui-backend/internal/app"
	"github.com/hrodrig/gfireui-backend/internal/bootstrap"
	"github.com/hrodrig/gfireui-backend/internal/config"
	"github.com/hrodrig/gfireui-backend/internal/gfire"
	"github.com/hrodrig/gfireui-backend/internal/store/postgres"
)

func main() {
	cfgPath := os.Getenv("GFIREUI_BACKEND_CONFIG")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfgPath, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cfgPath string, args []string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	switch {
	case len(args) == 0:
		return runServe(ctx, cfg)
	case args[0] == "serve":
		if len(args) > 1 {
			return fmt.Errorf("serve does not accept arguments: %v", args[1:])
		}
		return runServe(ctx, cfg)
	case args[0] == "user":
		return runUser(ctx, cfg, args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runServe(ctx context.Context, cfg *config.Config) error {
	app.LogStartup(cfg)

	deps := api.Deps{
		JWTSecret:          []byte(cfg.Auth.JWTSecret),
		TokenTTL:           cfg.Auth.TokenTTL,
		CORSAllowedOrigins: cfg.Server.CORSOrigins(),
	}

	var store *postgres.Store
	if cfg.Database.DSN != "" {
		var err error
		store, err = postgres.Open(ctx, cfg.Database.DSN)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		defer store.Close()
		deps.Store = store
		deps.AuditStore = store
		deps.Audit = store

		bootstrapped, err := bootstrap.BootstrapAdministrator(ctx, store, cfg.Bootstrap)
		if err != nil {
			return err
		}
		if bootstrapped {
			slog.Info("bootstrapped administrator", "email", cfg.Bootstrap.AdminEmail)
		}
	} else {
		slog.Warn("database.dsn empty — /api/auth/* will fail until configured")
	}

	if strings.TrimSpace(cfg.GFire.BaseURL) == "" {
		slog.Warn("gfire.base_url empty — /api/gfire/* and /api/ops/summary will fail until configured")
	} else {
		client, err := gfire.NewClient(cfg.GFire.BaseURL, cfg.GFire.Token, nil)
		if err != nil {
			return fmt.Errorf("configure gfire client: %w", err)
		}
		deps.GFire = client
		if strings.TrimSpace(cfg.GFire.Token) == "" {
			slog.Warn("gfire.token empty — proxying to GFire without Authorization (auth disabled upstream)")
		}
	}

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           api.NewServer(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		app.LogListening(cfg)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func runUser(ctx context.Context, cfg *config.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing user subcommand")
	}

	switch args[0] {
	case "create":
		return runUserCreate(ctx, cfg, args[1:])
	default:
		return fmt.Errorf("unknown user subcommand %q", args[0])
	}
}

func runUserCreate(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("user create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var input bootstrap.CreateUserInput
	fs.StringVar(&input.Email, "email", "", "")
	fs.StringVar(&input.Password, "password", "", "")
	fs.StringVar(&input.Role, "role", "", "")
	fs.StringVar(&input.FirstName, "first-name", "", "")
	fs.StringVar(&input.LastName, "last-name", "", "")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	if strings.TrimSpace(cfg.Database.DSN) == "" {
		return fmt.Errorf("database.dsn is required for user create")
	}

	store, err := postgres.Open(ctx, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer store.Close()

	created, err := bootstrap.CreateUser(ctx, store, input)
	if err != nil {
		return err
	}

	fmt.Printf("created user %s (%s) id=%s\n", created.Email, created.Role, created.ID)
	return nil
}
