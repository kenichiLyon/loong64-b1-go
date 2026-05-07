package assistant

import "encoding/json"

func defaultJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return mustJSON(map[string]any{})
	}
	return raw
}
