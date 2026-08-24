# Webhookie Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Execute phases in order. Do not start Phase N+1 until Phase N verification passes. Each task is independently reviewable.

**Goal:** Ship a single-container Mailpit analog for webhooks: provider-faithful sinks (Slack, Teams, PagerDuty, Discord, generic HTTP), signed source firing, live inbox with previews, replay, chaos, and a CI assertion API.

**Architecture:** One Go process (chi) accepts `/hooks/*` POSTs, validates against provider packs, persists to SQLite, streams events over SSE, and serves an embedded Vite UI from the same origin. Outbound `/api/v1/send` signs fixtures and POSTs them at a target. No Postgres, no Redis, no tunnel, no SaaS.

**Tech Stack:** Go 1.24, chi v5, modernc.org/sqlite, santhosh-tekuri/jsonschema v6, slog, embed.FS. Frontend: **Astronomer chrome** — Vite 7, React 19, TypeScript, Tailwind 3.4 + HSL token theme, Inter Variable + JetBrains Mono Variable, TanStack Router + Query + Table + Virtual + Form + Store, lucide-react, sonner, cmdk. Copy primitives from `../astronomer/frontend`, do not invent a second design system. Tests: `go test`, vitest, Playwright. Package: multi-stage Docker, Compose, Makefile.

**Spec:** This file is the spec. `PRODUCT.md` is written in Phase 0 as the one-page contract and must not contradict this plan.

## Global Constraints

- Single binary, single container, single port `8080`, data dir `/data` (SQLite file `/data/webhookie.db`).
- JSON API envelopes: success `{"data": ...}`, list `{"data": ..., "pagination": {total, limit, offset, hasMore, nextOffset}}`, error `{"error": {"code": "...", "message": "...", "details": ...}}`. CamelCase JSON. No `camelizeKeys`.
- OpenAPI is source of truth for `/api/v1/*`. Hand-written chi handlers. `oapi-codegen` generates **models + client only** (copy Sierra: `oapi-codegen.yaml` with `generate.models` + `generate.client`, not chi-server).
- Provider packs live under `internal/fixtures/<provider>/` as JSON Schema + sample events + response templates. Adding a provider later must not require rewriting `internal/httpapi`.
- Hook URLs are path-prefix emulation on one port: `http://webhookie:8080/hooks/slack/services/{T}/{B}/{token}`. Do not bind `hooks.slack.com`.
- Airgap: self-hosted fonts, no Google CDN, no runtime network required.
- Auth v1: none by default; optional `WEBHOOKIE_PASSWORD` basic auth on UI + `/api/v1/*` only. **Never** require auth on `/hooks/*` (that would break the product).
- License Apache-2.0. Module path `github.com/alphabravo-oss/webhookie`.
- Logs: `log/slog` JSON. Probes: `GET /healthz`, `GET /readyz`. Metrics: `GET /metrics` Prometheus text.
- Max body default 1 MiB (`WEBHOOKIE_MAX_BODY_BYTES`). Retention default 7 days **or** 10_000 events, whichever first (`WEBHOOKIE_RETENTION_DAYS`, `WEBHOOKIE_MAX_EVENTS`).
- SSE for live inbox. Do not add WebSockets.
- Do **not** add: Postgres, Redis, NATS, Gin/Fiber, JS plugin runtime, in-process tunnel, SSO/RBAC, email-to-webhook, production retry gateway, Monaco unless the raw JSON view is proven inadequate.

### Locked product decisions

| Decision | Choice |
|---|---|
| Job to be done | Destination emulator (sinks) first; sources/replay second. Not webhook.site clone. Not Svix/Hookdeck. |
| Hosting v1 | Local + CI Docker only |
| DB | SQLite WAL via `modernc.org/sqlite`, `MaxOpenConns=1`, copy Wagon DSN pragmas |
| First sink after generic | Slack Incoming Webhooks, then PagerDuty Events API v2, then Teams (MessageCard + Adaptive Card), then Discord |
| Teams | Dual: legacy `@type: MessageCard` **and** Adaptive Card envelope |
| PagerDuty sink | Events API v2 enqueue + change-events; collapse trigger/ack/resolve by `dedup_key` |
| Sources v1 | Generic Standard Webhooks, GitHub, Slack Events API, Stripe, PagerDuty webhooks v3 |
| CLI | HTTP API first; no `webhookie` CLI in v1 unless assertion API is painful |
| Tunnel | Document cloudflared/ngrok sidecar. Do not build one. |

### What this is not (do not build)

Webhook.site workflows, Beeceptor OpenAPI mock server, Svix production delivery, Slack Bolt modals/slash commands, Teams Bot Framework / Graph subscriptions, Opsgenie/Mattermost/Telegram (v1.1), public SaaS.

### Landscape (why this product exists)

Inbound inspectors (Webhook.site, RequestBin, Svix Play, Hookdeck Console, Beeceptor, tarampampam/webhook-tester, smee/ngrok) catch generic HTTP. Production gateways (Svix, Hookdeck Event Gateway, Convoy, Hook0) deliver at scale. Mock servers (Mockoon, WireMock, Prism) have no Slack/Teams/PD UX.

**Missing analog:** Mailpit for webhooks — especially destination APIs products actually POST to. Closest cousins: webhook-tester (generic catcher) + Hookdeck samples (inbound fixtures) + Beeceptor (chaos). Webhookie is all three plus provider-faithful sinks, rendered previews, and a Mailpit-style assertion API.

---

## Visual identity (Astronomer chrome lock)

Webhookie is an AlphaBravo product. The UI **is** Astronomer's chrome with webhookie content — not a cousin, not a restyle. Copy files from `../astronomer/frontend`; do not reimplement tokens, sidebar, topbar, DataTable, or page primitives from memory.

**Source of truth:** `../astronomer/frontend/src/styles/globals.css`, `tailwind.config.ts`, `src/components/layout/{sidebar,topbar,command-palette}.tsx`, `src/components/ui/{page,data-table,table,status-badge,empty-state,action-button,action-menu,metric-card,drawer-shell,modal-shell,overlay-shell,code-block,confirm-dialog}.tsx`, `src/lib/{theme,utils,link,navigation,persisted-store,store-hook}.tsx`.

