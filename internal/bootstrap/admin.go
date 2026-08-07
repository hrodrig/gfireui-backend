package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/config"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

// UserStore is the minimal persistence contract needed for bootstrap and CLI user creation.
type UserStore interface {
	ListUsers(context.Context) ([]domain.User, error)
	CreateUser(context.Context, *domain.User) error
}

// CreateUserInput captures the fields required to create a local user.
type CreateUserInput struct {
	Email     string
	Password  string
	Role      string
	FirstName string
	LastName  string
}

// BootstrapAdministrator creates the first Administrator account when the database is empty.
// It returns true when a user was created.
func BootstrapAdministrator(ctx context.Context, store UserStore, cfg config.BootstrapConfig) (bool, error) {
	if err := validateBootstrapConfig(cfg); err != nil {
		return false, err
	}
	if isBootstrapDisabled(cfg) {
		return false, nil
	}
	if store == nil {
		return false, fmt.Errorf("user store is nil")
	}

	users, err := store.ListUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("list users: %w", err)
	}
	if len(users) > 0 {
		return false, nil
	}

	created, err := CreateUser(ctx, store, CreateUserInput{
		Email:     cfg.AdminEmail,
		Password:  cfg.AdminPassword,
		Role:      string(domain.RoleAdministrator),
		FirstName: cfg.AdminFirstName,
		LastName:  cfg.AdminLastName,
	})
	if err != nil {
		return false, err
	}
	_ = created
	return true, nil
}

// CreateUser hashes the password, validates the requested role, and persists a local user.
func CreateUser(ctx context.Context, store UserStore, input CreateUserInput) (*domain.User, error) {
	if store == nil {
		return nil, fmt.Errorf("user store is nil")
	}

	user, err := buildUser(input)
	if err != nil {
		return nil, err
	}
	if err := store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func buildUser(input CreateUserInput) (*domain.User, error) {
	email := strings.TrimSpace(input.Email)
	password := input.Password
	role := domain.Role(strings.TrimSpace(input.Role))
	firstName := strings.TrimSpace(input.FirstName)
	lastName := strings.TrimSpace(input.LastName)

	switch {
	case email == "":
		return nil, fmt.Errorf("email is required")
	case password == "":
		return nil, fmt.Errorf("password is required")
	case firstName == "":
		return nil, fmt.Errorf("first name is required")
	case lastName == "":
		return nil, fmt.Errorf("last name is required")
	case !role.Valid():
		return nil, fmt.Errorf("invalid role %q", role)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	return &domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		FirstName:    firstName,
		LastName:     lastName,
		Enabled:      true,
	}, nil
}

func validateBootstrapConfig(cfg config.BootstrapConfig) error {
	values := []string{
		strings.TrimSpace(cfg.AdminEmail),
		strings.TrimSpace(cfg.AdminPassword),
		strings.TrimSpace(cfg.AdminFirstName),
		strings.TrimSpace(cfg.AdminLastName),
	}
	anySet := false
	allSet := true
	for _, value := range values {
		if value == "" {
			allSet = false
			continue
		}
		anySet = true
	}
	if anySet && !allSet {
		return fmt.Errorf("bootstrap admin requires admin_email, admin_password, admin_first_name, and admin_last_name")
	}
	return nil
}

func isBootstrapDisabled(cfg config.BootstrapConfig) bool {
	return strings.TrimSpace(cfg.AdminEmail) == "" &&
		strings.TrimSpace(cfg.AdminPassword) == "" &&
		strings.TrimSpace(cfg.AdminFirstName) == "" &&
		strings.TrimSpace(cfg.AdminLastName) == ""
}
