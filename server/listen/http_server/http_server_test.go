package http_server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNormalizeLeadingSlashes(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantPath    string
		wantRawPath string
	}{
		{
			name:     "unchanged root",
			target:   "http://pmail.test/",
			wantPath: "/",
		},
		{
			name:     "two leading slashes",
			target:   "http://pmail.test//login?next=%2F",
			wantPath: "/login",
		},
		{
			name:     "several leading slashes",
			target:   "http://pmail.test////api/ping",
			wantPath: "/api/ping",
		},
		{
			name:     "internal slashes are unchanged",
			target:   "http://pmail.test/assets//app.js",
			wantPath: "/assets//app.js",
		},
		{
			name:        "escaped slash is unchanged",
			target:      "http://pmail.test/%2Fadmin",
			wantPath:    "//admin",
			wantRawPath: "/%2Fadmin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotRawPath, gotQuery string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotRawPath = r.URL.RawPath
				gotQuery = r.URL.RawQuery
				w.WriteHeader(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, tt.target, nil)
			response := httptest.NewRecorder()
			normalizeLeadingSlashes(next).ServeHTTP(response, request)

			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
			}
			if gotPath != tt.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tt.wantPath)
			}
			if gotRawPath != tt.wantRawPath {
				t.Errorf("raw path = %q, want %q", gotRawPath, tt.wantRawPath)
			}
			if tt.name == "two leading slashes" && gotQuery != "next=%2F" {
				t.Errorf("query = %q, want %q", gotQuery, "next=%2F")
			}
		})
	}
}

func TestNormalizeLeadingSlashesPreventsServeMuxRedirect(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "http://pmail.test//login", nil)
	response := httptest.NewRecorder()
	normalizeLeadingSlashes(mux).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; location = %q", response.Code, http.StatusNoContent, response.Header().Get("Location"))
	}
}
