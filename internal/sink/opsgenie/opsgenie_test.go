package opsgenie

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestValidateTable(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodPost, "/hooks/opsgenie/v2/alerts", nil)
	tests := []struct {
		name    string
		body    string
		valid   bool
		path    string
		message string
	}{
		{
			name:  "extras allowed",
			body:  `{"message":"cpu","alias":"cpu-1","priority":"P3","description":"load 4","entity":"api-2","source":"nagios","user":"oncall","note":"page","details":{"region":"us"}}`,
			valid: true,
		},
		{
			name:  "tags at limit",
			body:  `{"message":"x","tags":["a","b","c"]}`,
			valid: true,
		},
		{
			name:    "message too long",
			body:    `{"message":"` + strings.Repeat("m", maxMessage+1) + `"}`,
			path:    "/message",
			message: "130",
		},
		{
			name:    "alias too long",
			body:    `{"message":"x","alias":"` + strings.Repeat("a", maxAlias+1) + `"}`,
			path:    "/alias",
			message: "512",
		},
		{
			name:    "description too long",
			body:    `{"message":"x","description":"` + strings.Repeat("d", maxDescription+1) + `"}`,
			path:    "/description",
			message: "15000",
		},
		{
			name:    "bad priority",
			body:    `{"message":"x","priority":"P0"}`,
			path:    "/priority",
			message: "P1|P2|P3|P4|P5",
		},
		{
			name:    "too many tags",
			body:    `{"message":"x","tags":["1","2","3","4","5","6","7","8","9","10","11","12","13","14","15","16","17","18","19","20","21"]}`,
			path:    "/tags",
			message: "20",
		},
		{
			name:    "tag too long",
			body:    `{"message":"x","tags":["` + strings.Repeat("t", maxTagLen+1) + `"]}`,
			path:    "/tags/0",
			message: "50",
		},
		{
			name:    "missing message",
			body:    `{"priority":"P1"}`,
			path:    "/message",
			message: "required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := s.Validate(req, []byte(tc.body))
			if tc.valid {
				if !v.Valid {
					t.Fatalf("%+v", v)
				}
				return
			}
			if v.Valid || !v.Has(tc.path, tc.message) {
				t.Fatalf("want %s %q got %+v", tc.path, tc.message, v.Errors)
			}
			w := httptest.NewRecorder()
			_ = s.Respond(w, req, []byte(tc.body), store.Chaos{})
			if w.Code != 422 {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), `"errors"`) {
				t.Fatal(w.Body.String())
			}
		})
	}
}
