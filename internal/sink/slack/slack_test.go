package slack

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func slackReq() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/hooks/slack/services/T00000000/B00000000/webhookie", nil)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestTextOnly(t *testing.T) {
	s := Sink{}
	body := []byte(`{"text":"hello"}`)
	req := slackReq()
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
	req := slackReq()
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
	req := slackReq()
	v := s.Validate(req, body)
	if v.Valid {
		t.Fatal("expected invalid")
	}
	if !v.Has("/", "text, blocks, or attachments") {
		t.Fatalf("%+v", v)
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 400 || !strings.Contains(w.Body.String(), `"invalid_payload"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "blocks") {
		t.Fatal("slack envelope must not leak validation details")
	}
}

func TestGETNotMatched(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodGet, "/hooks/slack/services/T00000000/B00000000/webhookie", nil)
	if s.Match(req) {
		t.Fatal("GET should not match")
	}
}

func TestValidateTable(t *testing.T) {
	s := Sink{}
	longText := strings.Repeat("x", maxText+1)
	tooMany := make([]string, maxBlocks+1)
	for i := range tooMany {
		tooMany[i] = `{"type":"divider"}`
	}
	tests := []struct {
		name    string
		body    string
		valid   bool
		path    string
		message string
	}{
		{
			name:  "text plus extra fields",
			body:  `{"text":"hi","username":"deploy-bot","icon_emoji":":rocket:","channel":"#alerts"}`,
			valid: true,
		},
		{
			name:  "form payload still decoded as json",
			body:  `payload={"text":"from form"}`,
			valid: true,
		},
		{
			name:  "legacy attachment only",
			body:  `{"attachments":[{"color":"#ff0000","text":"disk full","ts":"1"}]}`,
			valid: true,
		},
		{
			name:  "actions with button — typical two-way payload",
			body:  `{"text":"approve this","blocks":[{"type":"actions","elements":[{"type":"button","action_id":"approve","text":{"type":"plain_text","text":"Approve"},"value":"yes"}]}]}`,
			valid: true,
		},
		{
			name:  "section fields",
			body:  `{"blocks":[{"type":"section","fields":[{"type":"mrkdwn","text":"*env*\nprod"},{"type":"plain_text","text":"ok"}]}]}`,
			valid: true,
		},
		{
			name:    "empty object",
			body:    `{}`,
			path:    "/",
			message: "text, blocks, or attachments",
		},
		{
			name:    "whitespace text only",
			body:    `{"text":"   "}`,
			path:    "/",
			message: "text, blocks, or attachments",
		},
		{
			name:    "text not a string",
			body:    `{"text":123}`,
			path:    "/text",
			message: "string",
		},
		{
			name:    "text over 40000",
			body:    `{"text":"` + longText + `"}`,
			path:    "/text",
			message: "40000",
		},
		{
			name:    "unknown block type",
			body:    `{"blocks":[{"type":"not_a_block"}]}`,
			path:    "/blocks/0/type",
			message: "unknown",
		},
		{
			name:    "block missing type",
			body:    `{"blocks":[{"text":{"type":"mrkdwn","text":"x"}}]}`,
			path:    "/blocks/0/type",
			message: "required",
		},
		{
			name:    "block not object",
			body:    `{"blocks":["section"]}`,
			path:    "/blocks/0",
			message: "object",
		},
		{
			name:    "too many blocks",
			body:    `{"blocks":[` + strings.Join(tooMany, ",") + `]}`,
			path:    "/blocks",
			message: "50",
		},
		{
			name:    "empty section",
			body:    `{"blocks":[{"type":"section"}]}`,
			path:    "/blocks/0",
			message: "text, fields, or accessory",
		},
		{
			name:    "section text wrong type",
			body:    `{"blocks":[{"type":"section","text":{"type":"markdown","text":"hi"}}]}`,
			path:    "/blocks/0/text/type",
			message: "plain_text or mrkdwn",
		},
		{
			name:    "actions missing elements",
			body:    `{"blocks":[{"type":"actions"}]}`,
			path:    "/blocks/0/elements",
			message: "required",
		},
		{
			name:    "button missing text",
			body:    `{"blocks":[{"type":"actions","elements":[{"type":"button","action_id":"x"}]}]}`,
			path:    "/blocks/0/elements/0/text",
			message: "required",
		},
		{
			name:    "header must be plain_text",
			body:    `{"blocks":[{"type":"header","text":{"type":"mrkdwn","text":"Title"}}]}`,
			path:    "/blocks/0/text/type",
			message: "plain_text",
		},
		{
			name:    "image missing alt_text",
			body:    `{"blocks":[{"type":"image","image_url":"https://example.com/a.png"}]}`,
			path:    "/blocks/0/alt_text",
			message: "required",
		},
		{
			name:    "invalid json",
			body:    `{`,
			path:    "/",
			message: "invalid json",
		},
		{
			name:  "image accessory",
			body:  `{"blocks":[{"type":"section","text":{"type":"mrkdwn","text":"pic"},"accessory":{"type":"image","image_url":"https://example.com/a.png","alt_text":"a"}}]}`,
			valid: true,
		},
		{
			name:  "overflow menu",
			body:  `{"blocks":[{"type":"actions","elements":[{"type":"overflow","options":[{"text":{"type":"plain_text","text":"A"},"value":"a"},{"text":{"type":"plain_text","text":"B"},"value":"b"}]}]}]}`,
			valid: true,
		},
		{
			name:    "button style invalid",
			body:    `{"blocks":[{"type":"actions","elements":[{"type":"button","text":{"type":"plain_text","text":"X"},"style":"green"}]}]}`,
			path:    "/blocks/0/elements/0/style",
			message: "primary or danger",
		},
		{
			name:    "overflow missing options",
			body:    `{"blocks":[{"type":"actions","elements":[{"type":"overflow"}]}]}`,
			path:    "/blocks/0/elements/0/options",
			message: "required",
		},
		{
			name:    "video missing title",
			body:    `{"blocks":[{"type":"video","video_url":"https://example.com/v.mp4","thumbnail_url":"https://example.com/t.png","alt_text":"v"}]}`,
			path:    "/blocks/0/title",
			message: "required",
		},
		{
			name:  "rich_text",
			body:  `{"blocks":[{"type":"rich_text","elements":[{"type":"rich_text_section","elements":[{"type":"text","text":"hello"}]}]}]}`,
			valid: true,
		},
		{
			name:    "static_select needs options",
			body:    `{"blocks":[{"type":"actions","elements":[{"type":"static_select","placeholder":{"type":"plain_text","text":"pick"}}]}]}`,
			path:    "/blocks/0/elements/0",
			message: "options or option_groups",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := slackReq()
			if tc.name == "form payload still decoded as json" {
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
			v := s.Validate(r, []byte(tc.body))
			if tc.valid {
				if !v.Valid {
					t.Fatalf("expected valid: %+v", v)
				}
				return
			}
			if v.Valid {
				t.Fatal("expected invalid")
			}
			if !v.Has(tc.path, tc.message) {
				t.Fatalf("want %s containing %q; got %+v", tc.path, tc.message, v.Errors)
			}
		})
	}
}
