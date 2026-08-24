package pagerduty

import (
	"net/http"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

// Source emits PagerDuty webhooks v3 (outbound from PD to your app).
// Signature: HMAC-SHA256 hex of the raw body, header X-PagerDuty-Signature: v1=<hex>.
type Source struct{}

func (Source) Provider() string { return "pagerduty-v3" }

func (Source) Events() []source.Fixture {
	return []source.Fixture{
		{Name: "incident.triggered", Description: "PD v3 incident.triggered", ContentType: "application/json", Body: []byte(`{"event":{"event_type":"incident.triggered","resource_type":"incident","data":{"id":"PT1","type":"incident","status":"triggered","title":"disk"}}}`)},
		{Name: "incident.acknowledged", Description: "PD v3 incident.acknowledged", ContentType: "application/json", Body: []byte(`{"event":{"event_type":"incident.acknowledged","resource_type":"incident","data":{"id":"PT1","type":"incident","status":"acknowledged"}}}`)},
		{Name: "incident.resolved", Description: "PD v3 incident.resolved", ContentType: "application/json", Body: []byte(`{"event":{"event_type":"incident.resolved","resource_type":"incident","data":{"id":"PT1","type":"incident","status":"resolved"}}}`)},
	}
}

func (Source) Sign(body []byte, secret string, _ time.Time, _ string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-PagerDuty-Signature", "v1="+source.HMACSHA256Hex(secret, string(body)))
	return h
}
