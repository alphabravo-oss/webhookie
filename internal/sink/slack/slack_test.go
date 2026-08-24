package slack

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestTextOnly(t *testing.T) {
	s := Sink{}
	body := []byte(`{"text":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/slack/services/T00000000/B00000000/webhookie", nil)
	req.Header.Set("Content-Type", "application/json")
	if !s.Match(req) {
		t.Fatal("match")
	}
	v := s.Validate(req, body)
	if !v.Valid {
		t.Fatalf("%+v", v)
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	b, _ := io.ReadAll(w.Body)
	if w.Code != 200 || string(b) != "ok" {
		t.Fatalf("%d %q", w.Code, b)
	}
}

func TestBlocks(t *testing.T) {
	s := Sink{}
	body := []byte(`{"blocks":[{"type":"section","text":{"type":"mrkdwn","text":"*deploy failed*"}},{"type":"divider"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/slack/services/T00000000/B00000000/webhookie", nil)
	req.Header.Set("Content-Type", "application/json")
	if !s.Validate(req, body).Valid {
		t.Fatal("expected valid")
	}
	if s.Summarize(req, body).Text != "*deploy failed*" {
		t.Fatalf("summary %q", s.Summarize(req, body).Text)
	}
}

func TestMissingTextAndBlocks(t *testing.T) {
	s := Sink{}
	body := []byte(`{"username":"bot"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/slack/services/T00000000/B00000000/webhookie", nil)
	req.Header.Set("Content-Type", "application/json")
	v := s.Validate(req, body)
	if v.Valid {
		t.Fatal("expected invalid")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 400 || !strings.Contains(w.Body.String(), `"invalid_payload"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestGETNotMatched(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodGet, "/hooks/slack/services/T00000000/B00000000/webhookie", nil)
	if s.Match(req) {
		t.Fatal("GET should not match")
	}
}
