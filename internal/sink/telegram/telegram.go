package telegram

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

func (Sink) Provider() string { return "telegram" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// hooks / telegram / bot / {token} / sendMessage
	return len(parts) == 5 && parts[0] == "hooks" && parts[1] == "telegram" && parts[2] == "bot" && parts[4] == "sendMessage"
}

func parse(body []byte) (map[string]any, error) {
	var m map[string]any
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	err := json.Unmarshal(body, &m)
	return m, err
}

func chatID(m map[string]any) string {
	switch v := m["chat_id"].(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	case json.Number:
		return v.String()
	default:
		return ""
	}
}

func (Sink) Validate(_ *http.Request, body []byte) sink.Validation {
	m, err := parse(body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	if chatID(m) == "" {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/chat_id", Message: "chat_id is required"}}}
	}
	text, _ := m["text"].(string)
	if strings.TrimSpace(text) == "" {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/text", Message: "text is required"}}}
	}
	return sink.Validation{Valid: true}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		msg := "Bad Request: chat_id is empty"
		for _, e := range v.Errors {
			if e.Path == "/text" {
				msg = "Bad Request: message text is empty"
			}
		}
		sink.WriteJSON(w, http.StatusBadRequest, fmt.Sprintf(`{"ok":false,"error_code":400,"description":%q}`, msg))
		return nil
	}
	m, _ := parse(body)
	text, _ := m["text"].(string)
	id := chatID(m)
	var chat any = id
	if _, err := json.Number(id).Int64(); err == nil {
		chat = json.Number(id)
	}
	out := map[string]any{
		"ok": true,
		"result": map[string]any{
			"message_id": 1,
			"from":       map[string]any{"id": 0, "is_bot": true, "first_name": "Webhookie"},
			"chat":       map[string]any{"id": chat, "type": "private"},
			"date":       time.Now().Unix(),
			"text":       text,
		},
	}
	if rm, ok := m["reply_markup"]; ok {
		if result, ok := out["result"].(map[string]any); ok {
			result["reply_markup"] = rm
		}
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
		return sink.Summary{Text: "invalid telegram payload"}
	}
	if t, _ := m["text"].(string); strings.TrimSpace(t) != "" {
		return sink.Summary{Text: sink.FirstN(t, 80)}
	}
	return sink.Summary{Text: "sendMessage"}
}
