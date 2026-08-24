package source

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/alphabravo-oss/webhookie/internal/store"
	"github.com/google/uuid"
)

type Fixture struct {
	Name        string
	Description string
	Headers     map[string]string
	Body        []byte
	ContentType string
}

type Source interface {
	Provider() string
	Events() []Fixture
	Sign(body []byte, secret string, ts time.Time, id string) http.Header
}

type Registry struct{ items []Source }

func (r *Registry) Register(s Source) { r.items = append(r.items, s) }

func (r *Registry) Get(provider string) (Source, bool) {
	for _, s := range r.items {
		if s.Provider() == provider {
			return s, true
		}
	}
	return nil, false
}

func (r *Registry) All() []Source { return r.items }

func HMACSHA256Hex(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func HMACSHA256(secret []byte, payload string) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func NewID() string { return uuid.NewString() }

func Deliver(ctx context.Context, target string, hdr http.Header, body []byte, timeout time.Duration) store.SendAttempt {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		ms := int(time.Since(start).Milliseconds())
		return store.SendAttempt{Error: err.Error(), LatencyMS: ms}
	}
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	res, err := client.Do(req)
	ms := int(time.Since(start).Milliseconds())
	if err != nil {
		return store.SendAttempt{Error: err.Error(), LatencyMS: ms, RequestHeaders: hdr, Body: body}
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	st := res.StatusCode
	return store.SendAttempt{Status: &st, LatencyMS: ms, RequestHeaders: hdr, Body: body}
}

func CloneHeader(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, vs := range h {
		out[k] = append([]string(nil), vs...)
	}
	return out
}
