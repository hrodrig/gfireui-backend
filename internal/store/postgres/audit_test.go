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
	dsn := os.Getenv("GFIREUI_TEST_DSN")
	if dsn == "" {
		t.Skip("GFIREUI_TEST_DSN unset")
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

	listed, err := store.ListAudit(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("events = %d, want 2", len(listed))
	}
	if listed[0].Action != second.Action {
		t.Fatalf("first listed action = %q, want %q", listed[0].Action, second.Action)
	}
	if listed[1].Action != first.Action {
		t.Fatalf("second listed action = %q, want %q", listed[1].Action, first.Action)
	}
	if listed[0].ActorUserID == nil || *listed[0].ActorUserID != actor.ID {
		t.Fatalf("first listed actor = %v, want %s", listed[0].ActorUserID, actor.ID)
	}
	if got := string(listed[1].Payload); got != `{"email":"ada@example.com"}` {
		t.Fatalf("payload = %s", got)
	}

	paginated, err := store.ListAudit(ctx, 1, 1)
	if err != nil {
		t.Fatalf("list audit paginated: %v", err)
	}
	if len(paginated) != 1 {
		t.Fatalf("paginated events = %d, want 1", len(paginated))
	}
	if paginated[0].Action != first.Action {
		t.Fatalf("paginated action = %q, want %q", paginated[0].Action, first.Action)
	}
}
