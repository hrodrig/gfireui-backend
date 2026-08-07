package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/api"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

func TestUsersAdminAPI(t *testing.T) {
	t.Parallel()

	admin := testRoleUser(t, domain.RoleAdministrator)
	other := &domain.User{
		ID:           uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f21"),
		FirstName:    "Grace",
		LastName:     "Hopper",
		Email:        "grace@example.com",
		Role:         domain.RoleOperator,
		Enabled:      true,
		PasswordHash: mustHashPassword(t, "correct-password"),
	}

	t.Run("list users", func(t *testing.T) {
		t.Parallel()

		store := &fakeUserStore{
			usersByEmail: map[string]*domain.User{
				admin.Email: admin,
				other.Email: other,
			},
			usersByID: map[uuid.UUID]*domain.User{
				admin.ID: admin,
				other.ID: other,
			},
		}

		rec := performAdminRequest(t, store, admin, http.MethodGet, "/api/users", nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var body []map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode users list: %v", err)
		}
		if len(body) != 2 {
			t.Fatalf("users = %d, want 2", len(body))
		}
		for _, user := range body {
			if _, ok := user["password_hash"]; ok {
				t.Fatal("password_hash should not be present")
			}
		}
	})

	t.Run("create user", func(t *testing.T) {
		t.Parallel()

		store := &fakeUserStore{
			usersByEmail: map[string]*domain.User{
				admin.Email: admin,
			},
			usersByID: map[uuid.UUID]*domain.User{
				admin.ID: admin,
			},
		}
		auditWriter := &fakeAuditWriter{}

		body := `{"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com","role":"Operator","enabled":true,"password":"secret-password"}`
		rec := performAdminRequestWithAudit(t, store, auditWriter, admin, http.MethodPost, "/api/users", bytes.NewBufferString(body))

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
		}

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("decode create response: %v", err)
		}
		if response["email"] != "ada@example.com" {
			t.Fatalf("email = %v, want ada@example.com", response["email"])
		}
		if _, ok := response["password_hash"]; ok {
			t.Fatal("password_hash should not be present")
		}

		created := store.usersByEmail["ada@example.com"]
		if created == nil {
			t.Fatal("expected user to be created")
		}
		if created.PasswordHash == "secret-password" {
			t.Fatal("password should be hashed")
		}
		if len(auditWriter.events) != 1 || auditWriter.events[0].Action != "users.create" {
			t.Fatalf("audit events = %#v", auditWriter.events)
		}
	})

	t.Run("get user", func(t *testing.T) {
		t.Parallel()

		store := &fakeUserStore{
			usersByEmail: map[string]*domain.User{
				admin.Email: admin,
				other.Email: other,
			},
			usersByID: map[uuid.UUID]*domain.User{
				admin.ID: admin,
				other.ID: other,
			},
		}

		rec := performAdminRequest(t, store, admin, http.MethodGet, "/api/users/"+other.ID.String(), nil)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("decode get response: %v", err)
		}
		if response["email"] != other.Email {
			t.Fatalf("email = %v, want %s", response["email"], other.Email)
		}
		if _, ok := response["password_hash"]; ok {
			t.Fatal("password_hash should not be present")
		}
	})

	t.Run("patch user", func(t *testing.T) {
		t.Parallel()

		store := &fakeUserStore{
			usersByEmail: map[string]*domain.User{
				admin.Email: admin,
				other.Email: other,
			},
			usersByID: map[uuid.UUID]*domain.User{
				admin.ID: admin,
				other.ID: other,
			},
		}
		auditWriter := &fakeAuditWriter{}

		body := `{"first_name":"Grace","last_name":"Hopper","email":"grace.ada@example.com","role":"Auditor","enabled":false}`
		rec := performAdminRequestWithAudit(t, store, auditWriter, admin, http.MethodPatch, "/api/users/"+other.ID.String(), bytes.NewBufferString(body))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		updated := store.usersByID[other.ID]
		if updated.Email != "grace.ada@example.com" || updated.Role != domain.RoleAuditor || updated.Enabled {
			t.Fatalf("updated user = %#v", updated)
		}
		if len(auditWriter.events) != 1 || auditWriter.events[0].Action != "users.update" {
			t.Fatalf("audit events = %#v", auditWriter.events)
		}
	})

	t.Run("update password", func(t *testing.T) {
		t.Parallel()

		store := &fakeUserStore{
			usersByEmail: map[string]*domain.User{
				admin.Email: admin,
				other.Email: other,
			},
			usersByID: map[uuid.UUID]*domain.User{
				admin.ID: admin,
				other.ID: other,
			},
		}
		auditWriter := &fakeAuditWriter{}
		previousHash := other.PasswordHash

		body := `{"password":"new-secret-password"}`
		rec := performAdminRequestWithAudit(t, store, auditWriter, admin, http.MethodPost, "/api/users/"+other.ID.String()+"/password", bytes.NewBufferString(body))

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}

		updated := store.usersByID[other.ID]
		if updated.PasswordHash == previousHash {
			t.Fatal("password hash should change")
		}
		if len(auditWriter.events) != 1 || auditWriter.events[0].Action != "users.password.update" {
			t.Fatalf("audit events = %#v", auditWriter.events)
		}
	})
}

func TestRBACMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		role       domain.Role
		method     string
		path       string
		wantStatus int
		wantError  string
	}{
		{name: "administrator can list users", role: domain.RoleAdministrator, method: http.MethodGet, path: "/api/users", wantStatus: http.StatusOK},
		{name: "operator cannot list users", role: domain.RoleOperator, method: http.MethodGet, path: "/api/users", wantStatus: http.StatusForbidden, wantError: "forbidden"},
		{name: "auditor cannot list users", role: domain.RoleAuditor, method: http.MethodGet, path: "/api/users", wantStatus: http.StatusForbidden, wantError: "forbidden"},
		{name: "guest cannot list users", role: domain.RoleGuest, method: http.MethodGet, path: "/api/users", wantStatus: http.StatusForbidden, wantError: "forbidden"},
		{name: "administrator can read audit", role: domain.RoleAdministrator, method: http.MethodGet, path: "/api/audit", wantStatus: http.StatusOK},
		{name: "auditor can read audit", role: domain.RoleAuditor, method: http.MethodGet, path: "/api/audit", wantStatus: http.StatusOK},
		{name: "operator cannot read audit", role: domain.RoleOperator, method: http.MethodGet, path: "/api/audit", wantStatus: http.StatusForbidden, wantError: "forbidden"},
		{name: "guest can reach me", role: domain.RoleGuest, method: http.MethodGet, path: "/api/auth/me", wantStatus: http.StatusOK},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			user := testRoleUser(t, tt.role)
			rec := performAdminRequest(t, &fakeUserStore{
				usersByEmail: map[string]*domain.User{
					user.Email: user,
				},
				usersByID: map[uuid.UUID]*domain.User{
					user.ID: user,
				},
			}, user, tt.method, tt.path, nil)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantError != "" {
				assertErrorResponse(t, rec, tt.wantError)
			}
		})
	}
}

func performAdminRequest(t *testing.T, store *fakeUserStore, actor *domain.User, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	return performAdminRequestWithAudit(t, store, nil, actor, method, path, body)
}

func performAdminRequestWithAudit(t *testing.T, store *fakeUserStore, auditWriter *fakeAuditWriter, actor *domain.User, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	deps := api.Deps{
		Store:      store,
		AuditStore: &fakeAuditReadStore{},
		JWTSecret:  []byte("test-secret"),
		TokenTTL:   0,
	}
	if auditWriter != nil {
		deps.Audit = auditWriter
	}

	server := api.NewServer(deps)
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+mustIssueToken(t, actor))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return rec
}
