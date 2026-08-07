package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/api"
	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/domain"
	"github.com/hrodrig/gfireui-backend/internal/store"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	hash := mustHashPassword(t, "correct-password")
	activeUser := testUser(hash, true)
	disabledUser := testUser(hash, false)

	tests := []struct {
		name            string
		body            string
		usersByEmail    map[string]*domain.User
		wantStatus      int
		wantError       string
		wantAuditAction string
		wantAuditReason string
		wantActorID     *uuid.UUID
		wantToken       bool
	}{
		{
			name: "success",
			body: `{"email":"ada@example.com","password":"correct-password"}`,
			usersByEmail: map[string]*domain.User{
				activeUser.Email: activeUser,
			},
			wantStatus:      http.StatusOK,
			wantAuditAction: "auth.login.success",
			wantActorID:     &activeUser.ID,
			wantToken:       true,
		},
		{
			name: "unknown email",
			body: `{"email":"missing@example.com","password":"correct-password"}`,
			usersByEmail: map[string]*domain.User{
				activeUser.Email: activeUser,
			},
			wantStatus:      http.StatusUnauthorized,
			wantError:       "invalid credentials",
			wantAuditAction: "auth.login.failure",
			wantAuditReason: "invalid_credentials",
		},
		{
			name: "invalid password",
			body: `{"email":"ada@example.com","password":"wrong-password"}`,
			usersByEmail: map[string]*domain.User{
				activeUser.Email: activeUser,
			},
			wantStatus:      http.StatusUnauthorized,
			wantError:       "invalid credentials",
			wantAuditAction: "auth.login.failure",
			wantAuditReason: "invalid_credentials",
			wantActorID:     &activeUser.ID,
		},
		{
			name: "disabled user",
			body: `{"email":"ada@example.com","password":"correct-password"}`,
			usersByEmail: map[string]*domain.User{
				disabledUser.Email: disabledUser,
			},
			wantStatus:      http.StatusForbidden,
			wantError:       "user is disabled",
			wantAuditAction: "auth.login.failure",
			wantAuditReason: "disabled",
			wantActorID:     &disabledUser.ID,
		},
		{
			name:       "invalid json",
			body:       `{"email":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			auditWriter := &fakeAuditWriter{}
			server := api.NewServer(api.Deps{
				Store: &fakeUserStore{
					usersByEmail: tt.usersByEmail,
					usersByID:    indexUsersByID(tt.usersByEmail),
				},
				JWTSecret: []byte("test-secret"),
				TokenTTL:  time.Hour,
				Audit:     auditWriter,
			})

			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "handlers-auth-test")
			req.RemoteAddr = "203.0.113.10:1234"
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var body struct {
					Token string                 `json:"token"`
					User  map[string]interface{} `json:"user"`
				}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decode login response: %v", err)
				}
				if body.Token == "" {
					t.Fatal("expected a token")
				}
				claims, err := auth.ParseToken([]byte("test-secret"), body.Token)
				if err != nil {
					t.Fatalf("parse token: %v", err)
				}
				if claims.Sub != activeUser.ID {
					t.Fatalf("claims.Sub = %s, want %s", claims.Sub, activeUser.ID)
				}
				if got := body.User["email"]; got != activeUser.Email {
					t.Fatalf("user.email = %v, want %q", got, activeUser.Email)
				}
				if got := body.User["role"]; got != string(activeUser.Role) {
					t.Fatalf("user.role = %v, want %q", got, activeUser.Role)
				}
				if _, ok := body.User["password_hash"]; ok {
					t.Fatal("user.password_hash should not be present")
				}
			} else {
				assertErrorResponse(t, rec, tt.wantError)
			}

			assertAudit(t, auditWriter.events, tt.wantAuditAction, tt.wantAuditReason, tt.wantActorID)
		})
	}
}

func TestMe(t *testing.T) {
	t.Parallel()

	hash := mustHashPassword(t, "correct-password")
	activeUser := testUser(hash, true)
	disabledUser := testUser(hash, false)
	unknownUser := &domain.User{
		ID:           uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f10"),
		FirstName:    "Grace",
		LastName:     "Hopper",
		Email:        "grace@example.com",
		Role:         domain.RoleOperator,
		Enabled:      true,
		PasswordHash: hash,
	}

	activeToken := mustIssueToken(t, activeUser)
	disabledToken := mustIssueToken(t, disabledUser)
	unknownUserToken := mustIssueToken(t, unknownUser)

	tests := []struct {
		name         string
		authHeader   string
		usersByEmail map[string]*domain.User
		usersByID    map[uuid.UUID]*domain.User
		wantStatus   int
		wantError    string
	}{
		{
			name:       "missing bearer token",
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing or invalid bearer token",
		},
		{
			name:       "invalid bearer format",
			authHeader: "Basic abc123",
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing or invalid bearer token",
		},
		{
			name:       "invalid token",
			authHeader: "Bearer definitely-not-a-jwt",
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing or invalid bearer token",
		},
		{
			name:       "token user not found",
			authHeader: "Bearer " + unknownUserToken,
			usersByID: map[uuid.UUID]*domain.User{
				activeUser.ID: activeUser,
			},
			wantStatus: http.StatusUnauthorized,
			wantError:  "missing or invalid bearer token",
		},
		{
			name:       "disabled user",
			authHeader: "Bearer " + disabledToken,
			usersByID: map[uuid.UUID]*domain.User{
				disabledUser.ID: disabledUser,
			},
			wantStatus: http.StatusForbidden,
			wantError:  "user is disabled",
		},
		{
			name:       "success",
			authHeader: "Bearer " + activeToken,
			usersByID: map[uuid.UUID]*domain.User{
				activeUser.ID: activeUser,
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := api.NewServer(api.Deps{
				Store: &fakeUserStore{
					usersByEmail: tt.usersByEmail,
					usersByID:    tt.usersByID,
				},
				JWTSecret: []byte("test-secret"),
				TokenTTL:  time.Hour,
			})

			req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var body map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decode me response: %v", err)
				}
				if got := body["email"]; got != activeUser.Email {
					t.Fatalf("user.email = %v, want %q", got, activeUser.Email)
				}
				if _, ok := body["password_hash"]; ok {
					t.Fatal("user.password_hash should not be present")
				}
			} else {
				assertErrorResponse(t, rec, tt.wantError)
			}
		})
	}
}

type fakeUserStore struct {
	usersByEmail map[string]*domain.User
	usersByID    map[uuid.UUID]*domain.User
}

func (f *fakeUserStore) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	if user, ok := f.usersByEmail[email]; ok {
		return user, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeUserStore) GetUserByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if user, ok := f.usersByID[id]; ok {
		return user, nil
	}
	return nil, store.ErrNotFound
}

type fakeAuditWriter struct {
	events []*domain.AuditEvent
}

func (f *fakeAuditWriter) WriteAudit(_ context.Context, event *domain.AuditEvent) error {
	f.events = append(f.events, event)
	return nil
}

func testUser(hash string, enabled bool) *domain.User {
	return &domain.User{
		ID:           uuid.MustParse("018f1f0f-0e3b-7c0c-9b77-1c0c0f0f0f0f"),
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Email:        "ada@example.com",
		Role:         domain.RoleAdministrator,
		Enabled:      enabled,
		PasswordHash: hash,
	}
}

func mustHashPassword(t *testing.T, plain string) string {
	t.Helper()

	hash, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	return hash
}

func mustIssueToken(t *testing.T, user *domain.User) string {
	t.Helper()

	token, err := auth.IssueToken([]byte("test-secret"), time.Hour, user)
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}
	return token
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body["error"] != want {
		t.Fatalf("error = %q, want %q", body["error"], want)
	}
}

func assertAudit(t *testing.T, events []*domain.AuditEvent, wantAction, wantReason string, wantActorID *uuid.UUID) {
	t.Helper()

	if wantAction == "" {
		if len(events) != 0 {
			t.Fatalf("unexpected audit events: %d", len(events))
		}
		return
	}

	if len(events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(events))
	}

	event := events[0]
	if event.Action != wantAction {
		t.Fatalf("audit action = %q, want %q", event.Action, wantAction)
	}

	if wantActorID == nil {
		if event.ActorUserID != nil {
			t.Fatalf("audit actor = %v, want nil", *event.ActorUserID)
		}
	} else {
		if event.ActorUserID == nil {
			t.Fatal("expected audit actor user ID")
		}
		if *event.ActorUserID != *wantActorID {
			t.Fatalf("audit actor = %s, want %s", *event.ActorUserID, *wantActorID)
		}
	}

	if wantReason != "" {
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode audit payload: %v", err)
		}
		if payload["reason"] != wantReason {
			t.Fatalf("audit reason = %q, want %q", payload["reason"], wantReason)
		}
	}
}

func indexUsersByID(usersByEmail map[string]*domain.User) map[uuid.UUID]*domain.User {
	usersByID := make(map[uuid.UUID]*domain.User, len(usersByEmail))
	for _, user := range usersByEmail {
		usersByID[user.ID] = user
	}
	return usersByID
}
