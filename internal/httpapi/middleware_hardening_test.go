package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlatformKeyMiddleware_RequireKeyUnconfigured(t *testing.T) {
	d := &Deps{RequirePlatformAPIKey: true, PlatformAPIKey: ""}
	mw := d.platformKeyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestCORSMiddleware_Preflight(t *testing.T) {
	h := corsMiddleware([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/v1/health", nil)
	req.Header.Set("Origin", "https://app.example.com")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("ACAO = %q", got)
	}
}
