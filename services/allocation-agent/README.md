# allocation-agent

The Python service of timesheet-bot. It has two roles:

| Role | Phase | Status |
|---|---|---|
| Slack activity collector (`collect`) | P1.5 | [built] code and tests; [planned] first live run, after the Slack connector is authenticated and `normalize_messages` (`TODO(owner)`) is written |
| AI allocation agent (`PolicyService` over gRPC) | P3 | [planned: P3] |

## Slack activity collector [built]

Decision: [ADR-0011](../../docs/adr/0011-collect-slack-activity-through-claude-slack-mcp-connector.md) (Proposed).

The owner cannot create a Slack app, so there is no Slack token. `collect` drives the Claude Code CLI through `claude-agent-sdk` on your claude.ai login, and the claude.ai Slack connector searches your own messages for one ISO week. The result is written to `<out-dir>/<week>.json`, and the Go core reads that file when `activity.source` is `file`.

- Contract: [`api/activity/activity-week.v1.schema.json`](../../api/activity/activity-week.v1.schema.json), schema version 1.
- Search window: the Go collector's `from:<@you> after:<Sunday> before:<Saturday>`. The dates are exclusive, so the search covers Monday to Friday. Messages are then clamped to Monday to Friday in your timezone.
- Tools: only the connector's read/search tools are allowed. Built-in tools and every other connector tool (send, react, schedule, canvas, upload, ...) are denied. A preflight that sends no prompt checks that the connector is `connected` before any Claude usage.
- Output: the file is written atomically with mode 0600 to `<repo root>/data/activity/` by default. `data/activity/*.json` is gitignored. `--dry-run` prints counts per day and per channel only, never message text.

### Before the first run

1. Log in to Claude Code with your claude.ai account (`claude`, then `/login`). `ANTHROPIC_API_KEY` must not be set in that shell, or the connector does not load.
2. Authenticate the Slack connector: claude.ai > Settings > Connectors, or `/mcp` in Claude Code. On 2026-10-10 it showed `needs-auth`.
3. Set `slack_user_id` and `timezone` in `services/timesheet-core/config/config.yaml`; both services read that one file. `SLACK_USER_ID` and `TIMEZONE` in the environment override it, and `--slack-user-id` and `--timezone` override both.
4. Implement `normalize_messages` in `src/allocation_agent/activity/normalize.py` (`TODO(owner)`), then remove the skip in `tests/unit/activity/test_normalize.py`. Until then, `collect` exits with code 3.

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
- `2`: bad flag, week or config.
- `3`: `normalize_messages` is not implemented yet.

### Layout

| Path | Holds |
|---|---|
| `src/allocation_agent/activity/week.py` | ISO week and search window (mirrors `kernel.WeekID` and `slack.Query`) |
| `src/allocation_agent/activity/contract.py` | pydantic models for the file contract and the agent output |
| `src/allocation_agent/activity/normalize.py` | `normalize_messages`: `TODO(owner)` |
| `src/allocation_agent/activity/slack_mcp.py` | SDK session: preflight, tool allowlist, prompt, result validation |
| `src/allocation_agent/activity/store.py`, `summary.py`, `collector.py`, `config.py`, `cli.py` | atomic write, dry-run summary, orchestration, settings, CLI |
| `tests/` | pytest, `src` layout, `importlib` import mode ([testing](../../docs/architecture/testing.md)); the SDK client is faked, nothing calls Claude or Slack |

Tooling: Python 3.12, uv pinned to 0.12.21 with `uv.lock` (`uv sync --locked`). This follows the ADR-0007 ruling, where uv is CONDITIONAL and pip-tools is APPROVED; the owner still has to confirm the choice. ruff lints and formats, pytest runs the tests. `make ci` here runs `lint` and `test`. Commands: [build and test](../../docs/commands/build-and-test.md#python-service-allocation-agent), [lint and format](../../docs/commands/lint-and-format.md#python-ruff), [activity collection](../../docs/commands/activity-collection.md).

## AI allocation agent [planned: P3]

The AI allocation agent as its own service ([ADR-0007](../../docs/adr/0007-grpc-contracts-now-split-later.md)).

- Language: Python with LangGraph and the official Anthropic Python SDK.
- Interface: serves `PolicyService` from `api/proto/timesheetbot/allocation/v1/policy.proto` over gRPC.
- Caller: the Go core, through `services/timesheet-core/internal/allocation/infrastructure/grpcpolicy`. Its proposals still pass the `Timesheet` aggregate's validation, and failures fall back to `RulePolicy`.
- Secrets: only `ANTHROPIC_API_KEY`. No Slack, Jira or Tempo tokens.
- Data: reads prompts and skills from the agent workspace volume.

No allocation-agent code lives here until the Phase 3 plan has `status: final` (production-readiness review: packaging tool, lockfile, LangGraph evidence).
