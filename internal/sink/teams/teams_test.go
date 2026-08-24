package teams

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestMessageCard(t *testing.T) {
	s := Sink{}
	body := []byte(`{"@type":"MessageCard","@context":"http://schema.org/extensions","title":"Alert","text":"down"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/teams/incoming/webhookie", nil)
	if !s.Match(req) {
		t.Fatal("match")
	}
	if !s.Validate(req, body).Valid {
		t.Fatal("valid")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 200 || w.Body.String() != "1" {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestAdaptive(t *testing.T) {
	s := Sink{}
	body := []byte(`{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","body":[{"type":"TextBlock","text":"hi"}],"$schema":"http://adaptivecards.io/schemas/adaptive-card.json","version":"1.4"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/teams/workflow/webhookie", nil)
	if !s.Validate(req, body).Valid {
		t.Fatal("valid")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 200 || w.Body.String() != `{"statusCode":200}` {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestEmptyInvalid(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodPost, "/hooks/teams/workflow/webhookie", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, []byte(`{}`), store.Chaos{})
	if w.Code != 400 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestValidateTable(t *testing.T) {
	s := Sink{}
	req := httptest.NewRequest(http.MethodPost, "/hooks/teams/workflow/webhookie", nil)
	tests := []struct {
		name    string
		body    string
		valid   bool
		path    string
		message string
	}{
		{
			name:  "messagecard with actions extra fields",
			body:  `{"@type":"MessageCard","@context":"https://schema.org/extensions","themeColor":"FF0000","title":"Deploy","text":"failed","potentialAction":[{"@type":"HttpPOST","name":"Ack","target":"https://example.com"}]}`,
			valid: true,
		},
		{
			name:  "messagecard sections only",
			body:  `{"@type":"MessageCard","sections":[{"activityTitle":"host api-2","facts":[{"name":"env","value":"prod"}]}]}`,
			valid: true,
		},
		{
			name:    "messagecard empty",
			body:    `{"@type":"MessageCard"}`,
			path:    "/",
			message: "text, title, or sections",
		},
		{
			name:    "messagecard action missing name",
			body:    `{"@type":"MessageCard","text":"x","potentialAction":[{"@type":"OpenUri"}]}`,
			path:    "/potentialAction/0/name",
			message: "required",
		},
		{
			name:    "adaptive missing version",
			body:    `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","body":[{"type":"TextBlock","text":"hi"}]}}]}`,
			path:    "/attachments/0/content/version",
			message: "required",
		},
		{
			name:  "adaptive body optional per schema",
			body:  `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","version":"1.4","actions":[{"type":"Action.OpenUrl","title":"Open","url":"https://example.com"}]}}]}`,
			valid: true,
		},
		{
			name:  "msteams extra field allowed",
			body:  `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","version":"1.4","msteams":{"width":"Full"},"body":[{"type":"TextBlock","text":"hi","wrap":true}]}}]}`,
			valid: true,
		},
		{
			name:    "textblock missing text",
			body:    `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","version":"1.4","body":[{"type":"TextBlock"}]}}]}`,
			path:    "/attachments/0/content/body/0/text",
			message: "required",
		},
		{
			name:    "openurl missing url",
			body:    `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","version":"1.4","body":[{"type":"TextBlock","text":"x"}],"actions":[{"type":"Action.OpenUrl","title":"Go"}]}}]}`,
			path:    "/attachments/0/content/actions/0/url",
			message: "required",
		},
		{
			name:    "table requires 1.5",
			body:    `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","version":"1.0","body":[{"type":"Table","columns":[],"rows":[]}]}}]}`,
			path:    "/attachments/0/content/body/0/type",
			message: "1.5",
		},
		{
			name:    "unknown element type",
			body:    `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","version":"1.4","body":[{"type":"NotAThing"}]}}]}`,
			path:    "/attachments/0/content/body/0/type",
			message: "unknown",
		},
		{
			name:    "adaptive body item missing type",
			body:    `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.adaptive","content":{"type":"AdaptiveCard","version":"1.4","body":[{"text":"hi"}]}}]}`,
			path:    "/attachments/0/content/body/0/type",
			message: "required",
		},
		{
			name:    "wrong contentType",
			body:    `{"type":"message","attachments":[{"contentType":"application/vnd.microsoft.card.hero","content":{"type":"AdaptiveCard","version":"1.4","body":[]}}]}`,
			path:    "/attachments/0/contentType",
			message: "adaptive",
		},
		{
			name:    "empty envelope",
			body:    `{}`,
			path:    "/",
			message: "MessageCard or Adaptive",
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
