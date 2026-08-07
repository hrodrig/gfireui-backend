package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/hrodrig/gfireui-backend/internal/domain"
	"github.com/hrodrig/gfireui-backend/internal/store"
)

const userColumns = `id, first_name, last_name, email, role, enabled, password_hash, created_at, updated_at`

func newUserID() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate user id: %w", err)
	}
	return id, nil
}

func scanUser(row interface {
	Scan(dest ...any) error
}) (*domain.User, error) {
	var u domain.User
	if err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.Role,
		&u.Enabled,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts a new user row and sets the generated UUIDv7 on zero IDs.
func (s *Store) CreateUser(ctx context.Context, u *domain.User) error {
	if u == nil {
		return fmt.Errorf("user is nil")
	}
	if u.ID == uuid.Nil {
		id, err := newUserID()
		if err != nil {
			return err
		}
		u.ID = id
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (
			id, first_name, last_name, email, role, enabled, password_hash
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`, u.ID, u.FirstName, u.LastName, u.Email, u.Role, u.Enabled, u.PasswordHash)

	if err := row.Scan(&u.CreatedAt, &u.UpdatedAt); err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetUserByEmail looks up a user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// GetUserByID looks up a user by UUID.
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// ListUsers returns all users ordered by creation time.
func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("list users: %w", err)
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// UpdateUser updates a user row and refreshes the updated_at timestamp.
func (s *Store) UpdateUser(ctx context.Context, u *domain.User) error {
	if u == nil {
		return fmt.Errorf("user is nil")
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE users
		SET first_name = $2,
			last_name = $3,
			email = $4,
			role = $5,
			enabled = $6,
			password_hash = $7,
			updated_at = now()
		WHERE id = $1
		RETURNING created_at, updated_at
	`, u.ID, u.FirstName, u.LastName, u.Email, u.Role, u.Enabled, u.PasswordHash)

	if err := row.Scan(&u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrNotFound
		}
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}
