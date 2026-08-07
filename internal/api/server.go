package api

import (
	"encoding/json"
	"net/http"
)

type Deps struct{}

type Server struct {
	mux *http.ServeMux
}

func NewServer(_ Deps) http.Handler {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	return s.mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
