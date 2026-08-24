package mattermost

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestText(t *testing.T) {
	s := Sink{}
	body := []byte(`{"text":"Hello, this is some text","channel":"town-square","username":"bot"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/mattermost/hooks/webhookie", nil)
	req.Header.Set("Content-Type", "application/json")
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
	req := httptest.NewRequest(http.MethodPost, "/hooks/mattermost/hooks/webhookie", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, []byte(`{"username":"bot"}`), store.Chaos{})
	if w.Code != 400 || !strings.Contains(w.Body.String(), "Unable to parse") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
