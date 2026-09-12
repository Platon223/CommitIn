package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
		wantOK bool
	}{
		{"valid", "Bearer abc123", "abc123", true},
		{"missing header", "", "", false},
		{"wrong scheme", "Basic abc123", "", false},
		{"empty token", "Bearer ", "", false},
		{"empty token no space", "Bearer", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if c.header != "" {
				r.Header.Set("Authorization", c.header)
			}
			got, ok := bearerToken(r)
			if ok != c.wantOK || got != c.want {
				t.Fatalf("bearerToken(%q) = (%q, %v), want (%q, %v)", c.header, got, ok, c.want, c.wantOK)
			}
		})
	}
}
