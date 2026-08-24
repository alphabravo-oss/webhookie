package mattermost

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Sink struct{}

func (Sink) Provider() string { return "mattermost" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// hooks / mattermost / hooks / {token}
	return len(parts) == 4 && parts[0] == "hooks" && parts[1] == "mattermost" && parts[2] == "hooks"
}

func decode(r *http.Request, body []byte) (map[string]any, error) {
	ct := r.Header.Get("Content-Type")
	raw := strings.TrimSpace(string(body))
	if strings.Contains(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(raw, "payload=") {
		v, err := url.ParseQuery(raw)
		if err != nil {
			return nil, err
		}
		payload := v.Get("payload")
		if payload == "" {
			return map[string]any{}, nil
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	var m map[string]any
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (Sink) Validate(r *http.Request, body []byte) sink.Validation {
	m, err := decode(r, body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	text, _ := m["text"].(string)
	_, att := m["attachments"]
	if strings.TrimSpace(text) == "" && !att {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "text or attachments is required"}}}
	}
	return sink.Validation{Valid: true}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteText(w, http.StatusBadRequest, "Unable to parse incoming data", "text/plain")
		return nil
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
	return nil
}

func (Sink) Summarize(r *http.Request, body []byte) sink.Summary {
	m, err := decode(r, body)
	if err != nil {
		return sink.Summary{Text: "invalid mattermost payload"}
	}
	if t, _ := m["text"].(string); strings.TrimSpace(t) != "" {
		return sink.Summary{Text: sink.FirstN(t, 80)}
	}
	return sink.Summary{Text: "(attachments)"}
}
