package api

import (
	"net/http"

	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

// RequireRoles allows requests only when the authenticated user has one of the permitted roles.
func RequireRoles(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.UserFromContext(r.Context())
			if !ok || user == nil {
				writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
				return
			}

			if !hasAnyRole(user.Role, roles...) {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hasAnyRole(role domain.Role, allowed ...domain.Role) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}
