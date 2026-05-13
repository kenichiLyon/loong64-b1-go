package aigateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health/ready" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok", Service: "python-ai-gateway"})
	}))
	defer server.Close()

	client, err := New(server.URL, "", time.Second)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}

func TestParseArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/parse-artifact" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var request ParseArtifactRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.ArtifactID != "artifact-1" {
			t.Fatalf("unexpected artifact id: %s", request.ArtifactID)
		}
		_ = json.NewEncoder(w).Encode(ParseArtifactResponse{
			Status:      "succeeded",
			TextExcerpt: "stub",
			Metadata:    map[string]any{"artifact_id": request.ArtifactID},
		})
	}))
	defer server.Close()

	client, err := New(server.URL, "", time.Second)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	response, err := client.ParseArtifact(context.Background(), ParseArtifactRequest{
		ArtifactID:       "artifact-1",
		ArtifactKind:     "report",
		StoragePathOrURL: "/tmp/report.md",
	})
	if err != nil {
		t.Fatalf("ParseArtifact failed: %v", err)
	}
	if response.TextExcerpt != "stub" {
		t.Fatalf("unexpected response: %+v", response)
	}
}
