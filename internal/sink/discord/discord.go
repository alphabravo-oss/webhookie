package discord

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

const (
	maxContent     = 2000
	maxEmbeds      = 10
	maxEmbedTitle  = 256
	maxEmbedDesc   = 4096
	maxEmbedFields = 25
	maxFieldName   = 256
	maxFieldValue  = 1024
	maxFooter      = 2048
	maxAuthor      = 256
	maxEmbedChars  = 6000
	maxUsername    = 80
	maxActionRows  = 5
	maxRowChildren = 5
	maxButtonLabel = 80
	maxCustomID    = 100
	maxSelectOpts  = 25
)

type Sink struct{}

func (Sink) Provider() string { return "discord" }

func (Sink) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	return len(parts) == 6 && parts[0] == "hooks" && parts[1] == "discord" && parts[2] == "api" && parts[3] == "webhooks"
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
	dec, err := decode(r, body)
	if err != nil {
		return sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "invalid json"}}}
	}
	m := dec.m
	var p sink.Problems
	hasContent := false
	if raw, ok := m["content"]; ok && raw != nil {
		s, ok := raw.(string)
		if !ok {
			p.Add("/content", "must be a string")
		} else {
			if strings.TrimSpace(s) != "" {
				hasContent = true
			}
			p.MaxRunes("/content", s, maxContent)
		}
	}
	if raw, ok := m["username"]; ok && raw != nil {
		if s, ok := raw.(string); ok {
			p.MaxRunes("/username", s, maxUsername)
		}
	}

	hasEmbeds := false
	embedChars := 0
	if raw, ok := m["embeds"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/embeds")
		if ok && len(arr) > 0 {
			hasEmbeds = true
			_ = p.MaxItems("/embeds", len(arr), maxEmbeds)
			n := len(arr)
			if n > maxEmbeds {
				n = maxEmbeds
			}
			for i := 0; i < n; i++ {
				embedChars += validateEmbed(&p, sink.At("/embeds", i), arr[i])
			}
			if embedChars > maxEmbedChars {
				p.Add("/embeds", "combined embed text must be "+strconv.Itoa(maxEmbedChars)+" characters or fewer")
			}
		}
	}

	hasComponents := false
	if raw, ok := m["components"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/components")
		if ok && len(arr) > 0 {
			hasComponents = true
			_ = p.MaxItems("/components", len(arr), maxActionRows)
			n := len(arr)
			if n > maxActionRows {
				n = maxActionRows
			}
			for i := 0; i < n; i++ {
				validateActionRow(&p, sink.At("/components", i), arr[i])
			}
		}
	}

	hasAttachments := false
	if raw, ok := m["attachments"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, "/attachments")
		if ok && len(arr) > 0 {
			hasAttachments = true
		}
	}

	hasFiles := dec.files > 0
	if dec.files > maxFiles {
		p.Add("/files", "must have at most "+strconv.Itoa(maxFiles)+" files")
	}

	hasPoll := false
	if raw, ok := m["poll"]; ok && raw != nil {
		pm, ok := p.RequireObject(raw, "/poll")
		if ok {
			q, ok := p.RequireObject(pm["question"], "/poll/question")
			if ok {
				if t, _ := q["text"].(string); strings.TrimSpace(t) != "" {
					hasPoll = true
				} else {
					p.Add("/poll/question/text", "required")
				}
			}
		}
	}

	if !hasContent && !hasEmbeds && !hasComponents && !hasAttachments && !hasFiles && !hasPoll {
		p.Add("/", "content, embeds, components, files, or poll is required")
	}
	return p.Result()
}

func validateEmbed(p *sink.Problems, path string, v any) int {
	em, ok := p.RequireObject(v, path)
	if !ok {
		return 0
	}
	n := 0
	if title, ok := em["title"].(string); ok {
		p.MaxRunes(sink.Path(path, "title"), title, maxEmbedTitle)
		n += sink.RuneCount(title)
	}
	if desc, ok := em["description"].(string); ok {
		p.MaxRunes(sink.Path(path, "description"), desc, maxEmbedDesc)
		n += sink.RuneCount(desc)
	}
	if footer, ok := em["footer"].(map[string]any); ok {
		if t, ok := footer["text"].(string); ok {
			p.MaxRunes(sink.Path(path, "footer/text"), t, maxFooter)
			n += sink.RuneCount(t)
		}
	} else if em["footer"] != nil {
		p.Add(sink.Path(path, "footer"), "must be an object")
	}
	if author, ok := em["author"].(map[string]any); ok {
		if t, ok := author["name"].(string); ok {
			p.MaxRunes(sink.Path(path, "author/name"), t, maxAuthor)
			n += sink.RuneCount(t)
		}
	}
	if raw, ok := em["fields"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, sink.Path(path, "fields"))
		if ok {
			_ = p.MaxItems(sink.Path(path, "fields"), len(arr), maxEmbedFields)
			limit := len(arr)
			if limit > maxEmbedFields {
				limit = maxEmbedFields
			}
			for i := 0; i < limit; i++ {
				fp := sink.At(sink.Path(path, "fields"), i)
				f, ok := p.RequireObject(arr[i], fp)
				if !ok {
					continue
				}
				name, _ := f["name"].(string)
				value, _ := f["value"].(string)
				if strings.TrimSpace(name) == "" {
					p.Add(sink.Path(fp, "name"), "required")
				} else {
					p.MaxRunes(sink.Path(fp, "name"), name, maxFieldName)
					n += sink.RuneCount(name)
				}
				if strings.TrimSpace(value) == "" {
					p.Add(sink.Path(fp, "value"), "required")
				} else {
					p.MaxRunes(sink.Path(fp, "value"), value, maxFieldValue)
					n += sink.RuneCount(value)
				}
			}
		}
	}
	return n
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	case int:
		return n, true
	default:
		return 0, false
	}
}