| Piece | Astronomer contract | Webhookie adaptation |
|---|---|---|
| Theme | HSL CSS vars, `darkMode: 'class'`, default dark, light + system | Same tokens. Storage key `webhookie-theme` (never bare `theme`) |
| Fonts | `@fontsource-variable/inter`, `@fontsource-variable/jetbrains-mono` | Identical. No IBM Plex, no Geist |
| Sidebar | `bg-sidebar` `w-60` / collapsed `w-16`, `h-14` logo row, blue→violet gradient mark, "by AlphaBravo", collapsible groups, `nav-item` | Wordmark "Webhookie"; Lucide `Webhook` in the same gradient square; groups Capture (Inbox, Sinks) and Test (Send, Fixtures) |
| Topbar | sticky `h-14` `bg-background/80 backdrop-blur-lg`, breadcrumbs, `K` command button, theme cycle Sun/Moon/Monitor | Same. No user menu / alerting bell in v1 |
| Command palette | cmdk + OverlayShell, `⌘/Ctrl+K` | Pages only (Inbox, Sinks, Send, Fixtures) |
| Pages | `PageShell` (`space-y-6`), `PageHeader` (eyebrow + title + description + actions), `PageSection` | Required on every route |
| Lists | `DataTable` (TanStack Table + Pacer debounce + optional virtualizer) | Inbox, sinks, fixtures |
| Status | `StatusBadge` + `status-*` palette | valid/invalid/provider |
| Empty/error | `EmptyState` / `StatePanel` / `LoadingState` / `ErrorState` | Same |
| Overlay | `DrawerShell` right, `ModalShell` center, `ConfirmDialog`, `CodeBlock` `#0a0a0f` | Inspector drawer, curl blocks |
| Toasts | sonner bottom-right, `border border-border` | Same |
| Shell | `flex h-screen` sidebar + column; main `p-6 max-w-[1600px] mx-auto animate-fade-in` | Copy dashboard `route.tsx` layout, drop auth/extensions/window-manager |

**Do not:** copper/Plex packet-strip, Tailwind 4 `@theme`, Geist, a custom sidebar, shadcn from scratch, or a second token set. If Astronomer chrome changes, copy the delta.

**Motion:** Astronomer's `animate-fade-in` on page mount. Respect `prefers-reduced-motion` when adding inbox insert animation later.

---

## File map (create these; do not invent extra layers)

```
webhookie/
  PRODUCT.md
  README.md
  LICENSE                    # Apache-2.0
  Makefile
  Dockerfile
  .gitignore
  go.mod                     # module github.com/alphabravo-oss/webhookie
  oapi-codegen.yaml
  api/openapi.yaml
  cmd/webhookie/main.go
  internal/
    config/config.go
    config/config_test.go
    sqlite/sqlite.go         # Open, WAL, migrations — copy Wagon pattern
    sqlite/sqlite_test.go
    sqlite/migrations/0001_initial.sql
    store/store.go           # events, sinks, send attempts
    store/store_test.go
    sink/sink.go             # Sink interface + registry
    sink/generic/generic.go
    sink/generic/generic_test.go
    sink/slack/slack.go
    sink/slack/slack_test.go
    sink/discord/discord.go
    sink/discord/discord_test.go
    sink/teams/teams.go
    sink/teams/teams_test.go
    sink/pagerduty/pagerduty.go
    sink/pagerduty/pagerduty_test.go
    source/source.go         # Source interface + registry
    source/deliver.go
    source/standard/standard.go
    source/github/github.go
    source/slackevents/slackevents.go
    source/stripe/stripe.go
    source/pagerduty/pagerduty.go
    chaos/chaos.go
    chaos/chaos_test.go
    httpapi/server.go        # chi router, health, static, SSE, API
    httpapi/hooks.go
    httpapi/events.go
    httpapi/sinks.go
    httpapi/send.go
    httpapi/sse.go
    httpapi/static.go
    httpapi/server_test.go
    observe/observe.go       # slog + /metrics
    fixtures/                # embed.FS of schemas + samples
      generic/
      slack/
      discord/
      teams/
      pagerduty/
      github/
      stripe/
      standardwebhooks/
  frontend/
    package.json
    vite.config.ts
    tsconfig.json
    index.html
    src/main.tsx
    src/styles.css
    src/routeTree.gen.ts     # tanstack router plugin
    src/api.ts               # fetch wrappers matching OpenAPI
    src/sse.ts
    src/routes/__root.tsx
    src/routes/index.tsx     # inbox packet strip
    src/routes/sinks.tsx
    src/routes/sinks.$id.tsx
    src/routes/send.tsx
    src/components/shell.tsx
    src/components/packet-strip.tsx
    src/components/inspector.tsx
    src/components/json-view.tsx
    src/components/preview-slack.tsx
    src/components/preview-teams.tsx
    src/components/preview-discord.tsx
    src/components/preview-pagerduty.tsx
    src/lib/copy.ts
  deploy/docker-compose.yml
  examples/go/assert_test.go
  e2e/playwright.config.ts
  e2e/inbox.spec.ts
```

---

## Phase 0 — Product contract and repo scaffold

**What:** Lock the contract and empty module so later phases have a place to land.

**Docs to copy:**
- Wagon SQLite open + WAL DSN: `../wagon/backend/internal/sqlite/sqlite.go`
- Wagon Makefile targets: `../wagon/Makefile`
- Sierra oapi-codegen: `../sierra/oapi-codegen.yaml` (models+client only)
- Charlie PRODUCT.md tone: `../charlie/PRODUCT.md` (one-page contract, not a novel)

**Anti-pattern guards:** Do not add frontend dependencies yet. Do not add Postgres. Do not generate chi-server from OpenAPI.

### Task 0.1: PRODUCT.md + LICENSE + .gitignore

**Files:** Create `PRODUCT.md`, `LICENSE`, `.gitignore`

- [ ] Write `PRODUCT.md` with: one-paragraph promise; sinks vs sources; v1 provider table; explicit non-goals; `docker run` example; assertion API bullet list; health endpoints.
- [ ] Copy Apache-2.0 text into `LICENSE`.
- [ ] `.gitignore`: `bin/`, `frontend/dist/`, `frontend/node_modules/`, `frontend/src/routeTree.gen.ts` (if generated at build — actually **commit** generated route tree if the router plugin can be run in CI; prefer committing `routeTree.gen.ts` once the plugin exists), `*.db`, `*.db-wal`, `*.db-shm`, `.env`, `data/`.
- [ ] Commit: `chore: add product contract and license`

