package generic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestMatch(t *testing.T) {
	s := Sink{}
	ok := httptest.NewRequest(http.MethodPost, "/hooks/generic/default", nil)
	if !s.Match(ok) {
		t.Fatal("should match generic")
	}
	no := httptest.NewRequest(http.MethodPost, "/hooks/slack/x", nil)
	if s.Match(no) {
		t.Fatal("should not match slack")
	}
}

func TestRespond(t *testing.T) {
	s := Sink{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/hooks/generic/default", strings.NewReader(`{"hello":"world"}`))
	if err := s.Respond(w, req, []byte(`{"hello":"world"}`), store.Chaos{}); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || w.Body.String() != `{"ok":true}` {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
