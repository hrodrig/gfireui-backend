package api

import (
	"context"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/domain"
)

func TestNoopAuditWriter(t *testing.T) {
	t.Parallel()
	var w noopAuditWriter
	if err := w.WriteAudit(context.Background(), &domain.AuditEvent{Action: "x"}); err != nil {
		t.Fatal(err)
	}
}
