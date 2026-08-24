package googlechat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func gchatReq() *http.Request {
	return httptest.NewRequest(http.MethodPost, "/hooks/googlechat/v1/spaces/AAAAwebhookie/messages", nil)
}

func TestText(t *testing.T) {
	s := Sink{}
	body := []byte(`{"text":"Hello from Chat"}`)
	req := gchatReq()
	if !s.Match(req) {
		t.Fatal("match")
	}
	if !s.Validate(req, body).Valid {
		t.Fatal("valid")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if !strings.Contains(fmtString(out["name"]), "spaces/AAAAwebhookie/messages/") {
		t.Fatalf("name %v", out["name"])
	}
	if s.Summarize(req, body).Text != "Hello from Chat" {
		t.Fatal(s.Summarize(req, body).Text)
	}
}

func TestEmpty(t *testing.T) {
	s := Sink{}
	req := gchatReq()
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, []byte(`{}`), store.Chaos{})
	if w.Code != 400 {
		t.Fatalf("%d", w.Code)
	}
}

func TestValidateTable(t *testing.T) {
	s := Sink{}
	req := gchatReq()
	tests := []struct {
		name    string
		body    string
		valid   bool
		path    string
		message string
	}{
		{
			name:  "text plus extra",
			body:  `{"text":"hello","thread":{"name":"spaces/AAA/threads/t"}}`,
			valid: true,
		},
		{
			name:  "cardsV2 with header and widgets",
			body:  `{"cardsV2":[{"cardId":"alert","card":{"header":{"title":"Deploy"},"sections":[{"widgets":[{"textParagraph":{"text":"failed"}},{"buttonList":{"buttons":[{"text":"Ack","onClick":{"action":{"function":"ack"}}}]}}]}]}}]}`,
			valid: true,
		},
		{
			name:  "legacy cards",
			body:  `{"cards":[{"header":{"title":"old"}}]}`,
			valid: true,
		},
		{
			name:    "empty",
			body:    `{}`,
			path:    "/",
			message: "text, cards, or cardsV2",
		},
		{
			name:    "text too long",
			body:    `{"text":"` + strings.Repeat("a", maxText+1) + `"}`,
			path:    "/text",
			message: "4096",
		},
		{
			name:    "cardsV2 not array",
			body:    `{"cardsV2":{"card":{}}}`,
			path:    "/cardsV2",
			message: "array",
		},
		{
			name:    "cardsV2 missing card",
			body:    `{"cardsV2":[{"cardId":"x"}]}`,
			path:    "/cardsV2/0/card",
			message: "object",
		},
		{
			name:    "sections not array",
			body:    `{"cardsV2":[{"card":{"sections":{"widgets":[]}}}]}`,
			path:    "/cardsV2/0/card/sections",
			message: "array",
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
		})
	}
}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}
