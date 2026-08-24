package teams

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Sink struct{}

func (Sink) Provider() string { return "teams" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	p := r.URL.Path
	return strings.HasPrefix(p, "/hooks/teams/workflow/") || strings.HasPrefix(p, "/hooks/teams/incoming/")
}

func parse(body []byte) (map[string]any, error) {
	var m map[string]any
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	err := json.Unmarshal(body, &m)
	return m, err
}

func isMessageCard(m map[string]any) bool {
	t, _ := m["@type"].(string)
	return t == "MessageCard"
}

func isAdaptive(m map[string]any) bool {
	if t, _ := m["type"].(string); t == "AdaptiveCard" {
		return true
	}
	atts, _ := m["attachments"].([]any)
	for _, a := range atts {
		am, _ := a.(map[string]any)
		if am == nil {
			continue
		}
		ct, _ := am["contentType"].(string)
		content, _ := am["content"].(map[string]any)
		if ct == "application/vnd.microsoft.card.adaptive" && content != nil {
			if typ, _ := content["type"].(string); typ == "AdaptiveCard" {
				return true
			}
		}
	}
	return false
}

func (Sink) Validate(_ *http.Request, body []byte) sink.Validation {
	m, err := parse(body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	if isMessageCard(m) {
		_, text := m["text"].(string)
		_, title := m["title"].(string)
		_, sections := m["sections"].([]any)
		if !text && !title && !sections {
			return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "MessageCard needs text, title, or sections"}}}
		}
		return sink.Validation{Valid: true}
	}
	if isAdaptive(m) {
		return sink.Validation{Valid: true}
	}
	return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "expected MessageCard or Adaptive Card envelope"}}}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusBadRequest, `{"error":"invalid card"}`)
		return nil
	}
	m, _ := parse(body)
	if isMessageCard(m) {
		sink.WriteText(w, http.StatusOK, "1", "text/plain")
		return nil
	}
	sink.WriteJSON(w, http.StatusOK, `{"statusCode":200}`)
	return nil
}

func (Sink) Summarize(_ *http.Request, body []byte) sink.Summary {
	m, err := parse(body)
	if err != nil {
		return sink.Summary{Text: "invalid teams payload"}
	}
	if t, ok := m["title"].(string); ok && t != "" {
		return sink.Summary{Text: sink.FirstN(t, 80)}
	}
	if t, ok := m["text"].(string); ok && t != "" {
		return sink.Summary{Text: sink.FirstN(t, 80)}
	}
	if isAdaptive(m) {
		return sink.Summary{Text: "Adaptive Card"}
	}
	return sink.Summary{Text: "Teams card"}
}
