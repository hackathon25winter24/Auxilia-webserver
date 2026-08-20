package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsGitHubPagesPreflightByDefault(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("preflight request must not reach the API handler")
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/characters", nil)
	req.Header.Set("Origin", "https://hackathon25winter24.github.io")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://hackathon25winter24.github.io" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := res.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Fatalf("Access-Control-Allow-Headers = %q", got)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	handler := cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest(http.MethodGet, "/api/characters", nil)
	req.Header.Set("Origin", "https://example.invalid")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin = %q", got)
	}
}
