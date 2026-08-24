package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

func discReq() *http.Request {
	return httptest.NewRequest(http.MethodPost, "/hooks/discord/api/webhooks/0/webhookie", nil)
}

func Test204(t *testing.T) {
	s := Sink{}
	body := []byte(`{"content":"hi"}`)
	req := discReq()
	if !s.Match(req) {
		t.Fatal("match")
	}
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 204 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestWaitTrue(t *testing.T) {
	s := Sink{}
	body := []byte(`{"content":"hi"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/discord/api/webhooks/0/webhookie?wait=true", nil)
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"id"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestEmptyInvalid(t *testing.T) {
	s := Sink{}
	body := []byte(`{}`)
	req := discReq()
	w := httptest.NewRecorder()
	_ = s.Respond(w, req, body, store.Chaos{})
	if w.Code != 400 {
		t.Fatalf("code %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"code":50006`) {
		t.Fatalf("empty should be 50006, got %s", w.Body.String())
	}
}

func TestValidateTable(t *testing.T) {
	s := Sink{}
	req := discReq()
	longContent := strings.Repeat("a", maxContent+1)
	longTitle := strings.Repeat("t", maxEmbedTitle+1)
	tests := []struct {
		name    string
		body    string
		valid   bool
		path    string
		message string
		code    int
	}{
		{
			name:  "content plus extras",
			body:  `{"content":"deploy failed","username":"ci","avatar_url":"https://example.com/a.png","tts":false}`,
			valid: true,
		},
		{
			name:  "embed only",
			body:  `{"embeds":[{"title":"Alert","description":"disk 92%","color":15158332,"fields":[{"name":"host","value":"api-2","inline":true}]}]}`,
			valid: true,
		},
		{
			name:  "buttons",
			body:  `{"content":"ship it?","components":[{"type":1,"components":[{"type":2,"style":1,"label":"Approve","custom_id":"ok"},{"type":2,"style":5,"label":"Docs","url":"https://example.com"}]}]}`,
			valid: true,
		},
		{
			name:    "content too long",
			body:    `{"content":"` + longContent + `"}`,
			path:    "/content",
			message: "2000",
			code:    50035,
		},
		{
			name:    "too many embeds",
			body:    `{"embeds":[{},{},{},{},{},{},{},{},{},{},{"title":"11"}]}`,
			path:    "/embeds",
			message: "10",
			code:    50035,
		},
		{
			name:    "embed title too long",
			body:    `{"embeds":[{"title":"` + longTitle + `"}]}`,
			path:    "/embeds/0/title",
			message: "256",
			code:    50035,
		},
		{
			name:    "field missing value",
			body:    `{"embeds":[{"fields":[{"name":"host"}]}]}`,
			path:    "/embeds/0/fields/0/value",
			message: "required",
			code:    50035,
		},
		{
			name:    "too many action rows",
			body:    `{"content":"x","components":[{"type":1,"components":[{"type":2,"style":1,"label":"a","custom_id":"a"}]},{"type":1,"components":[{"type":2,"style":1,"label":"b","custom_id":"b"}]},{"type":1,"components":[{"type":2,"style":1,"label":"c","custom_id":"c"}]},{"type":1,"components":[{"type":2,"style":1,"label":"d","custom_id":"d"}]},{"type":1,"components":[{"type":2,"style":1,"label":"e","custom_id":"e"}]},{"type":1,"components":[{"type":2,"style":1,"label":"f","custom_id":"f"}]}]}`,
			path:    "/components",
			message: "5",
			code:    50035,
		},
		{
			name:    "button style out of range",
			body:    `{"content":"x","components":[{"type":1,"components":[{"type":2,"style":9,"label":"x","custom_id":"x"}]}]}`,
			path:    "/components/0/components/0/style",
			message: "1-5",
			code:    50035,
		},
		{
			name:    "custom_id too long",
			body:    `{"content":"x","components":[{"type":1,"components":[{"type":2,"style":1,"label":"x","custom_id":"` + strings.Repeat("c", maxCustomID+1) + `"}]}]}`,
			path:    "/components/0/components/0/custom_id",
			message: "100",
			code:    50035,
		},
		{
			name:    "link button missing url",
			body:    `{"content":"x","components":[{"type":1,"components":[{"type":2,"style":5,"label":"link"}]}]}`,
			path:    "/components/0/components/0/url",
			message: "required",
			code:    50035,
		},
		{
			name:    "attachments not array",
			body:    `{"attachments":{"filename":"a.png"}}`,
			path:    "/attachments",
			message: "array",
			code:    50035,
		},
		{
			name:    "empty",
			body:    `{}`,
			path:    "/",
			message: "content, embeds, components, files, or poll",
			code:    50006,
		},
		{
			name:  "poll is enough",
			body:  `{"poll":{"question":{"text":"ship it?"},"answers":[{"poll_media":{"text":"yes"}}]}}`,
			valid: true,
		},
		{
			name:    "poll missing question text",
			body:    `{"poll":{"question":{}}}`,
			path:    "/poll/question/text",
			message: "required",
			code:    50035,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := s.Validate(req, []byte(tc.body))
			if tc.valid {
				if !v.Valid {
					t.Fatalf("%+v", v)
				}
				return
			}
			if v.Valid || !v.Has(tc.path, tc.message) {
				t.Fatalf("want %s %q got %+v", tc.path, tc.message, v.Errors)
			}
			w := httptest.NewRecorder()
			_ = s.Respond(w, req, []byte(tc.body), store.Chaos{})
			if w.Code != 400 {
				t.Fatalf("code %d", w.Code)
			}
			var out map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			code, _ := out["code"].(float64)
			if int(code) != tc.code {
				t.Fatalf("discord code %v want %d body %s", out["code"], tc.code, w.Body.String())
			}
		})
	}
}

func TestMultipartFiles(t *testing.T) {
	s := Sink{}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("payload_json", `{"content":"see file"}`); err != nil {
		t.Fatal(err)
	}
	part, err := w.CreateFormFile("files[0]", "note.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(part, "hello")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/hooks/discord/api/webhooks/0/webhookie", nil)
	req.Header.Set("Content-Type", w.FormDataContentType())
	v := s.Validate(req, buf.Bytes())
	if !v.Valid {
		t.Fatalf("%+v", v)
	}

	var onlyFile bytes.Buffer
	w2 := multipart.NewWriter(&onlyFile)
	p2, _ := w2.CreateFormFile("files[0]", "a.png")
	_, _ = io.WriteString(p2, "PNG")
	_ = w2.Close()
	req2 := httptest.NewRequest(http.MethodPost, "/hooks/discord/api/webhooks/0/webhookie", nil)
	req2.Header.Set("Content-Type", w2.FormDataContentType())
	if !s.Validate(req2, onlyFile.Bytes()).Valid {
		t.Fatal("file-only webhook must be valid")
	}
	rec := httptest.NewRecorder()
	_ = s.Respond(rec, req2, onlyFile.Bytes(), store.Chaos{})
	if rec.Code != 204 {
		t.Fatalf("code %d", rec.Code)
	}
}
