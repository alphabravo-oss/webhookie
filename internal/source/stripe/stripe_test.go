package stripe

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

func TestSign(t *testing.T) {
	ts := time.Unix(1600000000, 0)
	body := []byte(`{"id":"evt_1"}`)
	h := (Source{}).Sign(body, "whsec_test", ts, "")
	sig := h.Get("Stripe-Signature")
	want := fmt.Sprintf("t=%d,v1=%s", ts.Unix(), source.HMACSHA256Hex("whsec_test", fmt.Sprintf("%d.%s", ts.Unix(), body)))
	if sig != want {
		t.Fatalf("got %s want %s", sig, want)
	}
	if !strings.HasPrefix(sig, "t=") {
		t.Fatal("prefix")
	}
}
