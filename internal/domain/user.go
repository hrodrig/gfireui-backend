package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is a local console account. New users should use uuid.NewV7() for ID.
type User struct {
	ID           uuid.UUID
	FirstName    string
	LastName     string
	Email        string
	Role         Role
	Enabled      bool
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
