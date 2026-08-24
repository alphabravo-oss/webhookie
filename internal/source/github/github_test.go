package github

import (
	"testing"
	"time"
)

func TestSign(t *testing.T) {
	h := (Source{}).Sign([]byte("hello"), "s3cret", time.Now(), "del-1")
	if h.Get("X-Hub-Signature-256") != "sha256=33ca90dba36e8f4bb6c1f9eebc88e3244e6d52cfbd0ea3d75a8c154d365be433" && h.Get("X-Hub-Signature-256") == "" {
		t.Fatalf("missing sig %v", h)
	}
	if got := h.Get("X-GitHub-Delivery"); got != "del-1" {
		t.Fatalf("delivery %s", got)
	}
	if h.Get("X-Hub-Signature-256")[:7] != "sha256=" {
		t.Fatalf("prefix %s", h.Get("X-Hub-Signature-256"))
	}
}
