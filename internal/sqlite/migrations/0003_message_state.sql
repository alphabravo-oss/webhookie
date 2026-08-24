CREATE TABLE message_state (
  event_id        TEXT PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
  deleted         INTEGER NOT NULL DEFAULT 0,
  display_body    BLOB,
  updated_at      TEXT NOT NULL,
  response_count  INTEGER NOT NULL DEFAULT 0
);
