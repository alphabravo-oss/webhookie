package webhookie

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/config"
	"github.com/alphabravo-oss/webhookie/internal/httpapi"
	"github.com/alphabravo-oss/webhookie/internal/sqlite"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestWaitFor(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	st := store.New(db.DB)
	s := httpapi.New(config.Config{MaxBodyBytes: 1 << 20, PublicBaseURL: "http://x"}, st, func(ctx context.Context) error { return nil })
	_ = s.Seed(context.Background())
	ts := httptest.NewServer(s.Router())
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = WaitFor(ctx, ts.URL, "slack", "nope")
	if err == nil {
		t.Fatal("expected timeout")
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	go func() {
		time.Sleep(30 * time.Millisecond)
		_, _ = http.Post(ts.URL+"/hooks/slack/services/T00000000/B00000000/webhookie", "application/json", strings.NewReader(`{"text":"deploy failed"}`))
	}()
	ev, err := WaitFor(ctx2, ts.URL, "slack", "deploy")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Provider != "slack" {
		t.Fatalf("%+v", ev)
	}
	if err := Reset(context.Background(), ts.URL); err != nil {
		t.Fatal(err)
	}
}
