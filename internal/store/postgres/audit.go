package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

const auditColumns = `id, actor_user_id, action, resource_type, resource_id, ip, user_agent, payload, created_at`

func scanAuditEvent(row interface {
	Scan(dest ...any) error
}) (*domain.AuditEvent, error) {
	var event domain.AuditEvent
	var actorID pgtype.UUID
	var resourceID pgtype.Text
	var ip pgtype.Text
	var userAgent pgtype.Text
	var payload []byte

	if err := row.Scan(
		&event.ID,
		&actorID,
		&event.Action,
		&event.ResourceType,
		&resourceID,
		&ip,
		&userAgent,
		&payload,
		&event.CreatedAt,
	); err != nil {
		return nil, err
	}

	if actorID.Valid {
		id, err := uuid.FromBytes(actorID.Bytes[:])
		if err != nil {
			return nil, fmt.Errorf("decode actor user id: %w", err)
		}
		event.ActorUserID = &id
	}

	if resourceID.Valid {
		value := resourceID.String
		event.ResourceID = &value
	}
	if ip.Valid {
		value := ip.String
		event.IP = &value
	}
	if userAgent.Valid {
		value := userAgent.String
		event.UserAgent = &value
	}
	if len(payload) != 0 {
		event.Payload = append(json.RawMessage(nil), payload...)
	} else {
		event.Payload = json.RawMessage("{}")
	}

	return &event, nil
}

// WriteAudit inserts a new immutable audit event row.
func (s *Store) WriteAudit(ctx context.Context, event *domain.AuditEvent) error {
	if event == nil {
		return fmt.Errorf("audit event is nil")
	}
	if event.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate audit event id: %w", err)
		}
		event.ID = id
	}
	if len(event.Payload) == 0 {
		event.Payload = json.RawMessage("{}")
	}

	row := s.pool.QueryRow(ctx, `
		INSERT INTO audit_events (
			id, actor_user_id, action, resource_type, resource_id, ip, user_agent, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`, event.ID, event.ActorUserID, event.Action, event.ResourceType, event.ResourceID, event.IP, event.UserAgent, event.Payload)

	if err := row.Scan(&event.CreatedAt); err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// ListAudit returns audit events ordered from newest to oldest.
func (s *Store) ListAudit(ctx context.Context, limit, offset int) ([]domain.AuditEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.pool.Query(ctx, `
		SELECT `+auditColumns+`
		FROM audit_events
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.AuditEvent, 0)
	for rows.Next() {
		event, err := scanAuditEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("list audit events: %w", err)
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	return events, nil
}
