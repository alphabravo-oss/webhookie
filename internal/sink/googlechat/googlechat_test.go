package googlechat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestText(t *testing.T) {
	s := Sink{}
	body := []byte(`{"text":"Hello from Chat"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/googlechat/v1/spaces/AAAAwebhookie/messages", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/hooks/googlechat/v1/spaces/AAAAwebhookie/messages", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, []byte(`{}`), store.Chaos{})
	if w.Code != 400 {
		t.Fatalf("%d", w.Code)
	}
}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}
