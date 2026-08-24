# Sinks

A sink is an HTTP path that pretends to be a provider. `POST` to it from your app. Capture is `HandleFunc("/hooks/*")` — unknown shapes 404.

Validation is **shallow** (required fields / JSON). It is not a full Block Kit, Adaptive Card, or Discord component schema. Invalid-but-structured payloads often still `200`.

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

Success: `200` `text/plain` body `ok`.  
Invalid: `400` `{"ok":false,"error":"invalid_payload"}`.

## Microsoft Teams

`POST /hooks/teams/workflow/{token}` or `/hooks/teams/incoming/{token}`

Default: `/hooks/teams/workflow/webhookie`

Accepts Office 365 Connector `{"@type":"MessageCard",...}` or Adaptive Card (`type: AdaptiveCard` or `attachments[].contentType = application/vnd.microsoft.card.adaptive`).

Success: MessageCard `200` body `1`; Adaptive `200` `{"statusCode":200}`.  
Invalid: `400` JSON error.

## Discord incoming webhooks

`POST /hooks/discord/api/webhooks/{id}/{token}`

Default: `/hooks/discord/api/webhooks/0/webhookie`

Needs `content` or `embeds`. Query `wait=true` returns `200` `{"id":"0","content":"ok"}`; otherwise `204` empty.

Invalid: `400` `{"message":"Cannot send an empty message","code":50006}`.

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

`event_action`: `trigger` | `acknowledge` | `resolve`. Trigger needs `payload.summary`, `source`, `severity` in `info|warning|error|critical`. Ack/resolve need `dedup_key`. `routing_key` must be 32 characters.

Success: `202` `{"status":"success","message":"Event processed","dedup_key":"..."}`. Header `X-Webhookie-Dedup-Key` is set. Events with the same `dedup_key` group in the PagerDuty destination.

Lookup: body `routing_key` maps to a sink token when present.

## Telegram Bot API (`sendMessage` only)

`POST /hooks/telegram/bot/{token}/sendMessage`

Default: `/hooks/telegram/bot/123456:AAWebhookie/sendMessage`

Required JSON: `chat_id` (number or string) and `text`. Optional `reply_markup.inline_keyboard` for destination buttons.

Success: `200` `{"ok":true,"result":{message_id,from,chat,date,text,...}}`.  
Invalid: `400` `{"ok":false,"error_code":400,"description":"Bad Request: ..."}`.

No other Bot API methods (`getUpdates`, `sendPhoto`, `answerCallbackQuery`, …).

## Google Chat incoming webhooks

`POST /hooks/googlechat/v1/spaces/{space}/messages`

Default: `/hooks/googlechat/v1/spaces/AAAAwebhookie/messages`

Real Chat webhook URLs also have `?key=&token=`; webhookie matches **path only**. Needs `text`, `cards`, or `cardsV2`.

Success: `200` Message-shaped JSON (`name`, `text`, `createTime`, `thread.name`).  
Invalid: `400` `{"error":{"code":400,"message":"text or cards required","status":"INVALID_ARGUMENT"}}`.

Real Google incoming webhooks are one-way. Destination clicks still fire a synthetic `CARD_CLICKED` if you set an Interactivity URL.

## Mattermost incoming webhooks

`POST /hooks/mattermost/hooks/{token}`

Default: `/hooks/mattermost/hooks/mattermost-webhookie`

JSON `{text, channel, username, attachments, ...}` or Slack-compatible `payload=`. Needs `text` or `attachments`.

Success: `200` `text/plain` `ok`.  
Invalid: `400` `Unable to parse incoming data`.

## Opsgenie Alert API v2

`POST /hooks/opsgenie/v2/alerts`  
`POST /hooks/opsgenie/v2/alerts/{id}/acknowledge`  
`POST /hooks/opsgenie/v2/alerts/{id}/close`

Default: `/hooks/opsgenie/v2/alerts`  
Default GenieKey (optional, used to pick a team sink): `eb243592-faa2-4ba2-a551-webhookie01`

Create requires `message`. Optional `alias`, `priority` (`P1`–`P5`).

Success: `202` `{"result":"Request will be processed","took":0.01,"requestId":"..."}`.  
Invalid create: `422`.

`Authorization: GenieKey {token}` selects the sink when the path is shared.

## Listing sinks

`GET /api/v1/sinks` returns each sink with `url` = `publicBaseUrl + path`. `POST /api/v1/sinks` creates a **generic** bin only; provider channels are created from destination UIs (`POST /api/v1/workspaces/{id}/channels`).
