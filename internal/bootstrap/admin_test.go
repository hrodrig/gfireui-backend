package bootstrap_test

import (
	"context"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/bootstrap"
	"github.com/hrodrig/gfireui-backend/internal/config"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

type fakeUserStore struct {
	users       []domain.User
	createCalls int
	created     *domain.User
}

func (f *fakeUserStore) ListUsers(context.Context) ([]domain.User, error) {
	out := make([]domain.User, len(f.users))
	copy(out, f.users)
	return out, nil
}

func (f *fakeUserStore) CreateUser(_ context.Context, user *domain.User) error {
	f.createCalls++
	copyUser := *user
	f.created = &copyUser
	return nil
}

func TestBootstrapAdministratorCreatesOnlyWhenEmpty(t *testing.T) {
	t.Run("creates admin when empty", func(t *testing.T) {
		store := &fakeUserStore{}
		created, err := bootstrap.BootstrapAdministrator(context.Background(), store, config.BootstrapConfig{
			AdminEmail:     "admin@example.com",
			AdminPassword:  "secret-password",
			AdminFirstName: "Ada",
			AdminLastName:  "Lovelace",
		})
		if err != nil {
			t.Fatalf("bootstrap admin: %v", err)
		}
		if !created {
			t.Fatal("expected bootstrap to create admin")
		}
		if store.createCalls != 1 {
			t.Fatalf("create calls = %d, want 1", store.createCalls)
		}
		if store.created == nil {
			t.Fatal("expected created user")
		}
		if store.created.Role != domain.RoleAdministrator {
			t.Fatalf("role = %q, want %q", store.created.Role, domain.RoleAdministrator)
		}
		if !auth.CheckPassword(store.created.PasswordHash, "secret-password") {
			t.Fatal("password hash does not match bootstrap password")
		}
	})

	t.Run("skips when users already exist", func(t *testing.T) {
		store := &fakeUserStore{
			users: []domain.User{{Email: "existing@example.com"}},
		}
		created, err := bootstrap.BootstrapAdministrator(context.Background(), store, config.BootstrapConfig{
			AdminEmail:     "admin@example.com",
			AdminPassword:  "secret-password",
			AdminFirstName: "Ada",
			AdminLastName:  "Lovelace",
		})
		if err != nil {
			t.Fatalf("bootstrap admin: %v", err)
		}
		if created {
			t.Fatal("expected bootstrap to skip existing database")
		}
		if store.createCalls != 0 {
			t.Fatalf("create calls = %d, want 0", store.createCalls)
		}
	})
}
