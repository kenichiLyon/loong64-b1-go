package teaching

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/kenichiLyon/loong64-b1-go/internal/aigateway"
	"github.com/kenichiLyon/loong64-b1-go/internal/storage"
)

type fakeArtifactParser struct {
	response aigateway.ParseArtifactResponse
	err      error
}

func (f fakeArtifactParser) ParseArtifact(context.Context, aigateway.ParseArtifactRequest) (aigateway.ParseArtifactResponse, error) {
	return f.response, f.err
}

func TestStoreUploadedArtifactCanBeEnrichedByArtifactParser(t *testing.T) {
	store := storage.NewLocal(t.TempDir())
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{}, WithArtifactStore(store), WithUploadLimits(32*1024, 2), WithArtifactParser(fakeArtifactParser{
		response: aigateway.ParseArtifactResponse{
			Status:      "succeeded",
			TextExcerpt: "python excerpt",
			Metadata: map[string]any{
				"parser": "python",
			},
			Sections: []map[string]any{{
				"title":   "Overview",
				"content": "section body",
			}},
			Evidence: []map[string]any{{
				"kind": "keyword",
				"text": "api test",
			}},
		},
	}))
	stored, err := service.storeUploadedArtifact(ArtifactUploadInput{
		FileName: "report.txt",
		Reader:   bytes.NewBufferString("local report"),
	}, "artifact-1", "submission-1")
	if err != nil {
		t.Fatalf("storeUploadedArtifact failed: %v", err)
	}
	enriched, err := service.maybeParseWithArtifactParser(context.Background(), "artifact-1", stored)
	if err != nil {
		t.Fatalf("maybeParseWithArtifactParser failed: %v", err)
	}
	if enriched.TextExcerpt != "python excerpt" {
		t.Fatalf("expected python excerpt, got %q", enriched.TextExcerpt)
	}
	var metadata map[string]any
	if err := json.Unmarshal(enriched.Metadata, &metadata); err != nil {
		t.Fatalf("metadata should be valid JSON: %v", err)
	}
	if metadata["parser_source"] != "python_ai_gateway" {
		t.Fatalf("unexpected metadata: %+v", metadata)
	}
	if len(metadata["sections"].([]any)) != 1 || len(metadata["evidence"].([]any)) != 1 {
		t.Fatalf("expected sections and evidence to be preserved: %+v", metadata)
	}
}
