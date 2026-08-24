package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestSendMessage(t *testing.T) {
	s := Sink{}
	body := []byte(`{"chat_id":123,"text":"hello","reply_markup":{"inline_keyboard":[[{"text":"Approve","callback_data":"ok"}]]}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/telegram/bot/123456:AAWebhookie/sendMessage", nil)
	if !s.Match(req) {
		t.Fatal("match")
	}
	if !s.Validate(req, body).Valid {
		t.Fatal("valid")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 200 {
		t.Fatalf("code %d %s", w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["ok"] != true {
		t.Fatalf("%s", w.Body.String())
	}
	if s.Summarize(req, body).Text != "hello" {
		t.Fatalf("summary %q", s.Summarize(req, body).Text)
	}
}

func TestMissingText(t *testing.T) {
	s := Sink{}
	body := []byte(`{"chat_id":1}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/telegram/bot/t/sendMessage", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 400 || !strings.Contains(w.Body.String(), "message text is empty") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestGETNotMatched(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodGet, "/hooks/telegram/bot/t/sendMessage", nil)
	if s.Match(req) {
		t.Fatal("GET")
	}
}