### Task 0.2: Go module + main that serves health

**Files:** Create `go.mod`, `cmd/webhookie/main.go`, `internal/config/config.go`, `internal/config/config_test.go`, `Makefile`

**Interfaces:**

```go
package config

type Config struct {
    Addr            string // default ":8080"
    DataDir         string // default "./data"
    DBPath          string // DataDir + "/webhookie.db"
    MaxBodyBytes    int64  // default 1 << 20
    RetentionDays   int    // default 7
    MaxEvents       int    // default 10000
    Password        string // empty = off
    PublicBaseURL   string // default "http://localhost:8080" for copyable hook URLs
}

func FromEnv() Config
```

Env vars: `WEBHOOKIE_ADDR`, `WEBHOOKIE_DATA_DIR`, `WEBHOOKIE_MAX_BODY_BYTES`, `WEBHOOKIE_RETENTION_DAYS`, `WEBHOOKIE_MAX_EVENTS`, `WEBHOOKIE_PASSWORD`, `WEBHOOKIE_PUBLIC_BASE_URL`.

- [ ] `go mod init github.com/alphabravo-oss/webhookie`
- [ ] Write `config_test.go`: empty env → defaults; `WEBHOOKIE_ADDR=:9090` overrides.
- [ ] `go test ./internal/config/...` — fail, then implement `FromEnv`.
- [ ] `cmd/webhookie/main.go` loads config, listens, `GET /healthz` returns `{"status":"ok"}`, `GET /readyz` returns `{"status":"ready"}` (ready becomes false later if DB ping fails). Graceful shutdown on SIGINT/SIGTERM (copy Wagon signal handling idea).
- [ ] Makefile: `build` (`go build -o bin/webhookie ./cmd/webhookie`), `test` (`go test ./...`), `run` (`go run ./cmd/webhookie`).
- [ ] Verify: `make build && ./bin/webhookie` then `curl -s localhost:8080/healthz` → `{"status":"ok"}`.
- [ ] Commit: `feat: scaffold go module with health probes`

### Task 0.3: OpenAPI skeleton

**Files:** Create `api/openapi.yaml`, `oapi-codegen.yaml`

Cover at least (full paths, later tasks fill schemas):

```
GET  /api/v1/meta
GET  /api/v1/sinks
POST /api/v1/sinks                 # create extra generic bin
GET  /api/v1/sinks/{id}
PATCH /api/v1/sinks/{id}           # chaos + custom response
GET  /api/v1/events
GET  /api/v1/events/{id}
DELETE /api/v1/events
POST /api/v1/events/{id}/replay
GET  /api/v1/events/stream         # SSE documented as text/event-stream
POST /api/v1/send
GET  /api/v1/send/attempts
GET  /api/v1/fixtures
```

Event model (lock names now — later tasks must not rename):

```yaml
Event:
  id: string
  receivedAt: string  # RFC3339Nano
  sinkId: string
  provider: string    # generic|slack|discord|teams|pagerduty
  method: string
  path: string
  query: object
  headers: object     # map[string][]string, Authorization redacted at read if flag on
  contentType: string
  body: string        # raw, not prettified
  bodyTruncated: boolean
  status: integer     # status we returned
  latencyMs: integer
  valid: boolean
  validationErrors: array of {path: string, message: string}
  summary: string
  groupKey: string    # PD dedup_key or empty
```

- [ ] Write YAML with `openapi: 3.1.0`, `info.title: Webhookie`, servers `/`.
- [ ] `oapi-codegen.yaml` copied from Sierra pattern: package `apiclient`, output `internal/apiclient/apiclient.gen.go`, models+client only.
- [ ] Do **not** run codegen until a handler needs the types (Phase 1). Spec must still be valid YAML.
- [ ] Commit: `docs: add openapi skeleton for v1 API`

**Phase 0 verification:** `go test ./...` pass; `curl /healthz` ok; PRODUCT.md exists; OpenAPI lists every `/api/v1` path above.

---

## Phase 1 — SQLite store, generic sink, hook capture

**What:** POST any HTTP request to a generic bin, persist it, GET it back. No UI yet.

**Docs to copy:** Wagon `internal/sqlite/sqlite.go` (`file:` DSN, `_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)`, `SetMaxOpenConns(1)`, embed migrations).

**Anti-pattern guards:** No ORM. No sqlc unless queries get noisy (they will not in v1). Do not store pretty-printed JSON; store raw bytes as TEXT/BLOB.

### Task 1.1: SQLite open + migrations

**Files:** Create `internal/sqlite/sqlite.go`, `internal/sqlite/sqlite_test.go`, `internal/sqlite/migrations/0001_initial.sql`

Schema:

```sql
CREATE TABLE sinks (
  id            TEXT PRIMARY KEY,
  provider      TEXT NOT NULL,
  name          TEXT NOT NULL,
  token         TEXT NOT NULL UNIQUE,
  path          TEXT NOT NULL UNIQUE,
  chaos_json    TEXT NOT NULL DEFAULT '{}',
  created_at    TEXT NOT NULL
);

CREATE TABLE events (
  id              TEXT PRIMARY KEY,
  sink_id         TEXT NOT NULL REFERENCES sinks(id),
  provider        TEXT NOT NULL,
  received_at     TEXT NOT NULL,
  method          TEXT NOT NULL,
  path            TEXT NOT NULL,
  query_json      TEXT NOT NULL,
  headers_json    TEXT NOT NULL,
  content_type    TEXT NOT NULL,
  body            BLOB NOT NULL,
  body_truncated  INTEGER NOT NULL DEFAULT 0,
  status          INTEGER NOT NULL,
  latency_ms      INTEGER NOT NULL,
  valid           INTEGER NOT NULL,
  validation_json TEXT NOT NULL DEFAULT '[]',
  summary         TEXT NOT NULL DEFAULT '',
  group_key       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX events_received_at ON events(received_at DESC);
CREATE INDEX events_provider ON events(provider, received_at DESC);
CREATE INDEX events_sink ON events(sink_id, received_at DESC);
CREATE INDEX events_group ON events(group_key, received_at DESC);

CREATE TABLE send_attempts (
  id           TEXT PRIMARY KEY,
  created_at   TEXT NOT NULL,
  provider     TEXT NOT NULL,
  event_name   TEXT NOT NULL,
  target       TEXT NOT NULL,
  request_headers_json TEXT NOT NULL,
  body         BLOB NOT NULL,
  status       INTEGER,
  error        TEXT,
  latency_ms   INTEGER NOT NULL
);
```

