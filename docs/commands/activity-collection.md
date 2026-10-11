# Activity collection

Commands for the Slack activity collector in `services/allocation-agent` ([ADR-0011](../adr/0011-collect-slack-activity-through-claude-slack-mcp-connector.md)): checking the claude.ai Slack connector, collecting a week of your timesheet reports (summary channel and self-DM), and checking the weekly file against its contract. The collector needs no Claude credential. It uses your Claude Code claude.ai login, and `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` must be unset, or the connector does not load; `collect` checks this at startup. Its own settings and secrets (`DEEPSEEK_API_KEY` when `report.parser` is `llm`) live in `services/allocation-agent/.env` ([ADR-0013](../adr/0013-per-service-env-files.md)).

## Check that the Slack connector is usable

```bash
cd services/allocation-agent && uv run --locked python -I - <<'PY'
import asyncio
from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions

async def main():
    opts = ClaudeAgentOptions(tools=[], allowed_tools=[], permission_mode="dontAsk", setting_sources=[])
    async with ClaudeSDKClient(opts) as client:
        await asyncio.sleep(15)  # connectors connect asynchronously
        for s in (await client.get_mcp_status())["mcpServers"]:
            if s.get("scope") == "claudeai":
                print(s["name"], s["status"])
            if s["name"] == "claude.ai Slack":
                for t in s.get("tools", []):
                    print("   ", t["name"], t.get("annotations", {}))
asyncio.run(main())
PY
```

- **Run from**: the repository root.
- **What it does**: starts a Claude Code session through the SDK without sending a prompt, so it uses no Claude usage and reads no Slack data. It prints each claude.ai connector's status and, for `claude.ai Slack`, its tool names and read-only annotations. `collect` runs the same check (its preflight) before every collection.
- **When to use it**: before the first collection, and when `collect` fails its preflight.
- **Expected**: `claude.ai Slack connected`, followed by its tools. On 2026-10-10 the first form of this check (with `cwd` set, default settings) listed 27 servers, and `claude.ai Slack` was `needs-auth`. Fix that at claude.ai > Settings > Connectors or with `/mcp` in Claude Code. If no `claude.ai` line appears at all, you are not on a claude.ai login (check that `ANTHROPIC_API_KEY` is unset).

## Check which auth variables are set (names only)

```bash
for v in ANTHROPIC_API_KEY CLAUDE_CODE_OAUTH_TOKEN ANTHROPIC_AUTH_TOKEN ENABLE_CLAUDEAI_MCP_SERVERS; do
  if [ -n "${(P)v}" ]; then echo "$v=set"; else echo "$v=unset"; fi
done
```

- **Run from**: anywhere, in zsh (`${(P)v}` is zsh indirection; in bash use `${!v}`).
- **What it does**: reports whether each variable that disables the claude.ai connectors is set. It never prints a value.
- **Expected** for `collect`: all four `unset` (confirmed 2026-10-10). This checks your shell only; `collect` itself also sees `services/allocation-agent/.env` and exits 2 if one of the first three is set there.

## Collect a week

```bash
make collect WEEK=2026-W41 ARGS=--dry-run           # from the root: counts per day and source only
make collect WEEK=2026-W41                          # writes data/activity/2026-W41.json
cd services/allocation-agent && uv run --locked collect --week 2026-W41 --dry-run
uv run --locked collect --week 2026-W41 --dry-run --report-channel timesheet-reports --no-include-self-dm
uv run --locked collect --help                      # every flag
# with report.parser: llm in config.yaml:
uv run --locked collect --week 2026-W41 --dry-run                               # also calls the LLM
uv run --locked collect --week 2026-W41 --llm-model deepseek-v4-pro              # writes contract v2
```

- **Run from**: the repository root (`make`), or `services/allocation-agent/` (`uv run`). A relative `--out-dir` is relative to the current folder; through `make` that is `services/allocation-agent/`, so pass an absolute path in `ARGS`.
- **Env file**: `make collect` first exports `services/allocation-agent/.env` if it exists, then restores every variable your shell already exports, so the shell wins (the same `LOAD_DOTENV` macro as `make run`). Plain `uv run` does not read `.env`.
- **Startup check**: before anything else, `collect` exits 2 if `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` is non-empty, with `collect: <NAMES> is set. The claude.ai Slack connector loads only on a claude.ai subscription login, ...`. It names the variables and never prints a value. `DEEPSEEK_API_KEY` is not checked.
- **What it does**: runs the preflight, then one Claude session with only the connector's search tools. The session runs one search per report source:
  - `in:#<channel> from:<@you> after:<Sunday> before:<Saturday>` for `report.channel`;
  - `in:<@you> from:<@you> after:<Sunday> before:<Saturday>` for the self-DM, when `report.include_self_dm` is true. This modifier is unverified through the connector.

  Afterwards it:
  - validates each source's structured result;
  - merges them, deduplicates by `(channel_id, ts)`, and sets `channel_name` to the channel name or `self-dm`;
  - applies `normalize_messages`;
  - with `report.parser: llm`, sends each message to the configured LLM, one call per message, and validates every returned line ([ADR-0014](../adr/0014-llm-provider-port-with-deepseek-for-report-parsing.md));
  - writes `<out-dir>/<week>.json` atomically (mode 0600): contract v1 with `manual`, and contract v2 (plus `parser` and `report_lines`) with `llm`.

  `--dry-run` writes nothing. It prints the sources and the counts per day and per channel, never message text. With `llm`, it still calls the LLM, then also prints the minutes per day and project, never notes or message text.
