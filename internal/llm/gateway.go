package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type Config struct {
	BaseURL    string
	Model      string
	APIKey     string
	HTTPClient *http.Client
	Timeout    time.Duration
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
}

type CompletionResponse struct {
	Model            string
	Content          string
	PromptTokens     int
	CompletionTokens int
	Latency          time.Duration
}

type Gateway struct {
	endpoint string
	model    string
	apiKey   string
	client   *http.Client
}

func NewOpenAICompatible(config Config) (*Gateway, error) {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		return nil, errors.New("llm base url is required")
	}
	endpoint, err := chatCompletionsEndpoint(baseURL)
	if err != nil {
		return nil, err
	}
	client := config.HTTPClient
	if client == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		client = &http.Client{Timeout: timeout}
	}
	return &Gateway{endpoint: endpoint, model: strings.TrimSpace(config.Model), apiKey: strings.TrimSpace(config.APIKey), client: client}, nil
}

func (g *Gateway) CompleteJSON(ctx context.Context, request CompletionRequest) (CompletionResponse, error) {
	if g == nil || g.client == nil || g.endpoint == "" {
		return CompletionResponse{}, errors.New("llm gateway is not configured")
	}
	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = g.model
	}
	if model == "" {
		return CompletionResponse{}, errors.New("llm model is required")
	}
	if len(request.Messages) == 0 {
		return CompletionResponse{}, errors.New("llm messages are required")
	}
	body := openAIChatRequest{
		Model:       model,
		Messages:    request.Messages,
		Temperature: request.Temperature,
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
	}
	if request.MaxTokens > 0 {
		body.MaxTokens = request.MaxTokens
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal llm request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(payload))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create llm request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if g.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)
	}
	started := time.Now()
	httpResp, err := g.client.Do(httpReq)
	latency := time.Since(started)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("call llm gateway: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return CompletionResponse{}, fmt.Errorf("llm gateway returned %s: %s", httpResp.Status, limitedBody(httpResp.Body, 512))
	}
	var decoded openAIChatResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&decoded); err != nil {
		return CompletionResponse{}, fmt.Errorf("decode llm response: %w", err)
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return CompletionResponse{}, errors.New("llm response did not contain message content")
	}
	if decoded.Model == "" {
		decoded.Model = model
	}
	return CompletionResponse{Model: decoded.Model, Content: decoded.Choices[0].Message.Content, PromptTokens: decoded.Usage.PromptTokens, CompletionTokens: decoded.Usage.CompletionTokens, Latency: latency}, nil
}

func chatCompletionsEndpoint(base string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", fmt.Errorf("parse llm base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("llm base url must include scheme and host")
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		parsed.Path = path
	case strings.HasSuffix(path, "/v1"):
		parsed.Path = path + "/chat/completions"
	default:
		parsed.Path = path + "/v1/chat/completions"
	}
	return parsed.String(), nil
}

func limitedBody(reader io.Reader, limit int64) string {
	data, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

type openAIChatRequest struct {
	Model          string            `json:"model"`
	Messages       []Message         `json:"messages"`
	Temperature    float64           `json:"temperature"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}
