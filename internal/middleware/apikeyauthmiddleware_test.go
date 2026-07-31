package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiKeyAuthMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		wantStatus int
		wantNext   bool
	}{
		{name: "missing key", wantStatus: http.StatusUnauthorized},
		{name: "invalid key", apiKey: "invalid-key", wantStatus: http.StatusUnauthorized},
		{name: "similar key", apiKey: "api-key-11", wantStatus: http.StatusUnauthorized},
		{name: "first configured key", apiKey: "api-key-1", wantStatus: http.StatusNoContent, wantNext: true},
		{name: "second configured key", apiKey: "api-key-2", wantStatus: http.StatusNoContent, wantNext: true},
	}

	middleware := NewApiKeyAuthMiddleware([]string{"api-key-1", "api-key-2"})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nextCalled := false
			handler := middleware.Handle(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/images/sync", nil)
			if test.apiKey != "" {
				request.Header.Set("X-API-Key", test.apiKey)
			}
			recorder := httptest.NewRecorder()

			handler(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if nextCalled != test.wantNext {
				t.Fatalf("nextCalled = %t, want %t", nextCalled, test.wantNext)
			}
			if test.wantStatus == http.StatusUnauthorized {
				var body map[string]string
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if body["code"] != "UNAUTHORIZED" || body["message"] != "缺少或无效的 API Key" {
					t.Fatalf("response = %#v", body)
				}
			}
		})
	}
}

func TestApiKeyAuthMiddlewareRejectsWhenNoKeysConfigured(t *testing.T) {
	handler := NewApiKeyAuthMiddleware(nil).Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called")
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/images/sync", nil)
	request.Header.Set("X-API-Key", "any-key")
	recorder := httptest.NewRecorder()

	handler(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
