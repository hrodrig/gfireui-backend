package gfire_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/gfire"
)

func TestClientDoUsesServiceBearer(t *testing.T) {
	t.Parallel()

	var (
		gotMethod        string
		gotPath          string
		gotQuery         string
		gotAuthorization string
		gotContentType   string
		gotAccept        string
		gotBody          string
	)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read upstream body: %v", err)
		}

		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuthorization = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotAccept = r.Header.Get("Accept")
		gotBody = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	client, err := gfire.NewClient(upstream.URL+"/api", "service-token", upstream.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	resp, err := client.Do(context.Background(), http.MethodPost, "/v1/jobs?queue=default", strings.NewReader(`{"job":"demo"}`))
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/api/v1/jobs" {
		t.Fatalf("path = %q, want %q", gotPath, "/api/v1/jobs")
	}
	if gotQuery != "queue=default" {
		t.Fatalf("query = %q, want %q", gotQuery, "queue=default")
	}
	if gotAuthorization != "Bearer service-token" {
		t.Fatalf("authorization = %q, want %q", gotAuthorization, "Bearer service-token")
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type = %q, want %q", gotContentType, "application/json")
	}
	if gotAccept != "application/json" {
		t.Fatalf("accept = %q, want %q", gotAccept, "application/json")
	}
	if gotBody != `{"job":"demo"}` {
		t.Fatalf("body = %q, want %q", gotBody, `{"job":"demo"}`)
	}
}

func TestClientOpsHelpers(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/queues":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"queues":[{"name":"default","depth":3},{"name":"bulk","jobs_count":7}]}`))
		case r.URL.Path == "/api/v1/jobs" && r.URL.Query().Get("state") == "pending":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jobs":[{"id":"1"},{"id":"2"}]}`))
		case r.URL.Path == "/api/v1/jobs" && r.URL.Query().Get("state") == "processing":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total":5}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer upstream.Close()

	client, err := gfire.NewClient(upstream.URL+"/api", "service-token", upstream.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	queues, err := client.ListQueues(context.Background())
	if err != nil {
		t.Fatalf("ListQueues() error = %v", err)
	}
	if len(queues) != 2 {
		t.Fatalf("queues = %d, want 2", len(queues))
	}
	if queues[0].Name != "default" || queues[0].Depth != 3 {
		t.Fatalf("first queue = %#v", queues[0])
	}
	if queues[1].Name != "bulk" || queues[1].Depth != 7 {
		t.Fatalf("second queue = %#v", queues[1])
	}

	count, err := client.CountJobsByState(context.Background(), "pending")
	if err != nil {
		t.Fatalf("CountJobsByState(pending) error = %v", err)
	}
	if count != 2 {
		t.Fatalf("pending count = %d, want 2", count)
	}

	count, err = client.CountJobsByState(context.Background(), "processing")
	if err != nil {
		t.Fatalf("CountJobsByState(processing) error = %v", err)
	}
	if count != 5 {
		t.Fatalf("processing count = %d, want 5", count)
	}
}

func TestClientCountServersRecurringAndVersion(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/servers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"servers":[{"id":"a"},{"id":"b"},{"id":"c"}]}`))
		case "/api/v1/recurring":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"recurring":[{"id":"nightly"}]}`))
		case "/api/healthz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","version":"1.0.2","commit":"abc123"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer upstream.Close()

	client, err := gfire.NewClient(upstream.URL+"/api", "service-token", upstream.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	servers, err := client.CountServers(context.Background())
	if err != nil {
		t.Fatalf("CountServers() error = %v", err)
	}
	if servers != 3 {
		t.Fatalf("servers = %d, want 3", servers)
	}

	recurring, err := client.CountRecurring(context.Background())
	if err != nil {
		t.Fatalf("CountRecurring() error = %v", err)
	}
	if recurring != 1 {
		t.Fatalf("recurring = %d, want 1", recurring)
	}

	ver, err := client.FetchVersion(context.Background())
	if err != nil {
		t.Fatalf("FetchVersion() error = %v", err)
	}
	if ver.Version != "1.0.2" || ver.Commit != "abc123" {
		t.Fatalf("version = %#v", ver)
	}
}

func TestClientAllowsEmptyToken(t *testing.T) {
	t.Parallel()

	var gotAuthorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer upstream.Close()

	client, err := gfire.NewClient(upstream.URL, "", upstream.Client())
	if err != nil {
		t.Fatalf("NewClient() with empty token error = %v", err)
	}
	resp, err := client.Do(context.Background(), http.MethodGet, "/v1/queues", nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()
	if gotAuthorization != "" {
		t.Fatalf("authorization = %q, want empty", gotAuthorization)
	}
}
