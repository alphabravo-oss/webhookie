package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/sqlite"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return New(db.DB)
}

func TestStoreEvents(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	sk := Sink{ID: "sink-generic", Provider: "generic", Name: "g", Token: "default", Path: "/hooks/generic/default", CreatedAt: time.Now()}
	if err := s.UpsertSink(ctx, sk); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSink(ctx, "sink-generic")
	if err != nil || got.Path != sk.Path {
		t.Fatalf("%v %+v", err, got)
	}
	now := time.Now().UTC()
	e1 := Event{ID: "e1", SinkID: sk.ID, Provider: "generic", ReceivedAt: now.Add(-time.Second), Method: "POST", Path: sk.Path, Query: map[string]string{}, Headers: map[string][]string{"X": {"1"}}, Body: []byte(`{"a":1}`), Status: 200, Valid: true, ValidationErrors: []ValidationError{}, Summary: "a"}
	e2 := Event{ID: "e2", SinkID: sk.ID, Provider: "generic", ReceivedAt: now, Method: "POST", Path: sk.Path, Query: map[string]string{}, Headers: map[string][]string{}, Body: []byte(`{"b":2}`), Status: 200, Valid: true, ValidationErrors: []ValidationError{}, Summary: "b"}
	if err := s.InsertEvent(ctx, e1); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertEvent(ctx, e2); err != nil {
		t.Fatal(err)
	}
	items, total, err := s.ListEvents(ctx, EventFilter{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || items[0].ID != "e2" {
		t.Fatalf("newest first: total=%d first=%s", total, items[0].ID)
	}
	one, err := s.GetEvent(ctx, "e1")
	if err != nil || string(one.Body) != `{"a":1}` {
		t.Fatalf("%v %+v", err, one)
	}
	if err := s.DeleteEvents(ctx); err != nil {
		t.Fatal(err)
	}
	_, total, err = s.ListEvents(ctx, EventFilter{})
	if err != nil || total != 0 {
		t.Fatalf("deleted total=%d err=%v", total, err)
	}
}

func TestPruneKeepsNewest(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	sk := Sink{ID: "s", Provider: "generic", Name: "g", Token: "t", Path: "/hooks/generic/t", CreatedAt: time.Now()}
	_ = s.UpsertSink(ctx, sk)
	now := time.Now().UTC()
	for i, id := range []string{"a", "b", "c"} {
		_ = s.InsertEvent(ctx, Event{ID: id, SinkID: "s", Provider: "generic", ReceivedAt: now.Add(time.Duration(i) * time.Second), Method: "POST", Path: sk.Path, Query: map[string]string{}, Headers: map[string][]string{}, Body: []byte(id), Status: 200, Valid: true})
	}
	n, err := s.Prune(ctx, 24*time.Hour, 2)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("deleted %d", n)
	}
	items, total, _ := s.ListEvents(ctx, EventFilter{})
	if total != 2 || items[0].ID != "c" || items[1].ID != "b" {
		t.Fatalf("%+v", items)
	}
}

func TestContainsFilter(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	sk := Sink{ID: "s", Provider: "slack", Name: "s", Token: "t", Path: "/hooks/slack/x", CreatedAt: time.Now()}
	_ = s.UpsertSink(ctx, sk)
	_ = s.InsertEvent(ctx, Event{ID: "1", SinkID: "s", Provider: "slack", ReceivedAt: time.Now(), Method: "POST", Path: sk.Path, Query: map[string]string{}, Headers: map[string][]string{}, Body: []byte(`deploy failed`), Status: 200, Valid: true})
	_ = s.InsertEvent(ctx, Event{ID: "2", SinkID: "s", Provider: "slack", ReceivedAt: time.Now(), Method: "POST", Path: sk.Path, Query: map[string]string{}, Headers: map[string][]string{}, Body: []byte(`hello`), Status: 200, Valid: true})
	items, total, err := s.ListEvents(ctx, EventFilter{Contains: "deploy"})
	if err != nil || total != 1 || items[0].ID != "1" {
		t.Fatalf("total=%d items=%+v err=%v", total, items, err)
	}
}
