package generic

import (
	"net/http"
	"strings"

	"github.com/alphabravo-oss/webhookie/internal/sink"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Sink struct{}

func (Sink) Provider() string { return "generic" }

func (Sink) Match(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/hooks/generic/")
}

func (Sink) Validate(_ *http.Request, _ []byte) sink.Validation {
	return sink.Validation{Valid: true}
}

func (Sink) Respond(w http.ResponseWriter, _ *http.Request, _ []byte, _ store.Chaos) error {
	sink.WriteJSON(w, http.StatusOK, `{"ok":true}`)
	return nil
}

func (Sink) Summarize(r *http.Request, body []byte) sink.Summary {
	text := strings.TrimSpace(string(body))
	if text == "" {
		text = r.Method + " " + r.URL.Path
	}
	return sink.Summary{Text: sink.FirstN(text, 80)}
}
