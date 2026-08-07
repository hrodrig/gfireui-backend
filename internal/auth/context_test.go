package auth_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

func TestWithUserAndUserFromContext(t *testing.T) {
	user := &domain.User{ID: uuid.New(), Email: "ada@example.com"}
	ctx := auth.WithUser(context.Background(), user)

	got, ok := auth.UserFromContext(ctx)
	if !ok {
		t.Fatal("UserFromContext() = false, want true")
	}
	if got != user {
		t.Fatal("UserFromContext() returned a different pointer")
	}
}

func TestUserFromContextMissingUser(t *testing.T) {
	if got, ok := auth.UserFromContext(context.Background()); ok || got != nil {
		t.Fatal("UserFromContext() found a user in an empty context")
	}
}
