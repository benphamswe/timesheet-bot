# Claude Code tooling

How to test the hooks in `.claude/hooks/` without starting a session, and how to invoke the project's slash commands. Hook registration is in `.claude/settings.json`, except `verify-build.sh`, which is a `Stop` hook in the frontmatter of the `go-coder`, `test-engineer` and `qa-engineer` agents.

Run every command on this page from the repository root.

## Testing hooks

Each hook reads one JSON object on stdin, the same shape Claude Code sends. Exit 0 lets the action through; exit 2 blocks it and shows stderr to Claude.

Pipe JSON with `printf '%s'`, never `echo`. In zsh, `echo` turns `\n` into a real newline, `jq` rejects the payload (`parse error: Invalid string: control characters ...`), and hooks that do not use `set -e`, such as `route-request.sh`, then exit 0 with no routing, which looks like "no match". Confirmed 2026-10-09.

### Syntax check every hook

```bash
for f in .claude/hooks/*.sh; do bash -n "$f" && echo "ok $f"; done
```

- **What it does**: parses each script without running it.
- **When to use it**: after any hook edit.
- **Expected**: one `ok` line per hook (six on 2026-10-09).

### `route-request.sh` (UserPromptSubmit)

```bash
printf '%s' '{"prompt":"fix the failing test\nand update the docs"}' | bash .claude/hooks/route-request.sh
printf '%s' '{"prompt":"/learn what is DDD"}' | bash .claude/hooks/route-request.sh   # slash commands: no output
```

- **What it does**: prints the agents the orchestrator should dispatch for the prompt.
- **When to use it**: after changing a routing pattern, to check that it matches what you meant and nothing else.
- **Expected**: for the first prompt, the routing header and three lines: `superpowers:systematic-debugging ...`, `test-engineer ...`, `tech-writer`. The slash command prints nothing. Both confirmed 2026-10-09.

### `guard-bash.sh` (PreToolUse, Bash)

```bash
printf '%s' '{"tool_input":{"command":"git status --short"}}' | bash .claude/hooks/guard-bash.sh; echo "rc=$?"
printf '%s' '{"tool_input":{"command":"git com''mit -m x --no-verify"}}' | bash .claude/hooks/guard-bash.sh; echo "rc=$?"
printf '%s' '{"tool_input":{"command":"git pu''sh --force origin main"}}' | bash .claude/hooks/guard-bash.sh; echo "rc=$?"
```

- **What it does**: blocks AI attribution in commit messages, `--no-verify`, `--author`, and force pushes.
- **Expected**: `rc=0`, then `Blocked: --no-verify and --author are not allowed ...` with `rc=2`, then `Blocked: force push rewrites published history ...` with `rc=2`. Confirmed 2026-10-09.
- **Gotcha**: when you run this test from Claude Code, the live `guard-bash` hook checks your whole test command too. A literal `git commit` next to a blocked word blocks the test itself. Split the words with `''` as above, so the shell joins them but the live hook does not match.

### `guard-secrets.sh` (PreToolUse, Edit and Write)

```bash
jq -n --arg c "token: xox""b-000000000000" '{tool_input:{content:$c}}' | bash .claude/hooks/guard-secrets.sh; echo "rc=$?"
jq -n --arg c 'url: ${DATABASE_URL}' '{tool_input:{new_string:$c}}' | bash .claude/hooks/guard-secrets.sh; echo "rc=$?"
```

- **What it does**: blocks content that looks like a credential (Slack, Anthropic and Atlassian tokens, private keys, DSNs with a password). The local compose DSN `timesheet:timesheet@` is allowed.
- **Expected**: `Blocked: content matches a credential pattern (...)` with `rc=2` for the fake token, `rc=0` for the variable name. Confirmed 2026-10-09; the local DSN also passed.
- **Gotcha**: build fake tokens from split strings (`"xox""b-..."`) so the test file or doc you write is not itself blocked. The fake value is all zeros and is not a credential.

### `format-go.sh` (PostToolUse, Edit and Write)

```bash
printf '%s' '{"tool_input":{"file_path":"README.md"}}' | bash .claude/hooks/format-go.sh; echo "rc=$?"
```

- **What it does**: formats an edited `.go` file with `golangci-lint fmt` (falling back to `gofmt -w`); any other file is ignored.
- **Expected**: `rc=0`, no change. Confirmed 2026-10-09. Passing a real `.go` path rewrites that file, so do that only on a file you own.

### `session-context.sh` (SessionStart)

```bash
CLAUDE_PROJECT_DIR=$PWD bash .claude/hooks/session-context.sh
```

- **What it does**: prints the branch, the count of uncommitted files, every `TODO(owner)` location, the latest plan, and a reminder when coding-standards sources are older than 30 days.
- **Expected**: a `## timesheet-bot status` block, for example `- owner-only TODOs (do not implement): services/timesheet-core/internal/allocation/domain/rule.go:83`. Confirmed 2026-10-09.

### `verify-build.sh` (Stop, in agent frontmatter)

```bash
printf '%s' '{"stop_hook_active":false}' | CLAUDE_PROJECT_DIR=$PWD bash .claude/hooks/verify-build.sh; echo "rc=$?"
```

- **What it does**: runs `go build ./...`, `go vet ./...` and `go vet -tags=integration ./...` in `services/timesheet-core`, and blocks the agent from finishing if any fails.
- **Expected**: `rc=0` with no output. Confirmed 2026-10-09. On failure it prints `Build or vet failed; fix before finishing:` and the last 20 lines, with `rc=2`.

## Slash commands

Type these in a Claude Code session at the repository root. They run agents, so they were **not run while documenting**; arguments are taken from each skill's `argument-hint`.

| Command | Arguments | What it does | Defined in |
|---|---|---|---|
| `/sdlc` | `<feature or phase> [--from design\|implement\|review]` | Runs the full SDLC through `sdlc-orchestrator`. Without `--from` (or with `--from design`) it stops at GATE 1 for plan approval. `--from implement` needs a `status: final` plan. `--from review` reviews and tests uncommitted changes. | `.claude/skills/sdlc/SKILL.md` |
| `/learn` | `<question> [--feature FE-nnn]` | `learning-coach` answers from the code and verified sources and saves the answer to `docs/learn/`. | `.claude/skills/learn/SKILL.md` |
| `/backlog-sync` | `[EP-nnn \| FE-nnn \| SP-nn \| all]` (default `all`) | Drafts the GitHub Issues mirror in `docs/backlog/.sync/draft.md`, shows it with its sha256, and publishes only after you approve that draft in a new message. | `.claude/skills/backlog-sync/SKILL.md` |
| `/update-standards` | `add <patterns or topics> \| verify \| audit` (default: `verify`, then `audit`) | `standards-curator` updates `.claude/skills/coding-standards/` from official sources. Run `verify` monthly. | `.claude/skills/update-standards/SKILL.md` |
| `/review-fanout` | optional `{contexts: ["timesheet", ...]}` | Groups uncommitted files by bounded context, runs `ddd-reviewer` and `standards-reviewer` per group, and has two skeptics try to refute each finding. Returns confirmed, refuted and uncovered findings. | `.claude/workflows/review-fanout.js` |

Examples:

```text
/sdlc Phase 2
/sdlc Phase 2 --from implement
/learn why does the outbox exist --feature FE-001
/backlog-sync SP-01
/update-standards verify
/review-fanout
```

`/sdlc`, `/backlog-sync` and `/update-standards` set `disable-model-invocation: true`, so only you can start them; Claude cannot call them on its own.