- **LLM key**: `DEEPSEEK_API_KEY`, from the environment only (the env file above, or `~/.zshrc`). With `report.parser: llm` the client is built before any Claude or Slack usage, so a missing key fails first. **Data**: each summary's full text goes to DeepSeek, which stores data in China. Confirm the company policy first.
- **Settings**: flag > environment > `services/timesheet-core/config/config.yaml`.
  - `--report-channel` / `REPORT_CHANNEL` / `report.channel`.
  - `--include-self-dm`, `--no-include-self-dm` / `report.include_self_dm` (false when absent).
  - `--slack-user-id` / `SLACK_USER_ID` / `slack_user_id`.
  - `--timezone` / `TIMEZONE` / `timezone`.
  - `report.parser` (`manual` | `llm`, default `manual`) comes from `config.yaml` only. There is no flag and no environment variable, because the Go core reads the same key to choose contract v1 or v2. `REPORT_PARSER` is ignored, and `--parser` is rejected as an unrecognized argument.
  - `--llm-provider` / `LLM_PROVIDER` / `llm.provider` (default `deepseek`); `--llm-model` / `LLM_MODEL` / `llm.model` (default `deepseek-flash`).
  - `LLM_BASE_URL`, `LLM_TIMEOUT_SECONDS` and `LLM_MAX_RETRIES` override `llm.base_url` (https only), `llm.timeout_seconds` (default 60) and `llm.max_retries` (0-10, default 2). The `llm` block is read only with `llm`.
- **Exit codes**: `0` OK; `1` collection failed; `2` bad flag, week, config or credential, or an unknown `llm.provider`; `3` reserved (formerly: `normalize_messages` not implemented), no longer returned; `4` the LLM failed or returned an invalid line, and no file is written.
- **When to use it**: once per week before `timesheet plan` with `activity.source: file`.
- **Expected**:
  - Not run against Slack or DeepSeek while documenting (the connector was `needs-auth`, and no DeepSeek call was made).
  - The LLM checks below use a scratch config, `printf 'report:\n  parser: llm\n' > /tmp/llm-probe/config.yaml`.
    - With `DEEPSEEK_API_KEY` unset, `SLACK_USER_ID=U0123ABCD REPORT_CHANNEL=timesheet-reports uv run --locked collect --week 2026-W41 --dry-run --config /tmp/llm-probe/config.yaml` exits 4 with `collect: llm: DEEPSEEK_API_KEY is not set`, before any Claude session.
    - Adding `--llm-provider nope` exits 2 with `collect: unknown llm.provider 'nope'; registered providers: deepseek`.
    - With `LLM_MAX_RETRIES=11` it exits 2 with `collect: llm.max_retries must be a whole number from 0 to 10`.
    - A config with `parser: regex` exits 2 with `collect: report.parser 'regex' is unknown, want manual or llm`.
    - `--parser llm` exits 2 with `collect: error: unrecognized arguments: --parser llm`.
  - `make collect` without `WEEK` exits 2 with `usage: make collect WEEK=2026-W41 [ARGS=--dry-run]`.
  - With `ANTHROPIC_API_KEY` set in the env file, `make collect WEEK=2026-W41 ARGS=--dry-run DOTENV=/tmp/dotenv-probe/dummy.envfile` (a scratch file with a dummy value) exits 2 with `collect: ANTHROPIC_API_KEY is set. ...`; with `ANTHROPIC_API_KEY=` exported empty in the shell, the shell wins and the check passes. Confirmed 2026-10-10. `DOTENV=<path>` overrides the env file path for such tests.
  - `uv run --locked collect --week 2027-W53` exits 2 with `collect: week '2027-W53' does not exist`.
  - With no report source, `SLACK_USER_ID=U0123ABCD uv run --locked collect --week 2026-W41 --config /tmp/no-such-config.yaml` exits 2 with `collect: no report source: set report.channel and/or report.include_self_dm in config.yaml, REPORT_CHANNEL, --report-channel or --include-self-dm`.
  - Adding `--report-channel "Bad Name"` exits 2 with `collect: report channel 'Bad Name' is not a Slack channel name (lowercase letters, digits, '-', '_', '.')`.
  - All confirmed 2026-10-10.
