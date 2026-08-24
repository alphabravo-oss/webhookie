package opsgenie

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestCreate(t *testing.T) {
	s := Sink{}
	body := []byte(`{"message":"disk full","alias":"disk-api-2","priority":"P1"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/opsgenie/v2/alerts", nil)
	if !s.Match(req) {
		t.Fatal("match")
	}
	if !s.Validate(req, body).Valid {
		t.Fatal("valid")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 202 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["result"] != "Request will be processed" {
		t.Fatalf("%s", w.Body.String())
	}
	if w.Header().Get("X-Webhookie-Dedup-Key") != "disk-api-2" {
		t.Fatal(w.Header().Get("X-Webhookie-Dedup-Key"))
	}
	if s.Summarize(req, body).GroupKey != "disk-api-2" {
		t.Fatal(s.Summarize(req, body))
	}
}

func TestAckPath(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodPost, "/hooks/opsgenie/v2/alerts/disk-api-2/acknowledge", nil)
	if !s.Match(req) {
		t.Fatal("match")
	}
	if !s.Validate(req, []byte(`{}`)).Valid {
		t.Fatal("valid")
	}
	if s.Summarize(req, []byte(`{}`)).Text != "acknowledge" {
		t.Fatal(s.Summarize(req, []byte(`{}`)).Text)
	}
}

func TestMissingMessage(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodPost, "/hooks/opsgenie/v2/alerts", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, []byte(`{"priority":"P1"}`), store.Chaos{})
	if w.Code != 422 {
		t.Fatalf("%d", w.Code)
	}
}
