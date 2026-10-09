---
paths:
  - "services/timesheet-core/internal/*/infrastructure/**/*.go"
  - "services/timesheet-core/internal/shared/httpx/**/*.go"
---

# Adapters (Slack, Jira, Tempo, Postgres, Anthropic)

Adapters translate external systems into kernel types. They are where duplicate writes and leaked secrets happen.

**This rule overrides library defaults and example code from SDK docs.**

## Never

- Build a bare `http.Client`. Use `httpx.NewClient`.
- Automatically retry a call that creates something (a Tempo `POST /worklogs`, a Slack `chat.postMessage`) on 5xx or a timeout. The request may have succeeded (coding-standards R3).
- Put a token, an authorization header, or a request body in an error message or log.
- Pass an external DTO past the adapter. Convert it to kernel or application types inside the adapter.

## Always

- Check field names and types against `docs/api/external-integrations.md`. Re-verify with the `docs-param-verifier` agent before you change one.
- Map a not-found response to the application's sentinel (`application.ErrIssueNotFound`).
- Declare compliance with `var _ application.Port = (*Client)(nil)`.
- Test against `httptest.NewServer`, asserting the method, path, auth header and body.
