package main

import (
	"auxilia-webserver/internal/store"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTestModeRejectsPasswordBeforeDatabaseAccess(t *testing.T) {
	for _, tt := range []struct {
		configured, body string
		status           int
	}{
		{"", `{"password":""}`, 503},
		{"secret", `{"password":"wrong"}`, 403},
		{"secret", `{"password":""}`, 403},
		{"secret", `{"password":"secret","ownerId":"other"}`, 400},
	} {
		t.Setenv("TESTMODE_PASSWORD", tt.configured)
		s := &service{}
		response := httptest.NewRecorder()
		s.testMatch(response, httptest.NewRequest("POST", "/api/test-matches", strings.NewReader(tt.body)), &store.Guest{ID: "owner"})
		if response.Code != tt.status {
			t.Fatalf("got %d want %d", response.Code, tt.status)
		}
		if strings.Contains(response.Body.String(), "secret") {
			t.Fatal("password leaked")
		}
	}
}
