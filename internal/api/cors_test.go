package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hrodrig/gfireui-backend/internal/api"
)

func TestCORS_PreflightAndPOST(t *testing.T) {
	srv := api.NewServer(api.Deps{
		CORSAllowedOrigins: []string{"http://127.0.0.1:5173"},
	})

	preflight := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	preflight.Header.Set("Origin", "http://127.0.0.1:5173")
	preflight.Header.Set("Access-Control-Request-Method", "POST")
	preflightRec := httptest.NewRecorder()
	srv.ServeHTTP(preflightRec, preflight)
	if preflightRec.Code != http.StatusNoContent {
		t.Fatalf("preflight status=%d", preflightRec.Code)
	}
	if got := preflightRec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("preflight ACAO=%q", got)
	}

	get := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	get.Header.Set("Origin", "http://127.0.0.1:5173")
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, get)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status=%d", getRec.Code)
	}
	if got := getRec.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5173" {
		t.Fatalf("get ACAO=%q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	srv := api.NewServer(api.Deps{
		CORSAllowedOrigins: []string{"http://127.0.0.1:5173"},
	})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected ACAO=%q", got)
	}
}
