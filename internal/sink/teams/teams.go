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

func isAdaptiveEnvelope(m map[string]any) bool {
	if t, _ := m["type"].(string); t == "AdaptiveCard" {
		return true
	}
	if t, _ := m["type"].(string); t == "message" {
		return true
	}
	atts, _ := m["attachments"].([]any)
	return len(atts) > 0
}

func (Sink) Validate(_ *http.Request, body []byte) sink.Validation {
	m, err := parse(body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	var p sink.Problems
	if isMessageCard(m) {
		validateMessageCard(&p, m)
		return p.Result()
	}
	if isAdaptiveEnvelope(m) {
		validateAdaptive(&p, m)
		return p.Result()
	}
	p.Add("/", "expected MessageCard or Adaptive Card envelope")
	return p.Result()
}

func validateMessageCard(p *sink.Problems, m map[string]any) {
	_, text := m["text"].(string)
	_, title := m["title"].(string)
	sections, hasSections := m["sections"]
	if hasSections {
		arr, ok := p.RequireArray(sections, "/sections")
		if ok {
			hasSections = len(arr) > 0
			for i, s := range arr {
				p.RequireObject(s, sink.At("/sections", i))
			}
		} else {
			hasSections = false
		}
	}
	if !text && !title && !hasSections {
		p.Add("/", "MessageCard needs text, title, or sections")
	}
	if raw, ok := m["potentialAction"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/potentialAction")
		if !ok {
			return
		}
		for i, a := range arr {
			ap := sink.At("/potentialAction", i)
			obj, ok := p.RequireObject(a, ap)
			if !ok {
				continue
			}
			if _, ok := obj["@type"].(string); !ok {
				p.Add(sink.Path(ap, "@type"), "required")
			}
			if name, ok := obj["name"].(string); !ok || strings.TrimSpace(name) == "" {
				p.Add(sink.Path(ap, "name"), "required")
			}
		}
	}
}

func validateAdaptive(p *sink.Problems, m map[string]any) {
	if t, _ := m["type"].(string); t == "AdaptiveCard" {
		validateAdaptiveCard(p, "/", m, false)
		return
	}
	atts, ok := m["attachments"]
	if !ok {
		p.Add("/attachments", "required for Adaptive Card envelope")
		return
	}
	arr, ok := p.RequireArray(atts, "/attachments")
	if !ok {
		return
	}
	if len(arr) == 0 {
		p.Add("/attachments", "must have at least 1 item")
		return
	}
	found := false
	for i, a := range arr {
		ap := sink.At("/attachments", i)
		obj, ok := p.RequireObject(a, ap)
		if !ok {
			continue
		}
		ct, _ := obj["contentType"].(string)
		if ct != "application/vnd.microsoft.card.adaptive" {
			p.Add(sink.Path(ap, "contentType"), "must be application/vnd.microsoft.card.adaptive")
			continue
		}
		content, ok := p.RequireObject(obj["content"], sink.Path(ap, "content"))
		if !ok {
			continue
		}
		found = true
		validateAdaptiveCard(p, sink.Path(ap, "content"), content, false)
	}
	if !found && len(p.Result().Errors) == 0 {
		p.Add("/attachments", "expected an Adaptive Card attachment")
	}
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
	if isAdaptiveEnvelope(m) || isMessageCard(m) {
		if isMessageCard(m) {
			return sink.Summary{Text: "Teams card"}
		}
		return sink.Summary{Text: "Adaptive Card"}
	}
	return sink.Summary{Text: "Teams card"}
}
