package slackevents

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

type Source struct{}

func (Source) Provider() string { return "slack-events" }

func (Source) Events() []source.Fixture {
	return []source.Fixture{
		{Name: "url_verification", Description: "Slack url_verification challenge", ContentType: "application/json", Body: []byte(`{"token":"token","challenge":"challenge-value","type":"url_verification"}`)},
		{Name: "app_mention", Description: "Slack event_callback app_mention", ContentType: "application/json", Body: []byte(`{"token":"token","type":"event_callback","event":{"type":"app_mention","text":"<@U> hello","user":"U1","channel":"C1"}}`)},
	}
}

func (Source) Sign(body []byte, secret string, ts time.Time, _ string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-Slack-Request-Timestamp", fmt.Sprintf("%d", ts.Unix()))
	base := fmt.Sprintf("v0:%d:%s", ts.Unix(), body)
	h.Set("X-Slack-Signature", "v0="+source.HMACSHA256Hex(secret, base))
	return h
}
