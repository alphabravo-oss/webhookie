package stripe

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/source"
)

type Source struct{}

func (Source) Provider() string { return "stripe" }

func (Source) Events() []source.Fixture {
	return []source.Fixture{
		{Name: "invoice.paid", Description: "Stripe invoice.paid", ContentType: "application/json", Body: []byte(`{"id":"evt_1","object":"event","type":"invoice.paid","data":{"object":{"id":"in_1","paid":true}}}`)},
		{Name: "customer.subscription.updated", Description: "Stripe subscription updated", ContentType: "application/json", Body: []byte(`{"id":"evt_2","object":"event","type":"customer.subscription.updated","data":{"object":{"id":"sub_1","status":"active"}}}`)},
		{Name: "checkout.session.completed", Description: "Stripe checkout completed", ContentType: "application/json", Body: []byte(`{"id":"evt_3","object":"event","type":"checkout.session.completed","data":{"object":{"id":"cs_1","status":"complete"}}}`)},
	}
}

func (Source) Sign(body []byte, secret string, ts time.Time, _ string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	signed := fmt.Sprintf("%d.%s", ts.Unix(), body)
	h.Set("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s", ts.Unix(), source.HMACSHA256Hex(secret, signed)))
	return h
}
