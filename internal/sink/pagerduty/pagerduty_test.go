package pagerduty

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestTriggerGeneratesKey(t *testing.T) {
	s := Sink{}
	body := []byte(`{"routing_key":"0123456789abcdef0123456789abcdef","event_action":"trigger","payload":{"summary":"disk","source":"host","severity":"error"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/pagerduty/v2/enqueue", nil)
	if !s.Match(req) {
		t.Fatal("match")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 202 {
		t.Fatalf("code %d %s", w.Code, w.Body.String())
	}
	var out map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["dedup_key"] == "" || out["status"] != "success" {
		t.Fatalf("%v", out)
	}
}

func TestAckRequiresKey(t *testing.T) {
	s := Sink{}
	body := []byte(`{"routing_key":"0123456789abcdef0123456789abcdef","event_action":"acknowledge"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/pagerduty/v2/enqueue", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 400 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestMissingRoutingKey(t *testing.T) {
	s := Sink{}
	body := []byte(`{"event_action":"trigger","payload":{"summary":"x","source":"y","severity":"info"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/pagerduty/v2/enqueue", nil)
	if s.Validate(req, body).Valid {
		t.Fatal("expected invalid")
	}
}

func TestChangeEvent(t *testing.T) {
	s := Sink{}
	body := []byte(`{"routing_key":"0123456789abcdef0123456789abcdef","payload":{"summary":"deploy"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/pagerduty/v2/change", nil)
	if !s.Match(req) {
		t.Fatal("match")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 202 {
		t.Fatalf("code %d %s", w.Code, w.Body.String())
	}
}

func TestResolveSameKey(t *testing.T) {
	s := Sink{}
	key := "abc"
	body := []byte(`{"routing_key":"0123456789abcdef0123456789abcdef","event_action":"resolve","dedup_key":"` + key + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/pagerduty/v2/enqueue", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 202 || !strings.Contains(w.Body.String(), key) {
		t.Fatalf("%s", w.Body.String())
	}
	if s.Summarize(req, body).GroupKey != key {
		t.Fatal("group key")
	}
}
