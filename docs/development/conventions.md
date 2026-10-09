# Project conventions

Rules that apply across the codebase. Coding rules with their official sources are in `.claude/skills/coding-standards/SKILL.md`. File-type rules (domain, adapters, migrations, tests, containers, docs) are in `.claude/rules/` and load automatically when a matching file is touched.

## Non-negotiable

- **Secrets** come only from environment variables: `SLACK_USER_TOKEN`, `TEMPO_API_TOKEN`, `JIRA_BASE_URL`, `JIRA_EMAIL`, `JIRA_API_TOKEN`, `DATABASE_URL` (more per phase in [configuration](../operations/configuration.md)).
  - In this project the owner may keep the tokens in the repository-root `.env`, which is gitignored and written only by the owner. Exporting them in `~/.zshrc` still works, and the shell wins over `.env` ([where tokens can live](../operations/configuration.md#where-tokens-can-live)).
  - Agents never read, write, grep or print `.env` or any token value. `.claude/settings.json` denies Read, Edit and Write on `**/.env`, `.env.local`, `.env.*.local`, `.env.development`, `.env.production` and `.env.test`.
  - The committed template `.env.example` holds the token keys with empty values. The user-level settings deny `**/.env.*`, so the owner edits `.env.example` by hand.
  - `config.Env` reports missing names, never values. `docker-compose.yml` passes them through without values. The `guard-secrets` hook blocks credential patterns in any write.
- **Owner-only code**: `scaleDay` in `internal/allocation/domain/rule.go` is marked `TODO(owner)`. Only the owner implements it. `TestRulePolicyScalesOverCapDay` is skipped with a `TODO(owner)` message, and the owner removes the skip when `scaleDay` lands. The session-start hook lists every `TODO(owner)`.
- **Publishing**: nothing is committed, pushed, or posted to GitHub without the owner approving the exact content. `git commit`, `git push` and `gh` writes prompt for approval; the `guard-bash` hook blocks AI attribution, `--no-verify`, `--author` and force pushes.
- **Verification**: an external claim (API behaviour, library, best practice) is used only after `official-source-verifier` returns VERIFIED.
- **Plans**: coders execute only plans with `status: final`, which requires the production-readiness review.
- **Commands**: if you use a command that is not yet in [`docs/commands/`](../commands/README.md), add it to the file for its purpose in the same change, with where to run it, what it does, when to use it, and the expected result. If no purpose file fits, create `docs/commands/<purpose>.md` and add it to the index in `docs/commands/README.md`.

## Configuration and time

- Non-secret settings go in `services/timesheet-core/config/config.yaml` (gitignored); `config.example.yaml` next to it is the template.
- Go paths in the docs, `.claude/agents` and `.claude/skills` (`internal/...`, `cmd/...`, `migrations/`, `test/...`) are relative to `services/timesheet-core/`. Run Go tools there, or from the root with `go -C services/timesheet-core ...` (other root-safe forms: [commands](../commands/README.md)).
- Every Go test lives in `services/timesheet-core/test/<kind>/<area>/`, never beside the code, and is black-box (`package x_test`). This deviates from the Go convention by owner choice ([ADR-0009](../adr/0009-tests-in-separate-tree.md)); the layout and the folder mapping are in [testing](../architecture/testing.md#layout-by-test-type).
- Calendar days are computed once, at the edge (the Slack collector), in the configured timezone (default `Asia/Ho_Chi_Minh`). After that, `WorkDay` is a timezone-free date.
- The binary embeds `time/tzdata` because the distroless runtime image has no zoneinfo.

## Decisions and docs

- Decisions are ADRs in `docs/adr/`. Supersede an accepted ADR with a new one instead of editing it.
- The docs layout is in [docs/README.md](../README.md). A feature is not done until its feature doc and learning guide exist.

## Where Claude Code configuration lives

| What | Where | Maintained by |
|---|---|---|
| Agent team | `.claude/agents/` | owner |
| Preloaded conventions | `.claude/skills/{ddd-conventions,review-checklist,go-testing-conventions,tech-selection,architecture-references}` | owner, technical-architect |
| Coding standards and their sources | `.claude/skills/coding-standards/` | `standards-curator` (`/update-standards`) |
| Path-scoped rules | `.claude/rules/` | owner |
| Hooks (format, guards, routing, build gate) | `.claude/hooks/`, registered in `.claude/settings.json`, except `verify-build.sh`, which is a `Stop` hook in the frontmatter of the `go-coder`, `test-engineer` and `qa-engineer` agents | owner |
| Agent memory | `.claude/agent-memory/<agent>/MEMORY.md` | each agent |
