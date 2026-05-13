package teaching

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kenichiLyon/loong64-b1-go/internal/aigateway"
)

func (s *Service) maybeParseWithArtifactParser(ctx context.Context, artifactID string, stored storedArtifact) (storedArtifact, error) {
	if s == nil || s.artifactParser == nil {
		return stored, nil
	}
	filePath, err := s.store.Resolve(stored.StorageKey)
	if err != nil {
		return storedArtifact{}, unavailableError("resolve artifact storage key for ai gateway", err)
	}
	response, err := s.artifactParser.ParseArtifact(ctx, aigateway.ParseArtifactRequest{
		ArtifactID:       artifactID,
		ArtifactKind:     string(stored.Kind),
		StoragePathOrURL: filePath,
		ContentType:      stored.ContentType,
	})
	if err != nil {
		return storedArtifact{}, unavailableError("parse artifact through ai gateway", err)
	}
	if strings.ToLower(strings.TrimSpace(response.Status)) != "succeeded" {
		return storedArtifact{}, validationError(firstNonEmptyArtifactValue(response.Error, "ai gateway parse failed"))
	}
	metadata := response.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["artifact_id"] = artifactID
	metadata["artifact_kind"] = string(stored.Kind)
	metadata["parser_source"] = "python_ai_gateway"
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return storedArtifact{}, unavailableError("encode ai gateway parse metadata", err)
	}
	stored.Metadata = encoded
	stored.TextExcerpt = strings.TrimSpace(response.TextExcerpt)
	return stored, nil
}

func firstNonEmptyArtifactValue(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
