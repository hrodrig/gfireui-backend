package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/api"
	"github.com/hrodrig/gfireui-backend/internal/domain"
	"github.com/hrodrig/gfireui-backend/internal/gfire"
)

func TestOpsSummaryAccessAndAggregation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		role domain.Role
	}{
		{name: "administrator", role: domain.RoleAdministrator},
		{name: "operator", role: domain.RoleOperator},
		{name: "auditor", role: domain.RoleAuditor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			upstream := newOpsSummaryUpstream(t, false)
			defer upstream.Close()

			user := testRoleUser(t, tt.role)
			rec := performOpsRequest(t, user, upstream.URL, nil)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			var body struct {
				JobsByState map[string]int       `json:"jobs_by_state"`
				Queues      []gfire.QueueSummary `json:"queues"`
				GeneratedAt time.Time            `json:"generated_at"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode ops summary: %v", err)
			}

			wantJobs := map[string]int{
				"pending":    2,
				"processing": 1,
				"succeeded":  3,
				"failed":     4,
				"dead":       0,
			}
			for state, want := range wantJobs {
				if got := body.JobsByState[state]; got != want {
					t.Fatalf("jobs_by_state[%s] = %d, want %d", state, got, want)
				}
			}
			if len(body.Queues) != 2 {
				t.Fatalf("queues = %d, want 2", len(body.Queues))
			}
			if body.Queues[0].Name != "default" || body.Queues[0].Depth != 5 {
				t.Fatalf("first queue = %#v", body.Queues[0])
			}
			if body.GeneratedAt.IsZero() {
				t.Fatal("generated_at should be set")
			}
			if time.Since(body.GeneratedAt) > time.Minute {
				t.Fatalf("generated_at = %s, want recent timestamp", body.GeneratedAt)
			}
		})
	}
}

func TestOpsSummaryBestEffort(t *testing.T) {
	t.Parallel()

	upstream := newOpsSummaryUpstream(t, true)
	defer upstream.Close()

	user := testRoleUser(t, domain.RoleAdministrator)
	rec := performOpsRequest(t, user, upstream.URL, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body struct {
		JobsByState map[string]int `json:"jobs_by_state"`
		Queues      []struct {
			Name  string `json:"name"`
			Depth int    `json:"depth"`
		} `json:"queues"`
		GeneratedAt time.Time `json:"generated_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode ops summary: %v", err)
	}

	if got := body.JobsByState["pending"]; got != 2 {
		t.Fatalf("pending jobs = %d, want 2", got)
	}
	if got := body.JobsByState["failed"]; got != 0 {
		t.Fatalf("failed jobs = %d, want 0 after upstream error", got)
	}
	if len(body.Queues) != 2 {
		t.Fatalf("queues = %d, want 2", len(body.Queues))
	}
}

func performOpsRequest(t *testing.T, actor *domain.User, upstreamURL string, auditWriter *fakeAuditWriter) *httptest.ResponseRecorder {
	t.Helper()

	client, err := gfire.NewClient(upstreamURL, "service-token", nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	deps := api.Deps{
		Store: &fakeUserStore{
			usersByID: map[uuid.UUID]*domain.User{
				actor.ID: actor,
			},
		},
		JWTSecret: []byte("test-secret"),
		GFire:     client,
	}
	if auditWriter != nil {
		deps.Audit = auditWriter
	}

	server := api.NewServer(deps)
	req := httptest.NewRequest(http.MethodGet, "/api/ops/summary", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueToken(t, actor))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	return rec
}

func newOpsSummaryUpstream(t *testing.T, failDead bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/queues":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"queues":[{"name":"default","depth":5},{"name":"bulk","depth":2}]}`))
		case "/v1/jobs":
			switch r.URL.Query().Get("state") {
			case "pending":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"jobs":[{"id":"1"},{"id":"2"}]}`))
			case "processing":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"total":1}`))
			case "succeeded":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[{"id":"1"},{"id":"2"},{"id":"3"}]}`))
			case "failed":
				if failDead {
					w.WriteHeader(http.StatusBadGateway)
					_, _ = w.Write([]byte(`{"error":"temporarily unavailable"}`))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"count":4}`))
			case "dead":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"unexpected state"}`))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
}
