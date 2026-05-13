package teaching

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"github.com/kenichiLyon/loong64-b1-go/internal/aigateway"
)

type ArtifactParser interface {
	ParseArtifact(context.Context, aigateway.ParseArtifactRequest) (aigateway.ParseArtifactResponse, error)
}

var artifactParserRegistry sync.Map

func WithArtifactParser(parser ArtifactParser) ServiceOption {
	return func(s *Service) {
		if s == nil || parser == nil {
			return
		}
		artifactParserRegistry.Store(s, parser)
	}
}

func artifactParserFor(s *Service) ArtifactParser {
	if s == nil {
		return nil
	}
	value, ok := artifactParserRegistry.Load(s)
	if !ok {
		return nil
	}
	parser, _ := value.(ArtifactParser)
	return parser
}

func (s *Service) maybeParseWithArtifactParser(ctx context.Context, artifactID string, stored storedArtifact) (storedArtifact, error) {
	parser := artifactParserFor(s)
	if parser == nil {
		return stored, nil
	}
	filePath, err := s.store.Resolve(stored.StorageKey)
	if err != nil {
		return storedArtifact{}, unavailableError("resolve artifact storage key for ai gateway", err)
	}
	response, err := parser.ParseArtifact(ctx, aigateway.ParseArtifactRequest{
		ArtifactID:       artifactID,
		ArtifactKind:     string(stored.Kind),
		StoragePathOrURL: filePath,
		ContentType:      stored.ContentType,
	})
	if err != nil {
		return storedArtifact{}, unavailableError("parse artifact through ai gateway", err)
	}
	if strings.ToLower(strings.TrimSpace(response.Status)) != "succeeded" {
		return storedArtifact{}, validationError(firstNonEmpty(response.Error, "ai gateway parse failed"))
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
