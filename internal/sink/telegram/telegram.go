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

const (
	maxText         = 4096
	maxCallbackData = 64
	maxButtonText   = 64
)

var parseModes = map[string]bool{
	"Markdown":   true,
	"MarkdownV2": true,
	"HTML":       true,
}

type Sink struct{}

func (Sink) Provider() string { return "telegram" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
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
	var p sink.Problems
	if _, exists := m["chat_id"]; exists && m["chat_id"] != nil && chatID(m) == "" {
		p.Add("/chat_id", "must be a string or number")
	} else if chatID(m) == "" {
		p.Add("/chat_id", "chat_id is required")
	}
	text, _ := m["text"].(string)
	if _, exists := m["text"]; exists && m["text"] != nil {
		if _, ok := m["text"].(string); !ok {
			p.Add("/text", "must be a string")
		}
	}
	if strings.TrimSpace(text) == "" && (m["text"] == nil || m["text"] == "") {
		p.Add("/text", "text is required")
	} else if text != "" {
		p.MaxRunes("/text", text, maxText)
	}
	_, hasEntities := m["entities"]
	if pm, ok := m["parse_mode"].(string); ok && pm != "" {
		if !parseModes[pm] {
			p.Add("/parse_mode", "must be Markdown, MarkdownV2, or HTML")
		} else if !hasEntities && text != "" {
			if err := parseEntities(pm, text); err != nil {
				p.Add("/text", err.Error())
			}
		}
	}
	if hasEntities {
		validateMessageEntities(&p, text, m["entities"])
	}
	if raw, ok := m["reply_markup"]; ok && raw != nil {
		rm, ok := p.RequireObject(raw, "/reply_markup")
		if ok {
			if kb, ok := rm["inline_keyboard"]; ok {
				validateInlineKeyboard(&p, kb)
			}
		}
	}
	return p.Result()
}

func validateInlineKeyboard(p *sink.Problems, v any) {
	rows, ok := p.RequireArray(v, "/reply_markup/inline_keyboard")
	if !ok {
		return
	}
	for i, row := range rows {
		rp := sink.At("/reply_markup/inline_keyboard", i)
		buttons, ok := p.RequireArray(row, rp)
		if !ok {
			continue
		}
		for j, b := range buttons {
			bp := sink.At(rp, j)
			btn, ok := p.RequireObject(b, bp)
			if !ok {
				continue
			}
			text, _ := btn["text"].(string)
			if strings.TrimSpace(text) == "" {
				p.Add(sink.Path(bp, "text"), "required")
			} else {
				p.MaxRunes(sink.Path(bp, "text"), text, maxButtonText)
			}
			hasAction := false
			if cd, ok := btn["callback_data"].(string); ok {
				hasAction = true
				if cd == "" || len(cd) > maxCallbackData {
					p.Add(sink.Path(bp, "callback_data"), "must be 1-64 bytes")
				}
			}
			if u, ok := btn["url"].(string); ok && strings.TrimSpace(u) != "" {
				hasAction = true
			}
			for _, k := range []string{"switch_inline_query", "switch_inline_query_current_chat", "web_app", "login_url", "callback_game", "copy_text"} {
				if _, ok := btn[k]; ok {
					hasAction = true
				}
			}
			if pay, ok := btn["pay"].(bool); ok && pay {
				hasAction = true
			}
			if !hasAction {
				p.Add(bp, "inline button needs callback_data, url, or another action field")
			}
		}
	}
}

func telegramDescription(v sink.Validation) string {
	if len(v.Errors) == 0 {
		return "Bad Request"
	}
	e := v.Errors[0]
	switch {
	case e.Path == "/chat_id" && strings.Contains(e.Message, "required"):
		return "Bad Request: chat_id is empty"
	case e.Path == "/text" && strings.Contains(e.Message, "4096"):
		return "Bad Request: message is too long"
	case strings.Contains(e.Message, "can't parse entities"):
		return "Bad Request: " + e.Message
	case e.Path == "/text":
		return "Bad Request: message text is empty"
	case strings.Contains(e.Path, "callback_data"):
		return "Bad Request: BUTTON_DATA_INVALID"
	case e.Path == "/parse_mode":
		return "Bad Request: can't parse entities: unsupported start tag"
	default:
		return "Bad Request: " + e.Message
	}
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		sink.WriteJSON(w, http.StatusBadRequest, fmt.Sprintf(`{"ok":false,"error_code":400,"description":%q}`, telegramDescription(v)))
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
