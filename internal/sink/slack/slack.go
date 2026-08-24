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

// Documented incoming-webhook / Block Kit limits (public docs, not Slack's private validator).
const (
	maxText       = 40000
	maxBlocks     = 50
	maxAttach     = 100
	maxSection    = 3000
	maxFields     = 10
	maxFieldText  = 2000
	maxHeader     = 150
	maxActions    = 25
	maxContext    = 10
	maxButtonText = 75
	maxBlockID    = 255
	maxActionID   = 255
	maxButtonURL  = 3000
	maxButtonVal  = 2000
	maxOptionText = 75
	maxOptionVal  = 75
	maxOptions    = 100
	maxOverflow   = 5
	maxSelectPh   = 150
)

var blockTypes = map[string]bool{
	"section":   true,
	"divider":   true,
	"image":     true,
	"actions":   true,
	"context":   true,
	"header":    true,
	"input":     true,
	"file":      true,
	"rich_text": true,
	"video":     true,
	"markdown":  true,
	"table":     true,
}

var actionTypes = map[string]bool{
	"button":                     true,
	"overflow":                   true,
	"datepicker":                 true,
	"timepicker":                 true,
	"datetimepicker":             true,
	"checkboxes":                 true,
	"radio_buttons":              true,
	"static_select":              true,
	"external_select":            true,
	"users_select":               true,
	"conversations_select":       true,
	"channels_select":            true,
	"multi_static_select":        true,
	"multi_external_select":      true,
	"multi_users_select":         true,
	"multi_conversations_select": true,
	"multi_channels_select":      true,
	"workflow_button":            true,
	"plain_text_input":           true,
	"email_text_input":           true,
	"url_text_input":             true,
	"number_input":               true,
	"file_input":                 true,
}

type Sink struct{}

func (Sink) Provider() string { return "slack" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
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
	var p sink.Problems

	hasText, hasBlocks, hasAtt := false, false, false
	if raw, ok := m["text"]; ok && raw != nil {
		s, ok := raw.(string)
		if !ok {
			p.Add("/text", "must be a string")
		} else if strings.TrimSpace(s) != "" {
			hasText = true
			p.MaxRunes("/text", s, maxText)
		}
	}
	if raw, ok := m["blocks"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/blocks")
		if ok && len(arr) > 0 {
			hasBlocks = true
			_ = p.MaxItems("/blocks", len(arr), maxBlocks)
			n := len(arr)
			if n > maxBlocks {
				n = maxBlocks
			}
			for i := 0; i < n; i++ {
				validateBlock(&p, sink.At("/blocks", i), arr[i])
			}
		}
	}
	if raw, ok := m["attachments"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/attachments")
		if ok && len(arr) > 0 {
			hasAtt = true
			_ = p.MaxItems("/attachments", len(arr), maxAttach)
			n := len(arr)
			if n > maxAttach {
				n = maxAttach
			}
			for i := 0; i < n; i++ {
				validateAttachment(&p, sink.At("/attachments", i), arr[i])
			}
		}
	}
	if !hasText && !hasBlocks && !hasAtt {
		p.Add("/", "at least one of text, blocks, or attachments is required")
	}
	return p.Result()
}

