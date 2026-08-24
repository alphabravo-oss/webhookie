package googlechat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Sink struct{}

func (Sink) Provider() string { return "googlechat" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// hooks / googlechat / v1 / spaces / {space} / messages
	return len(parts) == 6 && parts[0] == "hooks" && parts[1] == "googlechat" && parts[2] == "v1" && parts[3] == "spaces" && parts[5] == "messages"
}

func parse(body []byte) (map[string]any, error) {
	var m map[string]any
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	err := json.Unmarshal(body, &m)
	return m, err
}

func spaceFrom(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 5 {
		return parts[4]
	}
	return "space"
}

func (Sink) Validate(_ *http.Request, body []byte) sink.Validation {
	m, err := parse(body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	text, _ := m["text"].(string)
	_, cards := m["cards"]
	_, cardsV2 := m["cardsV2"]
	if strings.TrimSpace(text) == "" && !cards && !cardsV2 {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "text, cards, or cardsV2 is required"}}}
	}
	return sink.Validation{Valid: true}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusBadRequest, `{"error":{"code":400,"message":"text or cards required","status":"INVALID_ARGUMENT"}}`)
		return nil
	}
	m, _ := parse(body)
	text, _ := m["text"].(string)
	space := spaceFrom(r.URL.Path)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	name := fmt.Sprintf("spaces/%s/messages/webhookie.1", space)
	out := map[string]any{
		"name":       name,
		"text":       text,
		"createTime": now,
		"thread":     map[string]any{"name": name + "/threads/webhookie"},
	}
	b, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b)
	return nil
}

func (Sink) Summarize(_ *http.Request, body []byte) sink.Summary {
	m, err := parse(body)
	if err != nil {
		return sink.Summary{Text: "invalid google chat payload"}
	}
	if t, _ := m["text"].(string); strings.TrimSpace(t) != "" {
		return sink.Summary{Text: sink.FirstN(t, 80)}
	}
	if cards, ok := m["cardsV2"].([]any); ok && len(cards) > 0 {
		if c, ok := cards[0].(map[string]any); ok {
			if card, ok := c["card"].(map[string]any); ok {
				if h, ok := card["header"].(map[string]any); ok {
					if title, _ := h["title"].(string); title != "" {
						return sink.Summary{Text: sink.FirstN(title, 80)}
					}
				}
			}
		}
		return sink.Summary{Text: "card"}
	}
	return sink.Summary{Text: "Google Chat message"}
}
