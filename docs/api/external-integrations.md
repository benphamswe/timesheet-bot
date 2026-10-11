# External integrations

Facts below were verified against official specs on 2026-10-09 (Mavenlink: read from the spec, not yet re-verified). Re-verify with the `docs-param-verifier` agent before changing an adapter.

## Slack

| Use | Token | API | Notes |
|---|---|---|---|
| Collect own messages [built] | user `xoxp-`, `search:read` | `search.messages` | Query `from:<@U...> after:<Sun> before:<Sat>`; dates are exclusive. slack-go v0.30.1: start from `slack.NewSearchParameters()`, page with `res.Paging.Pages` (`res.Page` is ambiguous). |
| Approval DM [P2] | bot `xoxb-`, `chat:write`, `im:write` | `chat.postMessage`, Block Kit | |
| Button clicks [P2] | app-level `xapp-`, `connections:write` | Socket Mode | [ADR-0003](../adr/0003-slack-socket-mode-for-approval.md) |

Adapter: `internal/activity/infrastructure/slack`.

## Activity week file [built]

A contract with `services/allocation-agent`, which fetches your Slack messages through Claude's Slack MCP connector and writes one file per ISO week. Used when `activity.source` is `file` ([configuration](../operations/configuration.md#activity-week-files)). Decision: ADR-0011, written with the allocation-agent side.

- **Path:** `<activity.file_dir>/<ISO week>.json`, for example `data/activity/2026-W41.json`.
- **Versions:** `1` (messages) and `2` (messages plus `parser` and `report_lines`, written when `report.parser: llm`). Any other `schema_version` is an error. The collector writes no file at all when its validation fails.
- **Top level:**

  | Field | Type | Checked |
  |---|---|---|
  | `schema_version` | int | must be `1` or `2`, otherwise an error |
  | `week` | string, `YYYY-Www` | must equal the requested week |
  | `slack_user_id` | string | must equal `slack_user_id` in `config.yaml` |
  | `generated_at` | RFC 3339 | not read |
  | `source` | string, `slack-mcp` | not read |
  | `messages` | array | see below |

