package slackevents

import (
	"fmt"
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

func TestSign(t *testing.T) {
	ts := time.Unix(1531420618, 0)
	body := []byte("body")
	h := (Source{}).Sign(body, "secret", ts, "")
	base := fmt.Sprintf("v0:%d:%s", ts.Unix(), body)
	want := "v0=" + source.HMACSHA256Hex("secret", base)
	if h.Get("X-Slack-Signature") != want {
		t.Fatalf("got %s want %s", h.Get("X-Slack-Signature"), want)
	}
}
