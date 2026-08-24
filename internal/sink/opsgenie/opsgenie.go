package opsgenie

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
	"github.com/google/uuid"
)

const (
	maxMessage     = 130
	maxAlias       = 512
	maxDescription = 15000
	maxTags        = 20
	maxTagLen      = 50
)

type Sink struct{}

func (Sink) Provider() string { return "opsgenie" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	p := r.URL.Path
	return p == "/hooks/opsgenie/v2/alerts" || strings.HasPrefix(p, "/hooks/opsgenie/v2/alerts/")
}

func parse(body []byte) (map[string]any, error) {
	var m map[string]any
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	err := json.Unmarshal(body, &m)
	return m, err
}

func actionOf(path string) string {
	p := strings.Trim(path, "/")
	switch {
	case strings.HasSuffix(p, "/acknowledge"):
		return "acknowledge"
	case strings.HasSuffix(p, "/close"):
		return "close"
	case strings.HasSuffix(p, "/unacknowledge"):
		return "unacknowledge"
	default:
		return "create"
	}
}

func (Sink) Validate(r *http.Request, body []byte) sink.Validation {
	act := actionOf(r.URL.Path)
	if act != "create" {
		return sink.Validation{Valid: true}
	}
	m, err := parse(body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	var p sink.Problems
	msg, _ := m["message"].(string)
	if strings.TrimSpace(msg) == "" {
		p.Add("/message", "required")
	} else {
		p.MaxRunes("/message", msg, maxMessage)
	}
	if alias, ok := m["alias"].(string); ok {
		p.MaxRunes("/alias", alias, maxAlias)
	}
	if desc, ok := m["description"].(string); ok {
		p.MaxRunes("/description", desc, maxDescription)
	}
	if pri, ok := m["priority"].(string); ok && pri != "" {
		switch pri {
		case "P1", "P2", "P3", "P4", "P5":
		default:
			p.Add("/priority", "must be P1|P2|P3|P4|P5")
		}
	}
	if raw, ok := m["tags"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/tags")
		if ok {
			_ = p.MaxItems("/tags", len(arr), maxTags)
			n := len(arr)
			if n > maxTags {
				n = maxTags
			}
			for i := 0; i < n; i++ {
				tag, ok := arr[i].(string)
				if !ok {
					p.Add(sink.At("/tags", i), "must be a string")
					continue
				}
				p.MaxRunes(sink.At("/tags", i), tag, maxTagLen)
			}
		}
	}
	return p.Result()
}

func ogErrorBody(v sink.Validation) string {
	fields := map[string][]string{}
	for _, e := range v.Errors {
		key := strings.TrimPrefix(e.Path, "/")
		if key == "" {
			key = "body"
		}
		fields[key] = append(fields[key], e.Message)
	}
	out, _ := json.Marshal(map[string]any{
		"message":   "Request body is not processable.",
		"took":      0.001,
		"requestId": "invalid",
		"errors":    fields,
	})
	return string(out)
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusUnprocessableEntity, ogErrorBody(v))
		return nil
	}
	id := uuid.NewString()
	m, _ := parse(body)
	if alias, _ := m["alias"].(string); alias != "" {
		w.Header().Set("X-Webhookie-Dedup-Key", alias)
	} else {
		w.Header().Set("X-Webhookie-Dedup-Key", id)
	}
	out, _ := json.Marshal(map[string]any{
		"result":    "Request will be processed",
		"took":      0.01,
		"requestId": id,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write(out)
	return nil
}

func (Sink) Summarize(r *http.Request, body []byte) sink.Summary {
	act := actionOf(r.URL.Path)
	m, _ := parse(body)
	key, _ := m["alias"].(string)
	if key == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 6 {
			key = parts[4]
		}
	}
	sum := act
	if msg, _ := m["message"].(string); strings.TrimSpace(msg) != "" {
		sum = act + " · " + msg
	}
	return sink.Summary{Text: sink.FirstN(sum, 80), GroupKey: key}
}
