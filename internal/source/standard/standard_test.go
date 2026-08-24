package standard

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

func TestSignVector(t *testing.T) {
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("secret"))
	body := []byte(`{"a":1}`)
	ts := time.Unix(1614264732, 0)
	id := "msg_test"
	h := (Source{}).Sign(body, secret, ts, id)
	toSign := fmt.Sprintf("%s.%d.%s", id, ts.Unix(), body)
	want := base64.StdEncoding.EncodeToString(source.HMACSHA256([]byte("secret"), toSign))
	if h.Get("webhook-signature") != "v1,"+want {
		t.Fatalf("got %s want v1,%s", h.Get("webhook-signature"), want)
	}
	if h.Get("webhook-id") != id {
		t.Fatal("id")
	}
}
