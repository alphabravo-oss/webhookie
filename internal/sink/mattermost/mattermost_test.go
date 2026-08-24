package mattermost

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func mmReq() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/hooks/mattermost/hooks/webhookie", nil)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestText(t *testing.T) {
	s := Sink{}
	body := []byte(`{"text":"Hello, this is some text","channel":"town-square","username":"bot"}`)
	req := mmReq()
	if !s.Match(req) {
		t.Fatal("match")
	}
	if !s.Validate(req, body).Valid {
		t.Fatal("valid")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	b, _ := io.ReadAll(w.Body)
	if w.Code != 200 || string(b) != "ok" {
		t.Fatalf("%d %q", w.Code, b)
	}
}

func TestPayloadForm(t *testing.T) {
	s := Sink{}
	body := []byte(`payload={"text":"from slack shape"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/mattermost/hooks/webhookie", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if !s.Validate(req, body).Valid {
		t.Fatal("valid")
	}
	if s.Summarize(req, body).Text != "from slack shape" {
		t.Fatal(s.Summarize(req, body).Text)
	}
}

func TestMissing(t *testing.T) {
	s := Sink{}
	req := mmReq()
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, []byte(`{"username":"bot"}`), store.Chaos{})
	if w.Code != 400 || !strings.Contains(w.Body.String(), "Unable to parse") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestValidateTable(t *testing.T) {
	s := Sink{}
	req := mmReq()
	tests := []struct {
		name    string
		body    string
		valid   bool
		path    string
		message string
	}{
		{
			name:  "attachments with actions",
			body:  `{"attachments":[{"title":"PR","text":"ready","actions":[{"type":"button","name":"Approve","integration":{"url":"https://example.com","context":{"id":"1"}}}]}]}`,
			valid: true,
		},
		{
			name:  "icon extras",
			body:  `{"text":"hi","icon_url":"https://example.com/a.png","props":{"card":"x"}}`,
			valid: true,
		},
		{
			name:    "empty",
			body:    `{"username":"bot"}`,
			path:    "/",
			message: "text or attachments",
		},
		{
			name:    "text too long",
			body:    `{"text":"` + strings.Repeat("a", maxText+1) + `"}`,
			path:    "/text",
			message: "16383",
		},
		{
			name:    "attachments not array",
			body:    `{"attachments":{"text":"x"}}`,
			path:    "/attachments",
			message: "array",
		},
		{
			name:    "action missing name",
			body:    `{"attachments":[{"actions":[{"type":"button"}]}]}`,
			path:    "/attachments/0/actions/0/name",
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
			if w.Code != 400 || !strings.Contains(w.Body.String(), "Unable to parse") {
				t.Fatalf("http envelope %d %s", w.Code, w.Body.String())
			}
		})
	}
}
