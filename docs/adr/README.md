# Architecture decision records

Each ADR records one decision that is hard to reverse. To add one, copy [0000-template.md](0000-template.md), take the next free number, set the status to `Proposed`, and add a row here. Only the owner moves an ADR to `Accepted`.

| ADR | Decision | Status |
|---|---|---|
| [ADR-0001](0001-ddd-package-by-bounded-context.md) | Organise code by bounded context with Clean Architecture layers | Accepted |
| [ADR-0002](0002-postgres-with-transactional-outbox.md) | Use Postgres with a transactional outbox and per-entry submit idempotency | Accepted |
| [ADR-0003](0003-slack-socket-mode-for-approval.md) | Receive Slack interactions over Socket Mode | Accepted (implementation: P2) |
| [ADR-0004](0004-allocation-policy-fallback-chain.md) | Allocate through an ordered policy chain validated by the aggregate | Accepted |
| [ADR-0005](0005-go-1-26-and-embedded-assets.md) | Target Go 1.26 and embed migrations and tzdata in a distroless image | Accepted |
| [ADR-0006](0006-personal-tool-scale-ready-aws-later.md) | Build a single-user tool with scale-ready seams; AWS is the future SaaS target | Accepted |
| [ADR-0007](0007-grpc-contracts-now-split-later.md) | Define gRPC contracts at context boundaries now; run in-process until a split is needed; the AI agent is a Python LangGraph service | Accepted (2026-10-09; conditions C1-C21 are mandatory for the Phase 3 plan) |
| [ADR-0008](0008-monorepo-service-layout.md) | Organise the repository as a monorepo of services with orchestration-only root | Accepted (2026-10-09) |
| [ADR-0009](0009-tests-in-separate-tree.md) | Keep every Go test in one `test/` tree, black-box only; a deliberate deviation from the Go convention | Accepted (2026-10-09, owner) |
| [ADR-0010](0010-add-mavenlink-as-a-second-submit-target.md) | Add Mavenlink as a second submit target through its REST API, with bearer or browser-cookie auth | Proposed |
| [ADR-0011](0011-collect-slack-activity-through-claude-slack-mcp-connector.md) | Collect Slack activity through Claude's Slack MCP connector into a versioned weekly JSON file | Proposed |
| [ADR-0012](0012-report-mode-slack-summaries-to-mavenlink-without-jira.md) | Report mode: each line of the owner's Slack timesheet summaries becomes one Mavenlink entry, without Jira; exact duplicates dropped, over-cap days warned, Tempo excluded | Proposed |
| [ADR-0013](0013-per-service-env-files.md) | Keep one `.env` per service (`services/timesheet-core/.env`, `services/allocation-agent/.env`); compose loads each with `env_file` and passes no secret through `environment:`; `collect` refuses Anthropic credentials; collector container credentials pending | Proposed |
| [ADR-0014](0014-llm-provider-port-with-deepseek-for-report-parsing.md) | Parse report summaries with an LLM behind a provider port (Strategy + registry); DeepSeek first; fail-stop on any LLM error (exit 4, no file); activity file v2 adds `parser` and `report_lines`; summaries go to a China-hosted third party, pending the owner's policy check | Proposed |

## Reviews

| Review | Covers |
|---|---|
| [0007-production-readiness.md](reviews/0007-production-readiness.md) | Architecture board and production-readiness review of ADR-0007, conditions C1-C21, and finding S1 (source of ADR-0008) |
