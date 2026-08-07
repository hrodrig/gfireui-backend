package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

const defaultTokenTTL = 24 * time.Hour

// UserStore is the minimal user persistence contract required by auth handlers.
type UserStore interface {
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// AuditWriter records auth events. It can be a no-op until a persistent audit store exists.
type AuditWriter interface {
	WriteAudit(ctx context.Context, event *domain.AuditEvent) error
}

// Deps are the runtime dependencies required by the HTTP server.
type Deps struct {
	Store     UserStore
	JWTSecret []byte
	TokenTTL  time.Duration
	Audit     AuditWriter
}

type Server struct {
	mux  *http.ServeMux
	deps Deps
}

func NewServer(deps Deps) http.Handler {
	if deps.TokenTTL <= 0 {
		deps.TokenTTL = defaultTokenTTL
	}
	if deps.Audit == nil {
		deps.Audit = noopAuditWriter{}
	}

	s := &Server{
		mux:  http.NewServeMux(),
		deps: deps,
	}
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.Handle("GET /api/auth/me", s.authMiddleware(http.HandlerFunc(s.handleMe)))
	return s.mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type noopAuditWriter struct{}

func (noopAuditWriter) WriteAudit(context.Context, *domain.AuditEvent) error {
	return nil
}
