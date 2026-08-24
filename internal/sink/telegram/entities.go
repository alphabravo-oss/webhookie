package telegram

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
)

// Formatting rules from https://core.telegram.org/bots/api#formatting-options
// We implement the documented grammars. This is not a byte-clone of Telegram's
// C++ parser; unmatched tags, unescaped MarkdownV2 reserved chars, and
// unsupported HTML tags are rejected as the Bot API documents.

var htmlTags = map[string]bool{
	"b": true, "strong": true, "i": true, "em": true, "u": true, "ins": true,
	"s": true, "strike": true, "del": true, "span": true, "tg-spoiler": true,
	"a": true, "tg-emoji": true, "tg-time": true, "code": true, "pre": true,
	"blockquote": true,
}

var namedHTMLEntities = map[string]bool{
	"lt": true, "gt": true, "amp": true, "quot": true,
}

var mdv2MustEscape = map[rune]bool{
	'_': true, '*': true, '[': true, ']': true, '(': true, ')': true,
	'~': true, '`': true, '>': true, '#': true, '+': true, '-': true,
	'=': true, '|': true, '{': true, '}': true, '.': true, '!': true,
}

func parseEntities(mode, text string) error {
	switch mode {
	case "HTML":
		return parseHTML(text)
	case "Markdown":
		return parseMarkdown(text)
	case "MarkdownV2":
		return parseMarkdownV2(text)
	default:
		return fmt.Errorf("unsupported parse_mode")
	}
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}

func byteOffset(s string, runeIndex int) int {
	if runeIndex <= 0 {
		return 0
	}
	i := 0
	for idx := range s {
		if i == runeIndex {
			return idx
		}
		i++
	}
	return len(s)
}

func parseHTML(s string) error {
	type frame struct {
		name string
		off  int
	}
	var stack []frame
	runes := []rune(s)
	for i := 0; i < len(runes); {
		c := runes[i]
		switch c {
		case '&':
			n, err := consumeHTMLEntity(runes, i)
			if err != nil {
				return fmt.Errorf("can't parse entities: %s at byte offset %d", err.Error(), byteOffset(s, i))
			}
			i = n
		case '<':
			name, closing, next, err := consumeHTMLTag(runes, i)
			if err != nil {
				return fmt.Errorf("can't parse entities: %s at byte offset %d", err.Error(), byteOffset(s, i))
			}
			if closing {
				if len(stack) == 0 || stack[len(stack)-1].name != name {
					return fmt.Errorf("can't parse entities: unmatched end tag </%s> at byte offset %d", name, byteOffset(s, i))
				}
				stack = stack[:len(stack)-1]
			} else {
				stack = append(stack, frame{name: name, off: byteOffset(s, i)})
			}
			i = next
		case '>':
			return fmt.Errorf("can't parse entities: unescaped '>' at byte offset %d", byteOffset(s, i))
		default:
			i++
		}
	}
	if len(stack) > 0 {
		f := stack[len(stack)-1]
		return fmt.Errorf("can't parse entities: can't find end tag </%s> starting at byte offset %d", f.name, f.off)
	}
	return nil
}

func consumeHTMLEntity(runes []rune, i int) (int, error) {
	if i+1 >= len(runes) {
		return 0, fmt.Errorf("unescaped '&'")
	}
	j := i + 1
	if runes[j] == '#' {
		j++
		hex := false
		if j < len(runes) && (runes[j] == 'x' || runes[j] == 'X') {
			hex = true
			j++
		}
		start := j
		for j < len(runes) && isAlnum(runes[j]) {
			j++
		}
		if start == j || j >= len(runes) || runes[j] != ';' {
			return 0, fmt.Errorf("invalid numeric HTML entity")
		}
		digits := string(runes[start:j])
		var n int64
		var err error
		if hex {
			n, err = strconv.ParseInt(digits, 16, 32)
		} else {
			n, err = strconv.ParseInt(digits, 10, 32)
		}
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid numeric HTML entity")
		}
		return j + 1, nil
	}
	start := j
	for j < len(runes) && isNameChar(runes[j]) {
		j++
	}
	if start == j || j >= len(runes) || runes[j] != ';' {
		return 0, fmt.Errorf("unescaped '&'")
	}
	name := string(runes[start:j])
	if !namedHTMLEntities[name] {
		return 0, fmt.Errorf("unsupported HTML entity &%s;", name)
	}
	return j + 1, nil
}

