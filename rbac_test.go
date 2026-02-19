package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// makeRequest creates a request with role set in context
func makeRequest(method, path, role string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if role != "" {
		ctx := context.WithValue(req.Context(), ctxUserID, 1)
		ctx = context.WithValue(ctx, ctxRole, role)
		req = req.WithContext(ctx)
	}
	return req
}

func TestRBAC(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	rbac := requireRBAC(okHandler)

	tests := []struct {
		name       string
		method     string
		path       string
		role       string
		wantStatus int
	}{
		// Admin: full access
		{"admin GET parts", "GET", "/api/v1/parts", "admin", 200},
		{"admin POST parts", "POST", "/api/v1/parts", "admin", 200},
		{"admin GET users", "GET", "/api/v1/users", "admin", 200},
		{"admin POST users", "POST", "/api/v1/users", "admin", 200},
		{"admin GET apikeys", "GET", "/api/v1/apikeys", "admin", 200},
		{"admin PUT email config", "PUT", "/api/v1/email/config", "admin", 200},
		{"admin PUT settings email", "PUT", "/api/v1/settings/email", "admin", 200},

		// User: CRUD on business objects, no admin endpoints
		{"user GET parts", "GET", "/api/v1/parts", "user", 200},
		{"user POST parts", "POST", "/api/v1/parts", "user", 200},
		{"user PUT ecos", "PUT", "/api/v1/ecos/123", "user", 200},
		{"user DELETE parts", "DELETE", "/api/v1/parts/123", "user", 200},
		{"user GET workorders", "GET", "/api/v1/workorders", "user", 200},
		{"user POST workorders", "POST", "/api/v1/workorders", "user", 200},
		{"user GET users DENIED", "GET", "/api/v1/users", "user", 403},
		{"user POST users DENIED", "POST", "/api/v1/users", "user", 403},
		{"user PUT users DENIED", "PUT", "/api/v1/users/1", "user", 403},
		{"user GET apikeys DENIED", "GET", "/api/v1/apikeys", "user", 403},
		{"user POST apikeys DENIED", "POST", "/api/v1/apikeys", "user", 403},
		{"user DELETE apikeys DENIED", "DELETE", "/api/v1/apikeys/1", "user", 403},
		{"user GET email config DENIED", "GET", "/api/v1/email/config", "user", 403},
		{"user PUT email config DENIED", "PUT", "/api/v1/email/config", "user", 403},
		{"user POST email test DENIED", "POST", "/api/v1/email/test", "user", 403},
		{"user GET settings email DENIED", "GET", "/api/v1/settings/email", "user", 403},
		{"user PUT settings email DENIED", "PUT", "/api/v1/settings/email", "user", 403},
		{"user GET email-log allowed", "GET", "/api/v1/email-log", "user", 200},

		// Readonly: GET only
		{"readonly GET parts", "GET", "/api/v1/parts", "readonly", 200},
		{"readonly GET users", "GET", "/api/v1/users", "readonly", 200},
		{"readonly POST parts DENIED", "POST", "/api/v1/parts", "readonly", 403},
		{"readonly PUT ecos DENIED", "PUT", "/api/v1/ecos/1", "readonly", 403},
		{"readonly DELETE parts DENIED", "DELETE", "/api/v1/parts/1", "readonly", 403},

		// No role (Bearer token): full access
		{"no role GET", "GET", "/api/v1/parts", "", 200},
		{"no role POST users", "POST", "/api/v1/users", "", 200},

		// Non-API paths pass through
		{"non-api path", "GET", "/auth/login", "readonly", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := makeRequest(tt.method, tt.path, tt.role)
			w := httptest.NewRecorder()
			rbac.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
