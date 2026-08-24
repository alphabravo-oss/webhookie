package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/alphabravo-oss/webhookie/internal/observe"
	"github.com/alphabravo-oss/webhookie/internal/source"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	limit, offset := queryLimit(r)
	items, total, err := s.store.ListEvents(r.Context(), store.EventFilter{
		Provider: r.URL.Query().Get("provider"),
		SinkID:   r.URL.Query().Get("sinkId"),
		Since:    parseSince(r.URL.Query().Get("since")),
		GroupKey: r.URL.Query().Get("groupKey"),
		Contains: r.URL.Query().Get("contains"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		writeError(w, 500, "list_failed", err.Error())
		return
	}
	if items == nil {
		items = []store.Event{}
	}
	unredact := boolQuery(r, "unredact")
	out := make([]store.Event, len(items))
	for i, ev := range items {
		out[i] = envelopeEvent(ev, unredact)
	}
	writeJSON(w, 200, map[string]any{
		"data": out,
		"pagination": map[string]any{
			"total":      total,
			"limit":      limit,
			"offset":     offset,
			"hasMore":    offset+len(out) < total,
			"nextOffset": offset + len(out),
		},
	})
}

func (s *Server) getEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ev, err := s.store.GetEvent(r.Context(), id)
	if err != nil {
		writeError(w, 404, "not_found", "event not found")
		return
	}
	writeJSON(w, 200, map[string]any{"data": envelopeEvent(ev, boolQuery(r, "unredact"))})
}

func (s *Server) deleteEvents(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteEvents(r.Context()); err != nil {
		writeError(w, 500, "delete_failed", err.Error())
		return
	}
	observe.Stored.Store(0)
	writeJSON(w, 200, map[string]any{"data": map[string]any{"ok": true}})
}

func (s *Server) listSinks(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSinks(r.Context())
	if err != nil {
		writeError(w, 500, "list_failed", err.Error())
		return
	}
	type sinkOut struct {
		store.Sink
		URL string `json:"url"`
	}
	out := make([]sinkOut, 0, len(items))
	for _, sk := range items {
		out = append(out, sinkOut{Sink: sk, URL: strings.TrimRight(s.cfg.PublicBaseURL, "/") + sk.Path})
	}
	writeJSON(w, 200, map[string]any{"data": out})
}

func (s *Server) getSink(w http.ResponseWriter, r *http.Request) {
	sk, err := s.store.GetSink(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 404, "not_found", "sink not found")
		return
	}
	writeJSON(w, 200, map[string]any{"data": sk})
}

func (s *Server) createSink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.Name == "" {
		in.Name = "Generic bin"
	}
	token := uuid.NewString()
	sk := store.Sink{
		ID:        "sink-" + token[:8],
		Provider:  "generic",
		Name:      in.Name,
		Token:     token,
		Path:      "/hooks/generic/" + token,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.UpsertSink(r.Context(), sk); err != nil {
		writeError(w, 500, "create_failed", err.Error())
		return
	}
	writeJSON(w, 201, map[string]any{"data": sk})
}

func (s *Server) patchSink(w http.ResponseWriter, r *http.Request) {
	sk, err := s.store.GetSink(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 404, "not_found", "sink not found")
		return
	}
	var in struct {
		Chaos *store.Chaos `json:"chaos"`
		Name  string       `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Chaos != nil {
		sk.Chaos = *in.Chaos
	}
	if in.Name != "" {
		sk.Name = in.Name
	}
	if err := s.store.UpsertSink(r.Context(), sk); err != nil {
		writeError(w, 500, "update_failed", err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{"data": sk})
}

func (s *Server) listFixtures(w http.ResponseWriter, r *http.Request) {
	type fx struct {
		Provider    string `json:"provider"`
		Event       string `json:"event"`
		Description string `json:"description"`
	}
	var out []fx
	for _, src := range s.srcs.All() {
		for _, e := range src.Events() {
			out = append(out, fx{Provider: src.Provider(), Event: e.Name, Description: e.Description})
		}
	}
	writeJSON(w, 200, map[string]any{"data": out})
}

func (s *Server) send(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Provider         string `json:"provider"`
		Event            string `json:"event"`
		Target           string `json:"target"`
		Secret           string `json:"secret"`
		TimestampSkewSec int    `json:"timestampSkewSec"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err.Error())
		return
	}
	src, ok := s.srcs.Get(in.Provider)
	if !ok {
		writeError(w, 400, "unknown_provider", "unknown source provider")
		return
	}
	var fx source.Fixture
	found := false
	for _, e := range src.Events() {
		if e.Name == in.Event {
			fx = e
			found = true
			break
		}
	}
	if !found {
		writeError(w, 400, "unknown_event", "unknown event")
		return
	}
	if in.Target == "" {
		writeError(w, 400, "target_required", "target URL is required")
		return
	}
	ts := time.Now().UTC().Add(-time.Duration(in.TimestampSkewSec) * time.Second)
	id := source.NewID()
	hdr := src.Sign(fx.Body, in.Secret, ts, id)
	for k, v := range fx.Headers {
		if hdr.Get(k) == "" {
			hdr.Set(k, v)
		}
	}
	if fx.ContentType != "" && hdr.Get("Content-Type") == "" {
		hdr.Set("Content-Type", fx.ContentType)
	}
	att := source.Deliver(r.Context(), in.Target, hdr, fx.Body, 10*time.Second)
	att.ID = newID()
	att.CreatedAt = time.Now().UTC()
	att.Provider = in.Provider
	att.EventName = in.Event
	att.Target = in.Target
	att.RequestHeaders = hdr
	att.Body = fx.Body
	att.BodyText = string(fx.Body)
	_ = s.store.InsertAttempt(r.Context(), att)
	observe.Sends.Add(1)
	writeJSON(w, 200, map[string]any{"data": att})
}

func (s *Server) listAttempts(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListAttempts(r.Context())
	if err != nil {
		writeError(w, 500, "list_failed", err.Error())
		return
	}
	if items == nil {
		items = []store.SendAttempt{}
	}
	writeJSON(w, 200, map[string]any{"data": items})
}

func (s *Server) replay(w http.ResponseWriter, r *http.Request) {
	ev, err := s.store.GetEvent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 404, "not_found", "event not found")
		return
	}
	var in struct {
		Target string `json:"target"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if in.Target == "" {
		writeError(w, 400, "target_required", "target URL is required")
		return
	}
	hdr := hopByHop(ev.Headers)
	if ev.ContentType != "" && hdr.Get("Content-Type") == "" {
		hdr.Set("Content-Type", ev.ContentType)
	}
	att := source.Deliver(r.Context(), in.Target, hdr, ev.Body, 10*time.Second)
	att.ID = newID()
	att.CreatedAt = time.Now().UTC()
	att.Provider = ev.Provider
	att.EventName = "replay:" + ev.ID
	att.Target = in.Target
	att.RequestHeaders = hdr
	att.Body = ev.Body
	att.BodyText = string(ev.Body)
	_ = s.store.InsertAttempt(r.Context(), att)
	observe.Sends.Add(1)
	writeJSON(w, 200, map[string]any{"data": att})
}
