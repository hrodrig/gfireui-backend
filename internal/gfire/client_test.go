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
