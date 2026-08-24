package standard

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

type Source struct{}

func (Source) Provider() string { return "standard" }

func (Source) Events() []source.Fixture {
	return []source.Fixture{{
		Name:        "generic.ping",
		Description: "Standard Webhooks ping",
		ContentType: "application/json",
		Body:        []byte(`{"type":"ping","data":{"ok":true}}`),
	}}
}

func (Source) Sign(body []byte, secret string, ts time.Time, id string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("webhook-id", id)
	h.Set("webhook-timestamp", fmt.Sprintf("%d", ts.Unix()))
	raw := strings.TrimPrefix(secret, "whsec_")
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		key = []byte(secret)
	}
	toSign := fmt.Sprintf("%s.%d.%s", id, ts.Unix(), body)
	sig := base64.StdEncoding.EncodeToString(source.HMACSHA256(key, toSign))
	h.Set("webhook-signature", "v1,"+sig)
	return h
}
