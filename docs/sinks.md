# Sinks

A sink is an HTTP path that pretends to be a provider. `POST` to it from your app. Capture is `HandleFunc("/hooks/*")` — unknown shapes 404.

Validation follows each provider's **documented public API**: required fields, known enums, child-object shape, published limits, Discord `multipart/form-data` files, Telegram HTML/Markdown/MarkdownV2, and Adaptive Card element types. Extra fields are allowed. It is not Slack's private validator, Adaptive Card `additionalProperties: false`, or a byte-clone of Telegram's C++ parser.

HTTP bodies stay provider-shaped. JSON-pointer errors (`{path, message}`) are stored on the event (`valid`, `validationErrors`) and shown in Inbox. Slack’s 400 is only `{"ok":false,"error":"invalid_payload"}`. Discord empty is `50006`; field errors are `50035` with an `errors` tree. PagerDuty 400 includes `errors[]`. Opsgenie 422 includes an `errors` map. Telegram `description` is `Bad Request: …`.

Chaos on a sink (`PATCH /api/v1/sinks/{id}`) overrides status/body/delay/hang **instead of** the provider envelope.

Default tokens below are created on first boot (`Seed`). Creating a channel in a destination UI mints a unique path; use that URL in the app.

Base: `http://localhost:8080` unless `WEBHOOKIE_PUBLIC_BASE_URL` says otherwise.

## Generic

`POST /hooks/generic/default` (or `/hooks/generic/{token}`)

Accepts any method body. Response `200` `{"ok":true}`.

## Slack incoming webhooks

`POST /hooks/slack/services/{team}/{bot}/{token}`

Default: `/hooks/slack/services/T00000000/B00000000/webhookie`

JSON or `application/x-www-form-urlencoded` with `payload=`. Needs at least one of `text`, `blocks`, `attachments`.

Also checked (documented Block Kit / incoming-webhook rules):

- `text` max 40,000 characters
- `blocks` max 50; each item needs a known `type` (`section`, `divider`, `image`, `actions`, `context`, `header`, `input`, `file`, `rich_text`, `video`, `markdown`, `table`)
- `section` needs `text`, `fields`, or `accessory`; section text max 3,000; `fields` max 10
- `actions.elements` required (max 25); `button` needs `plain_text` (style `primary`|`danger` if set; `url` max 3,000; `value` max 2,000); `overflow.options` 1–5; `static_select` needs `options` or `option_groups`
- header is `plain_text` max 150; image needs `alt_text` and `image_url` or `slack_file`; `video` needs `video_url`, `thumbnail_url`, `alt_text`, `title`; `file` needs `external_id` and `source`; `rich_text` needs typed section/list/quote elements
- `attachments` max 100; extra fields (`username`, `icon_emoji`, `channel`, `block_id`, …) allowed
- Not Slack’s private incoming-webhook 400s and not mrkdwn parsing

Success: `200` `text/plain` body `ok`.  
Invalid: `400` `{"ok":false,"error":"invalid_payload"}` (details only on the stored event).

## Microsoft Teams

`POST /hooks/teams/workflow/{token}` or `/hooks/teams/incoming/{token}`

Default: `/hooks/teams/workflow/webhookie`

Accepts Office 365 Connector `{"@type":"MessageCard",...}` or Adaptive Card (`type: AdaptiveCard` or `attachments[].contentType = application/vnd.microsoft.card.adaptive`).

Also checked:

