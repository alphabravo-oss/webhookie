package discord

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func Test204(t *testing.T) {
	s := Sink{}
	body := []byte(`{"content":"hi"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/discord/api/webhooks/0/webhookie", nil)
	if !s.Match(req) {
		t.Fatal("match")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 204 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestWaitTrue(t *testing.T) {
	s := Sink{}
	body := []byte(`{"content":"hi"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/discord/api/webhooks/0/webhookie?wait=true", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 200 || !stringsContains(w.Body.String(), `"id"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestEmptyInvalid(t *testing.T) {
	s := Sink{}
	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/discord/api/webhooks/0/webhookie", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 400 {
		t.Fatalf("code %d", w.Code)
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
