# ADR-0007: Define gRPC contracts at context boundaries now; run in-process until a split is needed; the AI agent is a Python LangGraph service

- Status: Accepted (2026-10-09, owner; conditions C1-C21 from the production-readiness review are mandatory for the Phase 3 plan)
- Date: 2026-10-09
- Deciders: owner
- Refines: [ADR-0006](0006-personal-tool-scale-ready-aws-later.md) (single-user and scale-ready still apply). Replaces section 7 of the design spec (an in-process Go `claudeagent`).
- Related: [ADR-0008](0008-monorepo-service-layout.md) records the repository layout (`services/`, orchestration-only root); this ADR covers only `api/proto` and `allocation-agent`.

## Context

The owner wants microservices with gRPC, so that any bounded context can become its own service later. The owner also wants the Phase 3 AI agent built with LangGraph. LangGraph's official libraries exist only for Python (`langgraph`) and TypeScript (`langgraphjs`) (github.com/langchain-ai, checked 2026-10-09), so a LangGraph agent cannot run inside the Go process.

## Decision

1. **Contracts first.** Every boundary between bounded contexts gets a versioned protobuf contract under `api/proto/timesheetbot/<context>/v1/`:
   - `allocation/v1/policy.proto`: `PolicyService.Allocate(AllocateRequest) returns (Proposal)`, mirroring `allocation/domain.Policy`.
   - `activity/v1/source.proto`: `ActivitySourceService.Collect`, mirroring `application.ActivitySource`.
   - Adapters to third-party systems (Tempo, Jira, Slack) stay plain Go ports with no proto, because they are not our services. *Open question 1.*
2. **In-process by default.** Each Go context keeps calling the others through Go interfaces. A context is split out only by swapping, in `cmd/timesheet/wire.go`, the in-process implementation for a generated gRPC client adapter. Callers do not change.
3. **The AI agent is the first and only remote service for now:** `services/allocation-agent/`, in Python with LangGraph and the official Anthropic Python SDK, serving `PolicyService` over gRPC. The Go core reaches it through `internal/allocation/infrastructure/grpcpolicy` (a `Policy` adapter). Its proposals still pass through the `Timesheet` aggregate's validation and the policy fallback chain ([ADR-0004](0004-allocation-policy-fallback-chain.md)), so an agent failure falls back to `RulePolicy`.
4. **Contract hygiene in CI:** `buf lint` (style guide) and `buf breaking` (no breaking changes on `v1`), plus generated code checked by `buf generate` with no diff.
5. **Runtime:** a second container in `docker-compose.yml` on the internal compose network, with no published port. Only the Go core has secrets for Slack, Jira and Tempo. The agent gets only `ANTHROPIC_API_KEY` and the read-only data it is sent.

## Technology decisions (evidence collected 2026-10-09)