- MessageCard needs `text`, `title`, or `sections`; `potentialAction[]` items need `@type` and `name` when present
- Adaptive envelope needs `content.type = AdaptiveCard` and `version` (body is optional per schema)
- Element types and required properties from [adaptivecards.io/explorer](https://adaptivecards.io/explorer/) (`TextBlock.text`, `Image.url`, `Action.OpenUrl.url`, `Container.items`, inputs need `id`, …)
- Version gates: `Media`/`Action.ToggleVisibility` 1.1, `ActionSet`/`RichTextBlock` 1.2, `Action.Execute` 1.4, `Table` 1.5
- Extra fields allowed (`msteams`, templating `${}`, …). Not the published schema’s `additionalProperties: false`, not Adaptive Card templating evaluation, not host fallback rendering

Success: MessageCard `200` body `1`; Adaptive `200` `{"statusCode":200}`.  
Invalid: `400` `{"error":"invalid card"}` (details on the stored event).

## Discord incoming webhooks

`POST /hooks/discord/api/webhooks/{id}/{token}`

Default: `/hooks/discord/api/webhooks/0/webhookie`

Needs one of `content`, `embeds`, `components`, `files[n]`, or `poll`. Query `wait=true` returns `200` `{"id":"0","content":"ok"}`; otherwise `204` empty.

Also checked:

- `content` max 2,000; `username` max 80
- `embeds` max 10; title 256, description 4,096, fields 25 (name 256, value 1,024), footer 2,048, author 256; combined embed text 6,000
- `components` max 5 action rows × 5 children; button `style` 1–5, `label` 80, `custom_id` 100; link buttons (`style` 5) need `url`
- `multipart/form-data` with `payload_json` and `files[n]` (max 10). A file part is enough for a non-empty message (`attachment://` URLs in embeds are stored, not rewritten)
- `poll.question.text` required when `poll` is present

Empty payload: `400` `{"message":"Cannot send an empty message","code":50006}`.  
Field errors: `400` `{"message":"Invalid Form Body","code":50035,"errors":{...}}`.

## PagerDuty Events API v2

`POST /hooks/pagerduty/v2/enqueue`  
also `/hooks/pagerduty/v2/enqueue/{token}` and `/hooks/pagerduty/v2/change`

Default routing key (32 chars): `0123456789abcdef0123456789abcdef`

Create:

```json
{
  "routing_key": "0123456789abcdef0123456789abcdef",
  "event_action": "trigger",
  "payload": { "summary": "disk", "source": "api-2", "severity": "error" }
}
```

`event_action`: `trigger` | `acknowledge` | `resolve`. Trigger needs `payload.summary` (max 1,024), `source`, `severity` in `info|warning|error|critical`. Ack/resolve need `dedup_key` (max 255). `routing_key` must be 32 characters. `images`/`links` max 8. Extra fields (`custom_details`, `client`, `timestamp`, …) allowed.

Success: `202` `{"status":"success","message":"Event processed","dedup_key":"..."}`. Header `X-Webhookie-Dedup-Key` is set. Events with the same `dedup_key` group in the PagerDuty destination.

Invalid: `400` `{"status":"invalid event","message":"Event object is invalid","errors":["/payload/summary required", …]}`.

Lookup: body `routing_key` maps to a sink token when present.

## Telegram Bot API (`sendMessage` only)

`POST /hooks/telegram/bot/{token}/sendMessage`

Default: `/hooks/telegram/bot/123456:AAWebhookie/sendMessage`

Required JSON: `chat_id` (number or string) and `text` (1–4,096 characters). `parse_mode` if set: `Markdown` | `MarkdownV2` | `HTML`, validated against [Bot API formatting options](https://core.telegram.org/bots/api#formatting-options) (unmatched HTML tags, unsupported tags, unescaped MarkdownV2 reserved chars). `entities[]` may be used instead (`type`, `offset`, `length` in UTF-16 code units). Inline buttons need `text` and an action (`callback_data` 1–64 bytes, `url`, …). Extra fields (`disable_notification`, …) allowed.

Success: `200` `{"ok":true,"result":{message_id,from,chat,date,text,...}}`.  
Invalid: `400` `{"ok":false,"error_code":400,"description":"Bad Request: ..."}`.

No other Bot API methods (`getUpdates`, `sendPhoto`, `answerCallbackQuery`, …).

## Google Chat incoming webhooks

`POST /hooks/googlechat/v1/spaces/{space}/messages`

Default: `/hooks/googlechat/v1/spaces/AAAAwebhookie/messages`

Real Chat webhook URLs also have `?key=&token=`; webhookie matches **path only**. Needs `text` (max 4,096), `cards`, or `cardsV2`. `cardsV2[]` items need a `card` object; `sections`/`widgets` if present must be arrays of objects.

Success: `200` Message-shaped JSON (`name`, `text`, `createTime`, `thread.name`).  
Invalid: `400` `{"error":{"code":400,"message":"…","status":"INVALID_ARGUMENT"}}`. Empty payload uses `text or cards required`; other failures use the first pointer message.

Real Google incoming webhooks are one-way. Destination clicks still fire a synthetic `CARD_CLICKED` if you set an Interactivity URL.

## Mattermost incoming webhooks

`POST /hooks/mattermost/hooks/{token}`

Default: `/hooks/mattermost/hooks/mattermost-webhookie`

JSON `{text, channel, username, attachments, ...}` or Slack-compatible `payload=`. Needs `text` (max 16,383) or `attachments`. Attachment `actions[]` need `name` when present. Extra fields (`icon_url`, `props`, …) allowed.

Success: `200` `text/plain` `ok`.  
Invalid: `400` `Unable to parse incoming data`.

## Opsgenie Alert API v2

`POST /hooks/opsgenie/v2/alerts`  
`POST /hooks/opsgenie/v2/alerts/{id}/acknowledge`  
`POST /hooks/opsgenie/v2/alerts/{id}/close`

Default: `/hooks/opsgenie/v2/alerts`  
Default GenieKey (optional, used to pick a team sink): `eb243592-faa2-4ba2-a551-webhookie01`

Create requires `message` (max 130). Optional `alias` (512), `description` (15,000), `priority` (`P1`–`P5`), `tags` (max 20 × 50 characters). Extra fields (`entity`, `source`, `details`, …) allowed. Invalid create includes an `errors` map in the 422 body.

Success: `202` `{"result":"Request will be processed","took":0.01,"requestId":"..."}`.  
Invalid create: `422`.

`Authorization: GenieKey {token}` selects the sink when the path is shared.

## Listing sinks

`GET /api/v1/sinks` returns each sink with `url` = `publicBaseUrl + path`. `POST /api/v1/sinks` creates a **generic** bin only; provider channels are created from destination UIs (`POST /api/v1/workspaces/{id}/channels`).
