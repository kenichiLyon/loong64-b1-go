package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatCompletionsEndpointNormalizesBaseURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{name: "root", base: "http://localhost:8000", want: "http://localhost:8000/v1/chat/completions"},
		{name: "v1", base: "http://localhost:8000/v1", want: "http://localhost:8000/v1/chat/completions"},
		{name: "complete", base: "http://localhost:8000/v1/chat/completions", want: "http://localhost:8000/v1/chat/completions"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := chatCompletionsEndpoint(tc.base)
			if err != nil {
				t.Fatalf("endpoint should normalize: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected endpoint: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestCompleteJSONUsesOpenAICompatibleChatAPI(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		var body openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body.Model != "qwen-local" || len(body.Messages) != 1 || body.ResponseFormat["type"] != "json_object" {
			t.Fatalf("unexpected request body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"qwen-local","choices":[{"message":{"role":"assistant","content":"{\"summary\":\"ok\"}"}}],"usage":{"prompt_tokens":7,"completion_tokens":3}}`))
	}))
	defer server.Close()

	gateway, err := NewOpenAICompatible(Config{BaseURL: server.URL + "/v1", Model: "qwen-local", APIKey: "test-key", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	response, err := gateway.CompleteJSON(context.Background(), CompletionRequest{Messages: []Message{{Role: "user", Content: "return json"}}})
	if err != nil {
		t.Fatalf("CompleteJSON should succeed: %v", err)
	}
	if gotPath != "/v1/chat/completions" || gotAuth != "Bearer test-key" {
		t.Fatalf("unexpected path/auth: %s %s", gotPath, gotAuth)
	}
	if !strings.Contains(response.Content, `"summary"`) || response.PromptTokens != 7 || response.CompletionTokens != 3 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestCompleteJSONRejectsEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	gateway, err := NewOpenAICompatible(Config{BaseURL: server.URL, Model: "model", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = gateway.CompleteJSON(context.Background(), CompletionRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}
