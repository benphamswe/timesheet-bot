# Threat model

Scope: single user, laptop-hosted, four external SaaS APIs. Method: data-flow review per trust boundary (STRIDE-lite).

## Trust boundaries

1. Laptop <-> Slack, Jira, Tempo, Anthropic (TLS, bearer/basic tokens).
2. Untrusted content -> domain: Slack message text, Jira fields, AI output [P3].
3. Learning loop [P4]: feedback -> lessons -> next week's prompt.
4. Core <-> allocation-agent over gRPC on the compose network (planned P3, [ADR-0007](../adr/0007-grpc-contracts-now-split-later.md)). Agent output is untrusted: it passes the `Timesheet` aggregate's validation, and an agent failure falls back to `RulePolicy` ([ADR-0004](../adr/0004-allocation-policy-fallback-chain.md)). The agent holds only `ANTHROPIC_API_KEY`; the Slack, Jira and Tempo tokens stay in the Go core.

## Threats and mitigations

Condition ids (C1-C21) refer to the [ADR-0007 production-readiness review](../adr/reviews/0007-production-readiness.md#7-conditions-for-the-phase-3-plan).

| ID | Threat | Mitigation | Status |
|---|---|---|---|
| T1 | Token leakage via files, logs, errors, commits | Env-only secrets, from one `.env` per service (owner-only, gitignored; ADR-0013) or `~/.zshrc`; `config.Env` names variables only; compose `env_file` per service, no secret in `environment:`; `printenv`/`env` denied for agents; `config.yaml` gitignored | built |
| T2 | Tampered or malformed API responses create bad worklogs | Kernel constructors, aggregate invariants, Jira 404 drop, Tempo id required | built |
| T3 | Duplicate or lost worklogs on crash/retry | Per-entry id persistence, outbox in save transaction | built |
| T4 | Prompt injection in Slack text or Jira fields makes the agent log wrong hours; the untrusted text now crosses the gRPC hop to the allocation-agent | Agent proposals pass the same validation as rule output; user approval before any write; agent has no write tools. Across the hop: `grpcpolicy` validates the proto `Proposal` before mapping (entry count, issue keys only from the request, days inside the week, length caps) and renders reasoning to Slack as plain text (C7); request content travels as structured fields, the agent puts it in provenance-labelled blocks, its only output is `submit_proposal`, and injection fixtures are in its tests (C8) | planned: P3 |
| T5 | Memory poisoning: injected text becomes a persistent lesson | Lessons only from user feedback signals (reject reason, thread correction, Tempo edit), never raw Slack text; 50-lesson cap; new skills require approval | planned: P4 |
| T6 | Runaway AI cost; the cost bounds now live in two processes (Go core and allocation-agent) | Turn limit, token budget, timeout per run; `agent_runs` accounting. Across both processes: one retry layer, with no gRPC `retryPolicy`, Anthropic SDK `max_retries=1` and an explicit timeout, and a second agent attempt only after a validation rejection (C4); explicit LangGraph `recursion_limit` (C20); `max_tokens` per call and a token budget per request, with usage recorded in `agent_runs` (C21) | planned: P3 |
| T7 | Over-privileged Slack app | User token limited to read scopes; bot token to chat/im write | planned: P2 |
| T8 | Local database exposure | Postgres bound to 127.0.0.1; holds no secrets | built |
| T9 | Lateral movement: a prompt-injected or compromised allocation-agent container reaches Postgres or other services on the shared compose network | Separate compose networks, `core` (app, db) and `agent` (app, allocation-agent), so the agent cannot reach `db` and publishes no port (C6); agent container least privilege: only `ANTHROPIC_API_KEY`, no `env_file`, non-root, `read_only: true`, `cap_drop: [ALL]`, `no-new-privileges`, workspace mounted read-only (C9) | planned: P3 |
| T10 | Data exfiltration: Slack text leaves through third-party tracing (LangSmith) or verbose logs | LangSmith tracing stays off: no `LANGSMITH_*` variables in compose, and the agent refuses to start if `LANGSMITH_TRACING` is true (C18); logs scrubbed of Slack text beyond what is needed (C5) | planned: P3 |

## Residual risks

- A compromised laptop exposes all tokens (accepted for a single-user tool).
- Tokens can sit in plaintext in `services/timesheet-core/.env` and `services/allocation-agent/.env`, inside the repository folder and inside each service's Docker build context, if the owner chooses that over `~/.zshrc`. Each service sees only its own file (ADR-0013). Mitigations: `.gitignore` excludes `.env` and `.env.*` at any depth (except `.env.example`); `services/timesheet-core/.dockerignore` and `services/allocation-agent/.dockerignore` exclude them from the build context, which is now the control that keeps them out of the images; `.claude/settings.json` denies agents Read, Edit and Write on `**/.env` and its local and per-environment variants, and denies `printenv` and `env`; the `guard-secrets` hook blocks credential patterns in any write. The owner keeps `.env` at `chmod 600`, and rotates a token if it ever leaks (accepted for a single-user tool).
- The collector container cannot hold a claude.ai login without exposing a subscription credential; no credential is mounted until the owner picks an option (ADR-0013). With `report.parser: llm`, report summaries leave for DeepSeek, a third party (owner confirms company policy).
- Slack clicks while offline are lost (CLI fallback documented).
- Compose cannot limit the allocation-agent's egress to `api.anthropic.com` without an egress proxy, so the `agent` network allows any outbound host (accepted for a single-user laptop; planned: P3).
