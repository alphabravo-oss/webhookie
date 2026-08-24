package pagerduty

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
	"github.com/google/uuid"
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
	rk, _ := m["routing_key"].(string)
	if len(rk) != 32 {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/routing_key", Message: "routing_key must be 32 characters"}}}
	}
	if r.URL.Path == "/hooks/pagerduty/v2/change" {
		payload, _ := m["payload"].(map[string]any)
		sum, _ := payload["summary"].(string)
		if strings.TrimSpace(sum) == "" {
			return sink.Validation{Errors: []store.ValidationError{{Path: "/payload/summary", Message: "required"}}}
		}
		return sink.Validation{Valid: true}
	}
	action, _ := m["event_action"].(string)
	switch action {
	case "trigger":
		payload, _ := m["payload"].(map[string]any)
		if payload == nil {
			return sink.Validation{Errors: []store.ValidationError{{Path: "/payload", Message: "required"}}}
		}
		sum, _ := payload["summary"].(string)
		src, _ := payload["source"].(string)
		sev, _ := payload["severity"].(string)
		if strings.TrimSpace(sum) == "" {
			return sink.Validation{Errors: []store.ValidationError{{Path: "/payload/summary", Message: "required"}}}
		}
		if strings.TrimSpace(src) == "" {
			return sink.Validation{Errors: []store.ValidationError{{Path: "/payload/source", Message: "required"}}}
		}
		switch sev {
		case "info", "warning", "error", "critical":
		default:
			return sink.Validation{Errors: []store.ValidationError{{Path: "/payload/severity", Message: "must be info|warning|error|critical"}}}
		}
	case "acknowledge", "resolve":
		if d, _ := m["dedup_key"].(string); strings.TrimSpace(d) == "" {
			return sink.Validation{Errors: []store.ValidationError{{Path: "/dedup_key", Message: "required for acknowledge/resolve"}}}
		}
	default:
		return sink.Validation{Errors: []store.ValidationError{{Path: "/event_action", Message: "must be trigger|acknowledge|resolve"}}}
	}
	return sink.Validation{Valid: true}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusBadRequest, `{"status":"invalid event","message":"Event object is invalid"}`)
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

// DedupFromResponse extracts the generated key after Respond (via header).
func DedupFromBody(body []byte, generated string) string {
	m, _ := parse(body)
	if k, _ := m["dedup_key"].(string); k != "" {
		return k
	}
	return generated
}
