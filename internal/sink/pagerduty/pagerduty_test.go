package pagerduty

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func pdReq() *http.Request {
	return httptest.NewRequest(http.MethodPost, "/hooks/pagerduty/v2/enqueue", nil)
}

func TestTriggerGeneratesKey(t *testing.T) {
	s := Sink{}
	body := []byte(`{"routing_key":"0123456789abcdef0123456789abcdef","event_action":"trigger","payload":{"summary":"disk","source":"host","severity":"error"}}`)
	req := pdReq()
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
	req := pdReq()
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 400 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestMissingRoutingKey(t *testing.T) {
	s := Sink{}
	body := []byte(`{"event_action":"trigger","payload":{"summary":"x","source":"y","severity":"info"}}`)
	req := pdReq()
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
	req := pdReq()
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 202 || !strings.Contains(w.Body.String(), key) {
		t.Fatalf("%s", w.Body.String())
	}
	if s.Summarize(req, body).GroupKey != key {
		t.Fatal("group key")
	}
}

func TestValidateTable(t *testing.T) {
	s := Sink{}
	req := pdReq()
	rk := "0123456789abcdef0123456789abcdef"
	tests := []struct {
		name    string
		path    string
		body    string
		valid   bool
		errPath string
		message string
	}{
		{
			name:  "trigger with custom_details extra",
			body:  `{"routing_key":"` + rk + `","event_action":"trigger","client":"ci","payload":{"summary":"cpu","source":"api-2","severity":"warning","custom_details":{"load":"4.2"},"timestamp":"2026-01-01T00:00:00Z"}}`,
			valid: true,
		},
		{
			name:    "routing_key short",
			body:    `{"routing_key":"short","event_action":"trigger","payload":{"summary":"x","source":"y","severity":"info"}}`,
			errPath: "/routing_key",
			message: "32",
		},
		{
			name:    "bad severity",
			body:    `{"routing_key":"` + rk + `","event_action":"trigger","payload":{"summary":"x","source":"y","severity":"fatal"}}`,
			errPath: "/payload/severity",
			message: "info|warning|error|critical",
		},
		{
			name:    "missing source",
			body:    `{"routing_key":"` + rk + `","event_action":"trigger","payload":{"summary":"x","severity":"error"}}`,
			errPath: "/payload/source",
			message: "required",
		},
		{
			name:    "summary too long",
			body:    `{"routing_key":"` + rk + `","event_action":"trigger","payload":{"summary":"` + strings.Repeat("s", maxSummary+1) + `","source":"y","severity":"info"}}`,
			errPath: "/payload/summary",
			message: "1024",
		},
		{
			name:    "bad event_action",
			body:    `{"routing_key":"` + rk + `","event_action":"snooze","payload":{"summary":"x","source":"y","severity":"info"}}`,
			errPath: "/event_action",
			message: "trigger",
		},
		{
			name:    "ack missing dedup",
			body:    `{"routing_key":"` + rk + `","event_action":"acknowledge"}`,
			errPath: "/dedup_key",
			message: "required",
		},
		{
			name:    "too many images",
			body:    `{"routing_key":"` + rk + `","event_action":"trigger","payload":{"summary":"x","source":"y","severity":"info"},"images":[{},{},{},{},{},{},{},{},{}]}`,
			errPath: "/images",
			message: "8",
		},
		{
			name:    "change missing summary",
			path:    "/hooks/pagerduty/v2/change",
			body:    `{"routing_key":"` + rk + `","payload":{}}`,
			errPath: "/payload/summary",
			message: "required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := req
			if tc.path != "" {
				r = httptest.NewRequest(http.MethodPost, tc.path, nil)
			}
			v := s.Validate(r, []byte(tc.body))
			if tc.valid {
				if !v.Valid {
					t.Fatalf("%+v", v)
				}
				return
			}
			if v.Valid || !v.Has(tc.errPath, tc.message) {
				t.Fatalf("want %s %q got %+v", tc.errPath, tc.message, v.Errors)
			}
			w := httptest.NewRecorder()
			_ = s.Respond(w, r, []byte(tc.body), store.Chaos{})
			if w.Code != 400 || !strings.Contains(w.Body.String(), `"invalid event"`) {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), `"errors"`) {
				t.Fatal("pagerduty envelope should include errors")
			}
		})
	}
}
