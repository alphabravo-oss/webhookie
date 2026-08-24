package slack

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

func (Sink) Provider() string { return "slack" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// hooks / slack / services / T / B / token
	return len(parts) == 6 && parts[0] == "hooks" && parts[1] == "slack" && parts[2] == "services"
}

func decodeBody(r *http.Request, body []byte) (map[string]any, error) {
	ct := r.Header.Get("Content-Type")
	if strings.Contains(ct, "application/x-www-form-urlencoded") {
		v, err := url.ParseQuery(string(body))
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
	m, err := decodeBody(r, body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	_, hasText := m["text"]
	blocks, hasBlocks := m["blocks"]
	_, hasAtt := m["attachments"]
	if hasText {
		if s, ok := m["text"].(string); ok && strings.TrimSpace(s) != "" {
			hasText = true
		} else if !hasBlocks && !hasAtt {
			hasText = false
		}
	}
	if hasBlocks {
		arr, ok := blocks.([]any)
		if !ok || len(arr) == 0 {
			hasBlocks = false
		} else {
			for i, b := range arr {
				bm, ok := b.(map[string]any)
				if !ok {
					return sink.Validation{Errors: []store.ValidationError{{Path: "/blocks/" + itoa(i) + "/type", Message: "block must be an object"}}}
				}
				if _, ok := bm["type"].(string); !ok {
					return sink.Validation{Errors: []store.ValidationError{{Path: "/blocks/" + itoa(i) + "/type", Message: "type is required"}}}
				}
			}
		}
	}
	if !hasText && !hasBlocks && !hasAtt {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "at least one of text, blocks, or attachments is required"}}}
	}
	return sink.Validation{Valid: true}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusBadRequest, `{"ok":false,"error":"invalid_payload"}`)
		return nil
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok")
	return nil
}

func (Sink) Summarize(r *http.Request, body []byte) sink.Summary {
	m, err := decodeBody(r, body)
	if err != nil {
		return sink.Summary{Text: "invalid slack payload"}
	}
	if t, ok := m["text"].(string); ok && strings.TrimSpace(t) != "" {
		return sink.Summary{Text: sink.FirstN(t, 80)}
	}
	if blocks, ok := m["blocks"].([]any); ok {
		for _, b := range blocks {
			bm, _ := b.(map[string]any)
			if bm == nil {
				continue
			}
			if text, ok := bm["text"].(map[string]any); ok {
				if ts, ok := text["text"].(string); ok && ts != "" {
					return sink.Summary{Text: sink.FirstN(ts, 80)}
				}
			}
		}
		return sink.Summary{Text: "(blocks)"}
	}
	return sink.Summary{Text: "(attachments)"}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [12]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	return string(b[n:])
}
