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

func TestCharacterResponsesReturnRawUsageAndStartedMatchCounts(t *testing.T) {
	responses := characterResponses(map[string]uint64{"sophie": 3, "sena": 1}, 2)
	byID := make(map[string]characterResponse, len(responses))
	for _, response := range responses {
		byID[response.ID] = response
	}
	if got := byID["sophie"]; got.UsageCount != 3 || got.TotalPickCount != 2 {
		t.Fatalf("sophie usage = %d / %d picks, want 3 / 2", got.UsageCount, got.TotalPickCount)
	}
	if got := byID["sena"]; got.UsageCount != 1 || got.TotalPickCount != 2 {
		t.Fatalf("sena usage = %d / %d picks, want 1 / 2", got.UsageCount, got.TotalPickCount)
	}
	if got := byID["jude"]; got.UsageCount != 0 || got.TotalPickCount != 2 {
		t.Fatalf("jude usage = %d / %d picks, want 0 / 2", got.UsageCount, got.TotalPickCount)
	}
}
