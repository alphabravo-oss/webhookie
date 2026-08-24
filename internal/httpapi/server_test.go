package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/config"
	"github.com/alphabravo-oss/webhookie/internal/sqlite"
	"github.com/alphabravo-oss/webhookie/internal/store"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	st := store.New(db.DB)
	cfg := config.Config{Addr: ":0", MaxBodyBytes: 1 << 20, PublicBaseURL: "http://example", RetentionDays: 7, MaxEvents: 10000, Version: "test"}
	s := New(cfg, st, func(ctx context.Context) error { return st.Ping(ctx) })
	if err := s.Seed(context.Background()); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Router())
	t.Cleanup(ts.Close)
	return ts
}

func TestHealth(t *testing.T) {
	ts := testServer(t)
	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
	res, _ = http.Get(ts.URL + "/readyz")
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
}

func TestGenericCaptureAndList(t *testing.T) {
	ts := testServer(t)
	res, err := http.Post(ts.URL+"/hooks/generic/default", "application/json", strings.NewReader(`{"hello":"world"}`))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(b) != "{\"ok\":true}\n" && string(b) != `{"ok":true}` {
		if res.StatusCode != 200 || !strings.Contains(string(b), `"ok":true`) {
			t.Fatalf("%d %s", res.StatusCode, b)
		}
	}
	res, err = http.Get(ts.URL + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out struct {
		Data []store.Event `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 || out.Data[0].Provider != "generic" {
		t.Fatalf("%+v", out.Data)
	}
	id := out.Data[0].ID
	res, err = http.Get(ts.URL + "/api/v1/events/" + id)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/events", nil)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	res, _ = http.Get(ts.URL + "/api/v1/events")
	defer res.Body.Close()
	out.Data = nil
	_ = json.NewDecoder(res.Body).Decode(&out)
	if len(out.Data) != 0 {
		t.Fatalf("expected empty %+v", out.Data)
	}
}

func TestUnknownHook404(t *testing.T) {
	ts := testServer(t)
	res, err := http.Post(ts.URL+"/hooks/nope", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("status %d", res.StatusCode)
	}
	res, _ = http.Get(ts.URL + "/api/v1/events")
	defer res.Body.Close()
	var out struct {
		Data []store.Event `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	if len(out.Data) != 0 {
		t.Fatal("should not store 404")
	}
}

func TestSlackAndPagerDuty(t *testing.T) {
	ts := testServer(t)
	res, err := http.Post(ts.URL+"/hooks/slack/services/T00000000/B00000000/webhookie", "application/json", strings.NewReader(`{"text":"deploy failed"}`))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(b) != "ok" {
		t.Fatalf("%d %q", res.StatusCode, b)
	}
	body := `{"routing_key":"0123456789abcdef0123456789abcdef","event_action":"trigger","payload":{"summary":"disk","source":"host","severity":"error"}}`
	res, err = http.Post(ts.URL+"/hooks/pagerduty/v2/enqueue", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 202 {
		t.Fatalf("pd %d %s", res.StatusCode, b)
	}
	var pd map[string]string
	_ = json.Unmarshal(b, &pd)
	if pd["dedup_key"] == "" {
		t.Fatal("missing dedup")
	}
	resolve := `{"routing_key":"0123456789abcdef0123456789abcdef","event_action":"resolve","dedup_key":"` + pd["dedup_key"] + `"}`
	res, _ = http.Post(ts.URL+"/hooks/pagerduty/v2/enqueue", "application/json", strings.NewReader(resolve))
	res.Body.Close()
	res, _ = http.Get(ts.URL + "/api/v1/events?groupKey=" + pd["dedup_key"])
	defer res.Body.Close()
	var out struct {
		Data []store.Event `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	if len(out.Data) != 2 {
		t.Fatalf("want 2 grouped got %d", len(out.Data))
	}
}

func TestChaos503(t *testing.T) {
	ts := testServer(t)
	payload := `{"chaos":{"status":503,"body":"down"}}`
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/sinks/sink-generic", strings.NewReader(payload))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	res, _ = http.Post(ts.URL+"/hooks/generic/default", "application/json", strings.NewReader(`{"x":1}`))
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 503 {
		t.Fatalf("%d %s", res.StatusCode, b)
	}
}

func TestSendToSelf(t *testing.T) {
	ts := testServer(t)
	in := map[string]string{"provider": "standard", "event": "generic.ping", "target": ts.URL + "/hooks/generic/default", "secret": "whsec_c2VjcmV0"}
	b, _ := json.Marshal(in)
	res, err := http.Post(ts.URL+"/api/v1/send", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
	time.Sleep(50 * time.Millisecond)
	res, _ = http.Get(ts.URL + "/api/v1/events?contains=ping")
	defer res.Body.Close()
	var out struct {
		Data []store.Event `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	if len(out.Data) == 0 {
		t.Fatal("expected captured send")
	}
	found := false
	for k := range out.Data[0].Headers {
		if strings.EqualFold(k, "Webhook-Signature") || strings.EqualFold(k, "webhook-signature") {
			found = true
		}
	}
	if !found {
		t.Fatalf("headers %+v", out.Data[0].Headers)
	}
}

func TestPasswordLeavesHooksOpen(t *testing.T) {
	db, _ := sqlite.Open(filepath.Join(t.TempDir(), "t.db"))
	st := store.New(db.DB)
	cfg := config.Config{MaxBodyBytes: 1 << 20, Password: "s3cret", PublicBaseURL: "http://x"}
	s := New(cfg, st, func(ctx context.Context) error { return nil })
	_ = s.Seed(context.Background())
	ts := httptest.NewServer(s.Router())
	t.Cleanup(ts.Close)
	res, _ := http.Get(ts.URL + "/api/v1/events")
	if res.StatusCode != 401 {
		t.Fatalf("api %d", res.StatusCode)
	}
	res.Body.Close()
	res, _ = http.Post(ts.URL+"/hooks/generic/default", "application/json", strings.NewReader(`{"ok":true}`))
	if res.StatusCode != 200 {
		t.Fatalf("hook %d", res.StatusCode)
	}
	res.Body.Close()
}

func TestReplay(t *testing.T) {
	ts := testServer(t)
	_, _ = http.Post(ts.URL+"/hooks/slack/services/T00000000/B00000000/webhookie", "application/json", strings.NewReader(`{"text":"hello-replay"}`))
	res, _ := http.Get(ts.URL + "/api/v1/events")
	var out struct {
		Data []store.Event `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	id := out.Data[0].ID
	body, _ := json.Marshal(map[string]string{"target": ts.URL + "/hooks/generic/default"})
	res, err := http.Post(ts.URL+"/api/v1/events/"+id+"/replay", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	res, _ = http.Get(ts.URL + "/api/v1/events?contains=hello-replay")
	defer res.Body.Close()
	out.Data = nil
	_ = json.NewDecoder(res.Body).Decode(&out)
	if len(out.Data) < 2 {
		t.Fatalf("want replay capture got %d", len(out.Data))
	}
}
