package discord

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Sink struct{}

func (Sink) Provider() string { return "discord" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// hooks / discord / api / webhooks / id / token
	return len(parts) == 6 && parts[0] == "hooks" && parts[1] == "discord" && parts[2] == "api" && parts[3] == "webhooks"
}

func parse(body []byte) (map[string]any, error) {
	var m map[string]any
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	err := json.Unmarshal(body, &m)
	return m, err
}

func (Sink) Validate(_ *http.Request, body []byte) sink.Validation {
	m, err := parse(body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	content, _ := m["content"].(string)
	embeds, _ := m["embeds"].([]any)
	if strings.TrimSpace(content) == "" && len(embeds) == 0 {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "content or embeds is required"}}}
	}
	return sink.Validation{Valid: true}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusBadRequest, `{"message":"Cannot send an empty message","code":50006}`)
		return nil
	}
	if r.URL.Query().Get("wait") == "true" {
		sink.WriteJSON(w, http.StatusOK, `{"id":"0","content":"ok"}`)
		return nil
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (Sink) Summarize(_ *http.Request, body []byte) sink.Summary {
	m, err := parse(body)
	if err != nil {
		return sink.Summary{Text: "invalid discord payload"}
	}
	if c, ok := m["content"].(string); ok && strings.TrimSpace(c) != "" {
		return sink.Summary{Text: sink.FirstN(c, 80)}
	}
	if embeds, ok := m["embeds"].([]any); ok && len(embeds) > 0 {
		if em, ok := embeds[0].(map[string]any); ok {
			if title, ok := em["title"].(string); ok && title != "" {
				return sink.Summary{Text: sink.FirstN(title, 80)}
			}
		}
	}
	return sink.Summary{Text: "(embed)"}
}
