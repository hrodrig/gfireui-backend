package main

import (
	"log"
	"net/http"

	"github.com/hrodrig/gfireui-backend/internal/api"
)

func main() {
	addr := ":8090"
	log.Printf("gfireui-backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, api.NewServer(api.Deps{})))
}
