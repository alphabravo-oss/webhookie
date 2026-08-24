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

const maxText = 4096

type Sink struct{}

func (Sink) Provider() string { return "googlechat" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
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
	var p sink.Problems
	hasText := false
	if raw, ok := m["text"]; ok && raw != nil {
		s, ok := raw.(string)
		if !ok {
			p.Add("/text", "must be a string")
		} else if strings.TrimSpace(s) != "" {
			hasText = true
			p.MaxRunes("/text", s, maxText)
		}
	}
	hasCards := false
	if raw, ok := m["cards"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/cards")
		if ok && len(arr) > 0 {
			hasCards = true
			for i, c := range arr {
				p.RequireObject(c, sink.At("/cards", i))
			}
		}
	}
	hasV2 := false
	if raw, ok := m["cardsV2"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/cardsV2")
		if ok && len(arr) > 0 {
			hasV2 = true
			for i, c := range arr {
				validateCardV2(&p, sink.At("/cardsV2", i), c)
			}
		}
	}
	if !hasText && !hasCards && !hasV2 {
		p.Add("/", "text, cards, or cardsV2 is required")
	}
	return p.Result()
}

func validateCardV2(p *sink.Problems, path string, v any) {
	item, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	card, ok := p.RequireObject(item["card"], sink.Path(path, "card"))
	if !ok {
		return
	}
	if raw, ok := card["sections"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, sink.Path(path, "card/sections"))
		if !ok {
			return
		}
		for i, sec := range arr {
			sp := sink.At(sink.Path(path, "card/sections"), i)
			sm, ok := p.RequireObject(sec, sp)
			if !ok {
				continue
			}
			if wraw, ok := sm["widgets"]; ok && wraw != nil {
				widgets, ok := p.RequireArray(wraw, sink.Path(sp, "widgets"))
				if !ok {
					continue
				}
				for j, w := range widgets {
					p.RequireObject(w, sink.At(sink.Path(sp, "widgets"), j))
				}
			}
		}
	}
}

func firstError(v sink.Validation, fallback string) string {
	if len(v.Errors) == 0 {
		return fallback
	}
	if v.Has("/", "text, cards, or cardsV2") {
		return "text or cards required"
	}
	return v.Errors[0].Message
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		msg, _ := json.Marshal(firstError(v, "text or cards required"))
		sink.WriteJSON(w, http.StatusBadRequest, fmt.Sprintf(`{"error":{"code":400,"message":%s,"status":"INVALID_ARGUMENT"}}`, msg))
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
