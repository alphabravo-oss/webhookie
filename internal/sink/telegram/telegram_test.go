package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func tgReq() *http.Request {
	return httptest.NewRequest(http.MethodPost, "/hooks/telegram/bot/123456:AAWebhookie/sendMessage", nil)
}

func TestSendMessage(t *testing.T) {
	s := Sink{}
	body := []byte(`{"chat_id":123,"text":"hello","reply_markup":{"inline_keyboard":[[{"text":"Approve","callback_data":"ok"}]]}}`)
	req := tgReq()
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

func TestValidateTable(t *testing.T) {
	s := Sink{}
	req := tgReq()
	tests := []struct {
		name    string
		body    string
		valid   bool
		path    string
		message string
		desc    string
	}{
		{
			name:  "string chat_id plus extras",
			body:  `{"chat_id":"@mychannel","text":"hello","disable_notification":true,"parse_mode":"HTML"}`,
			valid: true,
		},
		{
			name:  "url button",
			body:  `{"chat_id":1,"text":"docs","reply_markup":{"inline_keyboard":[[{"text":"Open","url":"https://example.com"}]]}}`,
			valid: true,
		},
		{
			name:    "missing chat_id",
			body:    `{"text":"hi"}`,
			path:    "/chat_id",
			message: "required",
			desc:    "chat_id is empty",
		},
		{
			name:    "text too long",
			body:    `{"chat_id":1,"text":"` + strings.Repeat("a", maxText+1) + `"}`,
			path:    "/text",
			message: "4096",
			desc:    "message is too long",
		},
		{
			name:    "bad parse_mode",
			body:    `{"chat_id":1,"text":"hi","parse_mode":"bbcode"}`,
			path:    "/parse_mode",
			message: "Markdown",
			desc:    "can't parse entities",
		},
		{
			name:    "callback_data too long",
			body:    `{"chat_id":1,"text":"hi","reply_markup":{"inline_keyboard":[[{"text":"x","callback_data":"` + strings.Repeat("c", maxCallbackData+1) + `"}]]}}`,
			path:    "/reply_markup/inline_keyboard/0/0/callback_data",
			message: "1-64",
			desc:    "BUTTON_DATA_INVALID",
		},
		{
			name:    "button missing action",
			body:    `{"chat_id":1,"text":"hi","reply_markup":{"inline_keyboard":[[{"text":"x"}]]}}`,
			path:    "/reply_markup/inline_keyboard/0/0",
			message: "callback_data",
		},
		{
			name:    "button missing text",
			body:    `{"chat_id":1,"text":"hi","reply_markup":{"inline_keyboard":[[{"callback_data":"ok"}]]}}`,
			path:    "/reply_markup/inline_keyboard/0/0/text",
			message: "required",
		},
		{
			name:  "html nested tags",
			body:  `{"chat_id":1,"text":"<b>deploy <i>failed</i></b>","parse_mode":"HTML"}`,
			valid: true,
		},
		{
			name:  "markdownv2 escaped punctuation",
			body:  `{"chat_id":1,"text":"disk is 92\\% full\\.","parse_mode":"MarkdownV2"}`,
			valid: true,
		},
		{
			name:    "markdownv2 unescaped bang",
			body:    `{"chat_id":1,"text":"Hello world!","parse_mode":"MarkdownV2"}`,
			path:    "/text",
			message: "can't parse entities",
			desc:    "can't parse entities",
		},
		{
			name:    "html unmatched tag",
			body:    `{"chat_id":1,"text":"<b>bold","parse_mode":"HTML"}`,
			path:    "/text",
			message: "can't parse entities",
			desc:    "can't parse entities",
		},
		{
			name:  "entities instead of parse_mode",
			body:  `{"chat_id":1,"text":"hello","entities":[{"type":"bold","offset":0,"length":5}]}`,
			valid: true,
		},
		{
			name:    "entity past end of text",
			body:    `{"chat_id":1,"text":"hi","entities":[{"type":"bold","offset":0,"length":10}]}`,
			path:    "/entities/0",
			message: "UTF-16",
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
			if tc.desc == "" {
				return
			}
			w := httptest.NewRecorder()
			_ = s.Respond(w, req, []byte(tc.body), store.Chaos{})
			if !strings.Contains(w.Body.String(), tc.desc) {
				t.Fatalf("description %s", w.Body.String())
			}
		})
	}
}
