package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/chaos"
	"github.com/alphabravo-oss/webhookie/internal/observe"
	"github.com/alphabravo-oss/webhookie/internal/sink"
	sinkdiscord "github.com/alphabravo-oss/webhookie/internal/sink/discord"
	sinkslack "github.com/alphabravo-oss/webhookie/internal/sink/slack"
	sinktg "github.com/alphabravo-oss/webhookie/internal/sink/telegram"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

const slackResponseTTL = 30 * time.Minute
const slackResponseMax = 5

func slackResponseEventID(r *http.Request) string {
	if r.Method != http.MethodPost {
		return ""
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 4 && parts[0] == "hooks" && parts[1] == "slack" && parts[2] == "response" {
		return parts[3]
	}
	return ""
}

func discordMessageID(r *http.Request) (string, bool) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 8 {
		return "", false
	}
	if parts[0] != "hooks" || parts[1] != "discord" || parts[2] != "api" || parts[3] != "webhooks" || parts[6] != "messages" {
		return "", false
	}
	if r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		return "", false
	}
	return parts[7], true
}

func isTelegramAnswer(r *http.Request) bool {
	return r.Method == http.MethodPost && strings.HasSuffix(strings.Trim(r.URL.Path, "/"), "/answerCallbackQuery")
}

func parseJSONMap(body []byte) map[string]any {
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return map[string]any{}
	}
	return m
}

func displayJSON(m map[string]any) []byte {
	out := map[string]any{}
	for k, v := range m {
		switch k {
		case "replace_original", "delete_original", "response_type", "unfurl_links", "unfurl_media":
			continue
		default:
			out[k] = v
		}
	}
	b, _ := json.Marshal(out)
	return b
}

func (s *Server) handleSlackResponse(w http.ResponseWriter, r *http.Request, origID string) {
	orig, err := s.store.GetEvent(r.Context(), origID)
	if err != nil || orig.Provider != "slack" {
		http.NotFound(w, r)
		return
	}
	sk, err := s.store.GetSink(r.Context(), orig.SinkID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	body, truncated, ok := s.readCaptureBody(w, r)
	if !ok {
		return
	}
	adapter := sinkslack.Sink{}
	v := adapter.Validate(r, body)
	st, _ := s.store.GetMessageState(r.Context(), orig.ID)
	if v.Valid && time.Since(orig.ReceivedAt) > slackResponseTTL {
		v = sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "response_url expired (30 minutes)"}}}
	} else if v.Valid && st.ResponseCount >= slackResponseMax {
		v = sink.Validation{Errors: []store.ValidationError{{Path: "/", Message: "response_url already used 5 times"}}}
	}
	path := r.URL.Path
	m := parseJSONMap(body)
	replace := sink.Truthy(m["replace_original"])
	del := sink.Truthy(m["delete_original"])
	if v.Valid && !replace && !del {
		path = orig.Path
	}
	if !v.Valid {
		r = r.WithContext(sink.WithValidation(r.Context(), v))
	}
	s.finishFollowup(w, r, sk, adapter, body, truncated, v, path, orig.ID, func() {
		next := st
		next.EventID = orig.ID
		next.ResponseCount = st.ResponseCount + 1
		if del {
			next.Deleted = true
		} else if replace {
			next.DisplayBody = displayJSON(m)
			next.Deleted = false
		}
		_ = s.store.UpsertMessageState(r.Context(), next)
		if replace || del {
			s.publishOriginal(orig.ID)
		}
	})
}

func (s *Server) handleDiscordMessage(w http.ResponseWriter, r *http.Request, origID string) {
	orig, err := s.store.GetEvent(r.Context(), origID)
	if err != nil || orig.Provider != "discord" {
		sink.WriteJSON(w, http.StatusNotFound, `{"message":"Unknown Message","code":10008}`)
		return
	}
	sk, err := s.store.GetSink(r.Context(), orig.SinkID)
	if err != nil {
		sink.WriteJSON(w, http.StatusNotFound, `{"message":"Unknown Message","code":10008}`)
		return
	}
	body, truncated, ok := s.readCaptureBody(w, r)
	if !ok {
		return
	}
	r = r.WithContext(sink.WithEventID(r.Context(), orig.ID))
	adapter := sinkdiscord.Sink{}
	v := adapter.Validate(r, body)
	if !v.Valid {
		r = r.WithContext(sink.WithValidation(r.Context(), v))
	}
	s.finishFollowup(w, r, sk, adapter, body, truncated, v, r.URL.Path, orig.ID, func() {
		st, _ := s.store.GetMessageState(r.Context(), orig.ID)
		st.EventID = orig.ID
		if r.Method == http.MethodDelete {
			st.Deleted = true
		} else {
			st.DisplayBody = body
			st.Deleted = false
		}
		_ = s.store.UpsertMessageState(r.Context(), st)
		s.publishOriginal(orig.ID)
	})
}

