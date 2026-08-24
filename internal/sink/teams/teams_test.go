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
