package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newMux().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestWsRouteRejectsNonWebSocketRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()

	newMux().ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected a non-200 status for a non-websocket request to /ws, got 200")
	}
}
