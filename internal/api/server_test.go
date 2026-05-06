package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestRootHandlerWithoutEmbeddedUIReturnsJSON(t *testing.T) {
	handler := rootHandler(nil, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["service"] != ServiceName || body["ready"] != "/health/ready" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestRootHandlerWithEmbeddedUIFallsBackToIndex(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":       {Data: []byte("<!doctype html><title>ui</title>")},
		"assets/app.js":    {Data: []byte("console.log('ok')")},
		"assets/app.css":   {Data: []byte("body{}")},
		"favicon.ico":      {Data: []byte("ico")},
		"nested/route.txt": {Data: []byte("route")},
	}
	handler := rootHandler(dist, true)

	req := httptest.NewRequest(http.MethodGet, "/teacher/submissions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "<!doctype html><title>ui</title>" {
		t.Fatalf("unexpected fallback body: %q", string(body))
	}
}

func TestRootHandlerWithEmbeddedUIServesAsset(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<!doctype html><title>ui</title>")},
		"assets/app.js": {Data: []byte("console.log('ok')")},
	}
	handler := rootHandler(dist, true)

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "console.log('ok')" {
		t.Fatalf("unexpected asset body: %q", got)
	}
}

func TestRootHandlerWithEmbeddedUIRejectsUnknownAPIPath(t *testing.T) {
	dist := fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>ui</title>")},
	}
	handler := rootHandler(dist, true)

	req := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