- [ ] Test: `Open(t.TempDir()+"/x.db")` then reopen; second open does not re-apply 0001. Insert a sink row.
- [ ] Implement `Open` by copying Wagon’s DSN and migration loop (parse `NNNN_name.sql` version prefix).
- [ ] `go test ./internal/sqlite/...` pass.
- [ ] Commit: `feat: sqlite open, wal, and initial schema`

### Task 1.2: Store API

**Files:** Create `internal/store/store.go`, `internal/store/store_test.go`

```go
package store

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store

func (s *Store) UpsertSink(ctx context.Context, sk Sink) error
func (s *Store) GetSink(ctx context.Context, id string) (Sink, error)
func (s *Store) GetSinkByPath(ctx context.Context, path string) (Sink, error)
func (s *Store) ListSinks(ctx context.Context) ([]Sink, error)

func (s *Store) InsertEvent(ctx context.Context, ev Event) error
func (s *Store) GetEvent(ctx context.Context, id string) (Event, error)
func (s *Store) ListEvents(ctx context.Context, f EventFilter) (items []Event, total int, err error)
func (s *Store) DeleteEvents(ctx context.Context) error
func (s *Store) Prune(ctx context.Context, maxAge time.Duration, maxN int) (deleted int, err error)

type EventFilter struct {
    Provider string
    SinkID   string
    Since    time.Time
    GroupKey string
    Limit    int // default 50, max 200
    Offset   int
}

type Sink struct {
    ID, Provider, Name, Token, Path string
    Chaos Chaos
    CreatedAt time.Time
}

type Chaos struct {
    DelayMS     int    `json:"delayMs"`
    Status      int    `json:"status"`      // 0 = adapter default
    Body        string `json:"body"`
    ContentType string `json:"contentType"`
    Hang        bool   `json:"hang"`        // never respond (test timeout)
}
```

IDs: `uuid.NewString()`. Paths for seeded sinks are fixed (see Task 1.4).

- [ ] Tests: insert + get; list newest-first; delete all; prune keeps newest `maxN`.
- [ ] Implement.
- [ ] Commit: `feat: event and sink store`

### Task 1.3: Sink interface + generic adapter

**Files:** Create `internal/sink/sink.go`, `internal/sink/generic/generic.go`, `internal/sink/generic/generic_test.go`

```go
package sink

type Validation struct {
    Valid  bool
    Errors []Error
}
type Error struct{ Path, Message string }

type Summary struct {
    Text     string
    GroupKey string
}

type Sink interface {
    Provider() string
    Match(r *http.Request) bool
    Validate(r *http.Request, body []byte) Validation
    Respond(w http.ResponseWriter, r *http.Request, body []byte, ch chaos.Chaos) error
    Summarize(r *http.Request, body []byte) Summary
}

type Registry struct{ items []Sink }
func (r *Registry) Register(s Sink)
func (r *Registry) Match(req *http.Request) (Sink, bool)
```

Generic:

- Match `POST|PUT|PATCH|GET|DELETE /hooks/generic/{token}` (any method).
- Validate: always valid (raw catcher). Empty body ok.
- Respond: `200` `{"ok":true}` JSON unless chaos overrides.
- Summarize: first 80 runes of body, or `{method} {path}`.

- [ ] Tests: match only generic prefix; respond 200 JSON; chaos status 503 wins.
- [ ] Commit: `feat: generic sink adapter`

### Task 1.4: Seed default sinks + hook handler

**Files:** Create `internal/httpapi/server.go`, `internal/httpapi/hooks.go`, `internal/httpapi/server_test.go`. Modify `cmd/webhookie/main.go`.

On startup, upsert these sinks if missing (stable tokens so Compose URLs do not churn across restarts):

| id | provider | path |
|---|---|---|
| `sink-generic` | generic | `/hooks/generic/default` |
| `sink-slack` | slack | `/hooks/slack/services/T00000000/B00000000/webhookie` |
| `sink-discord` | discord | `/hooks/discord/api/webhooks/0/webhookie` |
| `sink-teams` | teams | `/hooks/teams/workflow/webhookie` |
| `sink-pagerduty` | pagerduty | `/hooks/pagerduty/v2/enqueue` |

Until those adapters exist, unmatched provider paths 404. Generic works now. Slack/etc. register in later phases; seed rows anyway so the UI can show copyable URLs from day one.

Hook handler algorithm:

1. Limit body to `MaxBodyBytes` (`http.MaxBytesReader`). Truncate flag if over (return `413` with `{"error":{"code":"body_too_large"}}`).
2. `Registry.Match`. If none: `404`.
3. Apply chaos delay / hang **after** read, before respond.
4. `Validate` → `Summarize` → `Respond`.
5. `InsertEvent` with wall latency.
6. Notify SSE hub (no-op until Phase 2).

Chi mount: `r.HandleFunc("/hooks/*", h.Capture)` plus exact PD path.

- [ ] `httptest`: `POST /hooks/generic/default` with `{"hello":"world"}` → 200 `{"ok":true}`; `GET /api` not yet; store has 1 event; headers include `Content-Type`.
- [ ] `GET /hooks/generic/default` also captured (webhooks are usually POST; capturing GET is useful for challenge probes).
- [ ] Unknown path `/hooks/nope` → 404, **no** event row.
- [ ] Commit: `feat: capture generic webhooks into sqlite`

### Task 1.5: Events REST (read + reset)

**Files:** Create `internal/httpapi/events.go`. Extend OpenAPI models; run oapi-codegen if useful, or hand-write structs matching the YAML names.

```
GET /api/v1/events?provider=&sinkId=&since=&limit=&offset=
GET /api/v1/events/{id}
DELETE /api/v1/events
GET /api/v1/sinks
GET /api/v1/meta   → {version, publicBaseUrl}
```

Redact `Authorization`, `Cookie`, `X-Api-Key` values to `***` in API responses (store keeps originals).

- [ ] Tests via `httptest`: capture two events, list newest first, get by id, delete all, list empty.
- [ ] Pagination clamp: limit default 50, max 200 (copy Sierra queryLimit idea).
- [ ] Commit: `feat: events and sinks read API`

**Phase 1 verification:**