- **Common failures**:
  - `MCP server "claude.ai Slack" is 'needs-auth', not connected`: authenticate the connector (exit 1).
  - `... not found`: log in with claude.ai and unset `ANTHROPIC_API_KEY` (exit 1).
  - `the self-DM search returned messages outside a DM; the in:<@user> modifier may not work through the connector` (exit 1): the self-DM modifier did not work through the connector. Run with `--no-include-self-dm` and report it, so ADR-0011 can switch to the history-reader fallback.
  - `collect: report extraction stopped: message ts=<ts> channel=<id>: ...` (exit 4): the LLM failed or returned an invalid line for that message. The rest of the line says why, and it never quotes the message:
    - `the API key was rejected (HTTP 401)`: check `DEEPSEEK_API_KEY`.
    - `insufficient balance (HTTP 402)`: top up the DeepSeek account.
    - `request refused (HTTP 400/422 ...); check llm.model`: set `llm.model` to `deepseek-flash` or `deepseek-v4-pro`.
    - `rate limited (HTTP 429)` or `server error (HTTP 5xx) after N attempt(s)`: run again later, or raise `llm.max_retries`.
    - `no answer within 60s`: raise `llm.timeout_seconds`.
    - `invalid report line at: lines.0.minutes (value_error)`: the summary has a time that is not a multiple of 15 minutes (for example `20m`), or the LLM got it wrong. Fix the summary in Slack. Minutes are never rounded.
    - `lines.0.date`: the line's day is not a Monday-Friday of the week.

## Check the contract

```bash
make -n collect WEEK=2026-W41 ARGS=--dry-run
git check-ignore -v services/timesheet-core/.env services/allocation-agent/.env data/activity/2026-W41.json
jq 'del(.messages) + {msg_keys: (.messages[0]|keys), n: (.messages|length)}' \
  services/timesheet-core/test/unit/adapters/file/testdata/happy/2026-W41.json
jq '{schema_version, parser, n_lines: (.report_lines|length), line_keys: (.report_lines[0]|keys)}' \
  services/timesheet-core/test/unit/adapters/file/testdata/v2/2026-W41.json
jq 'has("report_lines")' services/timesheet-core/test/unit/adapters/file/testdata/v2-no-lines/2026-W41.json
```

- **Run from**: the repository root.
- **What it does**: previews the `make collect` commands; confirms that the env files and the weekly file are gitignored (it prints the matching `.gitignore` rule and reads no file content); shows the Go reader's test fixture without message text.
- **Expected** (2026-10-10):
  - The preview prints `uv sync --locked` and `uv run --locked collect --week 2026-W41 --dry-run`.
  - `check-ignore` prints `.gitignore:3:.env` for both service env paths and `.gitignore:10:data/activity/*.json` for the weekly file (re-confirmed 2026-10-10 after ADR-0013).
  - `jq` shows `schema_version` 1 and an extra `future_field`. The Go reader ignores unknown fields, and so does the schema. `tests/unit/activity/test_contract.py` validates this fixture against `api/activity/activity-week.v1.schema.json`.
  - The v2 fixture shows `schema_version` 2, `parser` `{kind: llm, provider: deepseek, model: ...}`, 3 lines, and the line keys `date, minutes, notes, project, source_channel_id, source_ts`. It shows no notes or message text.
  - `has("report_lines")` prints `false`: that fixture omits `report_lines`, which the v2 schema allows and the Go reader's `ReportLines` rejects.
  - `tests/unit/activity/test_contract_v2.py` checks every Go `v2*` fixture and `schema-v3` against `api/activity/activity-week.v2.schema.json`. A JSON Schema cannot tell which dates fall in the week, so `v2-outside-week` and `v2-weekend` pass the schema; the collector and the Go reader both reject them.

## Collect in a container [not usable yet]

```bash
docker compose --profile collect build collector
docker compose run --rm collector --week 2026-W41 --dry-run
```

- **Run from**: the repository root.
- **What it does**: builds and runs `collect` in the `collector` service: `services/allocation-agent/.env` through `env_file` (optional, the only secret source), `services/timesheet-core/config` mounted read-only and `data/activity` read-write, non-root, read-only root filesystem, own network ([ADR-0013](../adr/0013-per-service-env-files.md)).
- **When to use it**: not yet. The container has no claude.ai login, by design, until the owner picks a credential option ([ADR-0013](../adr/0013-per-service-env-files.md#collector-credentials-in-a-container)).
- **Expected**: the build exits 0 (confirmed 2026-10-10). `run` was **not run while documenting**: without a login it fails its preflight with `MCP server "claude.ai Slack" not found` (exit 1).
