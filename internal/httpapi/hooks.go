package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/chaos"
	"github.com/alphabravo-oss/webhookie/internal/observe"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

func (s *Server) Capture(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	adapter, ok := s.sinks.Match(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	limited := http.MaxBytesReader(w, r.Body, s.cfg.MaxBodyBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || int64(len(body)) > s.cfg.MaxBodyBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds limit")
			return
		}
		writeError(w, 400, "read_error", err.Error())
		return
	}
	truncated := false
	if int64(len(body)) > s.cfg.MaxBodyBytes {
		body = body[:s.cfg.MaxBodyBytes]
		truncated = true
		writeError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds limit")
		return
	}

	sk, err := s.store.GetSinkByPath(r.Context(), r.URL.Path)
	if adapter.Provider() == "pagerduty" {
		if rk := routingKeyFromBody(body); rk != "" {
			if byTok, tokErr := s.store.GetSinkByToken(r.Context(), rk); tokErr == nil {
				sk, err = byTok, nil
			}
		}
	}
	if adapter.Provider() == "opsgenie" {
		if key := genieKey(r); key != "" {
			if byTok, tokErr := s.store.GetSinkByToken(r.Context(), key); tokErr == nil {
				sk, err = byTok, nil
			}
		}
	}
	if err != nil {
		sk, err = s.store.GetSinkByProvider(r.Context(), adapter.Provider())
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	ov, err := chaos.Apply(r.Context(), sk.Chaos)
	if err != nil {
		return
	}

	v := adapter.Validate(r, body)
	sum := adapter.Summarize(r, body)

	rw := &statusRecorder{ResponseWriter: w, status: 200}
	if ov != nil {
		if ov.RetryAfter != "" {
			rw.Header().Set("Retry-After", ov.RetryAfter)
		}
		ct := ov.ContentType
		if ct == "" {
			ct = "text/plain"
		}
		rw.Header().Set("Content-Type", ct)
		rw.WriteHeader(ov.Status)
		_, _ = rw.Write(ov.Body)
	} else {
		_ = adapter.Respond(rw, r, body, sk.Chaos)
	}

	groupKey := sum.GroupKey
	if k := rw.Header().Get("X-Webhookie-Dedup-Key"); k != "" && groupKey == "" {
		groupKey = k
	}
	query := map[string]string{}
	for k, vs := range r.URL.Query() {
		if len(vs) > 0 {
			query[k] = vs[0]
		}
	}
	headers := map[string][]string{}
	for k, vs := range r.Header {
		headers[k] = append([]string(nil), vs...)
	}
	ev := store.Event{
		ID:               newID(),
		SinkID:           sk.ID,
		Provider:         adapter.Provider(),
		ReceivedAt:       time.Now().UTC(),
		Method:           r.Method,
		Path:             r.URL.Path,
		Query:            query,
		Headers:          headers,
		ContentType:      r.Header.Get("Content-Type"),
		Body:             body,
		BodyText:         string(body),
		BodyTruncated:    truncated,
		Status:           rw.status,
		LatencyMS:        int(time.Since(start).Milliseconds()),
		Valid:            v.Valid,
		ValidationErrors: v.Errors,
		Summary:          sum.Text,
		GroupKey:         groupKey,
	}
	if ev.ValidationErrors == nil {
		ev.ValidationErrors = []store.ValidationError{}
	}
	if err := s.store.InsertEvent(r.Context(), ev); err != nil {
		s.log.Error("insert event", "err", err)
	} else {
		observe.Captured.Add(1)
		if !ev.Valid {
			observe.Invalid.Add(1)
		}
		if n, err := s.store.CountEvents(r.Context()); err == nil {
			observe.Stored.Store(int64(n))
		}
		s.hub.Publish(ev)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	hdr    http.Header
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Header() http.Header {
	return s.ResponseWriter.Header()
}

func envelopeEvent(ev store.Event, unredact bool) store.Event {
	out := ev
	out.BodyText = string(ev.Body)
	if !unredact {
		out.Headers = store.RedactHeaders(ev.Headers)
	}
	return out
}

func writeEventJSON(w http.ResponseWriter, ev store.Event) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ev)
}

func hopByHop(h map[string][]string) http.Header {
	out := http.Header{}
	skip := map[string]bool{
		"Content-Length": true, "Connection": true, "Host": true, "Transfer-Encoding": true,
	}
	for k, vs := range h {
		if skip[http.CanonicalHeaderKey(k)] {
			continue
		}
		for _, v := range vs {
			out.Add(k, v)
		}
	}
	return out
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "sse_unsupported", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, cancel := s.hub.Subscribe()
	defer cancel()
	observe.SSEClients.Add(1)
	defer observe.SSEClients.Add(-1)
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			_, _ = io.WriteString(w, ": ping\n\n")
			fl.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(envelopeEvent(ev, false))
			_, _ = io.WriteString(w, "event: webhook\ndata: "+string(b)+"\n\n")
			fl.Flush()
		}
	}
}

func queryLimit(r *http.Request) (limit, offset int) {
	limit = atoiDefault(r.URL.Query().Get("limit"), 50)
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}
	offset = atoiDefault(r.URL.Query().Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}
	return
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return d
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func parseSince(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

func boolQuery(r *http.Request, key string) bool {
	v := strings.ToLower(r.URL.Query().Get(key))
	return v == "1" || v == "true"
}

func routingKeyFromBody(body []byte) string {
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return ""
	}
	rk, _ := m["routing_key"].(string)
	return rk
}

func genieKey(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const p = "GenieKey "
	if strings.HasPrefix(h, p) {
		return strings.TrimSpace(strings.TrimPrefix(h, p))
	}
	return ""
}