```
go test ./...
curl -s -X POST localhost:8080/hooks/generic/default -H 'content-type: application/json' -d '{"ping":true}'
curl -s localhost:8080/api/v1/events | jq '.data[0].provider'   # generic
curl -s -X DELETE localhost:8080/api/v1/events
```

---

## Phase 2 — SSE + frontend inbox + inspector

**What:** Live packet strip. This is the screenshot.

**Docs:** TanStack Router file routes + Vite plugin; TanStack Query; EventSource. Tailwind 4 `@import "tailwindcss"`. Chi CORS only for `http://localhost:5173` in dev.

**Anti-pattern guards:** No polling as the primary live path. No Monaco. No shadcn. No Inter/Geist. Do not fetch Google fonts.

### Task 2.1: SSE hub

**Files:** Create `internal/httpapi/sse.go`

```go
type Hub struct { /* mutex + map[chan Event]struct{} */ }
func (h *Hub) Publish(ev store.Event)
func (h *Hub) Subscribe() (ch <-chan store.Event, cancel func())
```

`GET /api/v1/events/stream` → `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`. Write `event: webhook\ndata: {json}\n\n` and ping comment every 15s. Context cancel removes subscriber.

- [ ] Test: two subscribers, publish one event, both receive; cancel one, publish, only remaining receives. Use `httptest.NewServer`.
- [ ] Hook handler calls `hub.Publish` after insert.
- [ ] Commit: `feat: sse hub for live inbox`

### Task 2.2: Static SPA embedding (placeholder)

**Files:** Create `internal/httpapi/static.go` serving `frontend/dist` via `embed.FS` **or** `os.DirFS` when `WEBHOOKIE_DEV_FRONTEND=1`. Until frontend build exists, serve a one-line `index.html` so `go test` does not require node. Switch to embed in Phase 2.6.

Pattern:

```go
//go:embed all:frontend/dist
var dist embed.FS
```

Use a `//go:embed` on a `dist` that always exists: check in `frontend/dist/.gitkeep` + tiny index until Vite writes over it. SPA fallback: unknown non-API/non-hook GET returns `index.html`.

- [ ] Commit: `feat: spa fallback static handler`

### Task 2.3: Vite app scaffold

**Files:** Create `frontend/package.json` and the rest of `frontend/` listed in the file map (shell + routes only; previews stubbed).

Dependencies (pin at install time to current latest that satisfies):

- `react`, `react-dom` ^19
- `@tanstack/react-router`, `@tanstack/router-plugin`, `@tanstack/react-query`, `@tanstack/react-table`, `@tanstack/react-virtual`, `@tanstack/react-form`
- `lucide-react`, `sonner`, `clsx`, `tailwind-merge`
- `tailwindcss`, `@tailwindcss/vite`
- `@fontsource/ibm-plex-sans`, `@fontsource/ibm-plex-mono`
- `adaptivecards` — add in Teams phase, not now
- Dev: `vite`, `@vitejs/plugin-react`, `typescript`, `vitest`, `jsdom`, `@testing-library/react`, `eslint`

Vite proxy `/api` and `/hooks` to `localhost:8080`.

Tokens from the visual identity table in `src/styles.css` as `@theme`.

- [ ] `npm install` in `frontend/`.
- [ ] Router routes: `/` inbox, `/sinks`, `/sinks/$id`, `/send`.
- [ ] Shell: left nav Inbox / Sinks / Send, copper accent, IBM Plex.
- [ ] Commit: `feat: vite react tailwind tanstack shell`

### Task 2.4: Packet strip + inspector (generic)

**Files:** `frontend/src/components/packet-strip.tsx`, `inspector.tsx`, `json-view.tsx`, `src/api.ts`, `src/sse.ts`, `src/routes/index.tsx`

Behavior:

- On load: `GET /api/v1/events?limit=50`.
- Open `EventSource('/api/v1/events/stream')`; prepend events; cap client list at 500.
- Row click loads `GET /api/v1/events/{id}` into inspector.
- Inspector tabs: Headers, Body (pretty JSON if parseable else raw + hex dump for non-utf8), Response (status we returned). Preview tab disabled until provider renderers exist.
- Copy as cURL button.
- Empty state: curl for generic default URL from `GET /api/v1/sinks`.
- Filter chips: All / generic / slack / teams / pagerduty / discord.

- [ ] Vitest: json-view pretty-prints `{"a":1}`; packet row renders provider + status.
- [ ] Commit: `feat: live inbox packet strip and inspector`

### Task 2.5: Sinks page (copy URL)

**Files:** `frontend/src/routes/sinks.tsx`, `sinks.$id.tsx`

- List seeded sinks with provider color chip and copy button (`navigator.clipboard.writeText(publicBaseUrl + path)`).
- Detail: last events for that sink, chaos form (delayMs, status, hang) via `PATCH /api/v1/sinks/{id}` — **PATCH can wait until Phase 3**; for now copy URL only if PATCH is not ready. Prefer implementing GET-only here and chaos in Phase 3.

- [ ] Commit: `feat: sinks list with copyable hook urls`

### Task 2.6: Wire embed + Makefile frontend targets

- [ ] `make frontend` → `npm ci && npm run build` in `frontend/`.
- [ ] `make build` depends on frontend then `go build` with embed.
- [ ] `make dev`: run vite + `go run` concurrently (document two terminals if a process helper is too much).
- [ ] Verify in browser: open `/`, POST with curl, row appears without refresh.
- [ ] Commit: `feat: embed frontend dist in go binary`

**Phase 2 verification:** Playwright **or** manual + curl: event appears live; inspector shows body; 375px layout does not hide the strip (stack inspector below). Desktop 1280px split view.

---

## Phase 3 — Chaos and custom responses

**What:** Per-sink delay, status override, body override, hang.

**Anti-pattern:** Hang must be cancellable via request context (client disconnect). Do not leak goroutines.

### Task 3.1: Chaos engine

**Files:** `internal/chaos/chaos.go`, `internal/chaos/chaos_test.go`. Modify hook handler.

```go
func Apply(ctx context.Context, ch Chaos) (override *Override, err error)
type Override struct{ Status int; Body []byte; ContentType string }
```

- `DelayMS`: `select` on `time.After` vs `ctx.Done()`.
- `Hang: true`: block until `ctx.Done()`, then return.
- `Status != 0`: adapter `Respond` is skipped; write override instead.

