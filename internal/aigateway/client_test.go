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

func TestEvaluateSubmission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate-submission" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var request EvaluateSubmissionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.SubmissionID != "submission-1" {
			t.Fatalf("unexpected submission id: %s", request.SubmissionID)
		}
		_ = json.NewEncoder(w).Encode(EvaluateSubmissionResponse{
			Summary: "gateway",
			MetricScores: []map[string]any{{
				"metric_code":     "quality",
				"suggested_score": 18,
				"confidence_bps":  7600,
				"rationale":       "looks good",
				"evidence_refs":   []string{"artifact:artifact-1"},
			}},
			RawModelMeta: map[string]any{"engine": "stub"},
		})
	}))
	defer server.Close()

	client, err := New(server.URL, "", time.Second)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	response, err := client.EvaluateSubmission(context.Background(), EvaluateSubmissionRequest{
		SubmissionID: "submission-1",
		Mode:         "rule_and_llm",
	})
	if err != nil {
		t.Fatalf("EvaluateSubmission failed: %v", err)
	}
	if response.Summary != "gateway" || len(response.MetricScores) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestBuildRetrievalIndexAndQuery(t *testing.T) {
	var createdIndexRef string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/build-retrieval-index":
			var request BuildRetrievalIndexRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode build request: %v", err)
			}
			if len(request.Chunks) != 1 {
				t.Fatalf("expected one chunk, got %+v", request.Chunks)
			}
			createdIndexRef = "idx:submission-1:test"
			_ = json.NewEncoder(w).Encode(BuildRetrievalIndexResponse{
				IndexRef:   createdIndexRef,
				ChunkCount: 1,
			})
		case "/internal/query-retrieval":
			var request QueryRetrievalRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode query request: %v", err)
			}
			if request.IndexRef != createdIndexRef {
				t.Fatalf("unexpected index_ref: %s", request.IndexRef)
			}
			_ = json.NewEncoder(w).Encode(QueryRetrievalResponse{
				Matches: []map[string]any{{
					"chunk_id": "artifact-1:1",
					"score":    3,
				}},
				Citations: []map[string]any{{
					"evidence_ref": "artifact:artifact-1",
				}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "", time.Second)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	buildResponse, err := client.BuildRetrievalIndex(context.Background(), BuildRetrievalIndexRequest{
		SubmissionID: "submission-1",
		ArtifactIDs:  []string{"artifact-1"},
		Chunks: []map[string]any{{
			"chunk_id":     "artifact-1:1",
			"artifact_id":  "artifact-1",
			"evidence_ref": "artifact:artifact-1",
			"text":         "implemented api tests",
		}},
	})
	if err != nil {
		t.Fatalf("BuildRetrievalIndex failed: %v", err)
	}
	queryResponse, err := client.QueryRetrieval(context.Background(), QueryRetrievalRequest{
		IndexRef: buildResponse.IndexRef,
		Query:    "api tests",
		TopK:     3,
	})
	if err != nil {
		t.Fatalf("QueryRetrieval failed: %v", err)
	}
	if len(queryResponse.Matches) != 1 || len(queryResponse.Citations) != 1 {
		t.Fatalf("unexpected retrieval response: %+v", queryResponse)
	}
}
