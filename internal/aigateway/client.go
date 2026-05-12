package aigateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func New(baseURL, apiKey string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("ai gateway base url is required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: baseURL,
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type ParseArtifactRequest struct {
	ArtifactID       string         `json:"artifact_id"`
	ArtifactKind     string         `json:"artifact_kind"`
	StoragePathOrURL string         `json:"storage_path_or_url"`
	ContentType      string         `json:"content_type"`
	ParseOptions     map[string]any `json:"parse_options,omitempty"`
}

type ParseArtifactResponse struct {
	Status      string           `json:"status"`
	TextExcerpt string           `json:"text_excerpt"`
	Metadata    map[string]any   `json:"metadata"`
	Sections    []map[string]any `json:"sections"`
	Evidence    []map[string]any `json:"evidence"`
	Error       string           `json:"error"`
}

type EvaluateSubmissionRequest struct {
	SubmissionID   string         `json:"submission_id"`
	Rubric         map[string]any `json:"rubric,omitempty"`
	SubmissionSpec map[string]any `json:"submission_spec,omitempty"`
	EvidenceBundle map[string]any `json:"evidence_bundle,omitempty"`
	Mode           string         `json:"mode"`
}

type EvaluateSubmissionResponse struct {
	Summary      string           `json:"summary"`
	Findings     []map[string]any `json:"findings"`
	MetricScores []map[string]any `json:"metric_scores"`
	Confidence   float64          `json:"confidence"`
	RawModelMeta map[string]any   `json:"raw_model_meta"`
	Error        string           `json:"error"`
}

type BuildRetrievalIndexRequest struct {
	SubmissionID string           `json:"submission_id"`
	ArtifactIDs  []string         `json:"artifact_ids"`
	Chunks       []map[string]any `json:"chunks"`
}

type BuildRetrievalIndexResponse struct {
	IndexRef   string `json:"index_ref"`
	ChunkCount int    `json:"chunk_count"`
	Error      string `json:"error"`
}

type QueryRetrievalRequest struct {
	IndexRef string `json:"index_ref"`
	Query    string `json:"query"`
	TopK     int    `json:"top_k"`
}

type QueryRetrievalResponse struct {
	Matches   []map[string]any `json:"matches"`
	Citations []map[string]any `json:"citations"`
	Error     string           `json:"error"`
}

func (c *Client) HealthCheck(ctx context.Context) error {
	var response HealthResponse
	if err := c.get(ctx, "/health/ready", &response); err != nil {
		return err
	}
	if response.Status != "ok" {
		return fmt.Errorf("ai gateway status is %q", response.Status)
	}
	return nil
}

func (c *Client) ParseArtifact(ctx context.Context, request ParseArtifactRequest) (ParseArtifactResponse, error) {
	var response ParseArtifactResponse
	err := c.post(ctx, "/internal/parse-artifact", request, &response)
	return response, err
}

func (c *Client) EvaluateSubmission(ctx context.Context, request EvaluateSubmissionRequest) (EvaluateSubmissionResponse, error) {
	var response EvaluateSubmissionResponse
	err := c.post(ctx, "/internal/evaluate-submission", request, &response)
	return response, err
}

func (c *Client) BuildRetrievalIndex(ctx context.Context, request BuildRetrievalIndexRequest) (BuildRetrievalIndexResponse, error) {
	var response BuildRetrievalIndexResponse
	err := c.post(ctx, "/internal/build-retrieval-index", request, &response)
	return response, err
}

func (c *Client) QueryRetrieval(ctx context.Context, request QueryRetrievalRequest) (QueryRetrievalResponse, error) {
	var response QueryRetrievalResponse
	err := c.post(ctx, "/internal/query-retrieval", request, &response)
	return response, err
}

func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.decorate(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call ai gateway %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ai gateway %s returned status %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode ai gateway %s response: %w", path, err)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, payload any, dst any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ai gateway %s request: %w", path, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.decorate(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call ai gateway %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ai gateway %s returned status %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode ai gateway %s response: %w", path, err)
	}
	return nil
}

func (c *Client) decorate(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

type Checker struct {
	Client *Client
}

func (c Checker) Name() string { return "python_ai_gateway" }

func (c Checker) Check(ctx context.Context) error {
	if c.Client == nil {
		return fmt.Errorf("ai gateway client is not configured")
	}
	return c.Client.HealthCheck(ctx)
}
