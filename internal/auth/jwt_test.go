package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

func TestIssueTokenAndParseToken(t *testing.T) {
	id := uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f0f")
	user := &domain.User{
		ID:    id,
		Email: "ada@example.com",
		Role:  domain.RoleAdministrator,
	}

	token, err := auth.IssueToken([]byte("secret"), time.Minute, user)
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("IssueToken() returned an empty token")
	}

	claims, err := auth.ParseToken([]byte("secret"), token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.Sub != id {
		t.Fatalf("Sub = %s, want %s", claims.Sub, id)
	}
	if claims.Email != user.Email {
		t.Fatalf("Email = %q, want %q", claims.Email, user.Email)
	}
	if claims.Role != user.Role {
		t.Fatalf("Role = %q, want %q", claims.Role, user.Role)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	user := &domain.User{
		ID:    uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f0f"),
		Email: "ada@example.com",
		Role:  domain.RoleAdministrator,
	}

	token, err := auth.IssueToken([]byte("secret"), time.Minute, user)
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}

	if _, err := auth.ParseToken([]byte("other-secret"), token); err == nil {
		t.Fatal("ParseToken() succeeded with the wrong secret")
	}
}
