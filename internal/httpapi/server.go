package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/alphabravo-oss/webhookie/internal/config"
	"github.com/alphabravo-oss/webhookie/internal/observe"
	"github.com/alphabravo-oss/webhookie/internal/sink"
	sinkdiscord "github.com/alphabravo-oss/webhookie/internal/sink/discord"
	sinkgeneric "github.com/alphabravo-oss/webhookie/internal/sink/generic"
	sinkgchat "github.com/alphabravo-oss/webhookie/internal/sink/googlechat"
	sinkmm "github.com/alphabravo-oss/webhookie/internal/sink/mattermost"
	sinkog "github.com/alphabravo-oss/webhookie/internal/sink/opsgenie"
	sinkpd "github.com/alphabravo-oss/webhookie/internal/sink/pagerduty"
	sinkslack "github.com/alphabravo-oss/webhookie/internal/sink/slack"
	sinkteams "github.com/alphabravo-oss/webhookie/internal/sink/teams"
	sinktg "github.com/alphabravo-oss/webhookie/internal/sink/telegram"
	"github.com/alphabravo-oss/webhookie/internal/source"
	srcgithub "github.com/alphabravo-oss/webhookie/internal/source/github"
	srcpd "github.com/alphabravo-oss/webhookie/internal/source/pagerduty"
	srcslack "github.com/alphabravo-oss/webhookie/internal/source/slackevents"
	srcstd "github.com/alphabravo-oss/webhookie/internal/source/standard"
	srcstripe "github.com/alphabravo-oss/webhookie/internal/source/stripe"
	"github.com/alphabravo-oss/webhookie/internal/store"
	"github.com/alphabravo-oss/webhookie/internal/webui"
	"github.com/google/uuid"
)

type Server struct {
	cfg   config.Config
	store *store.Store
	hub   *Hub
	sinks *sink.Registry
	srcs  *source.Registry
	ready func(context.Context) error
	web   fs.FS
	log   *slog.Logger
}

