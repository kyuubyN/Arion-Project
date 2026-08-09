// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyuubyN/Arion-Project/backend/security"
)

func TestLocalAPIDeniesMissingTokenAndCrossOriginRequests(t *testing.T) {
	api := &API{sessionToken: "secret", audit: security.NewAuditLogger(10)}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := api.securityMiddleware(next)

	tests := []struct {
		name   string
		token  string
		origin string
		want   int
	}{
		{name: "missing token", want: http.StatusUnauthorized},
		{name: "wrong token", token: "wrong", want: http.StatusUnauthorized},
		{name: "remote origin", token: "secret", origin: "https://example.com", want: http.StatusForbidden},
		{name: "same origin", token: "secret", origin: "http://127.0.0.1:8765", want: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/collections", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
		})
	}
}

func TestLocalAPIRejectsHostHeaderAttack(t *testing.T) {
	api := &API{sessionToken: "secret", audit: security.NewAuditLogger(10)}
	handler := api.securityMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	request := httptest.NewRequest(http.MethodGet, "http://evil.example/api/collections", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}
