package httputil

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadJSONRejectsInvalidBeforeMutation(t *testing.T) {
	for _, input := range []string{"", "null", "[]", `{\"name\":`, `{\"name\":\"x\"} {}`, `{\"name\":\"x\"}garbage`} {
		t.Run(input, func(t *testing.T) {
			var data *struct{ Name string }
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/", strings.NewReader(input))
			if ReadJSON(w, r, &data) || data != nil {
				t.Fatalf("invalid body accepted or decoded: %q", input)
			}
		})
	}
	var data struct{ Name string }
	if !ReadJSON(httptest.NewRecorder(), httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"中文","future":true}`)), &data) || data.Name != "中文" {
		t.Fatal("valid compatible JSON rejected")
	}
}

func TestReadJSONBodyLimit(t *testing.T) {
	var data map[string]any
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"data":"`+strings.Repeat("x", MaxJSONBodySize)+`"}`))
	if ReadJSON(httptest.NewRecorder(), r, &data) {
		t.Fatal("oversized body accepted")
	}
}

func TestPagination(t *testing.T) {
	for _, tc := range []struct{ page, size, wantPage, wantSize, wantOffset int }{
		{2, 0, 2, 15, 15}, {0, 0, 1, 15, 0}, {-1, -1, 1, 15, 0}, {2, 10000, 2, 200, 200},
	} {
		p, s, o, ok := Pagination(tc.page, tc.size)
		if !ok || p != tc.wantPage || s != tc.wantSize || o != tc.wantOffset {
			t.Fatalf("Pagination(%d,%d) = %d,%d,%d,%v", tc.page, tc.size, p, s, o, ok)
		}
	}
	if _, _, _, ok := Pagination(int(^uint(0)>>1), 200); ok {
		t.Fatal("offset overflow accepted")
	}
}
