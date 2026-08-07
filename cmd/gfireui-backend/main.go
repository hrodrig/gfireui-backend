package main

import (
	"log"
	"net/http"
	"os"

	"github.com/hrodrig/gfireui-backend/internal/api"
	"github.com/hrodrig/gfireui-backend/internal/config"
)

func main() {
	cfgPath := os.Getenv("GFIREUI_CONFIG")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("gfireui-backend listening on %s", cfg.Server.Addr)
	log.Fatal(http.ListenAndServe(cfg.Server.Addr, api.NewServer(api.Deps{})))
}
