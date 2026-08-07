package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/domain"
	"github.com/hrodrig/gfireui-backend/internal/store/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ensureAuditSchema(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS audit_events (
			id UUID PRIMARY KEY,
			actor_user_id UUID NULL REFERENCES users(id),
			action TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			resource_id TEXT NULL,
			ip TEXT NULL,
			user_agent TEXT NULL,
			payload JSONB NOT NULL DEFAULT '{}',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return err
	}

	_, err = pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS audit_events_created_at_idx
		ON audit_events (created_at DESC)
	`)
	return err
}

func TestWriteAuditAndListAudit(t *testing.T) {
	dsn := os.Getenv("GFIREUI_BACKEND_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_BACKEND_TEST_DSN unset")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := ensureUsersSchema(ctx, dsn); err != nil {
		t.Fatalf("ensure users schema: %v", err)
	}
	if err := ensureAuditSchema(ctx, dsn); err != nil {
		t.Fatalf("ensure audit schema: %v", err)
	}

	store, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(store.Close)

	actorEmail := fmt.Sprintf("auditor-%s@example.com", uuid.NewString())
	actor := &domain.User{
		FirstName:    "Audit",
		LastName:     "Actor",
		Email:        actorEmail,
		Role:         domain.RoleAuditor,
		Enabled:      true,
		PasswordHash: "hash",
	}
	if err := store.CreateUser(ctx, actor); err != nil {
		t.Fatalf("create actor: %v", err)
	}

	resourceID := "user-123"
	ip := "203.0.113.10"
	userAgent := "gfireui-backend-test"

	first := &domain.AuditEvent{
		ActorUserID:  &actor.ID,
		Action:       "auth.login.success",
		ResourceType: "user",
		ResourceID:   &resourceID,
		IP:           &ip,
		UserAgent:    &userAgent,
		Payload:      json.RawMessage(`{"email":"ada@example.com"}`),
	}
	second := &domain.AuditEvent{
		ActorUserID:  &actor.ID,
		Action:       "user.update",
		ResourceType: "user",
		Payload:      json.RawMessage(`{"field":"enabled"}`),
	}

	if err := store.WriteAudit(ctx, first); err != nil {
		t.Fatalf("write first audit: %v", err)
	}
	if first.ID == uuid.Nil {
		t.Fatal("expected WriteAudit to assign an ID")
	}
	if first.CreatedAt.IsZero() {
		t.Fatal("expected WriteAudit to set CreatedAt")
	}

	if err := store.WriteAudit(ctx, second); err != nil {
		t.Fatalf("write second audit: %v", err)
	}
	if second.ID == uuid.Nil {
		t.Fatal("expected WriteAudit to assign an ID")
	}

	listed, err := store.ListAudit(ctx, 100, 0)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	mine := filterAuditByActor(listed, actor.ID)
	if len(mine) < 2 {
		t.Fatalf("actor events = %d, want at least 2", len(mine))
	}
	if mine[0].Action != second.Action {
		t.Fatalf("newest action = %q, want %q", mine[0].Action, second.Action)
	}
	if mine[1].Action != first.Action {
		t.Fatalf("older action = %q, want %q", mine[1].Action, first.Action)
	}
	var payload map[string]any
	if err := json.Unmarshal(mine[1].Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["email"] != "ada@example.com" {
		t.Fatalf("payload = %s", mine[1].Payload)
	}

	// limit<=0 and offset<0 take defaults inside ListAudit
	if _, err := store.ListAudit(ctx, 0, -1); err != nil {
		t.Fatalf("list audit defaults: %v", err)
	}
}

func filterAuditByActor(events []domain.AuditEvent, actorID uuid.UUID) []domain.AuditEvent {
	out := make([]domain.AuditEvent, 0, len(events))
	for _, event := range events {
		if event.ActorUserID != nil && *event.ActorUserID == actorID {
			out = append(out, event)
		}
	}
	return out
}
