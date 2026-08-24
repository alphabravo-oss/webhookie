package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

type Chaos struct {
	DelayMS     int    `json:"delayMs"`
	Status      int    `json:"status"`
	Body        string `json:"body"`
	ContentType string `json:"contentType"`
	Hang        bool   `json:"hang"`
}

type Sink struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	Path      string    `json:"path"`
	Chaos     Chaos     `json:"chaos"`
	CreatedAt time.Time `json:"createdAt"`
}

type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type Event struct {
	ID               string            `json:"id"`
	SinkID           string            `json:"sinkId"`
	Provider         string            `json:"provider"`
	ReceivedAt       time.Time         `json:"receivedAt"`
	Method           string            `json:"method"`
	Path             string            `json:"path"`
	Query            map[string]string `json:"query"`
	Headers          map[string][]string `json:"headers"`
	ContentType      string            `json:"contentType"`
	Body             []byte            `json:"-"`
	BodyText         string            `json:"body"`
	BodyTruncated    bool              `json:"bodyTruncated"`
	Status           int               `json:"status"`
	LatencyMS        int               `json:"latencyMs"`
	Valid            bool              `json:"valid"`
	ValidationErrors []ValidationError `json:"validationErrors"`
	Summary          string            `json:"summary"`
	GroupKey         string            `json:"groupKey"`
}

type EventFilter struct {
	Provider string
	SinkID   string
	Since    time.Time
	GroupKey string
	Contains string
	Limit    int
	Offset   int
}

type SendAttempt struct {
	ID              string              `json:"id"`
	CreatedAt       time.Time           `json:"createdAt"`
	Provider        string              `json:"provider"`
	EventName       string              `json:"eventName"`
	Target          string              `json:"target"`
	RequestHeaders  map[string][]string `json:"requestHeaders"`
	Body            []byte              `json:"-"`
	BodyText        string              `json:"body"`
	Status          *int                `json:"status"`
	Error           string              `json:"error,omitempty"`
	LatencyMS       int                 `json:"latencyMs"`
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.QueryRowContext(ctx, `SELECT 1`).Scan(new(int))
}

func (s *Store) UpsertSink(ctx context.Context, sk Sink) error {
	chaos, err := json.Marshal(sk.Chaos)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sinks (id, provider, name, token, path, chaos_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			provider=excluded.provider,
			name=excluded.name,
			token=excluded.token,
			path=excluded.path,
			chaos_json=excluded.chaos_json
	`, sk.ID, sk.Provider, sk.Name, sk.Token, sk.Path, string(chaos), sk.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetSink(ctx context.Context, id string) (Sink, error) {
	return scanSink(s.db.QueryRowContext(ctx, `SELECT id, provider, name, token, path, chaos_json, created_at FROM sinks WHERE id=?`, id))
}

func (s *Store) GetSinkByPath(ctx context.Context, path string) (Sink, error) {
	return scanSink(s.db.QueryRowContext(ctx, `SELECT id, provider, name, token, path, chaos_json, created_at FROM sinks WHERE path=?`, path))
}

func (s *Store) GetSinkByProvider(ctx context.Context, provider string) (Sink, error) {
	return scanSink(s.db.QueryRowContext(ctx, `SELECT id, provider, name, token, path, chaos_json, created_at FROM sinks WHERE provider=? ORDER BY created_at LIMIT 1`, provider))
}

func (s *Store) GetSinkByToken(ctx context.Context, token string) (Sink, error) {
	return scanSink(s.db.QueryRowContext(ctx, `SELECT id, provider, name, token, path, chaos_json, created_at FROM sinks WHERE token=?`, token))
}

func (s *Store) ListSinks(ctx context.Context) ([]Sink, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, provider, name, token, path, chaos_json, created_at FROM sinks ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Sink
	for rows.Next() {
		sk, err := scanSinkRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *Store) InsertEvent(ctx context.Context, ev Event) error {
	q, err := json.Marshal(ev.Query)
	if err != nil {
		return err
	}
	h, err := json.Marshal(ev.Headers)
	if err != nil {
		return err
	}
	v, err := json.Marshal(ev.ValidationErrors)
	if err != nil {
		return err
	}
	valid := 0
	if ev.Valid {
		valid = 1
	}
	trunc := 0
	if ev.BodyTruncated {
		trunc = 1
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO events (id, sink_id, provider, received_at, method, path, query_json, headers_json, content_type, body, body_truncated, status, latency_ms, valid, validation_json, summary, group_key)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, ev.ID, ev.SinkID, ev.Provider, ev.ReceivedAt.UTC().Format(time.RFC3339Nano), ev.Method, ev.Path, string(q), string(h), ev.ContentType, ev.Body, trunc, ev.Status, ev.LatencyMS, valid, string(v), ev.Summary, ev.GroupKey)
	return err
}

func (s *Store) GetEvent(ctx context.Context, id string) (Event, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, sink_id, provider, received_at, method, path, query_json, headers_json, content_type, body, body_truncated, status, latency_ms, valid, validation_json, summary, group_key FROM events WHERE id=?`, id)
	return scanEvent(row)
}