- [ ] Tests: delay ~50ms; hang returns when context canceled; status 429 writes body `rate_limited` + header `Retry-After: 1` if status is 429 and no custom body.
- [ ] `PATCH /api/v1/sinks/{id}` with `{chaos:{delayMs,status,body,contentType,hang}}`.
- [ ] Sinks UI form: delay, status, hang checkbox, Save. Toast via sonner.
- [ ] Commit: `feat: per-sink chaos delay status hang`

**Phase 3 verification:** `curl` a generic sink with chaos 503; client sees 503; inbox row status 503.

---

## Phase 4 — Slack Incoming Webhooks sink + preview

**What:** Faithful Slack incoming webhook. This is the product screenshot.

**Docs (read before coding, do not guess):**
- Slack incoming webhooks: `https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks`
- Block Kit surface: `text`, `blocks[]`, `attachments[]`, `username`, `icon_url`, `channel` (ignored by incoming webhooks — still capture).
- Success body is the string `ok` (not JSON). Errors are JSON `{"ok":false,"error":"..."}`.

**Anti-pattern:** Do not implement Slack Events API here (that is a source). Do not require a signing secret on incoming webhooks (Slack incoming webhooks are URL-secret only).

### Task 4.1: Schema + adapter

**Files:** `internal/fixtures/slack/incoming.schema.json`, `internal/sink/slack/slack.go`, `internal/sink/slack/slack_test.go`

Match: `POST /hooks/slack/services/{team}/{bot}/{token}` (three path segments).

Validate:
- Content-Type application/json (also accept `application/x-www-form-urlencoded` with `payload=` JSON — Slack’s older form; if present, parse that JSON).
- At least one of `text` or `blocks` or `attachments`.
- `blocks` items require `type`.
- Extra fields allowed.

Respond:
- Valid: `200` `Content-Type: text/plain` body `ok`
- Invalid: `400` JSON `{"ok":false,"error":"invalid_payload"}` plus our validation details **not** leaked in the Slack-shaped body (put details only in the stored event). The sender should see Slack-like errors.

Summarize: `text` if present, else first `section` block text, else `(blocks)`.

- [ ] Golden tests: text-only; blocks section+divider; missing text and blocks → invalid + 400; GET not matched (only POST).
- [ ] Register adapter in `main`.
- [ ] Commit: `feat: slack incoming webhook sink`

### Task 4.2: Slack preview

**Files:** `frontend/src/components/preview-slack.tsx` + tests

Render subset: `header`, `section` (mrkdwn as plain text with `*bold*` stripped simply), `divider`, `context`, `image` (img tag), `actions` (buttons disabled). Attachments: color bar + pretext + text. Fallback: `text`.

Style: Slack-dark-adjacent panel inside the preview tab, not a full Slack clone. Use `--slack` chip on the packet row.

- [ ] Vitest: section text appears; divider renders `hr`.
- [ ] Inbox Preview tab enables when `provider==="slack"`.
- [ ] Commit: `feat: slack block kit preview`

**Phase 4 verification:**

```
curl -s -o /tmp/b -w '%{http_code}' -X POST localhost:8080/hooks/slack/services/T00000000/B00000000/webhookie \
  -H 'content-type: application/json' \
  -d '{"text":"deploy failed","blocks":[{"type":"section","text":{"type":"mrkdwn","text":"*deploy failed*"}}]}'
# 200 and /tmp/b is ok
```

UI shows rendered message. Invalid payload 400 and row `valid: false` with pointer errors.

---

## Phase 5 — Discord incoming webhooks sink + preview

**Docs:** Discord execute webhook: `POST /api/webhooks/{id}/{token}` fields `content`, `username`, `avatar_url`, `embeds[]` (`title`,`description`,`color`,`fields`,`footer`,`image`), query `wait=true`.

Success: `204` empty, **or** `200` message object if `wait=true`.
Invalid: `400` `{message, code}`.

### Task 5.1: Adapter

**Files:** `internal/sink/discord/*`, `internal/fixtures/discord/incoming.schema.json`

- Match `POST /hooks/discord/api/webhooks/{id}/{token}`.
- Require `content` or non-empty `embeds`.
- Summarize: `content` or embed title.

- [ ] Tests: 204 default; 200 JSON when `wait=true`; missing content+embeds → 400.
- [ ] Commit: `feat: discord webhook sink`

### Task 5.2: Preview

Embed card: color left bar (int color → hex), title, description, fields grid, footer. Content as message text.

- [ ] Commit: `feat: discord embed preview`

---

## Phase 6 — Microsoft Teams sink (MessageCard + Adaptive Card)

**Docs (read, Teams is a minefield):**
- O365 connectors retired; products still send MessageCard.
- Workflows expect Adaptive Card envelope: `{type:"message", attachments:[{contentType:"application/vnd.microsoft.card.adaptive", content:{type:"AdaptiveCard",...}}]}`.
- Support **both**. Detect by `@type==MessageCard` vs `type==message` / `AdaptiveCard`.

**Anti-pattern:** Do not implement Bot Framework. Do not require Power Automate SAS query validation in v1 (capture query string; do not reject missing SAS).

### Task 6.1: Adapter

**Files:** `internal/sink/teams/*`, fixtures schemas `messagecard.schema.json`, `adaptive.schema.json`

Match: `POST /hooks/teams/workflow/{token}` and `POST /hooks/teams/incoming/{token}`.

Valid MessageCard: `@type` MessageCard and (`text` or `title` or `sections`).
Valid Adaptive: attachment contentType correct and content.type AdaptiveCard.
Respond: `200` `1` (legacy connectors returned `1`) for MessageCard; `200` `{"statusCode":200}` JSON for workflow-style. Document this in PRODUCT.md.

- [ ] Tests for both shapes; empty body invalid.
- [ ] Commit: `feat: teams messagecard and adaptive card sink`

### Task 6.2: Preview

Use `adaptivecards` npm to render Adaptive Card into a host container (markdown fallback if render throws). MessageCard: title, text, themeColor bar, sections facts as definition list.

- [ ] Commit: `feat: teams card previews`

---

## Phase 7 — PagerDuty Events API v2 sink + incident grouping

**Docs:** `https://developer.pagerduty.com/docs/events-api-v2-overview` — `routing_key` 32 chars, `event_action` trigger|acknowledge|resolve, `payload.summary` required on trigger, `dedup_key` optional (PD generates if missing — **we generate a UUID and return it**).

