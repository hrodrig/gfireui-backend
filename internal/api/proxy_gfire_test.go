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
	"github.com/hrodrig/gfireui-backend/internal/gfire"
)

func TestGFireProxyReadUsesServiceBearer(t *testing.T) {
	t.Parallel()

	var (
		gotMethod        string
		gotPath          string
		gotQuery         string
		gotAuthorization string
		gotAccept        string
	)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuthorization = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jobs":[]}`))
	}))
	defer upstream.Close()

	server := newGFireProxyServer(t, domain.RoleAuditor, upstream.URL, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/gfire/v1/jobs?state=pending", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueToken(t, testRoleUser(t, domain.RoleAuditor)))
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodGet)
	}
	if gotPath != "/v1/jobs" {
		t.Fatalf("path = %q, want %q", gotPath, "/v1/jobs")
	}
	if gotQuery != "state=pending" {
		t.Fatalf("query = %q, want %q", gotQuery, "state=pending")
	}
	if gotAuthorization != "Bearer service-token" {
		t.Fatalf("authorization = %q, want %q", gotAuthorization, "Bearer service-token")
	}
	if gotAccept != "application/json" {
		t.Fatalf("accept = %q, want %q", gotAccept, "application/json")
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content-type = %q, want %q", rec.Header().Get("Content-Type"), "application/json")
	}
	if body := rec.Body.String(); body != "{\"jobs\":[]}" {
		t.Fatalf("body = %q, want %q", body, "{\"jobs\":[]}")
	}
}

func TestGFireProxyMutatingRBACAndAudit(t *testing.T) {
	t.Parallel()

	t.Run("auditor denied for mutating methods", func(t *testing.T) {
		t.Parallel()

		called := false
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusCreated)
		}))
		defer upstream.Close()

		server := newGFireProxyServer(t, domain.RoleAuditor, upstream.URL, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/gfire/v1/jobs", bytes.NewBufferString(`{"queue":"default"}`))
		req.Header.Set("Authorization", "Bearer "+mustIssueToken(t, testRoleUser(t, domain.RoleAuditor)))
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
		assertErrorResponse(t, rec, "forbidden")
		if called {
			t.Fatal("upstream should not be called for denied role")
		}
	})

	t.Run("operator mutation is proxied and audited", func(t *testing.T) {
		t.Parallel()

		var (
			gotMethod        string
			gotPath          string
			gotAuthorization string
			gotContentType   string
			gotBody          string
		)

		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upstream body: %v", err)
			}

			gotMethod = r.Method
			gotPath = r.URL.RequestURI()
			gotAuthorization = r.Header.Get("Authorization")
			gotContentType = r.Header.Get("Content-Type")
			gotBody = string(body)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"job-1"}`))
		}))
		defer upstream.Close()

		auditWriter := &fakeAuditWriter{}
		roleUser := testRoleUser(t, domain.RoleOperator)
		server := newGFireProxyServerWithUser(t, roleUser, upstream.URL, auditWriter)
		req := httptest.NewRequest(http.MethodPost, "/api/gfire/v1/jobs?queue=default", bytes.NewBufferString(`{"name":"demo"}`))
		req.Header.Set("Authorization", "Bearer "+mustIssueToken(t, roleUser))
		req.Header.Set("User-Agent", "proxy-test")
		req.RemoteAddr = "203.0.113.99:4567"
		rec := httptest.NewRecorder()

		server.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
		if gotMethod != http.MethodPost {
			t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
		}
		if gotPath != "/v1/jobs?queue=default" {
			t.Fatalf("path = %q, want %q", gotPath, "/v1/jobs?queue=default")
		}
		if gotAuthorization != "Bearer service-token" {
			t.Fatalf("authorization = %q, want %q", gotAuthorization, "Bearer service-token")
		}
		if gotContentType != "application/json" {
			t.Fatalf("content-type = %q, want %q", gotContentType, "application/json")
		}
		if gotBody != `{"name":"demo"}` {
			t.Fatalf("body = %q, want %q", gotBody, `{"name":"demo"}`)
		}
		if len(auditWriter.events) != 1 {
			t.Fatalf("audit events = %d, want 1", len(auditWriter.events))
		}
		event := auditWriter.events[0]
		if event.Action != "gfire.proxy" {
			t.Fatalf("audit action = %q, want %q", event.Action, "gfire.proxy")
		}
		if event.ResourceType != "gfire" {
			t.Fatalf("audit resource type = %q, want %q", event.ResourceType, "gfire")
		}
		if event.ActorUserID == nil || *event.ActorUserID != roleUser.ID {
			t.Fatalf("audit actor user ID = %v, want %s", event.ActorUserID, roleUser.ID)
		}
		if event.ResourceID == nil || *event.ResourceID != "/v1/jobs?queue=default" {
			t.Fatalf("audit resource ID = %v, want %q", event.ResourceID, "/v1/jobs?queue=default")
		}

		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode audit payload: %v", err)
		}
		if payload["method"] != http.MethodPost {
			t.Fatalf("audit method = %v, want %q", payload["method"], http.MethodPost)
		}
		if payload["path"] != "/v1/jobs?queue=default" {
			t.Fatalf("audit path = %v, want %q", payload["path"], "/v1/jobs?queue=default")
		}
		if payload["status_code"] != float64(http.StatusCreated) {
			t.Fatalf("audit status_code = %v, want %d", payload["status_code"], http.StatusCreated)
		}
	})
}

func newGFireProxyServer(t *testing.T, role domain.Role, upstreamURL string, auditWriter *fakeAuditWriter) http.Handler {
	t.Helper()
	return newGFireProxyServerWithUser(t, testRoleUser(t, role), upstreamURL, auditWriter)
}

func newGFireProxyServerWithUser(t *testing.T, user *domain.User, upstreamURL string, auditWriter *fakeAuditWriter) http.Handler {
	t.Helper()

	client, err := gfire.NewClient(upstreamURL, "service-token", nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	deps := api.Deps{
		Store: &fakeUserStore{
			usersByEmail: map[string]*domain.User{
				user.Email: user,
			},
			usersByID: map[uuid.UUID]*domain.User{
				user.ID: user,
			},
		},
		JWTSecret: []byte("test-secret"),
		GFire:     client,
	}
	if auditWriter != nil {
		deps.Audit = auditWriter
	}

	return api.NewServer(deps)
}