func New(cfg config.Config, st *store.Store, ready func(context.Context) error) *Server {
	sk := &sink.Registry{}
	sk.Register(sinkgeneric.Sink{})
	sk.Register(sinkslack.Sink{})
	sk.Register(sinkdiscord.Sink{})
	sk.Register(sinkteams.Sink{})
	sk.Register(sinkpd.Sink{})
	sk.Register(sinktg.Sink{})
	sk.Register(sinkgchat.Sink{})
	sk.Register(sinkmm.Sink{})
	sk.Register(sinkog.Sink{})
	src := &source.Registry{}
	src.Register(srcstd.Source{})
	src.Register(srcgithub.Source{})
	src.Register(srcslack.Source{})
	src.Register(srcstripe.Source{})
	src.Register(srcpd.Source{})
	web, err := fs.Sub(webui.Dist, "dist")
	if err != nil {
		web = webui.Dist
	}
	return &Server{
		cfg:   cfg,
		store: st,
		hub:   NewHub(),
		sinks: sk,
		srcs:  src,
		ready: ready,
		web:   web,
		log:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (s *Server) Seed(ctx context.Context) error {
	now := time.Now().UTC()
	seeds := []store.Sink{
		{ID: "sink-generic", Provider: "generic", Name: "Generic bin", Token: "default", Path: "/hooks/generic/default", CreatedAt: now},
		{ID: "sink-slack", Provider: "slack", Name: "alerts", Token: "slack-webhookie", Path: "/hooks/slack/services/T00000000/B00000000/webhookie", CreatedAt: now},
		{ID: "sink-discord", Provider: "discord", Name: "general", Token: "discord-webhookie", Path: "/hooks/discord/api/webhooks/0/webhookie", CreatedAt: now},
		{ID: "sink-teams", Provider: "teams", Name: "General", Token: "teams-webhookie", Path: "/hooks/teams/workflow/webhookie", CreatedAt: now},
		{ID: "sink-pagerduty", Provider: "pagerduty", Name: "default", Token: "0123456789abcdef0123456789abcdef", Path: "/hooks/pagerduty/v2/enqueue", CreatedAt: now},
		{ID: "sink-telegram", Provider: "telegram", Name: "alerts", Token: "123456:AAWebhookie", Path: "/hooks/telegram/bot/123456:AAWebhookie/sendMessage", CreatedAt: now},
		{ID: "sink-googlechat", Provider: "googlechat", Name: "Ops", Token: "googlechat-webhookie", Path: "/hooks/googlechat/v1/spaces/AAAAwebhookie/messages", CreatedAt: now},
		{ID: "sink-mattermost", Provider: "mattermost", Name: "town-square", Token: "mattermost-webhookie", Path: "/hooks/mattermost/hooks/mattermost-webhookie", CreatedAt: now},
		{ID: "sink-opsgenie", Provider: "opsgenie", Name: "default", Token: "eb243592-faa2-4ba2-a551-webhookie01", Path: "/hooks/opsgenie/v2/alerts", CreatedAt: now},
	}
	for _, sk := range seeds {
		if err := s.store.UpsertSink(ctx, sk); err != nil {
			return err
		}
	}
	workspaces := []store.Workspace{
		{ID: "ws-slack", Provider: "slack", Name: "Slack", CreatedAt: now},
		{ID: "ws-teams", Provider: "teams", Name: "Teams", CreatedAt: now},
		{ID: "ws-discord", Provider: "discord", Name: "Discord", CreatedAt: now},
		{ID: "ws-pagerduty", Provider: "pagerduty", Name: "PagerDuty", CreatedAt: now},
		{ID: "ws-telegram", Provider: "telegram", Name: "Telegram", CreatedAt: now},
		{ID: "ws-googlechat", Provider: "googlechat", Name: "Google Chat", CreatedAt: now},
		{ID: "ws-mattermost", Provider: "mattermost", Name: "Mattermost", CreatedAt: now},
		{ID: "ws-opsgenie", Provider: "opsgenie", Name: "Opsgenie", CreatedAt: now},
	}
	for _, ws := range workspaces {
		if err := s.store.UpsertWorkspace(ctx, ws); err != nil {
			return err
		}
	}
	channels := []store.Channel{
		{ID: "ch-slack-alerts", WorkspaceID: "ws-slack", SinkID: "sink-slack", Name: "alerts", Slug: "alerts", Kind: "channel", CreatedAt: now},
		{ID: "ch-teams-general", WorkspaceID: "ws-teams", SinkID: "sink-teams", Name: "General", Slug: "general", Kind: "channel", CreatedAt: now},
		{ID: "ch-discord-general", WorkspaceID: "ws-discord", SinkID: "sink-discord", Name: "general", Slug: "general", Kind: "channel", CreatedAt: now},
		{ID: "ch-pd-default", WorkspaceID: "ws-pagerduty", SinkID: "sink-pagerduty", Name: "default", Slug: "default", Kind: "service", CreatedAt: now},
		{ID: "ch-telegram-alerts", WorkspaceID: "ws-telegram", SinkID: "sink-telegram", Name: "alerts", Slug: "alerts", Kind: "chat", CreatedAt: now},
		{ID: "ch-gchat-ops", WorkspaceID: "ws-googlechat", SinkID: "sink-googlechat", Name: "Ops", Slug: "ops", Kind: "space", CreatedAt: now},
		{ID: "ch-mm-town-square", WorkspaceID: "ws-mattermost", SinkID: "sink-mattermost", Name: "town-square", Slug: "town-square", Kind: "channel", CreatedAt: now},
		{ID: "ch-og-default", WorkspaceID: "ws-opsgenie", SinkID: "sink-opsgenie", Name: "default", Slug: "default", Kind: "team", CreatedAt: now},
	}
	for _, ch := range channels {
		if _, err := s.store.GetChannel(ctx, ch.ID); err == nil {
			continue
		}
		if err := s.store.InsertChannel(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173", s.cfg.PublicBaseURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, map[string]any{"status": "ok"})
	})
	r.Get("/readyz", s.readyz)
	r.Get("/metrics", observe.Metrics)
	r.HandleFunc("/hooks/*", s.Capture)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.protectAPI)
		r.Post("/login", s.login)
		r.Get("/meta", s.meta)
		r.Get("/sinks", s.listSinks)
		r.Post("/sinks", s.createSink)
		r.Get("/sinks/{id}", s.getSink)
		r.Patch("/sinks/{id}", s.patchSink)
		r.Get("/events", s.listEvents)
		r.Get("/events/stream", s.stream)
		r.Get("/events/{id}", s.getEvent)
		r.Delete("/events", s.deleteEvents)
		r.Post("/events/{id}/replay", s.replay)
		r.Post("/send", s.send)
		r.Get("/send/attempts", s.listAttempts)
		r.Get("/fixtures", s.listFixtures)
		r.Get("/workspaces", s.listWorkspaces)
		r.Get("/workspaces/{id}", s.getWorkspace)
		r.Patch("/workspaces/{id}", s.patchWorkspace)
		r.Post("/workspaces/{id}/channels", s.createChannel)
		r.Get("/workspaces/{id}/channels/{channelId}/interactions", s.listInteractions)
		r.Post("/workspaces/{id}/channels/{channelId}/actions", s.postAction)
	})
	r.Handle("/*", s.spa())
	return r
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if s.ready != nil {
		if err := s.ready(r.Context()); err != nil {
			writeError(w, 503, "not_ready", err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]any{"status": "ready"})
}

func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"data": map[string]any{
		"version":       s.cfg.Version,
		"publicBaseUrl": s.cfg.PublicBaseURL,
	}})
}

func (s *Server) spa() http.Handler {
	fileServer := http.FileServer(http.FS(s.web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/hooks/") {
			http.NotFound(w, r)
			return
		}
		if s.cfg.Password != "" && !s.authorized(r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="webhookie"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(s.web, name); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) protectAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Password == "" || r.URL.Path == "/api/v1/login" {
			next.ServeHTTP(w, r)
			return
		}
		if s.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, 401, "unauthorized", "authentication required")
	})
}

func (s *Server) authorized(r *http.Request) bool {
	if s.cfg.Password == "" {
		return true
	}
	if tok := r.URL.Query().Get("access_token"); tok != "" && tok == s.cfg.Password {
		return true
	}
	if c, err := r.Cookie("webhookie_session"); err == nil && s.validSession(c.Value) {
		return true
	}
	u, p, ok := r.BasicAuth()
	return ok && u == "webhookie" && p == s.cfg.Password
}

func (s *Server) sessionMAC() string {
	mac := hmac.New(sha256.New, []byte(s.cfg.Password))
	_, _ = io.WriteString(mac, "webhookie")
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) validSession(v string) bool {
	return hmac.Equal([]byte(v), []byte(s.sessionMAC()))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	if s.cfg.Password == "" || in.Password != s.cfg.Password {
		writeError(w, 401, "unauthorized", "invalid password")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "webhookie_session",
		Value:    s.sessionMAC(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, 200, map[string]any{"data": map[string]any{"ok": true}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg}})
}

func newID() string { return uuid.NewString() }
