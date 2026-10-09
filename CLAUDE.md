# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

timesheet-bot fills Jira Tempo timesheets from the owner's Slack activity. It is a single-user tool that must stay scale-ready (ADR-0006), organised as a monorepo of services (ADR-0008): `services/timesheet-core` (Go, built) and `services/allocation-agent` (Python + LangGraph, planned P3), with gRPC contracts in `api/proto`. Go paths in the docs are relative to `services/timesheet-core/`; run Go commands there. Phase 1 (CLI) is built; Phases 2-4 are planned in `docs/superpowers/specs/2026-10-09-timesheet-bot-design.md`.

## Read before you work

| Task | Read |
|---|---|
| Any code change | `docs/architecture/project-structure.md`, `docs/architecture/code-map.md`, `docs/development/conventions.md` |
| Architecture or design | `docs/architecture/` (overview, bounded contexts, scalability), `docs/adr/` |
| An adapter (Slack, Jira, Tempo, Anthropic) | `docs/api/external-integrations.md`, `docs/architecture/network.md` |
| Tests | `docs/architecture/testing.md` |
| Security | `docs/security/threat-model.md` |
| Backlog, stories, sprint | `docs/backlog/README.md` |
| Running and commands | `docs/commands/` (first-time setup: `docs/development/setup.md`) |
| How the agent team works | `docs/development/sdlc-workflow.md` |

Coding rules and conventions are preloaded into the agents that need them (`.claude/skills/`). File-type rules load automatically from `.claude/rules/` when you touch matching files.

## Which agent to call

Act only as an orchestrator in the main session. Do not read or search code, investigate, edit files, or run builds or tests there; send all of that to subagents, including small edits and quick lookups.

- Do only these jobs in the main session: talk with the owner (clarifying questions, approvals), split the request and dispatch agents, relay the agents' conclusions, and run the approval step before any commit, push or publish.
- Split the request into independent parts and dispatch one agent per part **in a single message**, so they run concurrently.
- Use the specialist from the table below when one fits; otherwise use `general-purpose`.
- Keep only the agents' conclusions in the main context. The `route-request` hook adds the match to each prompt.

| Request | Agent or command |
|---|---|
| New feature or phase | `/sdlc <feature>` |
| Bug or failure | `superpowers:systematic-debugging`, then `go-coder` + `test-engineer` |
| Review changes | `/review-fanout` |
| Tests | `test-engineer`, `qa-engineer` |
| Security / networking / reliability | `security-engineer` / `networking-engineer` / `sre-engineer` |
| Architecture or design | `solution-architect`, `architecture-reviewer`, `technical-architect` |
| Library, version, upgrade | `technical-architect` + the tech-review agents |
| CI, Docker, release | `devops-engineer` |
| Backlog, stories, sprint | `business-analyst`; `/backlog-sync` for GitHub |
| Docs | `tech-writer` |
| "Explain", "what is", "why" | `/learn <question>` |
| Any external fact | `official-source-verifier` |
| Anything else | `general-purpose` |

## Non-negotiable

Details: `docs/development/conventions.md`.

- Secrets come from environment variables: the owner's gitignored root `.env` (this project only) or `~/.zshrc`. Agents never read, write or print `.env` or token values.
- Never implement code marked `TODO(owner)`.
- Never commit, push or publish without the owner approving the exact content.
- Use an external claim only after `official-source-verifier` returns VERIFIED. Code only from plans with `status: final`.
- New commands go into `docs/commands/<purpose>.md`, in the same change that first uses them.

## Verify

`make ci` from the root runs it in every service (lint, test, race, vuln, build). Other commands: `docs/commands/README.md`.