func (s *Server) handleTelegramAnswer(w http.ResponseWriter, r *http.Request) {
	body, truncated, ok := s.readCaptureBody(w, r)
	if !ok {
		return
	}
	adapter := sinktg.Sink{}
	v := adapter.Validate(r, body)
	m := parseJSONMap(body)
	qid, _ := m["callback_query_id"].(string)
	if v.Valid {
		if _, err := s.store.GetInteraction(r.Context(), qid); err != nil {
			v = sink.Validation{Errors: []store.ValidationError{{Path: "/callback_query_id", Message: "required"}}}
		}
	}
	sendPath := strings.TrimSuffix(strings.TrimSuffix(r.URL.Path, "answerCallbackQuery"), "/") + "/sendMessage"
	sk, err := s.store.GetSinkByPath(r.Context(), sendPath)
	if err != nil {
		if tok := telegramTokenFromPath(r.URL.Path); tok != "" {
			sk, err = s.store.GetSinkByToken(r.Context(), tok)
		}
	}
	if err != nil {
		sk, err = s.store.GetSinkByProvider(r.Context(), "telegram")
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !v.Valid {
		r = r.WithContext(sink.WithValidation(r.Context(), v))
	}
	s.finishFollowup(w, r, sk, adapter, body, truncated, v, r.URL.Path, "", nil)
}

func telegramTokenFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 && parts[1] == "telegram" && parts[2] == "bot" {
		return parts[3]
	}
	return ""
}

func (s *Server) finishFollowup(w http.ResponseWriter, r *http.Request, sk store.Sink, adapter sink.Sink, body []byte, truncated bool, v sink.Validation, storePath, groupKey string, apply func()) {
	start := time.Now()
	ov, err := chaos.Apply(r.Context(), sk.Chaos)
	if err != nil {
		return
	}
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
	} else if !v.Valid {
		_ = adapter.Respond(rw, r, body, sk.Chaos)
	} else {
		if apply != nil {
			apply()
		}
		_ = adapter.Respond(rw, r, body, sk.Chaos)
	}
	s.insertCapture(r, sk, adapter.Provider(), body, truncated, rw.status, int(time.Since(start).Milliseconds()), v, sum, storePath, groupKey, newID())
}

func (s *Server) publishOriginal(id string) {
	ev, err := s.store.GetEvent(context.Background(), id)
	if err != nil {
		return
	}
	s.hub.Publish(ev)
}

func (s *Server) insertCapture(r *http.Request, sk store.Sink, provider string, body []byte, truncated bool, status, latency int, v sink.Validation, sum sink.Summary, path, groupKey, id string) {
	if path == "" {
		path = r.URL.Path
	}
	if groupKey == "" {
		groupKey = sum.GroupKey
	}
	if id == "" {
		id = newID()
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
		ID:               id,
		SinkID:           sk.ID,
		Provider:         provider,
		ReceivedAt:       time.Now().UTC(),
		Method:           r.Method,
		Path:             path,
		Query:            query,
		Headers:          headers,
		ContentType:      r.Header.Get("Content-Type"),
		Body:             body,
		BodyText:         string(body),
		BodyTruncated:    truncated,
		Status:           status,
		LatencyMS:        latency,
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
		return
	}
	observe.Captured.Add(1)
	if !ev.Valid {
		observe.Invalid.Add(1)
	}
	if n, err := s.store.CountEvents(r.Context()); err == nil {
		observe.Stored.Store(int64(n))
	}
	s.hub.Publish(ev)
}
