package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/store"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.deps.Store == nil {
			writeError(w, http.StatusInternalServerError, "auth store is not configured")
			return
		}
		if len(s.deps.JWTSecret) == 0 {
			writeError(w, http.StatusInternalServerError, "jwt secret is not configured")
			return
		}

		token, ok := bearerTokenFromHeader(r.Header.Get("Authorization"))
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}

		claims, err := auth.ParseToken(s.deps.JWTSecret, token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}

		user, err := s.deps.Store.GetUserByID(r.Context(), claims.Sub)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load user")
			return
		}

		if !user.Enabled {
			writeError(w, http.StatusForbidden, "user is disabled")
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), user)))
	})
}

func bearerTokenFromHeader(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
