package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkspaceSeedAndChannelAction(t *testing.T) {
	ts := testServer(t)
	res, err := http.Get(ts.URL + "/api/v1/workspaces")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var list struct {
		Data []struct {
			ID       string `json:"id"`
			Provider string `json:"provider"`
			Channels []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"channels"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 8 {
		t.Fatalf("workspaces %d", len(list.Data))
	}

	body, _ := json.Marshal(map[string]string{"name": "deploys"})
	res, err = http.Post(ts.URL+"/api/v1/workspaces/ws-slack/channels", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 201 {
		t.Fatalf("create %d %s", res.StatusCode, b)
	}
	var created struct {
		Data struct {
			ID   string `json:"id"`
			Path string `json:"path"`
			URL  string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &created); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(created.Data.Path, "/hooks/slack/") {
		t.Fatalf("path %s", created.Data.Path)
	}

	msg := `{"text":"approve this","blocks":[{"type":"actions","elements":[{"type":"button","action_id":"approve","text":{"type":"plain_text","text":"Approve"},"value":"yes"}]}]}`
	res, err = http.Post(ts.URL+created.Data.Path, "application/json", strings.NewReader(msg))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("post %d", res.StatusCode)
	}

	clicked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(clicked.Close)
	patch, _ := json.Marshal(map[string]string{"interactivityUrl": clicked.URL})
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/workspaces/ws-slack", strings.NewReader(string(patch)))
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	evRes, _ := http.Get(ts.URL + "/api/v1/events?sinkId=unused")
	evRes.Body.Close()
	evRes, _ = http.Get(ts.URL + "/api/v1/events")
	var events struct {
		Data []struct {
			ID     string `json:"id"`
			SinkID string `json:"sinkId"`
		} `json:"data"`
	}
	_ = json.NewDecoder(evRes.Body).Decode(&events)
	evRes.Body.Close()
	if len(events.Data) == 0 {
		t.Fatal("no events")
	}
	act, _ := json.Marshal(map[string]string{"eventId": events.Data[0].ID, "kind": "button", "actionId": "approve", "value": "yes", "text": "Approve"})
	res, err = http.Post(ts.URL+"/api/v1/workspaces/ws-slack/channels/"+created.Data.ID+"/actions", "application/json", strings.NewReader(string(act)))
	if err != nil {
		t.Fatal(err)
	}
	ab, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("action %d %s", res.StatusCode, ab)
	}
}

func TestPagerDutyAck(t *testing.T) {
	ts := testServer(t)
	body := `{"routing_key":"0123456789abcdef0123456789abcdef","event_action":"trigger","payload":{"summary":"disk","source":"host","severity":"error"}}`
	res, _ := http.Post(ts.URL+"/hooks/pagerduty/v2/enqueue", "application/json", strings.NewReader(body))
	res.Body.Close()
	evRes, _ := http.Get(ts.URL + "/api/v1/events?provider=pagerduty")
	var events struct {
		Data []struct {
			ID       string `json:"id"`
			GroupKey string `json:"groupKey"`
		} `json:"data"`
	}
	_ = json.NewDecoder(evRes.Body).Decode(&events)
	evRes.Body.Close()
	if len(events.Data) == 0 {
		t.Fatal("no pd event")
	}
	act, _ := json.Marshal(map[string]string{"eventId": events.Data[0].ID, "kind": "ack", "value": events.Data[0].GroupKey})
	res, err := http.Post(ts.URL+"/api/v1/workspaces/ws-pagerduty/channels/ch-pd-default/actions", "application/json", strings.NewReader(string(act)))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("ack %d", res.StatusCode)
	}
	evRes, _ = http.Get(ts.URL + "/api/v1/events?provider=pagerduty&contains=acknowledge")
	events.Data = nil
	_ = json.NewDecoder(evRes.Body).Decode(&events)
	evRes.Body.Close()
	if len(events.Data) == 0 {
		t.Fatal("expected ack event")
	}
}

func TestOpsgenieAck(t *testing.T) {
	ts := testServer(t)
	body := `{"message":"disk full","alias":"disk-api-2","priority":"P1"}`
	res, _ := http.Post(ts.URL+"/hooks/opsgenie/v2/alerts", "application/json", strings.NewReader(body))
	if res.StatusCode != 202 {
		t.Fatalf("create %d", res.StatusCode)
	}
	res.Body.Close()
	evRes, _ := http.Get(ts.URL + "/api/v1/events?provider=opsgenie")
	var events struct {
		Data []struct {
			ID       string `json:"id"`
			GroupKey string `json:"groupKey"`
		} `json:"data"`
	}
	_ = json.NewDecoder(evRes.Body).Decode(&events)
	evRes.Body.Close()
	if len(events.Data) == 0 {
		t.Fatal("no og event")
	}
	act, _ := json.Marshal(map[string]string{"eventId": events.Data[0].ID, "kind": "ack", "value": events.Data[0].GroupKey})
	res, err := http.Post(ts.URL+"/api/v1/workspaces/ws-opsgenie/channels/ch-og-default/actions", "application/json", strings.NewReader(string(act)))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("ack %d", res.StatusCode)
	}
	evRes, _ = http.Get(ts.URL + "/api/v1/events?provider=opsgenie")
	var listed struct {
		Data []struct {
			Summary string `json:"summary"`
		} `json:"data"`
	}
	_ = json.NewDecoder(evRes.Body).Decode(&listed)
	evRes.Body.Close()
	found := false
	for _, ev := range listed.Data {
		if strings.Contains(ev.Summary, "acknowledge") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected ack event")
	}
}
