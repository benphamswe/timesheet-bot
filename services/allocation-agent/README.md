# allocation-agent

The Python service of timesheet-bot. It has two roles:

| Role | Phase | Status |
|---|---|---|
| Slack activity collector (`collect`) | P1.5 | [built] code and tests; [planned] first live run, after the Slack connector is authenticated |
| LLM report parsing (`report.parser: llm`, DeepSeek) | P1.5 | [built] code and tests, no live call yet; Proposed in [ADR-0014](../../docs/adr/0014-llm-provider-port-with-deepseek-for-report-parsing.md); the owner must confirm the data policy first |
| AI allocation agent (`PolicyService` over gRPC) | P3 | [planned: P3] |

## Slack activity collector [built]

Decision: [ADR-0011](../../docs/adr/0011-collect-slack-activity-through-claude-slack-mcp-connector.md) (Proposed).

The owner cannot create a Slack app, so there is no Slack token. You write your timesheet summaries in Slack, in a summary channel and in your self-DM (the "you" conversation). `collect` drives the Claude Code CLI through `claude-agent-sdk` on your claude.ai login. The claude.ai Slack connector then searches your own messages in those two places for one ISO week. The result is written to `<out-dir>/<week>.json`, and the Go core reads that file when `activity.source` is `file`.

- Contract: [`api/activity/activity-week.v1.schema.json`](../../api/activity/activity-week.v1.schema.json), schema version 1, with `report.parser: manual` (the default). With `report.parser: llm` the file is [`activity-week.v2.schema.json`](../../api/activity/activity-week.v2.schema.json), schema version 2: version 1 plus `parser` and `report_lines` ([LLM report parsing](#llm-report-parsing-built)).
- Sources: one search per configured source, both with the Go collector's window `after:<Sunday> before:<Saturday>`. The dates are exclusive, so the search covers Monday to Friday.
  - Summary channel: `in:#<channel> from:<@you> ...`.
  - Self-DM: `in:<@you> from:<@you> ...`, the `in:<@UserID>` DM form from Slack's `search.messages` reference. It is unverified through the connector until the connector is authenticated. The run fails closed if this search returns non-DM messages.
- Merge: the results are deduplicated by `(channel_id, ts)` only. `channel_name` is the channel name, or `self-dm` for self-DM messages, so the Go core can tell the sources apart. The Go core handles the same text appearing in both places.
- Tools: only the connector's search tools are allowed. Built-in tools, history readers (read channel, read thread) and every write tool (send, react, schedule, canvas, upload, ...) are denied. A preflight that sends no prompt checks that the connector is `connected` before any Claude usage.
- Output: the file is written atomically with mode 0600 to `<repo root>/data/activity/` by default. `data/activity/*.json` is gitignored. `--dry-run` prints counts per day and per channel only, never message text.

### Before the first run

1. Log in to Claude Code with your claude.ai account (`claude`, then `/login`). `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `CLAUDE_CODE_OAUTH_TOKEN` must not be set, in your shell or in `services/allocation-agent/.env`, or the connector does not load. `collect` checks this at startup and exits 2, naming the variable.
   - Optional: `services/allocation-agent/.env` (owner-only, gitignored, `chmod 600`; [ADR-0013](../../docs/adr/0013-per-service-env-files.md)) holds this service's environment: `SLACK_USER_ID`, `TIMEZONE`, `REPORT_CHANNEL`, and `DEEPSEEK_API_KEY` when `report.parser` is `llm`. `make collect` loads it, and your shell wins over it.
   - Container: `docker compose --profile collect build collector` builds the image, but it has no claude.ai login yet, so collect on the host ([activity collection](../../docs/commands/activity-collection.md#collect-in-a-container-not-usable-yet)).
2. Authenticate the Slack connector: claude.ai > Settings > Connectors, or `/mcp` in Claude Code. On 2026-10-10 it showed `needs-auth`.
3. Set `slack_user_id`, `timezone` and the report sources in `services/timesheet-core/config/config.yaml`; both services read that one file.

   ```yaml
   report:
     channel: ""            # summary channel name without '#'
     include_self_dm: true  # false when the key is absent
   ```

   The precedence is flag > environment > `config.yaml`:

   | Setting | Flag | Environment |
   |---|---|---|
   | `slack_user_id` | `--slack-user-id` | `SLACK_USER_ID` |
   | `timezone` | `--timezone` | `TIMEZONE` |
   | `report.channel` | `--report-channel` | `REPORT_CHANNEL` |
   | `report.include_self_dm` | `--include-self-dm` / `--no-include-self-dm` | (none) |

   With no channel and no self-DM, `collect` exits 2.
4. Optional: to parse the summaries with an LLM, see [LLM report parsing](#llm-report-parsing-built).

### Run

```bash
make collect WEEK=2026-W41 ARGS=--dry-run    # from the repository root: counts only
make collect WEEK=2026-W41                   # writes data/activity/2026-W41.json
# or, inside services/allocation-agent:
uv run --locked collect --week 2026-W41 --dry-run
```

Exit codes:

- `0`: OK.
- `1`: collection failed (connector missing or not connected, agent error, invalid or incomplete output).
- `2`: bad flag, week or config, no report source, a connector-blocking credential, or an unknown `llm.provider`.
- `3`: reserved (formerly: `normalize_messages` not implemented). It is no longer returned, and it is not reused.
- `4`: `report.parser: llm` only. The LLM failed (missing or rejected key, insufficient balance, rate limit or server error after the retries, timeout), or it returned an invalid answer or line. No file is written, and there is no fallback.

### Layout

| Path | Holds |
|---|---|
| `src/allocation_agent/activity/week.py` | ISO week and search window (mirrors `kernel.WeekID` and `slack.Query`) |
| `src/allocation_agent/activity/contract.py` | pydantic models for the file contract and the agent output |
| `src/allocation_agent/activity/normalize.py` | `normalize_messages`: which report-source messages are written |
| `src/allocation_agent/activity/sources.py` | report scope, the two search queries, merge and `(channel_id, ts)` dedup |
| `src/allocation_agent/activity/slack_mcp.py` | SDK session: preflight, tool allowlist, prompt, result validation |
| `src/allocation_agent/activity/store.py`, `summary.py`, `collector.py`, `config.py`, `cli.py` | atomic write, dry-run summary, orchestration, settings, CLI |
| `src/allocation_agent/llm/` | `port.py` (`LLMClient`, `LLMError` family), `registry.py` (`register`, `create`), `providers.py` (the place to add a provider), `deepseek.py` (DeepSeek adapter, httpx), `settings.py` |
| `src/allocation_agent/report/` | `models.py` (`ReportLine`, strict LLM answer, `ParserInfo`), `prompt.py` (system prompt, `FEW_SHOT_EXAMPLES`: `TODO(owner)`), `extractor.py` (`ReportExtractor`), `summary.py` (dry-run minutes table) |
| `tests/` | pytest, `src` layout, `importlib` import mode ([testing](../../docs/architecture/testing.md)); the SDK client and the LLM are faked (`tests/llm_fakes.py`, `httpx.MockTransport`), and an autouse fixture refuses every real HTTP request, so nothing calls Claude, Slack or DeepSeek |

Tooling: Python 3.12, uv pinned to 0.12.21 with `uv.lock` (`uv sync --locked`). This follows the ADR-0007 ruling, where uv is CONDITIONAL and pip-tools is APPROVED; the owner still has to confirm the choice. ruff lints and formats, pytest runs the tests. `make ci` here runs `lint` and `test`. Commands: [build and test](../../docs/commands/build-and-test.md#python-service-allocation-agent), [lint and format](../../docs/commands/lint-and-format.md#python-ruff), [activity collection](../../docs/commands/activity-collection.md).

## LLM report parsing [built]

Decision: [ADR-0014](../../docs/adr/0014-llm-provider-port-with-deepseek-for-report-parsing.md) (Proposed).

With `report.parser: llm`, `collect` runs these steps:

1. collect;
2. normalize;
3. send each summary message to the configured LLM, one call per message;
4. write contract version 2, atomically.

The Go core then reads `report_lines` instead of parsing the text itself.

- **Data residency:** every summary's full text goes to the provider. DeepSeek stores data in the People's Republic of China. Confirm that company policy allows this before you turn it on.
- **Fail-stop:** any LLM error or invalid line stops the run with exit 4. No file is written, there is no fallback, and minutes are never rounded.
- **Line rules:** these mirror the Go core:
  - minutes are a positive multiple of 15;
  - the date is a Monday-Friday of the week;
  - the project is not blank.
- **Errors** name the message `ts` and channel id only, never its text or notes.
- **`--dry-run`** with `llm` does call the LLM. It prints the message counts plus minutes per day and project, never notes or message text, and writes nothing.
- **Provider:** DeepSeek over httpx (`POST https://api.deepseek.com/chat/completions`). It uses JSON mode, thinking off, temperature 0, and a configurable timeout. 429 and 5xx are retried at most `llm.max_retries` times. The key is `DEEPSEEK_API_KEY`, from the environment only.
- **Another provider:** add `src/allocation_agent/llm/<name>.py`, import it in `llm/providers.py`, and set `llm.provider`.

Settings live in `services/timesheet-core/config/config.yaml`. `report.parser` is read from that file only, with no flag or environment override. The Go core reads the same key to decide whether it expects contract v1 or v2, so the two services must never disagree. The `llm.*` keys follow the precedence flag > environment > yaml; Go ignores them.

```yaml
report:
  parser: manual        # manual | llm
llm:                    # read only when report.parser is llm
  provider: deepseek
  model: deepseek-flash # empty: the provider default (deepseek-flash)
  base_url: ""          # empty: the provider default (https://api.deepseek.com)
  timeout_seconds: 60
  max_retries: 2        # 0-10; retries 429 and 5xx only
```

| Setting | Flag | Environment |
|---|---|---|
| `report.parser` | (none: config.yaml only) | (none) |
| `llm.provider` | `--llm-provider` | `LLM_PROVIDER` |
| `llm.model` | `--llm-model` | `LLM_MODEL` |
| `llm.base_url` | (none) | `LLM_BASE_URL` |
| `llm.timeout_seconds` | (none) | `LLM_TIMEOUT_SECONDS` |
| `llm.max_retries` | (none) | `LLM_MAX_RETRIES` |

DeepSeek's current models are `deepseek-flash` and `deepseek-v4-pro` (checked 2026-10-10). `deepseek-chat` is no longer listed, so a config that still names it fails with HTTP 400 or 422 and exit 4.

`TODO(owner)`: add 2-3 real, redacted summaries and their expected lines to `FEW_SHOT_EXAMPLES` in `src/allocation_agent/report/prompt.py`. The parser works without them.

## AI allocation agent [planned: P3]

The AI allocation agent as its own service ([ADR-0007](../../docs/adr/0007-grpc-contracts-now-split-later.md)).

- Language: Python with LangGraph and the official Anthropic Python SDK.
- Interface: serves `PolicyService` from `api/proto/timesheetbot/allocation/v1/policy.proto` over gRPC.
- Caller: the Go core, through `services/timesheet-core/internal/allocation/infrastructure/grpcpolicy`. Its proposals still pass the `Timesheet` aggregate's validation, and failures fall back to `RulePolicy`.
- Secrets: only `ANTHROPIC_API_KEY`. No Slack, Jira or Tempo tokens.
- Data: reads prompts and skills from the agent workspace volume.

No allocation-agent code lives here until the Phase 3 plan has `status: final` (production-readiness review: packaging tool, lockfile, LangGraph evidence).