Change events: `POST /hooks/pagerduty/v2/change` with `routing_key` + `payload.summary`.

Success envelope:

```json
{"status":"success","message":"Event processed","dedup_key":"..."}
```

### Task 7.1: Adapter + groupKey

**Files:** `internal/sink/pagerduty/*`

- Match enqueue and change paths.
- Validate routing_key length 32; trigger requires payload.summary, source, severity in `{info,warning,error,critical}`.
- If `dedup_key` empty on trigger, generate one and return it. Ack/resolve without dedup_key → invalid.
- `Summarize.GroupKey = dedup_key`. Store on event.

Respond 202 (PD is async accepted) with the JSON envelope. Invalid: 400 `{status:"invalid event","message":"..."}`.

- [ ] Tests: trigger creates key; second trigger same key; ack; resolve; missing routing_key 400; change-event path.
- [ ] Commit: `feat: pagerduty events api v2 sink with dedup`

### Task 7.2: Incident timeline preview

When inspector `groupKey` is set, fetch `GET /api/v1/events?groupKey=` and show vertical timeline trigger → ack → resolve with severity color.

Packet strip: optional “group” marker so PD events with the same key indent/link.

- [ ] Commit: `feat: pagerduty incident timeline preview`

**Phase 7 verification:** two curls (trigger + resolve) same `dedup_key`; UI one incident two events.

---

## Phase 8 — Source firer: Standard Webhooks

**What:** Webhookie POSTs at the app under test.

**Docs:** Standard Webhooks spec `https://www.standardwebhooks.com` — headers `webhook-id`, `webhook-timestamp`, `webhook-signature` (`v1=` base64 HMAC-SHA256 of `{id}.{timestamp}.{body}`), secret `whsec_` base64.

### Task 8.1: Source interface + HTTP deliverer

**Files:** `internal/source/source.go`, `internal/source/deliver.go`, `internal/source/standard/standard.go`, tests

```go
type Fixture struct {
    Name        string
    Description string
    Headers     map[string]string
    Body        []byte
    ContentType string
}

type Source interface {
    Provider() string
    Events() []Fixture
    Sign(body []byte, secret string, ts time.Time, id string) http.Header
}

type Attempt struct { /* store.SendAttempt fields */ }

func Deliver(ctx context.Context, target string, hdr http.Header, body []byte, timeout time.Duration) Attempt
```

- [ ] Sign test: known secret + body + ts → golden signature (use a vector from the spec repo).
- [ ] `POST /api/v1/send` `{provider, event, target, secret, timestamp?}`.
- [ ] Persist attempt; `GET /api/v1/send/attempts`.
- [ ] Timeout default 10s. Do not follow redirects more than 3.
- [ ] Commit: `feat: standard webhooks source signer and deliverer`

### Task 8.2: Send UI

**Files:** `frontend/src/routes/send.tsx`

Pick provider, event, target URL, secret. Submit. Show last attempts table (status, latency, error).

- [ ] Commit: `feat: send page for outbound fixtures`

**Phase 8 verification:** point target at `/hooks/generic/default`; send; inbox shows a new generic event whose headers include `webhook-signature`.

---

## Phase 9 — Remaining sources

Each source: fixture JSON under `internal/fixtures/<p>/`, `Sign` implementation, tests with a known vector, register in send API `GET /api/v1/fixtures`.

### Task 9.1: GitHub

Headers: `X-GitHub-Event`, `X-GitHub-Delivery`, `X-Hub-Signature-256` = `sha256=` hex HMAC. Events: `ping`, `push`, `pull_request` opened.

- [ ] Commit: `feat: github webhook source fixtures`

### Task 9.2: Slack Events API

Events: `url_verification` (body `{token,challenge,type}`; **deliverer must still POST JSON** — the target should echo challenge; we record the response). Then `event_callback` `app_mention`. Sign: `X-Slack-Signature` `v0=` hex HMAC of `v0:{ts}:{body}`, header `X-Slack-Request-Timestamp`.

Clock skew option: `timestampSkewSec` in send body to fire an old ts (for testing reject-old-requests).

- [ ] Commit: `feat: slack events api source`

### Task 9.3: Stripe

`Stripe-Signature`: `t={ts},v1={hex hmac of ts.body}`. Events: `invoice.paid`, `customer.subscription.updated`, `checkout.session.completed`. Keep fixtures small but structurally real (`id` `evt_`, `type`, `data.object`).

- [ ] Commit: `feat: stripe webhook source`

### Task 9.4: PagerDuty webhooks v3 (outbound from PD, inbound to your app)

This is **not** Events API v2. Fixtures: `incident.triggered`, `incident.acknowledged`, `incident.resolved`. Sign per PD v3 HMAC (read current PD docs at implementation time; put the algorithm comment + golden test in `source/pagerduty`).

- [ ] Commit: `feat: pagerduty webhooks v3 source`

**Phase 9 verification:** `GET /api/v1/fixtures` lists all names; each provider has a `go test` signature vector.

---

## Phase 10 — Replay

`POST /api/v1/events/{id}/replay` `{target}` re-POSTs the **captured raw body and headers** (minus hop-by-hop: `Content-Length`, `Connection`, `Host`, `Transfer-Encoding`) to target. Record as a send_attempt with `event_name=replay:{id}`.

- [ ] Test: capture Slack, replay to generic sink, second event body identical.
- [ ] Inspector button “Replay to…” with target defaulting to original path on this server (optional) or empty input.
- [ ] Commit: `feat: replay captured webhook to a url`

---

## Phase 11 — Assertion API, prune, examples

This is how CI uses Webhookie without the UI.

### Task 11.1: Assertion semantics

Already have list/get/delete. Add:

```
GET /api/v1/events?provider=slack&contains=deploy
```

`contains` is a case-sensitive substring over body. Keep it dumb. No regex in v1.

- [ ] Test: two events, filter contains.
- [ ] Commit: `feat: event contains filter for ci`

### Task 11.2: Background prune

On a 1-minute ticker, `Store.Prune(retention, maxEvents)`. Metrics: `webhookie_events_stored`, `webhookie_events_pruned_total`.

- [ ] Test prune with maxN=2 keeps two newest.
- [ ] Commit: `feat: retention prune and metrics gauges`

### Task 11.3: Go example helper

