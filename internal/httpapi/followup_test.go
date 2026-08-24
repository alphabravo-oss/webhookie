package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSlackResponseURLReplace(t *testing.T) {
	ts := testServer(t)
	msg := `{"text":"approve this","blocks":[{"type":"actions","elements":[{"type":"button","action_id":"approve","text":{"type":"plain_text","text":"Approve"},"value":"yes"}]}]}`
	res, err := http.Post(ts.URL+"/hooks/slack/services/T00000000/B00000000/webhookie", "application/json", strings.NewReader(msg))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("post %d", res.StatusCode)
	}
	ev := latestEvent(t, ts.URL, "slack")
	body := `{"replace_original":true,"text":"approved"}`
	res, err = http.Post(ts.URL+"/hooks/slack/response/"+ev.ID, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(b) != "ok" {
		t.Fatalf("response_url %d %q", res.StatusCode, b)
	}
	got := getEvent(t, ts.URL, ev.ID)
	if got.DisplayBody == "" || !strings.Contains(got.DisplayBody, "approved") {
		t.Fatalf("displayBody %q", got.DisplayBody)
	}
	if got.Body == `{"text":"approved"}` {
		t.Fatal("inbox body must stay the original packet")
	}
}

func TestSlackResponseURLDeleteAndUnknown(t *testing.T) {
	ts := testServer(t)
	res, _ := http.Post(ts.URL+"/hooks/slack/services/T00000000/B00000000/webhookie", "application/json", strings.NewReader(`{"text":"x"}`))
	res.Body.Close()
	ev := latestEvent(t, ts.URL, "slack")
	res, _ = http.Post(ts.URL+"/hooks/slack/response/"+ev.ID, "application/json", strings.NewReader(`{"delete_original":true}`))
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || string(b) != "ok" {
		t.Fatalf("delete %d %q", res.StatusCode, b)
	}
	got := getEvent(t, ts.URL, ev.ID)
	if !got.Deleted {
		t.Fatal("expected deleted")
	}
	res, _ = http.Post(ts.URL+"/hooks/slack/response/does-not-exist", "application/json", strings.NewReader(`{"text":"x"}`))
	res.Body.Close()
	if res.StatusCode != 404 {
		t.Fatalf("unknown %d", res.StatusCode)
	}
}

func TestDiscordWaitIDAndPatchDelete(t *testing.T) {
	ts := testServer(t)
	res, err := http.Post(ts.URL+"/hooks/discord/api/webhooks/0/webhookie?wait=true", "application/json", strings.NewReader(`{"content":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	var msg struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	_ = json.NewDecoder(res.Body).Decode(&msg)
	res.Body.Close()
	if res.StatusCode != 200 || msg.ID == "" || msg.ID == "0" || msg.Content != "hello" {
		t.Fatalf("%d %+v", res.StatusCode, msg)
	}
	ev := getEvent(t, ts.URL, msg.ID)
	if ev.ID != msg.ID {
		t.Fatalf("stored id %s want %s", ev.ID, msg.ID)
	}
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/hooks/discord/api/webhooks/0/webhookie/messages/"+msg.ID, strings.NewReader(`{"content":"edited"}`))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(b), `"id":"`+msg.ID+`"`) {
		t.Fatalf("patch %d %s", res.StatusCode, b)
	}
	got := getEvent(t, ts.URL, msg.ID)
	if !strings.Contains(got.DisplayBody, "edited") {
		t.Fatalf("display %q", got.DisplayBody)
	}
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/hooks/discord/api/webhooks/0/webhookie/messages/"+msg.ID, nil)
	res, _ = http.DefaultClient.Do(req)
	res.Body.Close()
	if res.StatusCode != 204 {
		t.Fatalf("delete %d", res.StatusCode)
	}
	got = getEvent(t, ts.URL, msg.ID)
	if !got.Deleted {
		t.Fatal("expected deleted")
	}
	req, _ = http.NewRequest(http.MethodPatch, ts.URL+"/hooks/discord/api/webhooks/0/webhookie/messages/missing", strings.NewReader(`{"content":"x"}`))
	res, _ = http.DefaultClient.Do(req)
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 404 || !strings.Contains(string(b), "10008") {
		t.Fatalf("unknown %d %s", res.StatusCode, b)
	}
}

func TestTelegramAnswerCallbackQuery(t *testing.T) {
	ts := testServer(t)
	res, _ := http.Post(ts.URL+"/hooks/telegram/bot/123456:AAWebhookie/sendMessage", "application/json", strings.NewReader(`{"chat_id":1,"text":"ok?","reply_markup":{"inline_keyboard":[[{"text":"Yes","callback_data":"yes"}]]}}`))
	res.Body.Close()
	ev := latestEvent(t, ts.URL, "telegram")
	act, _ := json.Marshal(map[string]string{"eventId": ev.ID, "kind": "button", "actionId": "yes", "value": "yes", "text": "Yes"})
	res, err := http.Post(ts.URL+"/api/v1/workspaces/ws-telegram/channels/ch-telegram-alerts/actions", "application/json", strings.NewReader(string(act)))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Data struct {
			ID      string `json:"id"`
			Payload string `json:"payload"`
		} `json:"data"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if res.StatusCode != 200 || out.Data.ID == "" {
		t.Fatalf("action %d %+v", res.StatusCode, out)
	}
	res, _ = http.Post(ts.URL+"/hooks/telegram/bot/123456:AAWebhookie/answerCallbackQuery", "application/json", strings.NewReader(`{"callback_query_id":"`+out.Data.ID+`","text":"ok"}`))
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(b), `"ok":true`) {
		t.Fatalf("answer %d %s", res.StatusCode, b)
	}
	res, _ = http.Post(ts.URL+"/hooks/telegram/bot/123456:AAWebhookie/answerCallbackQuery", "application/json", strings.NewReader(`{"callback_query_id":"nope"}`))
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 400 || !strings.Contains(string(b), "query is too old") {
		t.Fatalf("unknown %d %s", res.StatusCode, b)
	}
}

type eventDTO struct {
	ID          string `json:"id"`
	Body        string `json:"body"`
	DisplayBody string `json:"displayBody"`
	Deleted     bool   `json:"deleted"`
}

func latestEvent(t *testing.T, base, provider string) eventDTO {
	t.Helper()
	res, err := http.Get(base + "/api/v1/events?provider=" + provider)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var list struct {
		Data []eventDTO `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) == 0 {
		t.Fatal("no events")
	}
	return list.Data[0]
}

func getEvent(t *testing.T, base, id string) eventDTO {
	t.Helper()
	res, err := http.Get(base + "/api/v1/events/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("get %d", res.StatusCode)
	}
	var wrap struct {
		Data eventDTO `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&wrap); err != nil {
		t.Fatal(err)
	}
	return wrap.Data
}
