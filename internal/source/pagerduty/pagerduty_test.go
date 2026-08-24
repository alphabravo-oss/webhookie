package pagerduty

import (
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

func TestSign(t *testing.T) {
	body := []byte(`{"event":{}}`)
	h := (Source{}).Sign(body, "secret", time.Now(), "")
	want := "v1=" + source.HMACSHA256Hex("secret", string(body))
	if h.Get("X-PagerDuty-Signature") != want {
		t.Fatalf("got %s want %s", h.Get("X-PagerDuty-Signature"), want)
	}
}
