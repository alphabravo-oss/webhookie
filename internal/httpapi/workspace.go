package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/alphabravo-oss/webhookie/internal/observe"
	"github.com/alphabravo-oss/webhookie/internal/source"
	srcpd "github.com/alphabravo-oss/webhookie/internal/source/pagerduty"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListWorkspaces(r.Context())
	if err != nil {
		writeError(w, 500, "list_failed", err.Error())
		return
	}
	if items == nil {
		items = []store.Workspace{}
	}
	for i := range items {
		s.decorateChannels(items[i].Channels)
	}
	writeJSON(w, 200, map[string]any{"data": items})
}

func (s *Server) getWorkspace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ws, err := s.store.GetWorkspace(r.Context(), id)
	if err != nil {
		ws, err = s.store.GetWorkspaceByProvider(r.Context(), id)
	}
	if err != nil {
		writeError(w, 404, "not_found", "workspace not found")
		return
	}
	s.decorateChannels(ws.Channels)
	writeJSON(w, 200, map[string]any{"data": ws})
}

func (s *Server) patchWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, err := s.store.GetWorkspace(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 404, "not_found", "workspace not found")
		return
	}
	var in struct {
		Name             string `json:"name"`
		InteractivityURL string `json:"interactivityUrl"`
		SigningSecret    string `json:"signingSecret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Name != "" {
		ws.Name = in.Name
	}
	ws.InteractivityURL = in.InteractivityURL
	if in.SigningSecret != "" {
		ws.SigningSecret = in.SigningSecret
	}
	if err := s.store.UpsertWorkspace(r.Context(), ws); err != nil {
		writeError(w, 500, "update_failed", err.Error())
		return
	}
	s.decorateChannels(ws.Channels)
	writeJSON(w, 200, map[string]any{"data": ws})
}

func (s *Server) createChannel(w http.ResponseWriter, r *http.Request) {
	ws, err := s.store.GetWorkspace(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 404, "not_found", "workspace not found")
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if strings.TrimSpace(in.Name) == "" {
		writeError(w, 400, "name_required", "name is required")
		return
	}
	slug := store.Slugify(in.Name)
	token := strings.ReplaceAll(uuid.NewString(), "-", "")
	now := time.Now().UTC()
	sinkID := "sink-" + token[:12]
	path, kind := channelPath(ws.Provider, token, slug)
	sk := store.Sink{
		ID:        sinkID,
		Provider:  ws.Provider,
		Name:      in.Name,
		Token:     token,
		Path:      path,
		CreatedAt: now,
	}
	if ws.Provider == "pagerduty" {
		sk.Token = token
		if len(sk.Token) > 32 {
			sk.Token = token[:32]
		}
	}
	if err := s.store.UpsertSink(r.Context(), sk); err != nil {
		writeError(w, 500, "create_failed", err.Error())
		return
	}
	ch := store.Channel{
		ID:          store.NewChannelID(),
		WorkspaceID: ws.ID,
		SinkID:      sk.ID,
		Name:        in.Name,
		Slug:        slug,
		Kind:        kind,
		CreatedAt:   now,
		Path:        sk.Path,
		Token:       sk.Token,
	}
	if err := s.store.InsertChannel(r.Context(), ch); err != nil {
		writeError(w, 409, "duplicate", "a channel with that name already exists")
		return
	}
	created, err := s.store.GetChannel(r.Context(), ch.ID)
	if err != nil {
		writeError(w, 500, "create_failed", err.Error())
		return
	}
	s.decorateChannels([]store.Channel{created})
	writeJSON(w, 201, map[string]any{"data": created})
}

func channelPath(provider, token, slug string) (path, kind string) {
	kind = "channel"
	switch provider {
	case "slack":
		return "/hooks/slack/services/T00000000/B" + token[:8] + "/" + token, kind
	case "teams":
		return "/hooks/teams/workflow/" + token, kind
	case "discord":
		return "/hooks/discord/api/webhooks/" + token[:16] + "/" + token, kind
	case "pagerduty":
		if len(token) > 32 {
			token = token[:32]
		}
		return "/hooks/pagerduty/v2/enqueue/" + token, "service"
	case "telegram":
		return "/hooks/telegram/bot/" + token + "/sendMessage", "chat"
	case "googlechat":
		space := token
		if len(space) > 16 {
			space = "AAAA" + token[:12]
		}
		return "/hooks/googlechat/v1/spaces/" + space + "/messages", "space"
	case "mattermost":
		return "/hooks/mattermost/hooks/" + token, kind
	case "opsgenie":
		return "/hooks/opsgenie/v2/alerts/" + token, "team"
	default:
		return "/hooks/generic/" + token, kind
	}
}

func (s *Server) decorateChannels(chs []store.Channel) {
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	for i := range chs {
		chs[i].URL = base + chs[i].Path
	}
}

func (s *Server) listInteractions(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListInteractions(r.Context(), chi.URLParam(r, "channelId"))
	if err != nil {
		writeError(w, 500, "list_failed", err.Error())
		return
	}
	if items == nil {
		items = []store.Interaction{}
	}
	writeJSON(w, 200, map[string]any{"data": items})
}

type actionIn struct {
	EventID  string `json:"eventId"`
	Kind     string `json:"kind"`
	ActionID string `json:"actionId"`
	Value    string `json:"value"`
	Text     string `json:"text"`
	BlockID  string `json:"blockId"`
}

func (s *Server) postAction(w http.ResponseWriter, r *http.Request) {
	ws, err := s.store.GetWorkspace(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, 404, "not_found", "workspace not found")
		return
	}
	ch, err := s.store.GetChannel(r.Context(), chi.URLParam(r, "channelId"))
	if err != nil {
		writeError(w, 404, "not_found", "channel not found")
		return
	}
	var in actionIn
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Kind == "" {
		in.Kind = "button"
	}
	var ev store.Event
	if in.EventID != "" {
		ev, _ = s.store.GetEvent(r.Context(), in.EventID)
	}

	if ws.Provider == "pagerduty" && (in.Kind == "ack" || in.Kind == "resolve") {
		action := "acknowledge"
		if in.Kind == "resolve" {
			action = "resolve"
		}
		key := ev.GroupKey
		if key == "" {
			key = in.Value
		}
		body := fmt.Sprintf(`{"routing_key":%q,"event_action":%q,"dedup_key":%q}`, ch.Token, action, key)
		path := ch.Path
		if path == "" {
			path = "/hooks/pagerduty/v2/enqueue"
		}
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, "http://local"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.URL.Path = path
		rec := newRecorder()
		s.Capture(rec, req)
	}
	if ws.Provider == "opsgenie" && (in.Kind == "ack" || in.Kind == "close" || in.Kind == "resolve") {
		action := "acknowledge"
		if in.Kind == "close" || in.Kind == "resolve" {
			action = "close"
		}
		key := ev.GroupKey
		if key == "" {
			key = in.Value
		}
		if key == "" {
			key = ev.ID
		}
		path := "/hooks/opsgenie/v2/alerts/" + url.PathEscape(key) + "/" + action
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, "http://local"+path, strings.NewReader(`{"user":"webhookie","source":"webhookie"}`))
		req.Header.Set("Content-Type", "application/json")
		if ch.Token != "" {
			req.Header.Set("Authorization", "GenieKey "+ch.Token)
		}
		req.URL.Path = path
		rec := newRecorder()
		s.Capture(rec, req)
	}

	interID := newID()
	payload, contentType := buildInteraction(ws, ch, ev, in, s.cfg.PublicBaseURL, interID)
	target := ws.InteractivityURL
	inter := store.Interaction{
		ID:          interID,
		CreatedAt:   time.Now().UTC(),
		WorkspaceID: ws.ID,
		ChannelID:   ch.ID,
		EventID:     in.EventID,
		Kind:        in.Kind,
		ActionID:    in.ActionID,
		Payload:     string(payload),
		Target:      target,
	}
	if target != "" {
		hdr := http.Header{}
		hdr.Set("Content-Type", contentType)
		if ws.SigningSecret != "" && ws.Provider == "slack" {
			srcslackSign(hdr, payload, ws.SigningSecret)
		}
		if ws.Provider == "pagerduty" && ws.SigningSecret != "" {
			h := (srcpd.Source{}).Sign(payload, ws.SigningSecret, time.Now(), "")
			hdr = h
		}
		att := source.Deliver(r.Context(), target, hdr, payload, 10*time.Second)
		inter.Status = att.Status
		inter.Error = att.Error
		inter.LatencyMS = att.LatencyMS
		observe.Sends.Add(1)
		_ = s.store.InsertAttempt(r.Context(), store.SendAttempt{
			ID:             newID(),
			CreatedAt:      inter.CreatedAt,
			Provider:       ws.Provider,
			EventName:      "interaction:" + in.Kind,
			Target:         target,
			RequestHeaders: hdr,
			Body:           payload,
			Status:         att.Status,
			Error:          att.Error,
			LatencyMS:      att.LatencyMS,
		})
	}
	_ = s.store.InsertInteraction(r.Context(), inter)
	writeJSON(w, 200, map[string]any{"data": inter})
}

func srcslackSign(h http.Header, body []byte, secret string) {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	h.Set("X-Slack-Request-Timestamp", ts)
	base := fmt.Sprintf("v0:%s:%s", ts, body)
	h.Set("X-Slack-Signature", "v0="+source.HMACSHA256Hex(secret, base))
}

func buildInteraction(ws store.Workspace, ch store.Channel, ev store.Event, in actionIn, publicBase, callbackID string) ([]byte, string) {
	switch ws.Provider {
	case "slack":
		msg := json.RawMessage("{}")
		if ev.BodyText != "" {
			msg = json.RawMessage(ev.BodyText)
		} else if len(ev.Body) > 0 {
			msg = json.RawMessage(ev.Body)
		}
		payload := map[string]any{
			"type":       "block_actions",
			"user":       map[string]any{"id": "UHOOKIE", "username": "webhookie", "name": "Webhookie"},
			"api_app_id": "AHOOKIE",
			"token":      "webhookie",
			"trigger_id": newID(),
			"team":       map[string]any{"id": "T00000000", "domain": "webhookie"},
			"channel":    map[string]any{"id": ch.ID, "name": ch.Name},
			"message":    msg,
			"actions": []map[string]any{{
				"action_id": in.ActionID,
				"block_id":  in.BlockID,
				"value":     in.Value,
				"type":      "button",
				"text":      map[string]any{"type": "plain_text", "text": in.Text},
			}},
			"response_url": strings.TrimRight(publicBase, "/") + "/hooks/slack/response/" + ev.ID,
		}
		b, _ := json.Marshal(payload)
		form := url.Values{}
		form.Set("payload", string(b))
		return []byte(form.Encode()), "application/x-www-form-urlencoded"
	case "teams":
		b, _ := json.Marshal(map[string]any{
			"type":    "invoke",
			"name":    "adaptiveCard/action",
			"channel": map[string]any{"id": ch.ID, "name": ch.Name},
			"value":   map[string]any{"action": map[string]any{"type": "Action.Submit", "data": map[string]any{"actionId": in.ActionID, "value": in.Value}}},
			"message": json.RawMessage(ev.BodyText),
		})
		return b, "application/json"
	case "discord":
		b, _ := json.Marshal(map[string]any{
			"type":       3,
			"custom_id":  in.ActionID,
			"channel_id": ch.ID,
			"data":       map[string]any{"custom_id": in.ActionID, "component_type": 2, "values": []string{in.Value}},
			"message":    json.RawMessage(ev.BodyText),
		})
		return b, "application/json"
	case "pagerduty":
		status := "acknowledged"
		if in.Kind == "resolve" {
			status = "resolved"
		}
		b, _ := json.Marshal(map[string]any{
			"event": map[string]any{
				"event_type":    "incident." + status,
				"resource_type": "incident",
				"data":          map[string]any{"id": ev.GroupKey, "type": "incident", "status": status, "title": ev.Summary},
			},
		})
		return b, "application/json"
	case "telegram":
		b, _ := json.Marshal(map[string]any{
			"update_id": 1,
			"callback_query": map[string]any{
				"id":            callbackID,
				"from":          map[string]any{"id": 1, "is_bot": false, "first_name": "Webhookie"},
				"chat_instance": ch.ID,
				"data":          firstNonEmpty(in.Value, in.ActionID),
				"message": map[string]any{
					"message_id": 1,
					"chat":       map[string]any{"id": ch.Slug, "title": ch.Name, "type": "group"},
					"text":       ev.Summary,
					"date":       time.Now().Unix(),
				},
			},
		})
		return b, "application/json"
	case "googlechat":
		msg := json.RawMessage("{}")
		if ev.BodyText != "" {
			msg = json.RawMessage(ev.BodyText)
		}
		b, _ := json.Marshal(map[string]any{
			"type":      "CARD_CLICKED",
			"eventTime": time.Now().UTC().Format(time.RFC3339Nano),
			"space":     map[string]any{"name": "spaces/" + ch.Slug, "displayName": ch.Name, "type": "ROOM"},
			"message":   msg,
			"action": map[string]any{
				"actionMethodName": firstNonEmpty(in.ActionID, in.Text),
				"parameters":       []map[string]string{{"key": "value", "value": in.Value}},
			},
			"user": map[string]any{"name": "users/webhookie", "displayName": "Webhookie", "type": "HUMAN"},
		})
		return b, "application/json"
	case "mattermost":
		b, _ := json.Marshal(map[string]any{
			"user_id":      "webhookie",
			"user_name":    "webhookie",
			"channel_id":   ch.ID,
			"channel_name": ch.Name,
			"team_id":      ws.ID,
			"post_id":      ev.ID,
			"trigger_id":   newID(),
			"context":      map[string]any{"action": firstNonEmpty(in.ActionID, in.Text), "value": in.Value},
		})
		return b, "application/json"
	case "opsgenie":
		action := "Acknowledge"
		if in.Kind == "close" || in.Kind == "resolve" {
			action = "Close"
		}
		b, _ := json.Marshal(map[string]any{
			"action": action,
			"alert": map[string]any{
				"alertId": ev.GroupKey,
				"alias":   ev.GroupKey,
				"message": ev.Summary,
				"source":  "webhookie",
			},
			"source": map[string]any{"name": "webhookie", "type": "user"},
		})
		return b, "application/json"
	default:
		b, _ := json.Marshal(in)
		return b, "application/json"
	}
}

type rec struct {
	http.ResponseWriter
	code int
	hdr  http.Header
	buf  strings.Builder
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func newRecorder() *rec {
	return &rec{hdr: http.Header{}, code: 200}
}

func (r *rec) Header() http.Header         { return r.hdr }
func (r *rec) Write(b []byte) (int, error) { return r.buf.Write(b) }
func (r *rec) WriteHeader(c int)           { r.code = c }
