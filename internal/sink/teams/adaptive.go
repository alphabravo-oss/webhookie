package teams

import (
	"strconv"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
)

// Documented Adaptive Card element types and required properties from
// https://adaptivecards.io/explorer/ — extra fields (including msteams) allowed.
// Not a full JSON Schema (additionalProperties is false in the published schema;
// we do not reject extras). Version gates match the explorer "Version" column.

func versionCode(s string) (int, bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	if len(parts) == 0 || parts[0] == "" {
		return 0, false
	}
	maj, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}
	min := 0
	if len(parts) > 1 {
		min, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, false
		}
	}
	return maj*100 + min, true
}

var elementMin = map[string]int{
	"TextBlock":       100,
	"Image":           100,
	"Container":       100,
	"ColumnSet":       100,
	"Column":          100,
	"FactSet":         100,
	"ImageSet":        100,
	"Input.Text":      100,
	"Input.Number":    100,
	"Input.Date":      100,
	"Input.Time":      100,
	"Input.Toggle":    100,
	"Input.ChoiceSet": 100,
	"Media":           101,
	"ActionSet":       102,
	"RichTextBlock":   102,
	"Table":           105,
	"TableRow":        105,
	"TableCell":       105,
}

var actionMin = map[string]int{
	"Action.OpenUrl":          100,
	"Action.Submit":           100,
	"Action.ShowCard":         100,
	"Action.ToggleVisibility": 101,
	"Action.Execute":          104,
}

func needVersion(p *sink.Problems, path, typ string, have, want int) bool {
	if have < want {
		maj, min := want/100, want%100
		p.Add(path+"/type", typ+" requires Adaptive Card version "+strconv.Itoa(maj)+"."+strconv.Itoa(min))
		return false
	}
	return true
}

func validateAdaptiveCard(p *sink.Problems, path string, card map[string]any, nested bool) {
	if t, _ := card["type"].(string); t != "AdaptiveCard" {
		p.Add(sink.Path(path, "type"), "must be AdaptiveCard")
	}
	ver := 100
	if raw, ok := card["version"].(string); ok && raw != "" {
		v, ok := versionCode(raw)
		if !ok {
			p.Add(sink.Path(path, "version"), "must be a dotted version like 1.4")
		} else {
			ver = v
		}
	} else if !nested {
		p.Add(sink.Path(path, "version"), "required")
	}
	if raw, ok := card["body"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, sink.Path(path, "body"))
		if ok {
			for i, item := range arr {
				validateElement(p, sink.At(sink.Path(path, "body"), i), item, ver)
			}
		}
	}
	if raw, ok := card["actions"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, sink.Path(path, "actions"))
		if ok {
			for i, item := range arr {
				validateAction(p, sink.At(sink.Path(path, "actions"), i), item, ver)
			}
		}
	}
}

