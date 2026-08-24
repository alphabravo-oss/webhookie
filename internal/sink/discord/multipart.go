package discord

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
)

const maxFiles = 10

type decoded struct {
	m     map[string]any
	files int
}

func decode(r *http.Request, body []byte) (decoded, error) {
	ct := r.Header.Get("Content-Type")
	media, params, err := mime.ParseMediaType(ct)
	if err == nil && media == "multipart/form-data" {
		return decodeMultipart(body, params["boundary"])
	}
	m, err := parse(body)
	return decoded{m: m}, err
}

func decodeMultipart(body []byte, boundary string) (decoded, error) {
	out := decoded{m: map[string]any{}}
	if boundary == "" {
		return out, nil
	}
	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return decoded{}, err
		}
		slurp, err := io.ReadAll(part)
		if err != nil {
			return decoded{}, err
		}
		name := part.FormName()
		switch {
		case name == "payload_json":
			if err := json.Unmarshal(slurp, &out.m); err != nil {
				return decoded{}, err
			}
		case strings.HasPrefix(name, "files[") || part.FileName() != "":
			out.files++
		case name != "":
			trim := bytes.TrimSpace(slurp)
			if len(trim) > 0 && (trim[0] == '{' || trim[0] == '[') {
				var v any
				if json.Unmarshal(slurp, &v) == nil {
					out.m[name] = v
					continue
				}
			}
			out.m[name] = string(slurp)
		}
		_ = part.Close()
	}
	return out, nil
}