func validateBlock(p *sink.Problems, path string, v any) {
	bm, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, _ := bm["type"].(string)
	if typ == "" {
		p.Add(sink.Path(path, "type"), "type is required")
		return
	}
	if !blockTypes[typ] {
		p.Add(sink.Path(path, "type"), "unknown block type "+typ)
		return
	}
	if id, ok := bm["block_id"].(string); ok {
		p.MaxRunes(sink.Path(path, "block_id"), id, maxBlockID)
	}
	switch typ {
	case "section":
		_, text := bm["text"]
		fields, hasFields := bm["fields"]
		_, accessory := bm["accessory"]
		if !text && !hasFields && !accessory {
			p.Add(path, "section needs text, fields, or accessory")
		}
		if text {
			validateTextObj(p, sink.Path(path, "text"), bm["text"], maxSection, true)
		}
		if hasFields {
			arr, ok := p.RequireArray(fields, sink.Path(path, "fields"))
			if ok {
				_ = p.MaxItems(sink.Path(path, "fields"), len(arr), maxFields)
				n := len(arr)
				if n > maxFields {
					n = maxFields
				}
				for i := 0; i < n; i++ {
					validateTextObj(p, sink.At(sink.Path(path, "fields"), i), arr[i], maxFieldText, true)
				}
			}
		}
		if accessory {
			am, ok := p.RequireObject(bm["accessory"], sink.Path(path, "accessory"))
			if ok {
				if t, _ := am["type"].(string); t == "image" {
					validateImage(p, sink.Path(path, "accessory"), am)
				} else {
					validateAction(p, sink.Path(path, "accessory"), bm["accessory"])
				}
			}
		}
	case "actions":
		raw, ok := bm["elements"]
		if !ok {
			p.Add(sink.Path(path, "elements"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "elements"))
		if !ok {
			return
		}
		if len(arr) == 0 {
			p.Add(sink.Path(path, "elements"), "must have at least 1 item")
			return
		}
		_ = p.MaxItems(sink.Path(path, "elements"), len(arr), maxActions)
		n := len(arr)
		if n > maxActions {
			n = maxActions
		}
		for i := 0; i < n; i++ {
			validateAction(p, sink.At(sink.Path(path, "elements"), i), arr[i])
		}
	case "context":
		raw, ok := bm["elements"]
		if !ok {
			p.Add(sink.Path(path, "elements"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "elements"))
		if !ok {
			return
		}
		_ = p.MaxItems(sink.Path(path, "elements"), len(arr), maxContext)
		n := len(arr)
		if n > maxContext {
			n = maxContext
		}
		for i := 0; i < n; i++ {
			validateContextEl(p, sink.At(sink.Path(path, "elements"), i), arr[i])
		}
	case "header":
		validateTextObj(p, sink.Path(path, "text"), bm["text"], maxHeader, false)
	case "image":
		validateImage(p, path, bm)
	case "markdown":
		if s, ok := bm["text"].(string); !ok || strings.TrimSpace(s) == "" {
			p.Add(sink.Path(path, "text"), "required")
		}
	case "video":
		if u, _ := bm["video_url"].(string); strings.TrimSpace(u) == "" {
			p.Add(sink.Path(path, "video_url"), "required")
		}
		if u, _ := bm["thumbnail_url"].(string); strings.TrimSpace(u) == "" {
			p.Add(sink.Path(path, "thumbnail_url"), "required")
		}
		if a, _ := bm["alt_text"].(string); strings.TrimSpace(a) == "" {
			p.Add(sink.Path(path, "alt_text"), "required")
		}
		validateTextObj(p, sink.Path(path, "title"), bm["title"], 200, false)
	case "file":
		if id, _ := bm["external_id"].(string); strings.TrimSpace(id) == "" {
			p.Add(sink.Path(path, "external_id"), "required")
		}
		if src, _ := bm["source"].(string); strings.TrimSpace(src) == "" {
			p.Add(sink.Path(path, "source"), "required")
		}
	case "rich_text":
		raw, ok := bm["elements"]
		if !ok {
			p.Add(sink.Path(path, "elements"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "elements"))
		if !ok {
			return
		}
		for i, el := range arr {
			validateRichText(p, sink.At(sink.Path(path, "elements"), i), el)
		}
	case "table":
		if raw, ok := bm["rows"]; ok && raw != nil {
			p.RequireArray(raw, sink.Path(path, "rows"))
		} else {
			p.Add(sink.Path(path, "rows"), "required")
		}
	case "input":
		if _, ok := bm["element"]; !ok {
			p.Add(sink.Path(path, "element"), "required")
		} else {
			validateAction(p, sink.Path(path, "element"), bm["element"])
		}
		validateTextObj(p, sink.Path(path, "label"), bm["label"], 2000, false)
	}
}

func validateImage(p *sink.Problems, path string, bm map[string]any) {
	if a, ok := bm["alt_text"].(string); !ok || strings.TrimSpace(a) == "" {
		p.Add(sink.Path(path, "alt_text"), "required")
	}
	_, url := bm["image_url"].(string)
	_, file := bm["slack_file"]
	if !url && !file {
		p.Add(path, "image needs image_url or slack_file")
	}
}

func validateContextEl(p *sink.Problems, path string, v any) {
	el, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, _ := el["type"].(string)
	switch typ {
	case "image":
		validateImage(p, path, el)
	case "mrkdwn", "plain_text":
		validateTextObj(p, path, el, maxSection, typ == "mrkdwn")
	case "":
		p.Add(sink.Path(path, "type"), "required")
	default:
		p.Add(sink.Path(path, "type"), "must be image, plain_text, or mrkdwn")
	}
}

var richTextTypes = map[string]bool{
	"rich_text_section":      true,
	"rich_text_list":         true,
	"rich_text_preformatted": true,
	"rich_text_quote":        true,
}

var richLeafTypes = map[string]bool{
	"text": true, "link": true, "emoji": true, "user": true, "usergroup": true,
	"channel": true, "team": true, "date": true, "broadcast": true, "color": true,
}

func validateRichText(p *sink.Problems, path string, v any) {
	el, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, _ := el["type"].(string)
	if !richTextTypes[typ] {
		p.Add(sink.Path(path, "type"), "must be rich_text_section, rich_text_list, rich_text_preformatted, or rich_text_quote")
		return
	}
	raw, ok := el["elements"]
	if !ok {
		p.Add(sink.Path(path, "elements"), "required")
		return
	}
	arr, ok := p.RequireArray(raw, sink.Path(path, "elements"))
	if !ok {
		return
	}
	for i, leaf := range arr {
		lp := sink.At(sink.Path(path, "elements"), i)
		lm, ok := p.RequireObject(leaf, lp)
		if !ok {
			continue
		}
		lt, _ := lm["type"].(string)
		if !richLeafTypes[lt] {
			p.Add(sink.Path(lp, "type"), "unknown rich_text element type")
		}
	}
}

func validateAction(p *sink.Problems, path string, v any) {
	el, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, _ := el["type"].(string)
	if typ == "" {
		p.Add(sink.Path(path, "type"), "type is required")
		return
	}
	if !actionTypes[typ] {
		p.Add(sink.Path(path, "type"), "unknown action type "+typ)
		return
	}
	if id, ok := el["action_id"].(string); ok {
		p.MaxRunes(sink.Path(path, "action_id"), id, maxActionID)
	}
	switch typ {
	case "button", "workflow_button":
		validateTextObj(p, sink.Path(path, "text"), el["text"], maxButtonText, false)
		if u, ok := el["url"].(string); ok {
			p.MaxRunes(sink.Path(path, "url"), u, maxButtonURL)
		}
		if v, ok := el["value"].(string); ok {
			p.MaxRunes(sink.Path(path, "value"), v, maxButtonVal)
		}
		if st, ok := el["style"].(string); ok && st != "" && st != "primary" && st != "danger" {
			p.Add(sink.Path(path, "style"), "must be primary or danger")
		}
	case "overflow":
		validateOptions(p, path, el, 1, maxOverflow, false)
	case "checkboxes", "radio_buttons":
		validateOptions(p, path, el, 1, maxOptions, false)
	case "static_select", "multi_static_select":
		_, opts := el["options"]
		_, groups := el["option_groups"]
		if !opts && !groups {
			p.Add(path, "static_select needs options or option_groups")
		}
		if opts && groups {
			p.Add(path, "options and option_groups cannot both be set")
		}
		if opts {
			validateOptions(p, path, el, 1, maxOptions, false)
		}
		if groups {
			arr, ok := p.RequireArray(el["option_groups"], sink.Path(path, "option_groups"))
			if ok {
				for i, g := range arr {
					gp := sink.At(sink.Path(path, "option_groups"), i)
					gm, ok := p.RequireObject(g, gp)
					if !ok {
						continue
					}
					validateTextObj(p, sink.Path(gp, "label"), gm["label"], 75, false)
					validateOptions(p, gp, gm, 1, maxOptions, false)
				}
			}
		}
	case "datepicker", "timepicker", "datetimepicker", "users_select", "conversations_select", "channels_select", "external_select", "multi_users_select", "multi_conversations_select", "multi_channels_select", "multi_external_select", "plain_text_input", "email_text_input", "url_text_input", "number_input", "file_input":
		if ph, ok := el["placeholder"]; ok {
			validateTextObj(p, sink.Path(path, "placeholder"), ph, maxSelectPh, false)
		}
	case "image":
		validateImage(p, path, el)
	}
}

func validateOptions(p *sink.Problems, path string, el map[string]any, min, max int, _ bool) {
	raw, ok := el["options"]
	if !ok {
		p.Add(sink.Path(path, "options"), "required")
		return
	}
	arr, ok := p.RequireArray(raw, sink.Path(path, "options"))
	if !ok {
		return
	}
	if len(arr) < min {
		p.Add(sink.Path(path, "options"), "must have at least 1 item")
	}
	_ = p.MaxItems(sink.Path(path, "options"), len(arr), max)
	n := len(arr)
	if n > max {
		n = max
	}
	for i := 0; i < n; i++ {
		op := sink.At(sink.Path(path, "options"), i)
		om, ok := p.RequireObject(arr[i], op)
		if !ok {
			continue
		}
		validateTextObj(p, sink.Path(op, "text"), om["text"], maxOptionText, false)
		if val, ok := om["value"].(string); !ok {
			p.Add(sink.Path(op, "value"), "required")
		} else {
			p.MaxRunes(sink.Path(op, "value"), val, maxOptionVal)
		}
	}
}

func validateTextObj(p *sink.Problems, path string, v any, max int, allowMrkdwn bool) {
	if v == nil {
		p.Add(path, "required")
		return
	}
	m, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, _ := m["type"].(string)
	if allowMrkdwn {
		if typ != "plain_text" && typ != "mrkdwn" {
			p.Add(sink.Path(path, "type"), "must be plain_text or mrkdwn")
		}
	} else if typ != "plain_text" {
		p.Add(sink.Path(path, "type"), "must be plain_text")
	}
	text, ok := m["text"].(string)
	if !ok {
		p.Add(sink.Path(path, "text"), "required")
		return
	}
	p.MaxRunes(sink.Path(path, "text"), text, max)
}

func validateAttachment(p *sink.Problems, path string, v any) {
	am, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	if raw, ok := am["actions"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, sink.Path(path, "actions"))
		if !ok {
			return
		}
		for i, a := range arr {
			ap := sink.At(sink.Path(path, "actions"), i)
			obj, ok := p.RequireObject(a, ap)
			if !ok {
				continue
			}
			if _, ok := obj["type"].(string); !ok {
				p.Add(sink.Path(ap, "type"), "required")
			}
			if s, ok := obj["text"].(string); !ok || strings.TrimSpace(s) == "" {
				p.Add(sink.Path(ap, "text"), "required")
			}
		}
	}
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
