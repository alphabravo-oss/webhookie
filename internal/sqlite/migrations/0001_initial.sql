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
  id                   TEXT PRIMARY KEY,
  created_at           TEXT NOT NULL,
  provider             TEXT NOT NULL,
  event_name           TEXT NOT NULL,
  target               TEXT NOT NULL,
  request_headers_json TEXT NOT NULL,
  body                 BLOB NOT NULL,
  status               INTEGER,
  error                TEXT,
  latency_ms           INTEGER NOT NULL
);
