package chaos

import (
	"context"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Override struct {
	Status      int
	Body        []byte
	ContentType string
	RetryAfter  string
}

func Apply(ctx context.Context, ch store.Chaos) (*Override, error) {
	if ch.DelayMS > 0 {
		select {
		case <-time.After(time.Duration(ch.DelayMS) * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if ch.Hang {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if ch.Status == 0 {
		return nil, nil
	}
	ov := &Override{Status: ch.Status, Body: []byte(ch.Body), ContentType: ch.ContentType}
	if ov.ContentType == "" {
		ov.ContentType = "text/plain"
	}
	if ch.Status == 429 && ch.Body == "" {
		ov.Body = []byte("rate_limited")
		ov.RetryAfter = "1"
	}
	return ov, nil
}
