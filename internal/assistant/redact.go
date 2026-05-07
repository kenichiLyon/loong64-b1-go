package assistant

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

var secretValuePattern = regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)\s*[:=]\s*([^\s,;]+)`)

func redactText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	redacted := secretValuePattern.ReplaceAllString(value, `$1=******`)
	if strings.Contains(redacted, "://") {
		if masked, ok := redactDSN(redacted); ok {
			redacted = masked
		}
	}
	return redacted
}

func redactDSN(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Scheme == "" {
		return "", false
	}
	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.UserPassword(username, "******")
		}
	}
	return parsed.String(), true
}

func redactJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || !json.Valid(raw) {
		return mustJSON(map[string]any{})
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return mustJSON(map[string]any{"redacted_text": redactText(string(raw))})
	}
	return mustJSON(redactAny(value))
}

func redactAny(value any) any {
	switch current := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(current))
		for key, item := range current {
			lower := strings.ToLower(strings.TrimSpace(key))
			if lower == "database_url" || strings.HasSuffix(lower, "_url") && strings.Contains(lower, "database") {
				if text, ok := item.(string); ok {
					if masked, valid := redactDSN(text); valid {
						out[key] = masked
						continue
					}
				}
			}
			if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "api_key") {
				out[key] = "******"
				continue
			}
			out[key] = redactAny(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(current))
		for _, item := range current {
			out = append(out, redactAny(item))
		}
		return out
	case string:
		if masked, ok := redactDSN(current); ok {
			return masked
		}
		return redactText(current)
	default:
		return current
	}
}

func mustJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}
