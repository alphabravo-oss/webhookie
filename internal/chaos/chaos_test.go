package chaos

import (
	"context"
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func TestDelay(t *testing.T) {
	start := time.Now()
	_, err := Apply(context.Background(), store.Chaos{DelayMS: 50})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Fatal("delay too short")
	}
}

func TestHangCancels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := Apply(ctx, store.Chaos{Hang: true})
	if err == nil {
		t.Fatal("expected cancel")
	}
}

func TestStatus429(t *testing.T) {
	ov, err := Apply(context.Background(), store.Chaos{Status: 429})
	if err != nil {
		t.Fatal(err)
	}
	if ov == nil || ov.Status != 429 || string(ov.Body) != "rate_limited" || ov.RetryAfter != "1" {
		t.Fatalf("%+v", ov)
	}
}