func validateElement(p *sink.Problems, path string, v any, ver int) {
	el, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, _ := el["type"].(string)
	if typ == "" {
		p.Add(sink.Path(path, "type"), "required")
		return
	}
	min, known := elementMin[typ]
	if !known {
		p.Add(sink.Path(path, "type"), "unknown element type "+typ)
		return
	}
	if !needVersion(p, path, typ, ver, min) {
		return
	}
	switch typ {
	case "TextBlock":
		if _, ok := el["text"].(string); !ok {
			p.Add(sink.Path(path, "text"), "required")
		}
	case "Image":
		if u, _ := el["url"].(string); strings.TrimSpace(u) == "" {
			p.Add(sink.Path(path, "url"), "required")
		}
	case "Container", "Column", "TableCell":
		if raw, ok := el["items"]; ok && raw != nil {
			arr, ok := p.RequireArray(raw, sink.Path(path, "items"))
			if ok {
				for i, item := range arr {
					validateElement(p, sink.At(sink.Path(path, "items"), i), item, ver)
				}
			}
		} else if typ == "Container" {
			p.Add(sink.Path(path, "items"), "required")
		}
		if sa, ok := el["selectAction"]; ok && sa != nil {
			validateAction(p, sink.Path(path, "selectAction"), sa, ver)
		}
	case "ColumnSet":
		raw, ok := el["columns"]
		if !ok {
			p.Add(sink.Path(path, "columns"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "columns"))
		if !ok {
			return
		}
		for i, c := range arr {
			cp := sink.At(sink.Path(path, "columns"), i)
			col, ok := p.RequireObject(c, cp)
			if !ok {
				continue
			}
			if t, _ := col["type"].(string); t == "" {
				col["type"] = "Column"
			} else if t != "Column" {
				p.Add(sink.Path(cp, "type"), "must be Column")
				continue
			}
			validateElement(p, cp, col, ver)
		}
	case "FactSet":
		raw, ok := el["facts"]
		if !ok {
			p.Add(sink.Path(path, "facts"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "facts"))
		if !ok {
			return
		}
		for i, f := range arr {
			fp := sink.At(sink.Path(path, "facts"), i)
			fm, ok := p.RequireObject(f, fp)
			if !ok {
				continue
			}
			if _, ok := fm["title"].(string); !ok {
				p.Add(sink.Path(fp, "title"), "required")
			}
			if _, ok := fm["value"].(string); !ok {
				p.Add(sink.Path(fp, "value"), "required")
			}
		}
	case "ImageSet":
		raw, ok := el["images"]
		if !ok {
			p.Add(sink.Path(path, "images"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "images"))
		if !ok {
			return
		}
		for i, img := range arr {
			validateElement(p, sink.At(sink.Path(path, "images"), i), img, ver)
		}
	case "Input.Text", "Input.Number", "Input.Date", "Input.Time", "Input.Toggle", "Input.ChoiceSet":
		if id, _ := el["id"].(string); strings.TrimSpace(id) == "" {
			p.Add(sink.Path(path, "id"), "required")
		}
		if typ == "Input.ChoiceSet" {
			if raw, ok := el["choices"]; ok && raw != nil {
				arr, ok := p.RequireArray(raw, sink.Path(path, "choices"))
				if ok {
					for i, c := range arr {
						cp := sink.At(sink.Path(path, "choices"), i)
						cm, ok := p.RequireObject(c, cp)
						if !ok {
							continue
						}
						if _, ok := cm["title"].(string); !ok {
							p.Add(sink.Path(cp, "title"), "required")
						}
						if _, ok := cm["value"].(string); !ok {
							p.Add(sink.Path(cp, "value"), "required")
						}
					}
				}
			}
		}
	case "Media":
		if raw, ok := el["sources"]; ok && raw != nil {
			p.RequireArray(raw, sink.Path(path, "sources"))
		} else {
			p.Add(sink.Path(path, "sources"), "required")
		}
	case "ActionSet":
		raw, ok := el["actions"]
		if !ok {
			p.Add(sink.Path(path, "actions"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "actions"))
		if !ok {
			return
		}
		for i, a := range arr {
			validateAction(p, sink.At(sink.Path(path, "actions"), i), a, ver)
		}
	case "RichTextBlock":
		raw, ok := el["inlines"]
		if !ok {
			p.Add(sink.Path(path, "inlines"), "required")
			return
		}
		arr, ok := p.RequireArray(raw, sink.Path(path, "inlines"))
		if !ok {
			return
		}
		for i, in := range arr {
			ip := sink.At(sink.Path(path, "inlines"), i)
			if _, ok := in.(string); ok {
				continue
			}
			im, ok := p.RequireObject(in, ip)
			if !ok {
				continue
			}
			t, _ := im["type"].(string)
			if t == "TextRun" {
				if _, ok := im["text"].(string); !ok {
					p.Add(sink.Path(ip, "text"), "required")
				}
			}
		}
	case "Table":
		if raw, ok := el["columns"]; ok && raw != nil {
			p.RequireArray(raw, sink.Path(path, "columns"))
		}
		if raw, ok := el["rows"]; ok && raw != nil {
			arr, ok := p.RequireArray(raw, sink.Path(path, "rows"))
			if ok {
				for i, row := range arr {
					validateTableRow(p, sink.At(sink.Path(path, "rows"), i), row, ver)
				}
			}
		}
	}
}

func validateTableRow(p *sink.Problems, path string, v any, ver int) {
	row, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	if raw, ok := row["cells"]; ok && raw != nil {
		arr, ok := p.RequireArray(raw, sink.Path(path, "cells"))
		if ok {
			for i, c := range arr {
				validateElement(p, sink.At(sink.Path(path, "cells"), i), c, ver)
			}
		}
	}
}

func validateAction(p *sink.Problems, path string, v any, ver int) {
	a, ok := p.RequireObject(v, path)
	if !ok {
		return
	}
	typ, _ := a["type"].(string)
	if typ == "" {
		p.Add(sink.Path(path, "type"), "required")
		return
	}
	min, known := actionMin[typ]
	if !known {
		p.Add(sink.Path(path, "type"), "unknown action type "+typ)
		return
	}
	if !needVersion(p, path, typ, ver, min) {
		return
	}
	switch typ {
	case "Action.OpenUrl":
		if u, _ := a["url"].(string); strings.TrimSpace(u) == "" {
			p.Add(sink.Path(path, "url"), "required")
		}
	case "Action.ShowCard":
		card, ok := p.RequireObject(a["card"], sink.Path(path, "card"))
		if ok {
			validateAdaptiveCard(p, sink.Path(path, "card"), card, true)
		}
	case "Action.ToggleVisibility":
		if raw, ok := a["targetElements"]; !ok || raw == nil {
			p.Add(sink.Path(path, "targetElements"), "required")
		} else {
			p.RequireArray(raw, sink.Path(path, "targetElements"))
		}
	}
}
