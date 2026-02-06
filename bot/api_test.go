package bot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestHandler() http.HandlerFunc {
	return apiCreateChannelHandler(Context{
		Config: Config{
			APIKey: "test-secret-key",
		},
	})
}

func TestAPICreateChannel_MethodNotAllowed(t *testing.T) {
	handler := newTestHandler()

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/channels", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestAPICreateChannel_Unauthorized(t *testing.T) {
	handler := newTestHandler()

	tests := []struct {
		name string
		auth string
	}{
		{"missing header", ""},
		{"wrong token", "Bearer wrong-key"},
		{"no bearer prefix", "test-secret-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/channels", strings.NewReader(`{"name":"test","description":"desc"}`))
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestAPICreateChannel_BadRequest(t *testing.T) {
	handler := newTestHandler()

	tests := []struct {
		name string
		body string
	}{
		{"empty name", `{"name":"","description":"desc"}`},
		{"empty description", `{"name":"test","description":""}`},
		{"both empty", `{"name":"","description":""}`},
		{"invalid json", `{invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/channels", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer test-secret-key")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}
