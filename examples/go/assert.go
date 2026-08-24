package webhookie

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Event struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Body     string `json:"body"`
	Summary  string `json:"summary"`
	Status   int    `json:"status"`
	Valid    bool   `json:"valid"`
}

func Reset(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, baseURL+"/api/v1/events", nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("reset: %s", res.Status)
	}
	return nil
}

func WaitFor(ctx context.Context, baseURL, provider, contains string) (Event, error) {
	q := url.Values{}
	if provider != "" {
		q.Set("provider", provider)
	}
	if contains != "" {
		q.Set("contains", contains)
	}
	endpoint := baseURL + "/api/v1/events?" + q.Encode()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return Event{}, err
		}
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			var out struct {
				Data []Event `json:"data"`
			}
			b, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if json.Unmarshal(b, &out) == nil && len(out.Data) > 0 {
				return out.Data[0], nil
			}
		}
		select {
		case <-ctx.Done():
			return Event{}, ctx.Err()
		case <-tick.C:
		}
	}
}