- **Each message:**

  | Field | Type | Use |
  |---|---|---|
  | `ts` | string, Slack timestamp | error messages; tells a thread parent from a reply |
  | `channel_id`, `channel_name` | string | `WorkActivity.ChannelID`, `ChannelName` (`channel_defaults` matches the name) |
  | `posted_at` | RFC 3339 with offset | the calendar day in `timezone`; Saturday, Sunday and days outside the week are skipped |
  | `text` | string | `WorkActivity.Text`; Jira keys are extracted from it |
  | `permalink` | string | `WorkActivity.Permalink`; also the source of `ThreadTS` when it has `thread_ts=` |
  | `thread_ts` | string or null | `WorkActivity.ThreadTS` for a reply. A value equal to `ts` (the thread's parent) is ignored, to match what `search.messages` gives |

- **Version 2 only, `parser`:** an object, read but not used by Go.

  | Field | Type | Example |
  |---|---|---|
  | `kind` | string | `llm` |
  | `provider` | string | `deepseek` |
  | `model` | string | `deepseek-flash` |

- **Version 2 only, `report_lines`:** an array, one item per summary line the collector's LLM parsed. Read by `file.Collector.ReportLines` in `report.parser: llm` mode; absent, or a version 1 file, fails with `activity file has no report_lines: re-run the collector with report.parser: llm`. An empty array is a week with no summaries. Each line is checked again in Go, and the first invalid one fails `plan` with its index and `source_ts`, never its notes:

  | Field | Type | Checked in Go | Use |
  |---|---|---|---|
  | `date` | string, `YYYY-MM-DD` | a Monday to Friday of `week` | entry day |
  | `project` | string | not empty after trimming | entry project; `mavenlink.mappings` `project` rules |
  | `minutes` | int | positive multiple of 15 | entry duration |
  | `notes` | string | not checked, never logged | entry description, Mavenlink `notes` |
  | `source_ts` | string, Slack timestamp | not checked | names the line in errors |
  | `source_channel_id` | string | not checked | kept on `kernel.ReportLine` |

- Unknown fields are ignored at every level.
- In report mode ([ADR-0012](../adr/0012-report-mode-slack-summaries-to-mavenlink-without-jira.md)) the same file carries your timesheet summaries from `report.channel` and the self-DM. With `report.parser: manual` only `text` is parsed in Go (`reportparser.Parser.Parse`, `TODO(owner)`), and the message day is `WorkActivity.Day`; with `report.parser: llm` Go uses version 2's `report_lines` instead. Both sides read `report.channel` and `report.include_self_dm` from `config.yaml` with the same rules: a leading `#` is stripped, names must match `[a-z0-9._-]+`, and an absent `include_self_dm` is false.
- Both collectors map messages through `activity/domain.ToActivity`, so the same messages give the same evidence; `test/unit/adapters/file` checks this parity.
- **Adapter:** `internal/activity/infrastructure/file`.

## Jira Cloud REST v3

Used in `allocation.mode: jira_keys` only; report mode builds no Jira client.

- `GET {JIRA_BASE_URL}/rest/api/3/issue/{key}?fields=summary`, basic auth `email:api_token`.
- Response `id` is a **string**; it is parsed to int64 for Tempo. 404 maps to `application.ErrIssueNotFound`.
- Adapter: `internal/timesheet/infrastructure/jira` (per-process cache).

## Tempo Cloud REST v4

- Base URL is regional: `https://api.tempo.io/4` (universal), `https://api.eu.tempo.io/4`, `https://api.us.tempo.io/4`. A wrong region returns an authorization error.
- Auth: `Authorization: Bearer <TEMPO_API_TOKEN>`.
- `POST /worklogs` required: `authorAccountId` (string), `issueId` (int64), `startDate` (YYYY-MM-DD), `timeSpentSeconds` (int). Optional used: `startTime` (HH:mm:ss), `description`. `issueKey` is **not** accepted.
- Response: `tempoWorklogId` (int64), stored as text.
- `GET /worklogs/user/{accountId}?from&to&offset&limit` (limit default 50) [P3/P4].
- Rate limit about 5 requests/second; 429 on excess. No documented `Retry-After`, so backoff starts at 1s.
- Adapter: `internal/timesheet/infrastructure/tempo`.

## Mavenlink (Kantata OX) REST v1 [built]

Read from the official OpenAPI spec on 2026-10-09; not yet confirmed by `official-source-verifier`. Decision and auth modes: [ADR-0010](../adr/0010-add-mavenlink-as-a-second-submit-target.md).

- **Base URL:**
  - Bearer auth: `https://api.mavenlink.com/api/v1`.
  - Cookie auth: the account's host, `https://27global.mavenlink.com/api/v1`, observed in the browser.
- **Auth:**
  - `bearer`: `Authorization: Bearer <MAVENLINK_API_TOKEN>`.
  - `cookie` (undocumented): `Cookie: <MAVENLINK_SESSION_COOKIE>`, `X-CSRF-Token` taken from `<meta name="csrf-token">` on `/timesheets`, `X-Requested-With: XMLHttpRequest`, and `Referer: <csrf page>`.
- **`POST /time_entries.json`**, body `{"time_entry": {...}}`:
  - Required: `workspace_id` (int), `date_performed` (YYYY-MM-DD), `time_in_minutes` (int).
  - Optional fields used: `story_id`, `role_id`, `notes`, `billable`, and `external_reference` {`service_name`, `service_model`, `service_model_ref`}, all three required inside it.
  - Response: `results[]` of {`key`, `id`}, with objects keyed by id under `time_entries`.
- **`GET /time_entries.json`**:
  - Parameters: `date_performed_between=<from>:<to>`, `external_reference_service_name`, `include=external_references,workspace,active_submission`, `per_page` (max 200), `page`; the page count is in `meta.page_count`.
  - In responses, ids such as `workspace_id` and `active_submission_id` are **strings**, and `workspace_id` and `active_submission_id` appear only when included.
- **`POST /timesheet_submissions.json`**:
  - Body: `{"timesheet_submission": {"workspace_id": int, "line_item_ids": [int], "title": string}}`.
  - One project per submission.
- **Gotchas:**
  - No idempotency on creates: the adapter tags each entry with the external reference `timesheet-bot` / `timesheet_entry` / `<entry ID>`, and never re-sends a create.
  - Rate limits are not published in the open docs; 429 is retried by `httpx.RetryTransport` for GET only.
- **Adapter:** `internal/timesheet/infrastructure/mavenlink` (`client.go`, and `auth.go` for the `Bearer` and `SessionCookie` strategies).
- **Report mode:** a report entry's notes become `notes`, and its project picks `workspace_id` (and optional `story_id`, `role_id`) through `mavenlink.mappings` rules with `project`.

## Anthropic API [P3]

- Python SDK `anthropic` (PyPI), used in `services/allocation-agent` (planned P3, [ADR-0007](../adr/0007-grpc-contracts-now-split-later.md)). Model configured in `agent/agent.yaml` (default `claude-sonnet-5-5`).
- The Go SDK `github.com/anthropics/anthropic-sdk-go` is not used: ADR-0007 (Accepted) moves the agent out of the Go process.
- Load the `claude-api` skill before writing this adapter.
