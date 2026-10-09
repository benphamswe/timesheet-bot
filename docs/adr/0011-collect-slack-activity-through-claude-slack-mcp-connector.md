# ADR-0011: Collect Slack activity through Claude's Slack MCP connector

- Status: Proposed
- Date: 2026-10-10
- Deciders: owner
- Related: [ADR-0006](0006-personal-tool-scale-ready-aws-later.md) (single-user tool), [ADR-0007](0007-grpc-contracts-now-split-later.md) (the Python service and its packaging ruling), [ADR-0008](0008-monorepo-service-layout.md) (service layout, contracts at the root)

## Context

- The Go collector (`services/timesheet-core/internal/activity/infrastructure/slack`) calls Slack `search.messages`, which needs a user token (`xoxp-`, scope `search:read`) from a Slack app. The owner cannot create or install a Slack app in the workspace, so `SLACK_USER_TOKEN` cannot exist.
- Copying the browser session token and cookie (`xoxc-` plus the `d` cookie) out of a logged-in Slack client was considered and **rejected for security reasons**: it is a full-account credential with no scopes, it bypasses workspace admin controls, it violates the spirit of the workspace's app policy, and a leak exposes every channel the owner can read.
- The owner's Claude Code is logged in with a claude.ai account whose organisation offers the claude.ai "Slack" connector (official OAuth, managed by Anthropic and Slack).
- Verified on 2026-10-10 from the official docs:
  - "If you've logged into Claude Code with a claude.ai account, MCP servers you've added in claude.ai, known as connectors, are automatically available in Claude Code", including "Agent SDK sessions" ([MCP](https://code.claude.com/docs/en/mcp)).
  - Connectors are fetched only when the active authentication is a claude.ai subscription login. They are not loaded with `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `apiKeyHelper`, a third-party provider, or a `claude setup-token` token ([MCP](https://code.claude.com/docs/en/mcp)).
  - `setting_sources` does not control connectors; `strict_mcp_config=True`, `disableClaudeAiConnectors` or `ENABLE_CLAUDEAI_MCP_SERVERS=false` disables them ([Agent SDK: Claude Code features](https://code.claude.com/docs/en/agent-sdk/claude-code-features), [Python SDK](https://code.claude.com/docs/en/agent-sdk/python)).
  - The Python SDK has structured output: `output_format={"type": "json_schema", "schema": ...}`, read from `ResultMessage.structured_output` ([Python SDK](https://code.claude.com/docs/en/agent-sdk/python)).
  - MCP tools are named `mcp__<server>__<tool>`, and allow rules accept `mcp__<server>__*` ([Agent SDK MCP](https://code.claude.com/docs/en/agent-sdk/mcp)).
  - The Agent SDK overview says: "Unless previously approved, Anthropic does not allow third party developers to offer claude.ai login or rate limits for their products, including agents built on the Claude Agent SDK" ([overview](https://code.claude.com/docs/en/agent-sdk/overview)). This collector is the owner's private script on the owner's own login and is not offered to anyone. The page gives no explicit personal-use exemption, so this reading is unconfirmed.
- Smoke test on 2026-10-10 (`claude-agent-sdk` 0.2.165, bundled CLI 2.1.294; `get_mcp_status()` only, no prompt sent, nothing read): claude.ai connectors appear with `scope: claudeai` and `type: claudeai-proxy`, and `claude.ai MCP` and `claude.ai Claude Docs` were `connected`. **`claude.ai Slack` was `needs-auth`**, so its tool list could not be read.

## Decision

1. A Python CLI in `services/allocation-agent`, `collect --week YYYY-Www [--out-dir DIR] [--dry-run]`, drives the Claude Code CLI through `claude-agent-sdk` on the owner's claude.ai login, and uses the claude.ai Slack connector to search the owner's own messages.
2. Least privilege in the SDK session:
   - built-in tools off (`tools=[]`);
   - `setting_sources=[]`, so no user or project allow rule leaks in;
   - `permission_mode="dontAsk"`, so any tool not pre-approved is denied;
   - `allowed_tools` holds only the connector's read/search tools, and every other connector tool (send, schedule, react, canvas, upload, ...) is in `disallowed_tools`.

   The tool split is computed from the live tool list and annotations in a no-prompt preflight. An unknown tool is denied.
3. The search is the Go collector's: `from:<@user> after:<Sunday> before:<Saturday>` (exclusive dates, so Monday to Friday), and messages are clamped to Monday to Friday of the ISO week in the configured timezone.
4. The agent returns structured output that pydantic validates. Which hits count as work activity is decided in `normalize_messages` (`TODO(owner)`).
5. The file contract below is the only interface to the Go core. Go reads it when `activity.source` is `file`.

### File contract (schema version 1)

- Path: `<out-dir>/<ISO week>.json`, default `<repo root>/data/activity/`. The file is written atomically (temp file in the same folder, fsync, `os.replace`) with mode 0600, and `data/activity/*.json` is gitignored.
- Schema: [`api/activity/activity-week.v1.schema.json`](../../api/activity/activity-week.v1.schema.json) (JSON Schema 2020-12).

```json
{
  "schema_version": 1,
  "week": "2026-W41",
  "slack_user_id": "U0123ABCD",
  "generated_at": "2026-10-10T10:00:00+07:00",
  "source": "slack-mcp",
  "messages": [
    {"ts": "1760000000.000100", "channel_id": "C0123", "channel_name": "proj-x",
     "posted_at": "2026-10-05T09:00:00+07:00", "text": "...",
     "permalink": "https://...", "thread_ts": null}
  ]
}
```

- `posted_at` and `generated_at` are RFC 3339 with an offset. `posted_at` is the `ts` in the configured timezone, in whole seconds.
- `channel_name` and `permalink` are empty strings when the connector does not return them. `thread_ts` is null for a message that is not a thread reply.
- Readers ignore unknown fields, so adding an optional field keeps version 1. Removing, renaming or retyping a field needs `schema_version: 2` and a reader that accepts it first.

## Alternatives considered

- **Slack app with a user token (the built Go collector).** Rejected: the owner cannot create or install a Slack app.
- **Browser session token and cookie.** Rejected for security reasons (see Context). Not to be implemented.
- **Slack's hosted MCP server (`https://mcp.slack.com/mcp`) with its own OAuth.** Rejected for now. It needs a registered Slack app (directory-published or internal) with a fixed app id, confidential OAuth with a client secret, and workspace admin approval ([Slack MCP server](https://docs.slack.dev/ai/slack-mcp-server)). This is the same blocker as option 1. It becomes the preferred path if an internal app is ever approved, because it removes the claude.ai login and model usage from the loop.
- **Manual export (Slack data export or copy and paste).** Rejected: an export needs workspace admin rights, and copy and paste is error-prone weekly toil.

## Consequences

- Positive:
  - No Slack token or browser credential is stored anywhere.
  - Access is the connector's OAuth grant, which the owner or the organisation can revoke.
  - The Go core gets a plain, versioned, testable file instead of a vendor API.
- Negative / accepted risk:
  - **Needs a claude.ai login on the machine** in Claude Code, with the Slack connector authenticated. On 2026-10-10 it was `needs-auth`. `ANTHROPIC_API_KEY` must not be active, or the connector does not load.
  - **Not fully headless.** The OAuth grant is interactive and can expire, so it cannot run unattended in CI or a container without the owner's login. The Phase 2 cron flow must tolerate a missing file.
  - **Consumes Claude usage** from the owner's subscription limits for every run. Large weeks mean long structured outputs; `--max-turns` bounds the loop.
  - **Depends on the connector's behaviour.** Tool names, parameters, page size, whether results carry `permalink`, `channel_name` and `thread_ts`, and rate limits are not documented. They can change without notice. The preflight fails closed if the connector is missing, not connected, or has no read tool.
  - **The model is in the data path.** Text is copied by the model, so a long message could be altered or truncated, and message content is untrusted input (prompt injection). Mitigations: read-only tool allowlist, a system prompt that treats content as data, structured-output validation, and `complete: false` fails the run. Fidelity is not guaranteed byte-for-byte.
  - Message text is stored in plain files on disk (mode 0600, gitignored).
  - The third-party sentence in the Agent SDK policy has no explicit personal-use exemption (see Context).
- Follow-up:
  - The owner authenticates the Slack connector, then runs `make collect WEEK=... ARGS=--dry-run` once to confirm tool names and fields.
  - The owner implements `normalize_messages` and removes the skip in `tests/unit/activity/test_normalize.py`.
  - The owner confirms uv as the Python lock tool (ADR-0007 ruling: uv is CONDITIONAL; pinned to 0.12.21, `uv sync --locked`).
  - Add CI and Dependabot entries for `services/allocation-agent` (ADR-0008 follow-up).
