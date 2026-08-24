package sqlite

import (
	"path/filepath"
	"testing"
)

func TestOpenAppliesMigrationOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sinks (id, provider, name, token, path, chaos_json, created_at) VALUES ('s1','generic','g','t','/hooks/generic/t','{}','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	var n int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("migrations applied %d times", n)
	}
	var sinks int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM sinks`).Scan(&sinks); err != nil {
		t.Fatal(err)
	}
	if sinks != 1 {
		t.Fatalf("sinks %d", sinks)
	}
}
