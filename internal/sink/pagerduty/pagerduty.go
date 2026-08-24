package pagerduty

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
	"github.com/google/uuid"
)

const (
	routingKeyLen = 32
	maxSummary    = 1024
	maxDedupKey   = 255
	maxImages     = 8
	maxLinks      = 8
)

type Sink struct{}

func (Sink) Provider() string { return "pagerduty" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	p := r.URL.Path
	return p == "/hooks/pagerduty/v2/enqueue" || strings.HasPrefix(p, "/hooks/pagerduty/v2/enqueue/") || p == "/hooks/pagerduty/v2/change"
}

func parse(body []byte) (map[string]any, error) {
	var m map[string]any
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	err := json.Unmarshal(body, &m)
	return m, err
}

func (Sink) Validate(r *http.Request, body []byte) sink.Validation {
	m, err := parse(body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	var p sink.Problems
	rk, _ := m["routing_key"].(string)
	if len(rk) != routingKeyLen {
		p.Add("/routing_key", "routing_key must be 32 characters")
	}
	if r.URL.Path == "/hooks/pagerduty/v2/change" {
		payload, _ := m["payload"].(map[string]any)
		if payload == nil {
			p.Add("/payload", "required")
			return p.Result()
		}
		sum, _ := payload["summary"].(string)
		if strings.TrimSpace(sum) == "" {
			p.Add("/payload/summary", "required")
		} else {
			p.MaxRunes("/payload/summary", sum, maxSummary)
		}
		return p.Result()
	}
	action, _ := m["event_action"].(string)
	switch action {
	case "trigger":
		payload, ok := m["payload"].(map[string]any)
		if !ok || payload == nil {
			p.Add("/payload", "required")
			break
		}
		sum, _ := payload["summary"].(string)
		src, _ := payload["source"].(string)
		sev, _ := payload["severity"].(string)
		if strings.TrimSpace(sum) == "" {
			p.Add("/payload/summary", "required")
		} else {
			p.MaxRunes("/payload/summary", sum, maxSummary)
		}
		if strings.TrimSpace(src) == "" {
			p.Add("/payload/source", "required")
		}
		switch sev {
		case "info", "warning", "error", "critical":
		default:
			p.Add("/payload/severity", "must be info|warning|error|critical")
		}
	case "acknowledge", "resolve":
		d, _ := m["dedup_key"].(string)
		if strings.TrimSpace(d) == "" {
			p.Add("/dedup_key", "required for acknowledge/resolve")
		} else {
			p.MaxRunes("/dedup_key", d, maxDedupKey)
		}
	default:
		p.Add("/event_action", "must be trigger|acknowledge|resolve")
	}
	if d, ok := m["dedup_key"].(string); ok && action == "trigger" {
		p.MaxRunes("/dedup_key", d, maxDedupKey)
	}
	if raw, ok := m["images"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/images")
		if ok {
			_ = p.MaxItems("/images", len(arr), maxImages)
		}
	}
	if raw, ok := m["links"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/links")
		if ok {
			_ = p.MaxItems("/links", len(arr), maxLinks)
		}
	}
	return p.Result()
}

func pdErrorBody(v sink.Validation) string {
	msgs := make([]string, 0, len(v.Errors))
	for _, e := range v.Errors {
		msgs = append(msgs, e.Path+" "+e.Message)
	}
	out, _ := json.Marshal(map[string]any{
		"status":  "invalid event",
		"message": "Event object is invalid",
		"errors":  msgs,
	})
	return string(out)
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusBadRequest, pdErrorBody(v))
		return nil
	}
	m, _ := parse(body)
	key, _ := m["dedup_key"].(string)
	if key == "" {
		key = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	out, _ := json.Marshal(map[string]string{
		"status":    "success",
		"message":   "Event processed",
		"dedup_key": key,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Webhookie-Dedup-Key", key)
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(out)
	return nil
}

func (Sink) Summarize(r *http.Request, body []byte) sink.Summary {
	m, _ := parse(body)
	key, _ := m["dedup_key"].(string)
	action, _ := m["event_action"].(string)
	if r.URL.Path == "/hooks/pagerduty/v2/change" {
		action = "change"
	}
	sum := action
	if payload, ok := m["payload"].(map[string]any); ok {
		if s, ok := payload["summary"].(string); ok && s != "" {
			sum = action + " · " + s
		}
	}
	return sink.Summary{Text: sink.FirstN(sum, 80), GroupKey: key}
}

func DedupFromBody(body []byte, generated string) string {
	m, _ := parse(body)
	if k, _ := m["dedup_key"].(string); k != "" {
		return k
	}
	return generated
}
