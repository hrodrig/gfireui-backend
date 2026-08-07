package auth

import (
	"context"

	"github.com/hrodrig/gfireui-backend/internal/domain"
)

type userContextKey struct{}

// WithUser stores u in ctx for request-scoped auth helpers.
func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey{}, u)
}

// UserFromContext extracts a user pointer from ctx.
func UserFromContext(ctx context.Context) (*domain.User, bool) {
	u, ok := ctx.Value(userContextKey{}).(*domain.User)
	return u, ok
}
