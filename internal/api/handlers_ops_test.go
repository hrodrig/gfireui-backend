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
				JobsByState    map[string]int       `json:"jobs_by_state"`
				Queues         []gfire.QueueSummary `json:"queues"`
				ServersCount   int                  `json:"servers_count"`
				RecurringCount int                  `json:"recurring_count"`
				Versions       []struct {
					Name    string `json:"name"`
					Version string `json:"version"`
					URL     string `json:"url"`
				} `json:"versions"`
				GeneratedAt time.Time `json:"generated_at"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode ops summary: %v", err)
			}

			wantJobs := map[string]int{
				"Enqueued":   2,
				"Processing": 1,
				"Succeeded":  3,
				"Failed":     4,
				"Dead":       0,
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
			if body.ServersCount != 3 {
				t.Fatalf("servers_count = %d, want 3", body.ServersCount)
			}
			if body.RecurringCount != 1 {
				t.Fatalf("recurring_count = %d, want 1", body.RecurringCount)
			}
			if len(body.Versions) < 2 {
				t.Fatalf("versions = %d, want >= 2", len(body.Versions))
			}
			if body.Versions[1].Name != "gfire" || body.Versions[1].Version != "1.0.1" {
				t.Fatalf("gfire version = %#v", body.Versions[1])
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

	if got := body.JobsByState["Enqueued"]; got != 2 {
		t.Fatalf("Enqueued jobs = %d, want 2", got)
	}
	if got := body.JobsByState["Failed"]; got != 0 {
		t.Fatalf("Failed jobs = %d, want 0 after upstream error", got)
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

func newOpsSummaryUpstream(t *testing.T, failFailed bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.1","commit":"abc1234"}`))
		case "/v1/queues":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"queues":[{"name":"default","depth":5},{"name":"bulk","depth":2}]}`))
		case "/v1/servers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"servers":[{"id":"gfire-1"},{"id":"gfire-2"},{"id":"gfire-3"}]}`))
		case "/v1/recurring":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"recurring":[{"id":"nightly"}]}`))
		case "/v1/jobs":
			switch r.URL.Query().Get("state") {
			case "Enqueued":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"jobs":[{"id":"1"},{"id":"2"}]}`))
			case "Processing":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"total":1}`))
			case "Succeeded":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":[{"id":"1"},{"id":"2"},{"id":"3"}]}`))
			case "Failed":
				if failFailed {
					w.WriteHeader(http.StatusBadGateway)
					_, _ = w.Write([]byte(`{"error":"temporarily unavailable"}`))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"count":4}`))
			case "Dead", "Scheduled", "Cancelled", "Deleted", "Awaiting":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"jobs":[]}`))
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
