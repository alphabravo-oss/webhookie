package sink

import (
	"net/http"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Validation struct {
	Valid  bool
	Errors []store.ValidationError
}

type Summary struct {
	Text     string
	GroupKey string
}

type Sink interface {
	Provider() string
	Match(r *http.Request) bool
	Validate(r *http.Request, body []byte) Validation
	Respond(w http.ResponseWriter, r *http.Request, body []byte, ch store.Chaos) error
	Summarize(r *http.Request, body []byte) Summary
}

type Registry struct{ items []Sink }

func (r *Registry) Register(s Sink) { r.items = append(r.items, s) }

func (r *Registry) Match(req *http.Request) (Sink, bool) {
	for _, s := range r.items {
		if s.Match(req) {
			return s, true
		}
	}
	return nil, false
}

func FirstN(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func HasPrefixPath(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func WriteJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func WriteText(w http.ResponseWriter, status int, body, contentType string) {
	if contentType == "" {
		contentType = "text/plain"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
