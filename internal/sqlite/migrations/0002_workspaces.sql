CREATE TABLE workspaces (
  id                 TEXT PRIMARY KEY,
  provider           TEXT NOT NULL UNIQUE,
  name               TEXT NOT NULL,
  interactivity_url  TEXT NOT NULL DEFAULT '',
  signing_secret     TEXT NOT NULL DEFAULT '',
  created_at         TEXT NOT NULL
);

CREATE TABLE channels (
  id            TEXT PRIMARY KEY,
  workspace_id  TEXT NOT NULL REFERENCES workspaces(id),
  sink_id       TEXT NOT NULL REFERENCES sinks(id),
  name          TEXT NOT NULL,
  slug          TEXT NOT NULL,
  kind          TEXT NOT NULL DEFAULT 'channel',
  created_at    TEXT NOT NULL,
  UNIQUE (workspace_id, slug)
);

CREATE TABLE interactions (
  id            TEXT PRIMARY KEY,
  created_at    TEXT NOT NULL,
  workspace_id  TEXT NOT NULL,
  channel_id    TEXT NOT NULL,
  event_id      TEXT NOT NULL DEFAULT '',
  kind          TEXT NOT NULL,
  action_id     TEXT NOT NULL DEFAULT '',
  payload       BLOB NOT NULL,
  target        TEXT NOT NULL DEFAULT '',
  status        INTEGER,
  error         TEXT,
  latency_ms    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX interactions_channel ON interactions(channel_id, created_at DESC);
