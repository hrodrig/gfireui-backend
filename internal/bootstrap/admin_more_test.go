package bootstrap_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/bootstrap"
	"github.com/hrodrig/gfireui-backend/internal/config"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

func TestBootstrapAdministratorValidationAndDisabled(t *testing.T) {
	t.Parallel()

	t.Run("disabled when empty", func(t *testing.T) {
		t.Parallel()
		created, err := bootstrap.BootstrapAdministrator(context.Background(), &fakeUserStore{}, config.BootstrapConfig{})
		if err != nil || created {
			t.Fatalf("created=%v err=%v", created, err)
		}
	})

	t.Run("partial config", func(t *testing.T) {
		t.Parallel()
		_, err := bootstrap.BootstrapAdministrator(context.Background(), &fakeUserStore{}, config.BootstrapConfig{
			AdminEmail: "admin@example.com",
		})
		if err == nil {
			t.Fatal("expected validation error")
		}
	})

	t.Run("nil store", func(t *testing.T) {
		t.Parallel()
		_, err := bootstrap.BootstrapAdministrator(context.Background(), nil, config.BootstrapConfig{
			AdminEmail:     "admin@example.com",
			AdminPassword:  "secret-password",
			AdminFirstName: "Ada",
			AdminLastName:  "Lovelace",
		})
		if err == nil {
			t.Fatal("expected nil store error")
		}
	})
}

func TestCreateUserValidation(t *testing.T) {
	t.Parallel()

	store := &fakeUserStore{}
	cases := []bootstrap.CreateUserInput{
		{Password: "x", Role: "Operator", FirstName: "A", LastName: "B"},
		{Email: "a@b.c", Role: "Operator", FirstName: "A", LastName: "B"},
		{Email: "a@b.c", Password: "x", Role: "Operator", LastName: "B"},
		{Email: "a@b.c", Password: "x", Role: "Operator", FirstName: "A"},
		{Email: "a@b.c", Password: "x", Role: "Nope", FirstName: "A", LastName: "B"},
	}
	for _, input := range cases {
		if _, err := bootstrap.CreateUser(context.Background(), store, input); err == nil {
			t.Fatalf("expected error for %#v", input)
		}
	}

	if _, err := bootstrap.CreateUser(context.Background(), nil, bootstrap.CreateUserInput{
		Email: "a@b.c", Password: "secret", Role: "Operator", FirstName: "A", LastName: "B",
	}); err == nil {
		t.Fatal("expected nil store error")
	}

	created, err := bootstrap.CreateUser(context.Background(), store, bootstrap.CreateUserInput{
		Email: "op@example.com", Password: "secret-password", Role: "Operator", FirstName: "Op", LastName: "User",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Role != domain.RoleOperator || !created.Enabled {
		t.Fatalf("created = %#v", created)
	}
}

type errListStore struct{ fakeUserStore }

func (e *errListStore) ListUsers(context.Context) ([]domain.User, error) {
	return nil, errors.New("boom")
}

func TestBootstrapAdministratorListError(t *testing.T) {
	t.Parallel()
	_, err := bootstrap.BootstrapAdministrator(context.Background(), &errListStore{}, config.BootstrapConfig{
		AdminEmail:     "admin@example.com",
		AdminPassword:  "secret-password",
		AdminFirstName: "Ada",
		AdminLastName:  "Lovelace",
	})
	if err == nil {
		t.Fatal("expected list error")
	}
}
