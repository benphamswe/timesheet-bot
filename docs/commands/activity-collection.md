# Activity collection

Commands for the Slack activity collector in `services/allocation-agent` ([ADR-0011](../adr/0011-collect-slack-activity-through-claude-slack-mcp-connector.md)): checking the claude.ai Slack connector, collecting a week, and checking the weekly file against its contract. The collector needs no secret. It uses your Claude Code claude.ai login, and `ANTHROPIC_API_KEY` must be unset, or the connector does not load.

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
- **Expected** for `collect`: all four `unset` (confirmed 2026-10-10).

## Collect a week

```bash
make collect WEEK=2026-W41 ARGS=--dry-run           # from the root: counts per day and channel only
make collect WEEK=2026-W41                          # writes data/activity/2026-W41.json
cd services/allocation-agent && uv run --locked collect --week 2026-W41 --dry-run
uv run --locked collect --help                      # every flag
```

- **Run from**: the repository root (`make`), or `services/allocation-agent/` (`uv run`). A relative `--out-dir` is relative to the current folder; through `make` that is `services/allocation-agent/`, so pass an absolute path in `ARGS`.
- **What it does**: runs the preflight, then one Claude session that searches `from:<@you> after:<Sunday> before:<Saturday>` with only the connector's read/search tools. It validates the structured result, applies `normalize_messages`, and writes `<out-dir>/<week>.json` atomically (mode 0600). `--dry-run` writes nothing and never prints message text.
- **When to use it**: once per week before `timesheet plan` with `activity.source: file`.
- **Expected**:
  - Not run against Slack while documenting (the connector was `needs-auth`, and `normalize_messages` is `TODO(owner)`).
  - Until `normalize_messages` lands, the command exits 3 with `collect: normalize_messages is not implemented yet (TODO(owner)); see normalize.py`.
  - `make collect` without `WEEK` exits 2 with `usage: make collect WEEK=2026-W41 [ARGS=--dry-run]`.
  - `uv run --locked collect --week 2027-W53` exits 2 with `collect: week '2027-W53' does not exist`.
  - All confirmed 2026-10-10.
- **Common failures**:
  - `MCP server "claude.ai Slack" is 'needs-auth', not connected`: authenticate the connector (exit 1).
  - `... not found`: log in with claude.ai and unset `ANTHROPIC_API_KEY` (exit 1).

## Check the contract

```bash
make -n collect WEEK=2026-W41 ARGS=--dry-run
git check-ignore -v .env services/allocation-agent/.env data/activity/2026-W41.json
jq 'del(.messages) + {msg_keys: (.messages[0]|keys), n: (.messages|length)}' \
  services/timesheet-core/test/unit/adapters/file/testdata/happy/2026-W41.json
```

- **Run from**: the repository root.
- **What it does**: previews the `make collect` commands; confirms that the env files and the weekly file are gitignored (it prints the matching `.gitignore` rule and reads no file content); shows the Go reader's test fixture without message text.
- **Expected** (2026-10-10):
  - The preview prints `uv sync --locked` and `uv run --locked collect --week 2026-W41 --dry-run`.
  - `check-ignore` prints `.gitignore:3:.env` for both env paths and `.gitignore:10:data/activity/*.json` for the weekly file.
  - `jq` shows `schema_version` 1 and an extra `future_field`. The Go reader ignores unknown fields, and so does the schema. `tests/unit/activity/test_contract.py` validates this fixture against `api/activity/activity-week.v1.schema.json`.