func consumeHTMLTag(runes []rune, i int) (name string, closing bool, next int, err error) {
	if i+1 >= len(runes) {
		return "", false, 0, fmt.Errorf("unescaped '<'")
	}
	j := i + 1
	if runes[j] == '/' {
		closing = true
		j++
	}
	start := j
	for j < len(runes) && isNameChar(runes[j]) {
		j++
	}
	if start == j {
		return "", false, 0, fmt.Errorf("unescaped '<'")
	}
	name = strings.ToLower(string(runes[start:j]))
	if !htmlTags[name] {
		return "", false, 0, fmt.Errorf("unsupported start tag \"%s\"", name)
	}
	for j < len(runes) && runes[j] != '>' {
		if runes[j] == '<' {
			return "", false, 0, fmt.Errorf("unescaped '<'")
		}
		j++
	}
	if j >= len(runes) {
		return "", false, 0, fmt.Errorf("unterminated tag")
	}
	inner := strings.TrimSpace(string(runes[start:j]))
	if !closing {
		if err := checkHTMLTagAttrs(name, inner); err != nil {
			return "", false, 0, err
		}
	}
	return name, closing, j + 1, nil
}

func checkHTMLTagAttrs(name, open string) error {
	lower := strings.ToLower(open)
	switch name {
	case "a":
		if !strings.Contains(lower, "href=") {
			return fmt.Errorf("tag <a> requires href")
		}
	case "tg-emoji":
		if !strings.Contains(lower, "emoji-id=") {
			return fmt.Errorf("tag <tg-emoji> requires emoji-id")
		}
	case "tg-time":
		if !strings.Contains(lower, "unix=") {
			return fmt.Errorf("tag <tg-time> requires unix")
		}
	case "span":
		if !strings.Contains(lower, `class="tg-spoiler"`) && !strings.Contains(lower, `class='tg-spoiler'`) {
			return fmt.Errorf(`tag <span> only supports class="tg-spoiler"`)
		}
	}
	return nil
}

func isNameChar(r rune) bool {
	return r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func parseMarkdown(s string) error {
	runes := []rune(s)
	in := ""
	start := 0
	for i := 0; i < len(runes); {
		if runes[i] == '\\' && in == "" {
			if i+1 >= len(runes) {
				return fmt.Errorf("can't parse entities: unescaped '\\' at byte offset %d", byteOffset(s, i))
			}
			i += 2
			continue
		}
		if in == "" && i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
			in = "```"
			start = i
			i += 3
			continue
		}
		if in == "```" {
			if i+2 < len(runes) && runes[i] == '`' && runes[i+1] == '`' && runes[i+2] == '`' {
				in = ""
				i += 3
				continue
			}
			i++
			continue
		}
		c := runes[i]
		switch in {
		case "":
			switch c {
			case '*', '_', '`':
				in = string(c)
				start = i
				i++
			case '[':
				in = "["
				start = i
				i++
			default:
				i++
			}
		case "*", "_", "`":
			if c == rune(in[0]) {
				if i == start+1 {
					return fmt.Errorf("can't parse entities: empty %s entity at byte offset %d", in, byteOffset(s, start))
				}
				in = ""
			}
			i++
		case "[":
			if c == ']' {
				if i+1 >= len(runes) || runes[i+1] != '(' {
					return fmt.Errorf("can't parse entities: can't find end of URL entity starting at byte offset %d", byteOffset(s, start))
				}
				i += 2
				closed := false
				for i < len(runes) {
					if runes[i] == ')' {
						closed = true
						i++
						break
					}
					i++
				}
				if !closed {
					return fmt.Errorf("can't parse entities: can't find end of URL entity starting at byte offset %d", byteOffset(s, start))
				}
				in = ""
			} else {
				i++
			}
		default:
			i++
		}
	}
	if in != "" {
		return fmt.Errorf("can't parse entities: can't find end of the entity starting at byte offset %d", byteOffset(s, start))
	}
	return nil
}