func (s *Store) ListEvents(ctx context.Context, f EventFilter) ([]Event, int, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	where, args := eventWhere(f)
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := s.db.QueryContext(ctx, `SELECT id, sink_id, provider, received_at, method, path, query_json, headers_json, content_type, body, body_truncated, status, latency_ms, valid, validation_json, summary, group_key FROM events`+where+` ORDER BY received_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		ev, err := scanEventRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, ev)
	}
	return out, total, rows.Err()
}

func eventWhere(f EventFilter) (string, []any) {
	var clauses []string
	var args []any
	if f.Provider != "" {
		clauses = append(clauses, `provider=?`)
		args = append(args, f.Provider)
	}
	if f.SinkID != "" {
		clauses = append(clauses, `sink_id=?`)
		args = append(args, f.SinkID)
	}
	if !f.Since.IsZero() {
		clauses = append(clauses, `received_at>=?`)
		args = append(args, f.Since.UTC().Format(time.RFC3339Nano))
	}
	if f.GroupKey != "" {
		clauses = append(clauses, `group_key=?`)
		args = append(args, f.GroupKey)
	}
	if f.Contains != "" {
		clauses = append(clauses, `CAST(body AS TEXT) LIKE ?`)
		args = append(args, "%"+f.Contains+"%")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (s *Store) DeleteEvents(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM events`)
	return err
}

func (s *Store) CountEvents(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&n)
	return n, err
}

func (s *Store) Prune(ctx context.Context, maxAge time.Duration, maxN int) (int, error) {
	cutoff := time.Now().UTC().Add(-maxAge).Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE received_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	deleted, _ := res.RowsAffected()
	if maxN > 0 {
		var total int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&total); err != nil {
			return int(deleted), err
		}
		if extra := total - maxN; extra > 0 {
			res, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE id IN (SELECT id FROM events ORDER BY received_at ASC LIMIT ?)`, extra)
			if err != nil {
				return int(deleted), err
			}
			n, _ := res.RowsAffected()
			deleted += n
		}
	}
	return int(deleted), nil
}

func (s *Store) InsertAttempt(ctx context.Context, a SendAttempt) error {
	h, err := json.Marshal(a.RequestHeaders)
	if err != nil {
		return err
	}
	var status any
	if a.Status != nil {
		status = *a.Status
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO send_attempts (id, created_at, provider, event_name, target, request_headers_json, body, status, error, latency_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.CreatedAt.UTC().Format(time.RFC3339Nano), a.Provider, a.EventName, a.Target, string(h), a.Body, status, a.Error, a.LatencyMS)
	return err
}

func (s *Store) ListAttempts(ctx context.Context) ([]SendAttempt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, created_at, provider, event_name, target, request_headers_json, body, status, error, latency_ms FROM send_attempts ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SendAttempt
	for rows.Next() {
		var a SendAttempt
		var created, headers string
		var status sql.NullInt64
		var errText sql.NullString
		if err := rows.Scan(&a.ID, &created, &a.Provider, &a.EventName, &a.Target, &headers, &a.Body, &status, &errText, &a.LatencyMS); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		_ = json.Unmarshal([]byte(headers), &a.RequestHeaders)
		a.BodyText = string(a.Body)
		if status.Valid {
			v := int(status.Int64)
			a.Status = &v
		}
		a.Error = errText.String
		out = append(out, a)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSink(row rowScanner) (Sink, error) {
	var sk Sink
	var chaos, created string
	if err := row.Scan(&sk.ID, &sk.Provider, &sk.Name, &sk.Token, &sk.Path, &chaos, &created); err != nil {
		return Sink{}, err
	}
	_ = json.Unmarshal([]byte(chaos), &sk.Chaos)
	sk.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return sk, nil
}

func scanSinkRow(rows *sql.Rows) (Sink, error) { return scanSink(rows) }

func scanEvent(row rowScanner) (Event, error) {
	var ev Event
	var received, q, h, vjson string
	var trunc, valid int
	if err := row.Scan(&ev.ID, &ev.SinkID, &ev.Provider, &received, &ev.Method, &ev.Path, &q, &h, &ev.ContentType, &ev.Body, &trunc, &ev.Status, &ev.LatencyMS, &valid, &vjson, &ev.Summary, &ev.GroupKey); err != nil {
		return Event{}, err
	}
	ev.ReceivedAt, _ = time.Parse(time.RFC3339Nano, received)
	_ = json.Unmarshal([]byte(q), &ev.Query)
	_ = json.Unmarshal([]byte(h), &ev.Headers)
	_ = json.Unmarshal([]byte(vjson), &ev.ValidationErrors)
	if ev.ValidationErrors == nil {
		ev.ValidationErrors = []ValidationError{}
	}
	if ev.Query == nil {
		ev.Query = map[string]string{}
	}
	if ev.Headers == nil {
		ev.Headers = map[string][]string{}
	}
	ev.BodyTruncated = trunc == 1
	ev.Valid = valid == 1
	ev.BodyText = string(ev.Body)
	return ev, nil
}

func scanEventRow(rows *sql.Rows) (Event, error) { return scanEvent(rows) }

func RedactHeaders(h map[string][]string) map[string][]string {
	out := make(map[string][]string, len(h))
	for k, vals := range h {
		cp := append([]string(nil), vals...)
		switch strings.ToLower(k) {
		case "authorization", "cookie", "x-api-key":
			for i := range cp {
				cp[i] = "***"
			}
		}
		out[k] = cp
	}
	return out
}

func (s *Store) EventCount(ctx context.Context) (int, error) {
	n, err := s.CountEvents(ctx)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return n, nil
}
