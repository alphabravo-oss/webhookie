# Webhookie docs

> **Alpha.** Use at your own risk. This tree documents the current code, not a stable contract.

User-facing documentation. The implementation plan lives in repo-root `plan.md` and is not a user guide.

| Doc | Contents |
|---|---|
| [Getting started](getting-started.md) | Docker, Compose, local two-process dev, Makefile |
| [Sinks](sinks.md) | Destination URLs, documented validation, HTTP responses, `response_url` / Discord PATCH / `answerCallbackQuery` |
| [Destinations](destinations.md) | Channel UIs, two-way clicks, what is and is not real |
| [Sources](sources.md) | Signed outbound fixtures (`POST /api/v1/send`) |
| [HTTP API](api.md) | REST, SSE, chaos, assertion helpers |
| [Configuration](configuration.md) | Environment variables, auth, retention |

Product contract: [`PRODUCT.md`](../PRODUCT.md). Operator summary: [`README.md`](../README.md). OpenAPI skeleton: [`../api/openapi.yaml`](../api/openapi.yaml). License: [AGPL-3.0](../LICENSE).
