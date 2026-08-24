package sink

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

type ctxKey int

const (
	eventIDKey   ctxKey = 1
	forcedValKey ctxKey = 2
)

func WithEventID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, eventIDKey, id)
}

func EventID(ctx context.Context) string {
	id, _ := ctx.Value(eventIDKey).(string)
	return id
}

func WithValidation(ctx context.Context, v Validation) context.Context {
	return context.WithValue(ctx, forcedValKey, v)
}

func ForcedValidation(ctx context.Context) (Validation, bool) {
	v, ok := ctx.Value(forcedValKey).(Validation)
	return v, ok
}

func Truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "true" || s == "1" || s == "yes"
	case float64:
		return t != 0
	default:
		return false
	}
}

// Problems collects JSON-pointer validation errors for a sink payload.
type Problems []store.ValidationError

func (p *Problems) Add(path, message string) {
	*p = append(*p, store.ValidationError{Path: path, Message: message})
}

func (p Problems) Result() Validation {
	if len(p) == 0 {
		return Validation{Valid: true}
	}
	return Validation{Errors: []store.ValidationError(p)}
}

func (v Validation) Has(path, messageSubstr string) bool {
	for _, e := range v.Errors {
		if e.Path != path {
			continue
		}
		if messageSubstr == "" || strings.Contains(strings.ToLower(e.Message), strings.ToLower(messageSubstr)) {
			return true
		}
	}
	return false
}

func Path(parent, child string) string {
	child = strings.TrimPrefix(child, "/")
	if parent == "" || parent == "/" {
		return "/" + child
	}
	return parent + "/" + child
}

func At(parent string, i int) string {
	return Path(parent, strconv.Itoa(i))
}

func Object(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func Array(v any) ([]any, bool) {
	a, ok := v.([]any)
	return a, ok
}

func String(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func RuneCount(s string) int { return utf8.RuneCountInString(s) }

func (p *Problems) MaxRunes(path, s string, max int) {
	if RuneCount(s) > max {
		p.Add(path, "must be "+strconv.Itoa(max)+" characters or fewer")
	}
}

func (p *Problems) MaxItems(path string, n, max int) bool {
	if n > max {
		p.Add(path, "must have at most "+strconv.Itoa(max)+" items")
		return false
	}
	return true
}

func (p *Problems) RequireObject(v any, path string) (map[string]any, bool) {
	m, ok := Object(v)
	if !ok {
		p.Add(path, "must be an object")
		return nil, false
	}
	return m, true
}

func (p *Problems) RequireArray(v any, path string) ([]any, bool) {
	a, ok := Array(v)
	if !ok {
		p.Add(path, "must be an array")
		return nil, false
	}
	return a, true
}