func parseMarkdownV2(s string) error {
	p := mdv2{s: s, r: []rune(s)}
	if err := p.consume(""); err != nil {
		return err
	}
	return nil
}

type mdv2 struct {
	s string
	r []rune
	i int
}

func (p *mdv2) off() int { return byteOffset(p.s, p.i) }

func (p *mdv2) consume(stop string) error {
	for p.i < len(p.r) {
		if stop != "" && p.has(stop) {
			p.i += len([]rune(stop))
			return nil
		}
		if p.r[p.i] == '\\' {
			if p.i+1 >= len(p.r) {
				return fmt.Errorf("can't parse entities: unescaped '\\' at byte offset %d", p.off())
			}
			p.i += 2
			continue
		}
		if stop == "" && p.atLineStart() && p.has("**>") {
			p.i += 3
			continue
		}
		if stop == "" && p.atLineStart() && p.r[p.i] == '>' {
			p.i++
			continue
		}
		if p.has("```") {
			start := p.off()
			p.i += 3
			if err := p.consumeCode(true); err != nil {
				return fmt.Errorf("can't parse entities: can't find end of pre entity starting at byte offset %d", start)
			}
			continue
		}
		if p.r[p.i] == '`' {
			start := p.off()
			p.i++
			if err := p.consumeCode(false); err != nil {
				return fmt.Errorf("can't parse entities: can't find end of code entity starting at byte offset %d", start)
			}
			continue
		}
		if p.has("||") {
			start := p.off()
			p.i += 2
			if err := p.consume("||"); err != nil {
				return fmt.Errorf("can't parse entities: can't find end of spoiler entity starting at byte offset %d", start)
			}
			continue
		}
		if p.has("__") {
			start := p.off()
			p.i += 2
			if err := p.consume("__"); err != nil {
				return fmt.Errorf("can't parse entities: can't find end of underline entity starting at byte offset %d", start)
			}
			continue
		}
		switch p.r[p.i] {
		case '*':
			start := p.off()
			p.i++
			if err := p.consume("*"); err != nil {
				return fmt.Errorf("can't parse entities: can't find end of bold entity starting at byte offset %d", start)
			}
		case '_':
			start := p.off()
			p.i++
			if err := p.consume("_"); err != nil {
				return fmt.Errorf("can't parse entities: can't find end of italic entity starting at byte offset %d", start)
			}
		case '~':
			start := p.off()
			p.i++
			if err := p.consume("~"); err != nil {
				return fmt.Errorf("can't parse entities: can't find end of strikethrough entity starting at byte offset %d", start)
			}
		case '[':
			if err := p.consumeLink(false); err != nil {
				return err
			}
		case '!':
			if p.i+1 < len(p.r) && p.r[p.i+1] == '[' {
				if err := p.consumeLink(true); err != nil {
					return err
				}
				break
			}
			return fmt.Errorf("can't parse entities: character '!' is reserved and must be escaped with the preceding '\\' at byte offset %d", p.off())
		default:
			c := p.r[p.i]
			if mdv2MustEscape[c] {
				if stop != "" && strings.HasPrefix(stop, string(c)) {
					return fmt.Errorf("can't parse entities: can't find end of the entity starting at byte offset %d", p.off())
				}
				return fmt.Errorf("can't parse entities: character '%c' is reserved and must be escaped with the preceding '\\' at byte offset %d", c, p.off())
			}
			p.i++
		}
	}
	if stop != "" {
		return fmt.Errorf("unclosed")
	}
	return nil
}

