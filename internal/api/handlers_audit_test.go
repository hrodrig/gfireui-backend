package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/api"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

func TestAuditAccess(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 6, 20, 0, 0, 0, time.UTC)
	auditEvent := domain.AuditEvent{
		ID:           uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f11"),
		Action:       "auth.login.success",
		ResourceType: "user",
		Payload:      json.RawMessage(`{"email":"ada@example.com"}`),
		CreatedAt:    now,
	}

	tests := []struct {
		name       string
		role       domain.Role
		wantStatus int
		wantError  string
	}{
		{
			name:       "administrator allowed",
			role:       domain.RoleAdministrator,
			wantStatus: http.StatusOK,
		},
		{
			name:       "auditor allowed",
			role:       domain.RoleAuditor,
			wantStatus: http.StatusOK,
		},
		{
			name:       "operator denied",
			role:       domain.RoleOperator,
			wantStatus: http.StatusForbidden,
			wantError:  "forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			user := testRoleUser(t, tt.role)
			token := mustIssueToken(t, user)
			auditStore := &fakeAuditReadStore{events: []domain.AuditEvent{auditEvent}}
			server := api.NewServer(api.Deps{
				Store: &fakeUserStore{
					usersByID: map[uuid.UUID]*domain.User{
						user.ID: user,
					},
				},
				AuditStore: auditStore,
				JWTSecret:  []byte("test-secret"),
				TokenTTL:   time.Hour,
			})

			req := httptest.NewRequest(http.MethodGet, "/api/audit?limit=1&offset=0", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var body []map[string]any
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decode audit response: %v", err)
				}
				if len(body) != 1 {
					t.Fatalf("events = %d, want 1", len(body))
				}
				if got := body[0]["action"]; got != auditEvent.Action {
					t.Fatalf("action = %v, want %q", got, auditEvent.Action)
				}
				if got := body[0]["resource_type"]; got != auditEvent.ResourceType {
					t.Fatalf("resource_type = %v, want %q", got, auditEvent.ResourceType)
				}
				if !auditStore.called {
					t.Fatal("expected audit store to be called")
				}
				if auditStore.limit != 1 || auditStore.offset != 0 {
					t.Fatalf("pagination = (%d,%d), want (1,0)", auditStore.limit, auditStore.offset)
				}
			} else {
				assertErrorResponse(t, rec, tt.wantError)
				if auditStore.called {
					t.Fatal("audit store should not be called for denied role")
				}
			}
		})
	}
}

type fakeAuditReadStore struct {
	events []domain.AuditEvent
	called bool
	limit  int
	offset int
}

func (f *fakeAuditReadStore) ListAudit(_ context.Context, limit, offset int) ([]domain.AuditEvent, error) {
	f.called = true
	f.limit = limit
	f.offset = offset
	return f.events, nil
}

func testRoleUser(t *testing.T, role domain.Role) *domain.User {
	t.Helper()

	return &domain.User{
		ID:           uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f20"),
		FirstName:    "Tess",
		LastName:     "User",
		Email:        "tess@example.com",
		Role:         role,
		Enabled:      true,
		PasswordHash: mustHashPassword(t, "correct-password"),
	}
}