func validateActionRow(p *sink.Problems, path string, v any) {
	row, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	if typ, ok := asInt(row["type"]); !ok || typ != 1 {
		p.Add(sink.Path(path, "type"), "must be 1 (action row)")
	}
	raw, ok := row["components"]
	if !ok {
		p.Add(sink.Path(path, "components"), "required")
		return
	}
	arr, ok := p.RequireArray(raw, sink.Path(path, "components"))
	if !ok {
		return
	}
	if len(arr) == 0 {
		p.Add(sink.Path(path, "components"), "must have at least 1 item")
		return
	}
	_ = p.MaxItems(sink.Path(path, "components"), len(arr), maxRowChildren)
	n := len(arr)
	if n > maxRowChildren {
		n = maxRowChildren
	}
	for i := 0; i < n; i++ {
		validateComponent(p, sink.At(sink.Path(path, "components"), i), arr[i])
	}
}

func validateComponent(p *sink.Problems, path string, v any) {
	c, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, ok := asInt(c["type"])
	if !ok {
		p.Add(sink.Path(path, "type"), "required")
		return
	}
	switch typ {
	case 2: // button
		if label, ok := c["label"].(string); ok {
			p.MaxRunes(sink.Path(path, "label"), label, maxButtonLabel)
		} else if c["emoji"] == nil {
			p.Add(sink.Path(path, "label"), "required")
		}
		style, ok := asInt(c["style"])
		if !ok {
			p.Add(sink.Path(path, "style"), "required")
		} else if style < 1 || style > 5 {
			p.Add(sink.Path(path, "style"), "must be 1-5")
		}
		if style == 5 {
			if u, ok := c["url"].(string); !ok || strings.TrimSpace(u) == "" {
				p.Add(sink.Path(path, "url"), "required for link buttons")
			}
		} else if cid, ok := c["custom_id"].(string); ok {
			p.MaxRunes(sink.Path(path, "custom_id"), cid, maxCustomID)
		} else {
			p.Add(sink.Path(path, "custom_id"), "required")
		}
	case 3: // string select
		if cid, ok := c["custom_id"].(string); ok {
			p.MaxRunes(sink.Path(path, "custom_id"), cid, maxCustomID)
		} else {
			p.Add(sink.Path(path, "custom_id"), "required")
		}
		if raw, ok := c["options"]; ok {
			arr, ok := p.RequireArray(raw, sink.Path(path, "options"))
			if ok {
				_ = p.MaxItems(sink.Path(path, "options"), len(arr), maxSelectOpts)
			}
		}
	}
}

func isEmptyMessage(v sink.Validation) bool {
	return len(v.Errors) == 1 && v.Has("/", "content, embeds, components, files, or poll")
}

func writeInvalid(w http.ResponseWriter, v sink.Validation) {
	if isEmptyMessage(v) {
		sink.WriteJSON(w, http.StatusBadRequest, `{"message":"Cannot send an empty message","code":50006}`)
		return
	}
	out := map[string]any{
		"message": "Invalid Form Body",
		"code":    50035,
		"errors":  discordErrorTree(v.Errors),
	}
	b, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write(b)
}

func discordErrorTree(errs []store.ValidationError) map[string]any {
	root := map[string]any{}
	for _, e := range errs {
		path := strings.Trim(e.Path, "/")
		cur := root
		if path != "" {
			for _, part := range strings.Split(path, "/") {
				next, ok := cur[part].(map[string]any)
				if !ok {
					next = map[string]any{}
					cur[part] = next
				}
				cur = next
			}
		}
		list, _ := cur["_errors"].([]any)
		list = append(list, map[string]string{"code": "INVALID", "message": e.Message})
		cur["_errors"] = list
	}
	return root
}

func (s Sink) Respond(w http.ResponseWriter, r *http.Request, body []byte, _ store.Chaos) error {
	v := s.Validate(r, body)
	if !v.Valid {
		writeInvalid(w, v)
		return nil
	}
	if r.URL.Query().Get("wait") == "true" {
		sink.WriteJSON(w, http.StatusOK, `{"id":"0","content":"ok"}`)
		return nil
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (Sink) Summarize(r *http.Request, body []byte) sink.Summary {
	dec, err := decode(r, body)
	if err != nil {
		return sink.Summary{Text: "invalid discord payload"}
	}
	m := dec.m
	if c, ok := m["content"].(string); ok && strings.TrimSpace(c) != "" {
		return sink.Summary{Text: sink.FirstN(c, 80)}
	}
	if embeds, ok := m["embeds"].([]any); ok && len(embeds) > 0 {
		if em, ok := embeds[0].(map[string]any); ok {
			if title, ok := em["title"].(string); ok && title != "" {
				return sink.Summary{Text: sink.FirstN(title, 80)}
			}
		}
	}
	if dec.files > 0 {
		return sink.Summary{Text: "(file)"}
	}
	return sink.Summary{Text: "(embed)"}
}
