package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditEvent is an append-only audit log entry. New events should use uuid.NewV7() for ID.
type AuditEvent struct {
	ID           uuid.UUID
	ActorUserID  *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *string
	IP           *string
	UserAgent    *string
	Payload      json.RawMessage
	CreatedAt    time.Time
}
