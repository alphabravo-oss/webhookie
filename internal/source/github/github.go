package github

import (
	"net/http"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
	"github.com/google/uuid"
)

type Source struct{}

func (Source) Provider() string { return "github" }

func (Source) Events() []source.Fixture {
	return []source.Fixture{
		{Name: "ping", Description: "GitHub ping", ContentType: "application/json", Headers: map[string]string{"X-GitHub-Event": "ping"}, Body: []byte(`{"zen":"Keep it logically awesome.","hook_id":1}`)},
		{Name: "push", Description: "GitHub push", ContentType: "application/json", Headers: map[string]string{"X-GitHub-Event": "push"}, Body: []byte(`{"ref":"refs/heads/main","commits":[{"id":"abc","message":"wip"}],"repository":{"full_name":"acme/app"}}`)},
		{Name: "pull_request", Description: "GitHub PR opened", ContentType: "application/json", Headers: map[string]string{"X-GitHub-Event": "pull_request"}, Body: []byte(`{"action":"opened","number":1,"pull_request":{"title":"Add webhookie","html_url":"https://github.com/acme/app/pull/1"}}`)},
	}
}

func (Source) Sign(body []byte, secret string, _ time.Time, id string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	if id == "" {
		id = uuid.NewString()
	}
	h.Set("X-GitHub-Delivery", id)
	h.Set("X-Hub-Signature-256", "sha256="+source.HMACSHA256Hex(secret, string(body)))
	return h
}
