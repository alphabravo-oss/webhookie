# Destinations

Destination pages are operator views of the same events as Inbox, filtered by that channel’s sink. Inbox stays the packet log.

Nav group in the UI is labeled **Mocks**. The pages are destinations, not full Slack/Teams/Telegram.

| UI | Workspace id | Default channel | Kind |
|---|---|---|---|
| `/slack` | `ws-slack` | `#alerts` | channel |
| `/teams` | `ws-teams` | `General` | channel |
| `/discord` | `ws-discord` | `#general` | channel |
| `/pagerduty` | `ws-pagerduty` | `default` | service |
| `/telegram` | `ws-telegram` | `alerts` | chat |
| `/googlechat` | `ws-googlechat` | `Ops` | space |
| `/mattermost` | `ws-mattermost` | `#town-square` | channel |
| `/opsgenie` | `ws-opsgenie` | `default` | team |

`GET /api/v1/workspaces` and `GET /api/v1/workspaces/{id}` (id or provider name) return channels with `path` and `url`.

Create a channel: `POST /api/v1/workspaces/{id}/channels` `{"name":"deploys"}` — also the **New channel** form in the UI. That upserts a sink with a unique path.

## Two-way clicks

`POST /api/v1/workspaces/{id}/channels/{channelId}/actions`

```json
{ "eventId": "...", "kind": "button", "actionId": "approve", "value": "yes", "text": "Approve" }
```

Always inserts an `interaction` row. If `interactivityUrl` is set on the workspace (`PATCH /api/v1/workspaces/{id}`), webhookie POSTs a provider-shaped body there (10s timeout) and stores the attempt.

| Provider | Click payload |
|---|---|
| Slack | `application/x-www-form-urlencoded` `payload=` `type=block_actions`. Includes a `response_url` at `/hooks/slack/response/{eventId}` (POST JSON `text`/`blocks`, `replace_original`, `delete_original`; 5 uses / 30 minutes). Slack signing headers only if `signingSecret` is set. |
| Teams | JSON Adaptive Card `invoke` / `adaptiveCard/action` |
| Discord | JSON interaction `type: 3` (component). Incoming-webhook `PATCH`/`DELETE /hooks/discord/api/webhooks/{id}/{token}/messages/{eventId}` edits/deletes the original (use `wait=true` so execute returns that id). Not a full Interactions API (no 3s ack, no application follow-ups). |
| Telegram | JSON `Update` with `callback_query` (`id` is the interaction id). `POST /hooks/telegram/bot/{token}/answerCallbackQuery` with that `callback_query_id`. |
| Google Chat | JSON `CARD_CLICKED`. Real incoming webhooks cannot receive this. |
| Mattermost | JSON interactive-message style (`user_id`, `post_id`, `context`). |
| PagerDuty | Also captures a local Events API v2 ack/resolve onto the same sink, then POSTs a v3-shaped `incident.acknowledged\|resolved` if Interactivity URL is set. |
| Opsgenie | Captures local `/v2/alerts/{alias}/acknowledge\|close`, then POSTs `{action, alert}` if Interactivity URL is set. |

Click chrome updates **locally** and does **not** wait for or apply your Interactivity URL handler’s HTTP body. If the handler then calls a follow-up URL (Slack `response_url`, Discord webhook message PATCH/DELETE, Telegram `answerCallbackQuery`), the original transcript message updates via `displayBody` / `deleted` (SSE). Inbox still shows the original capture `body`. Follow-up POSTs themselves are extra Inbox packets; the destination transcript only lists events whose `path` equals the channel webhook path (except Slack `response_url` without replace/delete, which is stored as a new channel message).

| Platform | After click (UI) |
|---|---|
| Slack | Clicked button `✓`, others disabled, “webhookie clicked …” |
| Teams | Buttons removed, “Your response was sent to the app” |
| Discord | Components disabled/faded, italic “webhookie used …” |
| Telegram | Inline keyboard removed, “You pressed …” |
| Mattermost | Actions removed, “{action} by webhookie” |
| Google Chat | Buttons replaced with a completed chip |
| PagerDuty / Opsgenie | Ack hides Acknowledge; Resolve/Close remain until resolved/closed, then no action buttons |

State is stored as interactions (`GET .../interactions`) and survives refresh.

## What this is not

- Slack Bolt, slash commands, modals, `chat.update` (bot token)
- Teams Bot Framework, Graph subscriptions, Adaptive Card refresh from the bot
- Discord application commands and Interactions follow-up tokens (webhook `PATCH`/`DELETE` of the captured message is implemented)
- Telegram Bot API besides `sendMessage` and `answerCallbackQuery`
- A production on-call console

Capture validation (Block Kit, Discord files, Telegram markup, Adaptive Card elements) is on the sink, not these pages. See [sinks.md](sinks.md). Use Inbox to see the exact bytes and `validationErrors`. Use destinations to copy URLs and exercise a click.