| Tech | Version | Evidence | Verdict (production-readiness review) |
|---|---|---|---|
| `google.golang.org/grpc` | v1.84.0 (proxy.golang.org latest, 2026-09-17) | Apache-2.0; OpenSSF Scorecard 8.5 (deps.dev); its `go.mod` needs `go 1.25.0`, ours is `go 1.26.0` | CONDITIONAL: deadlines, retries and message limits need explicit settings (C3, C4, C12) |
| `google.golang.org/protobuf` | v1.36.12 (2026-08-10) | BSD-3-Clause; Scorecard 5.2 | APPROVED |
| buf CLI | v1.73.0 (2026-09-11, GitHub releases and proxy agree) | Apache-2.0; Scorecard 5.1; lint and breaking docs live (buf.build/docs); releases ship `sha256.txt` and `sha256.txt.minisig` | APPROVED: C15 pins the version and verifies the checksum |
| `langgraph` (PyPI) | 1.2.14 (2026-10-06), the minimum allowed version | MIT (PyPI, deps.dev, GitHub); requires Python 3.10 or later; no Scorecard on deps.dev, so the review filled P3 with the organisation-wide security policy, published advisories with patched versions, and weekly releases. [GHSA-3fx2-cqw9-xr3c](https://github.com/langchain-ai/langgraph/security/advisories/GHSA-3fx2-cqw9-xr3c) is fixed in 1.2.14; [GHSA-fjqc-hq36-qh5p](https://github.com/langchain-ai/langgraph/security/advisories/GHSA-fjqc-hq36-qh5p) is fixed in `langgraph-checkpoint` 4.1.1, but 1.2.14 still allows `langgraph-checkpoint>=4.1.0` | CONDITIONAL, resolved by the review: `langgraph>=1.2.14` and `langgraph-checkpoint>=4.1.1` in the lockfile (C20); LangSmith tracing off (C18); no persistent checkpointer (C19) |
| `grpcio`, `grpcio-tools` (PyPI) | 1.84.0 (2026-09-14) | Apache-2.0; gRPC core Scorecard 6.6 | APPROVED |
| `grpcio-health-checking` (PyPI) | 1.84.0 (2026-09-14) | Apache-2.0; Scorecard 6.6 (`grpc/grpc`); requires Python 3.10 or later; no advisories (PyPI) | APPROVED, for the `grpc.health.v1` service (C13) |
| `protobuf` (PyPI) | 7.36.2 (2026-09-17) | Package licence BSD-3-Clause; the source repository `protocolbuffers/protobuf` has Scorecard **4.8**, below the 5.0 bar; deps.dev lists the project licence as "non-standard"; `grpcio-tools` 1.84.0 requires `protobuf>=7.35.1,<8.0.0` | CONDITIONAL: P3 is below 5.0, accepted because it is Google's official protobuf runtime and `grpcio-tools` requires it; hash-pinned under C15 |
| `anthropic` (PyPI) | 1.12.1 (2026-10-08, GitHub release agrees) | MIT; official vendor SDK; **no Scorecard on deps.dev** for `anthropics/anthropic-sdk-python` | CONDITIONAL: P3 has no score, accepted because it is the official Anthropic SDK (P6); retries, timeout and token limits are set explicitly (C4, C21) |

Style and design references were fetched live (HTTP 200) by the production-readiness review on 2026-10-09: the protobuf style guide (protobuf.dev), Google AIP (google.aip.dev), the buf lint and breaking rules (buf.build/docs), the gRPC guides (grpc.io/docs/guides), and the LangGraph docs (docs.langchain.com).

Conditions C1 to C21 in the [production-readiness review](reviews/0007-production-readiness.md#7-conditions-for-the-phase-3-plan) are mandatory: the Phase 3 plan must carry each one as a task.

## Alternatives considered

- Microservice per bounded context now: rejected by the owner. It means distributed data and transactions for a single user.
- Go agent with no framework (the original spec): rejected by the owner's choice of LangGraph.
- No contracts until a split: rejected. Contracts now make the later split a wiring change.

## Consequences

- Positive: the agent can use the LangGraph ecosystem. Any context can split without touching callers. The contracts document the boundaries.
- Negative: two languages and two toolchains (Go and Python), protobuf code generation, a second container, and mapping code between proto messages and domain types at every boundary.
- Risks to review: prompt-injection surface unchanged, but now across a network hop (security-engineer); gRPC deadlines, retries and keepalive (networking-engineer); agent service down at Friday run time, which falls back to the rule policy (sre-engineer).

## Open questions for the review (answered)

All three questions were answered by the [production-readiness review on 2026-10-09](reviews/0007-production-readiness.md#open-questions). The questions are kept as asked; the rulings are in the review.

1. Should third-party adapter ports (Tempo, Jira, Slack) also get proto contracts? The recommendation is no (YAGNI), since they would never be our services. **Answered: no.** The review APPROVED the recommendation; no condition.
2. Should the Go core run as a gRPC server at all before a second consumer exists? The recommendation is no. **Answered: no.** The review APPROVED the recommendation and added [C2](reviews/0007-production-readiness.md#7-conditions-for-the-phase-3-plan): no `grpc.NewServer` in `services/timesheet-core` in Phase 3, enforced by the architecture tests or a lint rule.
3. Python packaging and lockfile tool for `services/allocation-agent/`: to be chosen in the plan with tech-selection evidence. **Answered: pip-tools** (APPROVED), or uv as CONDITIONAL if the owner prefers it; either way [C15](reviews/0007-production-readiness.md#7-conditions-for-the-phase-3-plan) applies (one lock tool, every transitive dependency hash-pinned).

## Reviews

- [Architecture board and production-readiness review, 2026-10-09](reviews/0007-production-readiness.md)