func (p *mdv2) has(s string) bool {
	rr := []rune(s)
	if p.i+len(rr) > len(p.r) {
		return false
	}
	for k, r := range rr {
		if p.r[p.i+k] != r {
			return false
		}
	}
	return true
}

func (p *mdv2) atLineStart() bool {
	return p.i == 0 || p.r[p.i-1] == '\n'
}

func (p *mdv2) consumeCode(pre bool) error {
	for p.i < len(p.r) {
		if p.r[p.i] == '\\' {
			if p.i+1 < len(p.r) && (p.r[p.i+1] == '\\' || p.r[p.i+1] == '`') {
				p.i += 2
				continue
			}
		}
		if pre && p.has("```") {
			p.i += 3
			return nil
		}
		if !pre && p.r[p.i] == '`' {
			p.i++
			return nil
		}
		p.i++
	}
	return fmt.Errorf("unclosed")
}

func (p *mdv2) consumeLink(bang bool) error {
	start := p.off()
	if bang {
		p.i++ // !
	}
	p.i++ // [
	if err := p.consume("]"); err != nil {
		return fmt.Errorf("can't parse entities: can't find end of URL entity starting at byte offset %d", start)
	}
	if !p.has("(") {
		return fmt.Errorf("can't parse entities: can't find end of URL entity starting at byte offset %d", start)
	}
	p.i++
	for p.i < len(p.r) {
		if p.r[p.i] == '\\' {
			if p.i+1 >= len(p.r) {
				return fmt.Errorf("can't parse entities: unescaped '\\' at byte offset %d", p.off())
			}
			p.i += 2
			continue
		}
		if p.r[p.i] == ')' {
			p.i++
			return nil
		}
		p.i++
	}
	return fmt.Errorf("can't parse entities: can't find end of URL entity starting at byte offset %d", start)
}

func validateMessageEntities(p *sink.Problems, text string, raw any) {
	arr, ok := p.RequireArray(raw, "/entities")
	if !ok {
		return
	}
	total := utf16Len(text)
	known := map[string]bool{
		"mention": true, "hashtag": true, "cashtag": true, "bot_command": true,
		"url": true, "email": true, "phone_number": true, "bold": true, "italic": true,
		"underline": true, "strikethrough": true, "spoiler": true, "blockquote": true,
		"expandable_blockquote": true, "code": true, "pre": true, "text_link": true,
		"text_mention": true, "custom_emoji": true, "date_time": true,
	}
	for i, item := range arr {
		ip := sink.At("/entities", i)
		obj, ok := p.RequireObject(item, ip)
		if !ok {
			continue
		}
		typ, _ := obj["type"].(string)
		if typ == "" {
			p.Add(sink.Path(ip, "type"), "required")
		} else if !known[typ] {
			p.Add(sink.Path(ip, "type"), "unknown entity type "+typ)
		}
		off, okOff := asInt(obj["offset"])
		ln, okLn := asInt(obj["length"])
		if !okOff {
			p.Add(sink.Path(ip, "offset"), "required")
		}
		if !okLn || ln < 1 {
			p.Add(sink.Path(ip, "length"), "must be >= 1")
		}
		if okOff && okLn && (off < 0 || off+ln > total) {
			p.Add(ip, "offset+length exceeds UTF-16 length of text")
		}
		if typ == "text_link" {
			if u, _ := obj["url"].(string); strings.TrimSpace(u) == "" {
				p.Add(sink.Path(ip, "url"), "required")
			}
		}
		if typ == "custom_emoji" {
			if id, _ := obj["custom_emoji_id"].(string); strings.TrimSpace(id) == "" {
				p.Add(sink.Path(ip, "custom_emoji_id"), "required")
			}
		}
	}
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}
