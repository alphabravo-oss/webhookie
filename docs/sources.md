# Sources (outbound)

Webhookie can POST a **signed fixture** at your app so you can test inbound handlers without Stripe/GitHub/Slack.

```bash
curl -s -X POST http://localhost:8080/api/v1/send \
  -H 'content-type: application/json' \
  -d '{
    "provider": "github",
    "event": "push",
    "target": "http://host.docker.internal:3000/webhook",
    "secret": "whsec_test"
  }'
```

`provider` must match the source pack id (not the display name). `event` is a fixture name from that pack. `timestampSkewSec` is optional (integer; used where the signer includes a timestamp).

`GET /api/v1/fixtures` lists `{provider, event, description}`. `GET /api/v1/send/attempts` lists recent deliveries.

## Packs

| `provider` | Fixture `event` names | Signing |
|---|---|---|
| `standard` | `generic.ping` | Standard Webhooks HMAC |
| `github` | `ping`, `push`, `pull_request` | `X-Hub-Signature-256` |
| `slack-events` | `url_verification`, `app_mention` | Slack Events `X-Slack-Signature` |
| `stripe` | `invoice.paid`, `customer.subscription.updated`, `checkout.session.completed` | `Stripe-Signature` `t,v1` |
| `pagerduty-v3` | `incident.triggered`, `incident.acknowledged`, `incident.resolved` | PagerDuty webhooks v3 HMAC |

These are **outbound samples**, not a complete event catalog. Bodies are small JSON fixtures in `internal/source/*/`.

The UI **Send** page (`/send`) and **Fixtures** (`/fixtures`) wrap the same API.

Delivery timeout is 10 seconds. Redirects: at most 3. Attempts are stored even on failure.
