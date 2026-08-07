package api

import (
	"net/http"
	"testing"
)

func TestShouldSkipProxyHeader(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	headers.Set("Connection", "Keep-Alive, X-Custom-Drop")
	headers.Set("Keep-Alive", "timeout=5")
	headers.Set("X-Custom-Drop", "1")
	headers.Set("X-Keep", "yes")

	if !shouldSkipProxyHeader("Connection", headers) {
		t.Fatal("Connection should be skipped")
	}
	if !shouldSkipProxyHeader("Keep-Alive", headers) {
		t.Fatal("Keep-Alive hop-by-hop should be skipped")
	}
	if !shouldSkipProxyHeader("X-Custom-Drop", headers) {
		t.Fatal("Connection token should be skipped")
	}
	if shouldSkipProxyHeader("X-Keep", headers) {
		t.Fatal("X-Keep should not be skipped")
	}
}

func TestConnectionHeaderTokens(t *testing.T) {
	t.Parallel()

	if tokens := connectionHeaderTokens(http.Header{}); tokens != nil {
		t.Fatalf("empty = %#v", tokens)
	}

	h := http.Header{}
	h.Add("Connection", " keep-alive , , Upgrade ")
	h.Add("Connection", "TE")
	got := connectionHeaderTokens(h)
	if len(got) != 3 {
		t.Fatalf("tokens = %#v", got)
	}
}

func TestGFireProxyPath(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodGet, "/api/gfire/v1/jobs?state=pending", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetPathValue("path", "v1/jobs")
	if got := gfireProxyPath(req); got != "/v1/jobs?state=pending" {
		t.Fatalf("path = %q", got)
	}

	req2, err := http.NewRequest(http.MethodGet, "/api/gfire", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := gfireProxyPath(req2); got != "/" {
		t.Fatalf("empty path = %q", got)
	}
}
