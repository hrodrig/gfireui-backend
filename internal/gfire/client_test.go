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