**Files:** `examples/go/assert.go`, `examples/go/assert_test.go`

```go
func WaitFor(ctx context.Context, baseURL string, provider string, contains string) (Event, error)
func Reset(ctx context.Context, baseURL string) error
```

Poll `GET /api/v1/events?provider=&contains=` every 50ms until ctx deadline. No `time.Sleep(5s)`.

- [ ] Example test starts `httptest` of the real server if cheap, or documents Docker. Prefer `httptest` against `httpapi.NewServer`.
- [ ] Commit: `feat: go assertion helper example`

### Task 11.4: Observe endpoints

`GET /metrics` Prometheus: events captured, validation failures, send attempts, SSE clients, db size.

Copy Wagon `/metrics` style if present; otherwise `promhttp` is **not** required — a tiny text renderer is enough. Prefer stdlib to adding prometheus client **if** a 40-line renderer will do. If adding `prometheus/client_golang`, justify in the commit.

- [ ] `/healthz` liveness; `/readyz` pings SQLite `SELECT 1`.
- [ ] Commit: `feat: metrics and ready ping`

**Phase 11 verification:** example test passes; `DELETE /api/v1/events` then WaitFor times out.

---

## Phase 12 — Docker, password, docs, e2e, polish

### Task 12.1: Dockerfile + Compose

Multi-stage: `node:22-alpine` build frontend → `golang:1.24-alpine` build static binary (`CGO_ENABLED=0`) → `gcr.io/distroless/static-debian12` or alpine with ca-certs (needed for **source** POSTs to HTTPS targets). Distroless + `ca-certificates` is correct because sources call Stripe-shaped HTTPS URLs.

```
EXPOSE 8080
VOLUME /data
USER nonroot
ENTRYPOINT ["/webhookie"]
ENV WEBHOOKIE_DATA_DIR=/data
```

`deploy/docker-compose.yml`:

```yaml
services:
  webhookie:
    build: ..
    ports: ["8080:8080"]
    volumes: [webhookie-data:/data]
    environment:
      WEBHOOKIE_PUBLIC_BASE_URL: http://localhost:8080
volumes:
  webhookie-data:
```

- [ ] `docker build` + `docker run` + curl slack sink.
- [ ] Commit: `feat: docker image and compose`

### Task 12.2: Optional password

If `WEBHOOKIE_PASSWORD` set, basic auth (`webhookie` / password) on `/`, `/assets/*`, `/api/v1/*`. **Hooks stay open.** SSE must receive `Authorization` (EventSource cannot set headers) — so also allow `?access_token=` matching password on the stream **or** cookie set by `POST /api/v1/login`. Prefer: `POST /api/v1/login` sets `HttpOnly` cookie `webhookie_session` HMAC’d with password; EventSource cookies work same-origin. Document this.

- [ ] Test: with password, `GET /api/v1/events` 401; `POST /hooks/generic/default` 200.
- [ ] Commit: `feat: optional ui password, hooks remain open`

### Task 12.3: README + runbook

README: promise, docker run, table of sink URLs, curl examples for Slack/PD/Teams/Discord, assertion snippet, chaos, non-goals, airgap note.

- [ ] Commit: `docs: readme with sink urls and ci examples`

### Task 12.4: Playwright e2e

**Files:** `e2e/inbox.spec.ts`

Start binary (or docker) in Playwright webServer. Specs:

1. Empty inbox shows curl.
2. `request.post` Slack payload; row appears; preview contains text.
3. Invalid Slack; row invalid.
4. Send page: fire generic Standard Webhook at generic sink; inbox increments.
5. Mobile 375 viewport: strip usable.

- [ ] `make test-e2e`
- [ ] Commit: `test: playwright inbox slack and send`

### Task 12.5: Header redaction toggle + max body

UI toggle “Show secrets” default off. `GET /api/v1/events/{id}?unredact=1` only when password unset **or** authenticated. When password unset, still redact by default.

- [ ] Commit: `feat: header redaction in inspector`

### Task 12.6: Search (P1 if time; otherwise stop)

`GET /api/v1/events?q=` matches body or path substring. Packet strip search box.

Skip if Phase 12.4 slipped. Do not start v1.1 providers.

- [ ] Commit: `feat: inbox search` (optional)

**Phase 12 verification (release gate):**

- [ ] `go test ./...` pass
- [ ] `cd frontend && npm test` pass
- [ ] Playwright pass
- [ ] `docker build` image < ~40MB compressed (frontend embed dominates; fail the gate only if > 80MB — something bundled fonts twice or included node_modules)
- [ ] `docker run` Slack curl → UI preview < 2s on localhost
- [ ] No outbound network during sink tests (`httptest`)
- [ ] PRODUCT.md, README, OpenAPI, this plan agree on path names

---

## Phase 13 — Final verification (do not skip)

- [ ] Spec coverage: every P0 in the original recommendation has a task (sinks, sources, SSE, inspector, chaos, replay, assertion API, health, docker, previews).
- [ ] Grep for forbidden deps: `gin`, `fiber`, `postgres`, `redis`, `ws.`, `monaco` — none in go.mod / package.json unless Teams adaptivecards pulled something unexpected (document it).
- [ ] Grep OpenAPI vs handlers: every `/api/v1` path implemented.
- [ ] Confirm Slack success body is `ok` not JSON.
- [ ] Confirm PD returns `dedup_key`.
- [ ] Confirm `/hooks/*` ignores `WEBHOOKIE_PASSWORD`.
- [ ] Confirm fonts are local `@fontsource` imports.
- [ ] Browse inbox at 1280 and 375.

---

## v1.1 backlog (not this plan)

Opsgenie, Grafana OnCall, Mattermost, Google Chat, Telegram, Splunk HEC, Datadog, Jira, GitLab/Bitbucket/Shopify sources, Helm chart, Go+TS client modules, `webhookie` CLI wait, interactivity/slash commands, Bot Framework.

---

## Suggested commit order (for humans)

Phase 0 → 1 → 2 is the first demoable vertical (generic Mailpit). Phase 4 (Slack) is the first **product** demo. Do not open-source screenshot generic JSON only.

## Execution

After this file is reviewed:

1. **Subagent-driven (recommended)** — one subagent per task, review between tasks.
2. **Inline** — same session, stop at each phase verification.

Do not implement Tasks out of phase order. Do not “just add Postgres.” Do not start the Teams Adaptive renderer before Slack preview works.
