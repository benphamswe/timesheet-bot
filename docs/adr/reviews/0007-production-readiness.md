# Review: ADR-0007 production readiness and architecture board

- Reviewed: [ADR-0007](../0007-grpc-contracts-now-split-later.md) (status Proposed), measured against [ADR-0006](../0006-personal-tool-scale-ready-aws-later.md) and [ADR-0004](../0004-allocation-policy-fallback-chain.md)
- Date: 2026-10-09. All evidence below was fetched live on this date; the commands are in [Evidence log](#evidence-log).
- Roles: architecture-reviewer (five lenses), security-engineer, networking-engineer and sre-engineer (mode `tech-review`), with the `tech-selection` criteria P1 to P12 and the `architecture-references` rule IDs.
- Scope: the decision, the technology table, the three open questions, and coordinator finding S1 (repository layout).
- This review does not change the ADR status. Only the owner decides.

Verdict scale: APPROVED, CONDITIONAL (conditions listed, each becomes a Phase 3 task), REJECTED. A claim marked UNVERIFIED had no source that could be fetched; none of the verdicts rest on an UNVERIFIED claim.

## Verdict summary

| Item | Verdict | Conditions |
|---|---|---|
| Lens `aws-well-architected` | CONDITIONAL | C4, C5, C14, C15 |
| Lens `azure-well-architected` | CONDITIONAL | C1, C5, C16 |
| Lens `google-cloud-architecture` | CONDITIONAL | C3, C4, C5, C13 |
| Lens `ddd-hexagonal` | CONDITIONAL | C1, C2, C16, C17 |
| Lens `security-architecture` | CONDITIONAL | C6 to C11 |
| `google.golang.org/grpc` v1.84.0 | CONDITIONAL | C3, C4, C12 |
| `google.golang.org/protobuf` v1.36.12 | APPROVED | none |
| buf CLI v1.73.0 | APPROVED | general C15 only |
| `langgraph` 1.2.14 (was CONDITIONAL) | CONDITIONAL, resolved | C18, C19, C20 |
| `grpcio`, `grpcio-tools` 1.84.0 | APPROVED | general C15 only |
| `protobuf` (PyPI) 7.36.2 | CONDITIONAL | C15 (P3 below 5.0, with reason) |
| `anthropic` (PyPI) 1.12.1 | CONDITIONAL | C4, C21 |
| `grpcio-health-checking` 1.84.0 (new row) | APPROVED | C13 |
| Open question 1 (no proto for third-party adapters) | APPROVED as recommended | none |
| Open question 2 (Go core is not a gRPC server) | APPROVED as recommended | C2 |
| Open question 3 (Python packaging and lockfile) | Ruled: pip-tools, or uv as CONDITIONAL | C15 |
| Security review | CONDITIONAL | C6 to C11 |
| Networking review | CONDITIONAL | C3, C4, C12 |
| SRE review | CONDITIONAL | C5, C13, C14 |
| Finding S1 (repository layout) | Recommend option (b): ADR-0008 | draft below |

**Recommendation:** the owner can move ADR-0007 to Accepted. No finding contradicts the decision itself; every finding is a design constraint that the Phase 3 plan must carry as a task. Two notes for the owner before accepting: the technology table has three evidence errors (see [Corrections to the ADR table](#corrections-to-the-adr-table)), and the repository layout should be recorded separately as ADR-0008 (finding S1).

## 1. Architecture review (five lenses)

The decision is measured against ADR-0006: a single-user, laptop-hosted tool that must keep the five scale-ready seams. Distributed-scale machinery is not required (`architecture-references` scale filter).

### Seam check (ADR-0006)

| Seam | Holds? | Evidence |
|---|---|---|
| 1. Everything keyed by owner id | **No, not yet.** `allocation/domain.Input` has no owner id (`services/timesheet-core/internal/allocation/domain/policy.go`), and ADR-0007 says the proto mirrors it. In process the owner is implicit; across the wire it must be explicit, or a multi-tenant agent cannot tell callers apart. | C1 |
| 2. No mutable global state; state in Postgres | Holds only if the agent uses no persistent LangGraph checkpointer or store. LangGraph's persistence layer is optional ([docs.langchain.com persistence](https://docs.langchain.com/oss/python/langgraph/durable-execution)). | C19 |
| 3. External systems behind ports | Holds. The agent sits behind the existing `allocation.Policy` port, and `grpcpolicy` is the adapter. | sound |
| 4. Idempotent writes; side effects through the outbox | Holds. `Allocate` has no side effects; the agent writes nothing. Mark the RPC `idempotency_level = NO_SIDE_EFFECTS` ([descriptor.proto](https://github.com/protocolbuffers/protobuf/blob/main/src/google/protobuf/descriptor.proto), enum `IdempotencyLevel`). | C1 |
| 5. Config and credentials resolved at the composition root | Holds if the agent address, deadlines and message limits are config read in `cmd/timesheet/wire.go`. | C3 |

### Lens `aws-well-architected`: CONDITIONAL

- **[AWS-1] Retry layers multiply** (ADR-0007 Decision 3, ADR-0004). Risk: the Anthropic Python SDK retries 2 times by default and times out after 10 minutes ([platform.claude.com Python SDK](https://platform.claude.com/docs/en/api/sdks/python): "automatically retried 2 times by default", "By default requests time out after 10 minutes"). The chain `[agent, agent, rule]` calls the agent twice. If the second agent attempt also runs after a transport failure, one Friday run makes up to 2 x 3 = 6 Anthropic calls per model turn, and a hung agent holds the run for 20 minutes. Reference: [AWS REL05-BP03](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/rel_mitigate_interaction_failure_limit_retries.html). Recommendation: C4.
- **[AWS-10] Two languages, one telemetry format** (Decision 3). Risk: Go `slog` lines and Python logs cannot be joined for one run, so a fallback cannot be explained. Reference: [AWS OPS04-BP02](https://docs.aws.amazon.com/wellarchitected/latest/framework/ops_observability_application_telemetry.html). Recommendation: C5.
- **[AWS-9] CI covers Go only** (Decision 4). Risk: a vulnerable Python dependency ships unnoticed; `.github/dependabot.yml` has `gomod`, `github-actions`, `docker` and `docker-compose`, but no `pip` ecosystem. Reference: [AWS SEC11-BP02](https://docs.aws.amazon.com/wellarchitected/latest/framework/sec_appsec_automate_testing_throughout_lifecycle.html). Recommendation: C15.
- **[AWS-11] Recovery is assumed, not tested.** Risk: the fallback to `RulePolicy` is the whole reliability story, and no test proves it. Reference: [AWS reliability design principles](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/design-principles.html). Recommendation: C14.
- Sound: [AWS-5] the agent is a new component behind an ACL (`grpcpolicy`); [AWS-7] the ACL is permanent; the scale filter is respected (one remote service, no mesh, no saga).

### Lens `azure-well-architected`: CONDITIONAL

- **[AZ-1] Keep it simple.** The second language is justified by a requirement (LangGraph exists only for Python and TypeScript), and the agent is correctly separated as a component that may degrade. But Decision 1 also asks for `activity/v1/source.proto` with no consumer. Risk: a `v1` contract that nothing calls is frozen by `buf breaking` before it is exercised. Reference: [Azure reliability principles#keep-it-simple](https://learn.microsoft.com/en-us/azure/well-architected/reliability/principles#keep-it-simple). Recommendation: C16.
- **[AZ-2] Retries fit the end-to-end budget.** Same root cause as AWS-1. Reference: [Azure transient faults#avoid-anti-patterns](https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults#avoid-anti-patterns). Recommendation: C4.
- **[AZ-5] The ACL validates and sanitises input.** `grpcpolicy` must validate the proto `Proposal` before mapping, not only rely on the aggregate. Reference: [Azure ACL#problems-and-considerations](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer#problems-and-considerations). Recommendation: C7.
- Sound: [AZ-11] infrastructure stays out of the domain if C17 adds the archtest rule.

### Lens `google-cloud-architecture`: CONDITIONAL

- **[GCP-6] Deadlines are propagated, not invented per hop.** gRPC sets no deadline by default ([grpc.io deadlines](https://grpc.io/docs/guides/deadlines/): "By default, gRPC does not set a deadline which means it is possible for a client to end up waiting for a response effectively forever"). Automatic propagation exists in Go and Java but not in every language; the Python agent must read the remaining time itself. Reference: [SRE book, cascading failures](https://sre.google/sre-book/addressing-cascading-failures/). Recommendation: C3.
- **[GCP-3, GCP-4] At most 3 attempts, retry only at the layer above the rejection.** The Anthropic API rejects; the SDK is the layer directly above it. gRPC should not add a retry policy on top. Reference: [SRE book, handling overload](https://sre.google/sre-book/handling-overload/). Recommendation: C4.
- **[GCP-7] Name what degrades.** ADR-0007 names it (agent falls back to `RulePolicy`). Sound, but the degradation must be visible. Recommendation: C5.
- **[GCP-8] Request id.** `AllocateRequest` should carry a `request_id` (UUID4) so that `agent_runs`, Go logs and Python logs share one key. Reference: [AIP-155](https://google.aip.dev/155). Recommendation: C1.

### Lens `ddd-hexagonal`: CONDITIONAL

- **[DDD-1, DDD-11] Generated types stay out of the domain.** Risk: generated `*pb.Proposal` types leak into `allocation/domain` or `timesheet/application`, which couples the domain to protobuf. `grpcpolicy` is the only place that imports generated code. Reference: [AWS hexagonal architecture](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/hexagonal-architecture.html), [AWS ACL](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/acl.html). Recommendation: C17.
- **[DDD-4] The proto is a published language, not a copy of the internal struct.** "Mirroring `allocation/domain.Policy`" is acceptable only if mapping code exists in both directions, so the domain can change without a `v2`. Recommendation: C1 (field list), C17.
- **Seam 1** (owner id) is a finding here too. Recommendation: C1.
- **[DDD-2]** Sound: swapping the in-process implementation for the gRPC adapter in `cmd/timesheet/wire.go` only is the correct composition-root pattern.
- Open question 2 ruling (no gRPC server in the Go core) is consistent with DDD-3: a port and adapter pair needs a second consumer to justify it.

### Lens `security-architecture`: CONDITIONAL

Findings are in [section 3](#3-security). Summary: [SEC-1] the threat model needs the new boundary and two new threats; [SEC-3] and [SEC-4] the network hop does not change the rule that model output is untrusted; [SEC-6] the agent container can currently reach Postgres; [SEC-8] LangSmith tracing must stay off.

## 2. Production-readiness review of the technology table

### Corrections to the ADR table

| Row | ADR says | Live evidence says | Source |
|---|---|---|---|
| `protobuf` (PyPI) | BSD-3-Clause, no Scorecard given | Package BSD-3-Clause; source repo `protocolbuffers/protobuf` Scorecard **4.8** (below 5.0); deps.dev project licence field "non-standard" | [deps.dev project](https://api.deps.dev/v3/projects/github.com%2Fprotocolbuffers%2Fprotobuf), [deps.dev version](https://api.deps.dev/v3/systems/pypi/packages/protobuf/versions/7.36.2) |
| `anthropic` (PyPI) | MIT; official vendor SDK | MIT confirmed; **no Scorecard** on deps.dev for `anthropics/anthropic-sdk-python` | [deps.dev project](https://api.deps.dev/v3/projects/github.com%2Fanthropics%2Fanthropic-sdk-python) |
| `langgraph` | docs site unreachable | `docs.langchain.com` reachable (HTTP 200); also a fixed advisory makes 1.2.14 the floor, not just the latest | [docs.langchain.com](https://docs.langchain.com/oss/python/langgraph/overview), [GHSA-3fx2-cqw9-xr3c](https://github.com/langchain-ai/langgraph/security/advisories/GHSA-3fx2-cqw9-xr3c) |
| gRPC Go docs | grpc.io unverified | grpc.io reachable; every guide cited below returned HTTP 200 | [grpc.io guides](https://grpc.io/docs/guides/) |

Everything else in the table matched the live sources.

### Row verdicts

| Tech | Version (verified) | P1 vulns | P2 maint. | P3 Scorecard | P4 licence | P5 API | P6 provenance | P7 compat. | P12 currency | Verdict |
|---|---|---|---|---|---|---|---|---|---|---|
| `google.golang.org/grpc` | v1.84.0, 2026-09-17 | none | yes | 8.5 | Apache-2.0 | v1 | Google (grpc org) | `go 1.25.0` in its go.mod; ours is `go 1.26.0` | latest | CONDITIONAL: C3, C4, C12 (P10: deadlines, retries and limits need explicit settings) |
| `google.golang.org/protobuf` | v1.36.12, 2026-08-10 | none | yes | 5.2 | BSD-3-Clause | v1 | Go team | `go 1.23` | latest | APPROVED |
| buf CLI | v1.73.0, 2026-09-11 (proxy and GitHub release agree) | none | yes | 5.1 | Apache-2.0 | v1 | Buf (vendor) | n/a (CI tool) | latest | APPROVED. Releases ship `sha256.txt` and `sha256.txt.minisig`; C15 pins and verifies |
| `langgraph` | 1.2.14, 2026-10-06 | none on 1.2.14; see below | weekly releases | none on deps.dev | MIT | v1 | langchain-ai (official) | Python 3.10 or later | latest | CONDITIONAL, resolved: C18, C19, C20 |
| `grpcio`, `grpcio-tools` | 1.84.0, 2026-09-14 | none | yes | 6.6 (`grpc/grpc`) | Apache-2.0 | v1 | grpc org | Python 3.10 or later | latest | APPROVED |
| `protobuf` (PyPI) | 7.36.2, 2026-09-17 | none | yes | 4.8 | BSD-3-Clause | stable | Google | Python 3.10 or later; `grpcio-tools` 1.84.0 requires `protobuf>=7.35.1,<8.0.0` | latest | CONDITIONAL: P3 below 5.0, accepted with reason (official Google runtime, required by `grpcio-tools`); C15 |
| `anthropic` (PyPI) | 1.12.1, 2026-10-08 (GitHub release v1.12.1 agrees) | none | yes | none on deps.dev | MIT | v1 | Anthropic (official) | Python 3.10 or later | latest | CONDITIONAL: P3 has no score, accepted because P6 is the official vendor SDK; P10 needs C4 and C21 |
| `grpcio-health-checking` (new) | 1.84.0, 2026-09-14 | none (PyPI) | yes | 6.6 (`grpc/grpc`) | Apache-2.0 | v1 | grpc org | Python 3.10 or later | latest | APPROVED, for C13 |

Sources for every cell: [proxy.golang.org grpc](https://proxy.golang.org/google.golang.org/grpc/@latest), [proxy protobuf](https://proxy.golang.org/google.golang.org/protobuf/@latest), [proxy buf](https://proxy.golang.org/github.com/bufbuild/buf/@latest), [GitHub buf release v1.73.0](https://github.com/bufbuild/buf/releases/tag/v1.73.0), PyPI JSON for [langgraph](https://pypi.org/pypi/langgraph/json), [grpcio](https://pypi.org/pypi/grpcio/json), [grpcio-tools](https://pypi.org/pypi/grpcio-tools/json), [protobuf](https://pypi.org/pypi/protobuf/json), [anthropic](https://pypi.org/pypi/anthropic/json), [grpcio-health-checking](https://pypi.org/pypi/grpcio-health-checking/json), and the deps.dev version and project endpoints in the evidence log. Go compatibility: [grpc v1.84.0 go.mod](https://proxy.golang.org/google.golang.org/grpc/@v/v1.84.0.mod), [protobuf v1.36.12 go.mod](https://proxy.golang.org/google.golang.org/protobuf/@v/v1.36.12.mod).

### `langgraph`: the CONDITIONAL item, resolved

The P3 gap (no Scorecard) is filled with the evidence the criterion allows ("under 5.0 with a reason"):

- **Licence:** MIT on PyPI (`license_expression`), on deps.dev, and on the GitHub repository API (`license.spdx_id: MIT`). [PyPI](https://pypi.org/pypi/langgraph/json), [GitHub repo](https://github.com/langchain-ai/langgraph)
- **Releases:** 1.2.11 (2026-08-11), 1.2.12 (2026-09-21), 1.2.13 (2026-10-05), 1.2.14 (2026-10-06); not archived; last push 2026-10-09. [GitHub releases](https://github.com/langchain-ai/langgraph/releases)
- **Security policy:** no `SECURITY.md` in the repository (404 at the root, `.github/` and `libs/langgraph/`), but the organisation-wide `langchain-ai/.github/SECURITY.md` routes reports to the LangChain Open Source VDP on Intigriti. GitHub community profile health is 100 percent. [org SECURITY.md](https://github.com/langchain-ai/.github/blob/main/SECURITY.md)
- **Advisory handling:** the repository publishes GitHub Security Advisories with patched versions. Relevant to us:
  - [GHSA-3fx2-cqw9-xr3c](https://github.com/langchain-ai/langgraph/security/advisories/GHSA-3fx2-cqw9-xr3c), medium, published 2026-10-07, affects `langgraph` and `langgraph-sdk`, patched in **`langgraph` 1.2.14** and `langgraph-sdk` 0.4.6. deps.dev still showed no advisory on 1.2.13 at review time, so deps.dev lags GHSA by days. 1.2.14 is therefore a floor.
  - [GHSA-fjqc-hq36-qh5p](https://github.com/langchain-ai/langgraph/security/advisories/GHSA-fjqc-hq36-qh5p), unsafe JSON deserialisation in checkpoint loading, patched in `langgraph-checkpoint` 4.1.1. `langgraph` 1.2.14 allows `langgraph-checkpoint>=4.1.0,<5.0.0`, and deps.dev lists the advisory on 4.1.0. The lockfile must resolve 4.1.1 or later (latest is 4.2.0).
  - Several earlier advisories hit the SQLite and Postgres checkpointers (SQL injection, RCE through serialisers). The design does not need a persistent checkpointer, so C19 removes that surface entirely.
- **Footprint (P8):** 38 nodes in the resolved dependency graph, including `langchain-core`, `pydantic` and `xxhash` ([deps.dev dependencies](https://api.deps.dev/v3/systems/pypi/packages/langgraph/versions/1.2.14:dependencies)). Heavy, but justified by the owner's choice of LangGraph.
- **Defaults that matter (P9, P10):**
  - The default recursion limit is 1000 super-steps ([docs.langchain.com graph API](https://docs.langchain.com/oss/python/langgraph/graph-api): "Starting in version 1.0.6, the default recursion limit is set to 1000 steps"). That is far above a sane turn limit for one allocation, so C20 sets it explicitly.
  - LangSmith tracing turns on with `LANGSMITH_TRACING=true` and an API key ([overview](https://docs.langchain.com/oss/python/langgraph/overview)). Traces would send Slack text to a third party, so C18 keeps it off.

Verdict: **CONDITIONAL, resolved.** The P3 evidence is sufficient, and the remaining conditions are C18 to C20.

### Open questions

1. **Proto contracts for third-party adapters (Tempo, Jira, Slack): no.** Agreed with the ADR's recommendation. These are vendor APIs behind ACLs; a proto would be a second contract for a system we do not own, and DDD-11 already requires translation at the adapter. If an adapter is ever split out, its contract is written then. References: [AWS ACL](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/acl.html), [Azure ACL](https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer). Verdict: APPROVED.
2. **The Go core as a gRPC server before a second consumer exists: no.** Agreed. The core stays a client only and listens on no port. A server would add an inbound trust boundary, auth and a health endpoint with no caller. Corollary (C2): no `grpc.NewServer` in `services/timesheet-core` in Phase 3, and archtest or a lint rule enforces it. Verdict: APPROVED.
3. **Python packaging and lockfile tool: ruling.** The plan must choose one tool whose lockfile pins every transitive dependency with hashes, and both CI and the Docker build must install from it without re-resolving. Candidates on today's evidence:

   | Tool | Version | Licence | Scorecard | P5 | Fit |
   |---|---|---|---|---|---|
   | pip-tools | 7.6.2 (2026-10-07) | BSD-3-Clause | 6.6 | v7, stable | APPROVED. `pip-compile --generate-hashes` ([pip-tools README](https://github.com/jazzband/pip-tools)), installed with `pip install --require-hashes --only-binary :all:` ([pip secure installs](https://pip.pypa.io/en/stable/topics/secure-installs/)) |
   | uv | 0.12.24 (2026-10-08) | MIT OR Apache-2.0 | none on deps.dev | v0 | CONDITIONAL (P3 has no score, P5 is v0): pin the exact version, name an upgrade owner, use `uv sync --locked` ([uv docs](https://docs.astral.sh/uv/concepts/projects/sync/): "If the lockfile is not up-to-date, uv will raise an error") |
   | Poetry | 2.5.1 (2026-09-20) | MIT | 3.0 | v2 | CONDITIONAL (P3 is 3.0); no advantage over the other two here |

   Ruling: **pip-tools is the only candidate that is APPROVED on every criterion.** uv is acceptable as CONDITIONAL if the owner prefers it. Either way, C15 applies. The standard lockfile format `pylock.toml` exists ([packaging.python.org](https://packaging.python.org/en/latest/specifications/pylock-toml/)), but neither its tool support nor its use here was evaluated (UNVERIFIED for this project); it is not required.

## 3. Security

### Prompt injection across the network hop (OWASP Top 10 for LLM Applications)

The 2026 edition is the current one ([2026 final](https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/tree/main/2026/final)). IDs changed from 2025 ([2025 list](https://genai.owasp.org/llm-top-10/)), so both are given:

| Risk | 2026 ID | 2025 ID | Applies here |
|---|---|---|---|
| Prompt injection | LLM01:2026 | LLM01:2025 | Slack text and Jira fields in `AllocateRequest` |
| Sensitive information disclosure | LLM02:2026 | LLM02:2025 | Slack text sent to Anthropic, and possibly to LangSmith |
| Excessive agency | LLM03:2026 | LLM06:2025 | agent tools and container permissions |
| Supply chain | LLM04:2026 | LLM03:2025 | Python dependencies, images |
| Unbounded consumption | LLM06:2026 | LLM10:2025 | turn limit, tokens, retries |
| Improper output handling | LLM10:2026 | LLM05:2025 | `Proposal` fields and reasoning text rendered in Slack |

Findings, ranked:

1. **High: the agent container can reach Postgres.** Compose puts every service on one default network, and each container "is both reachable by other containers on that network" ([Compose networks](https://docs.docker.com/reference/compose-file/networks/)). The Postgres password is written in `docker-compose.yml`. A prompt-injected or compromised agent (LLM03:2026) could then read or change every timesheet. Fix: C6.
2. **High: the response is untrusted, whatever the transport.** gRPC gives the response a typed shape but not trust. LLM01:2026 mitigation 2: "Define a strict output schema and validate every response in trusted application code ... using structural validation rather than a second LLM call" ([LLM01:2026](https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM01_PromptInjection.md)). The ADR-0004 replay through the aggregate does this for entries; `grpcpolicy` must also bound sizes and treat `reasoning` as untrusted text for Slack (LLM10:2026, SEC-9). Fix: C7.
3. **Medium: untrusted input must stay in a labelled channel.** Slack text must cross the hop as structured `WorkActivity` fields, never as a pre-built prompt string, so the agent can place it in a provenance-labelled block such as a `tool_result` (SEC-4, [OWASP LLM prompt injection cheat sheet](https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html)). Fix: C1 (field list), C8.
4. **Medium: LangSmith tracing would export Slack text** (LLM02:2026). Fix: C18.
5. **Medium: excessive agency.** LLM03:2026: "Minimize tools", "Minimize tool permissions", and "Implement authorization in logic rather than relying on an LLM" ([LLM03:2026](https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM03_ExcessiveAgency.md)). The agent's only output is the `submit_proposal` structured result; it has no write tools and no network tools. Fix: C8, C9.
6. **Medium: unbounded consumption** (LLM06:2026: "token-aware cost controls, hard spending caps, agent-level circuit breakers" ([LLM06:2026](https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM06_UnboundedConsumption.md))). The default LangGraph limit of 1000 steps and the SDK's 2 retries and 10-minute timeout are all unbounded for our purpose. Fix: C4, C20, C21.
7. **Low: supply chain** (LLM04:2026 asks for hash-pinned, immutable references). Fix: C15.

### Least privilege: the agent gets only `ANTHROPIC_API_KEY`

The ADR's intent is right. To make it true in the runtime (C9):

- In `environment:`, list `ANTHROPIC_API_KEY:` as the only pass-through variable, with no `env_file:`. The Slack, Jira, Tempo and database variables stay on `app` only.
- Run as a non-root user with `read_only: true`, `cap_drop: [ALL]` and `security_opt: [no-new-privileges:true]` (all are Compose service attributes: [Compose services reference](https://docs.docker.com/reference/compose-file/services/)).
- Mount the agent workspace (prompts, skills) read-only. Lessons are written only by the Go core (`learning/infrastructure/fsworkspace`, T5 memory poisoning).
- No secret in the system prompt (SEC-8). The API key never appears in a log, error or trace.

### Trust boundary on the compose network

New boundary: Go core and allocation-agent over gRPC (already listed as boundary 4 in `docs/security/threat-model.md`). Data crossing it: one week of activity (Slack text, issue keys), channel defaults, caps and feedback. Nothing crossing it is a credential. C6 splits the network so that the boundary is real:

- `core` network: `app`, `db`.
- `agent` network: `app`, `allocation-agent`. This network needs egress to `api.anthropic.com`, so it cannot be `internal: true` ([Compose networks](https://docs.docker.com/reference/compose-file/networks/): "internal, when set to true, lets you create an externally isolated network").
- The agent publishes no port (already in the ADR), and `app` is the only client.

Residual risk: Compose cannot limit the agent's egress to one hostname without an egress proxy. Accepted for a single-user laptop; record it in the threat model.

### Does gRPC need TLS on the local network?

**Ruling: not in Phase 3; yes before any off-host deployment.**

- Why plaintext (h2c with `insecure.NewCredentials()`) is acceptable now: one host, a dedicated bridge network with two members (C6), no published port, and no credential crosses the hop. The data does include Slack text, but it never leaves the host's virtual network; an attacker who can sniff that bridge already has root on the laptop, which the threat model accepts as residual.
- Guard rail: grpc.io notes that "most gRPC language implementations will not let you send credentials on an unencrypted channel" ([grpc.io auth](https://grpc.io/docs/guides/auth/)). The design must therefore never add per-RPC credentials to this channel without TLS.
- Conditions (C10): config validation allows insecure credentials only when the target host is the compose service name (`allocation-agent`); any other host requires TLS. The AWS move ADR (ADR-0006 follow-up) must require TLS, and mutual authentication between services, before the agent runs on a different host. `insecure.NewCredentials` "disables transport security" ([grpc-go source v1.84.0](https://github.com/grpc/grpc-go/blob/v1.84.0/credentials/insecure/insecure.go)).

### Threat model updates (C11)

Add to `docs/security/threat-model.md`:

- T4 (extend): prompt injection now crosses a network hop; the mitigations are C7 and C8, and the status stays planned for P3.
- T9 (new): lateral movement from the agent container to Postgres or other services. Mitigation: C6 and C9.
- T10 (new): data exfiltration through third-party tracing (LangSmith) or verbose logs. Mitigation: C18 and C5 (log scrubbing).
- T6 (extend): cost bounds now live in two processes. Mitigation: C4, C20 and C21.

## 4. Networking

All settings below are Go client (`grpcpolicy`) settings unless marked "agent". Values are starting points to tune with data from `agent_runs`.

| Concern | Official guidance | Setting (C3, C4, C12) |
|---|---|---|
| Deadline | "By default, gRPC does not set a deadline ... you should always explicitly set a realistic deadline" ([grpc.io deadlines](https://grpc.io/docs/guides/deadlines/)) | Each `Allocate` call runs under `context.WithTimeout` taken from config (`agent.attempt_timeout`, start at 120 s). The whole policy chain runs under a run deadline (`agent.chain_timeout`, start at 300 s), so the total including `RulePolicy` stays far inside the 1-hour proposal-freshness SLO |
| Deadline in the agent | Propagation is automatic in Go and Java; "in some languages this behavior needs to be explicitly enabled" (same page). Python exposes `ServicerContext.time_remaining()` ([grpc Python API](https://grpc.github.io/grpc/python/grpc.html)) | Agent: before each model call, set the Anthropic SDK `timeout` to `min(per-call cap, time_remaining() - 5 s)`; stop the graph and return `DEADLINE_EXCEEDED` when too little time is left |
| Retry policy | "Retries are enabled by default, but there is no default retry policy"; without one, only transparent retries happen, when gRPC knows the RPC was not processed ([grpc.io retry](https://grpc.io/docs/guides/retry/)) | **No `retryPolicy`** for `PolicyService.Allocate`. Each attempt costs model tokens, and the policy chain is already the semantic retry layer (GCP-3, GCP-4). Keep the default transparent retries |
| Retry layering | as above, plus the SDK's default of 2 retries ([Python SDK](https://platform.claude.com/docs/en/api/sdks/python)) | Agent: `Anthropic(max_retries=1)`, set explicitly. Go: the chain's second agent attempt runs **only** after a validation rejection, never after `UNAVAILABLE`, `DEADLINE_EXCEEDED` or `RESOURCE_EXHAUSTED`; those go straight to `RulePolicy`. Worst case: 2 agent attempts x 2 SDK tries per model turn |
| Service config | Per-method timeout, wait-for-ready and retry or hedging policy ([grpc.io service config](https://grpc.io/docs/guides/service-config/)). The Go DNS resolver can read a service config from TXT records (same page) | `grpc.WithDefaultServiceConfig` with one `methodConfig` for `timesheetbot.allocation.v1.PolicyService`: `timeout` equal to `agent.attempt_timeout` as an upper bound, `waitForReady: false`, no `retryPolicy`. Add `grpc.WithDisableServiceConfig()` so a DNS TXT record cannot change it (both options exist in [grpc-go v1.84.0 dialoptions.go](https://github.com/grpc/grpc-go/blob/v1.84.0/dialoptions.go)) |
| Fail fast | With wait-for-ready, an RPC to an unreachable backend "will be delayed instead of immediately failing" ([service config](https://grpc.io/docs/guides/service-config/), [wait-for-ready](https://grpc.io/docs/guides/wait-for-ready/)) | `waitForReady: false` (the default), so a stopped agent returns `UNAVAILABLE` at once and the chain falls back |
| Keepalive | Client keepalive is disabled by default (`KEEPALIVE_TIME` `INT_MAX`); the server permits pings no more often than every 5 minutes by default and answers faster pings with GOAWAY `too_many_pings`; "Keepalive is less likely to be triggered for unary RPCs with quick replies" ([grpc.io keepalive](https://grpc.io/docs/guides/keepalive/)) | **No keepalive settings in Phase 3.** The call is one unary RPC per week. Create the client with `grpc.NewClient("dns:///allocation-agent:<port>", ...)` at run start and `Close` it at run end, so no idle connection survives laptop sleep. If keepalive is ever added, the client interval must be 5 minutes or more, or the agent's `PERMIT_KEEPALIVE_TIME` must be lowered to match |
| Message size | Default maximum receive size is 4 MiB on the Go client and server (`defaultClientMaxReceiveMessageSize`, `defaultServerMaxReceiveMessageSize`) and in gRPC core (`grpc.max_receive_message_length`, "Defaults to 4MB"); the default send size is unlimited ([grpc-go clientconn.go v1.84.0](https://github.com/grpc/grpc-go/blob/v1.84.0/clientconn.go), [server.go](https://github.com/grpc/grpc-go/blob/v1.84.0/server.go), [gRPC core channel_arg_names.h v1.84.0](https://github.com/grpc/grpc/blob/v1.84.0/include/grpc/impl/channel_arg_names.h)) | Keep 4 MiB, but set it explicitly on both sides: Go `grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4<<20), grpc.MaxCallSendMsgSize(4<<20))`; agent `grpc.max_receive_message_length` and `grpc.max_send_message_length` set to 4 MiB. The Go side caps the request before sending (per-activity text length, activity count), so an oversized week fails locally with `RESOURCE_EXHAUSTED` and falls back to `RulePolicy` |
| Graceful stop | The server "immediately notifies all clients to stop sending new RPCs" and lets in-flight RPCs finish until a deadline ([grpc.io graceful stop](https://grpc.io/docs/guides/server-graceful-stop/)) | Agent: on `SIGTERM`, set health to `NOT_SERVING`, then `server.stop(grace=10)` ([grpc Python API](https://grpc.github.io/grpc/python/grpc.html), `stop(grace)`) |
| Status mapping | [grpc.io status codes](https://grpc.io/docs/guides/status-codes/) | Agent returns `INVALID_ARGUMENT` for a bad request, `DEADLINE_EXCEEDED` when its budget runs out, `UNAVAILABLE` for an Anthropic outage after its own retry, `RESOURCE_EXHAUSTED` for token-budget stops. Go logs the code and falls back on every non-OK code |

Also for `docs/architecture/network.md`: choose the port, add the `agent` and `core` networks to the diagram, and record the timeouts and limits above in the egress table.

## 5. SRE

### The agent is down at the Friday run

| Failure | What the Go core sees | Time to fallback | Owner-visible effect |
|---|---|---|---|
| Container not running, or crashed | `UNAVAILABLE` at once (`waitForReady: false`) | under 1 s | rule-based proposal, flagged (C5) |
| Agent running but hung | `DEADLINE_EXCEEDED` | `agent.attempt_timeout` (120 s) | rule-based proposal, flagged |
| Anthropic outage or 429 | agent returns `UNAVAILABLE` after one SDK retry | seconds to the attempt deadline | rule-based proposal, flagged |
| Agent returns an invalid proposal twice | aggregate rejects both | two attempts | rule-based proposal with the validation errors |
| Agent starts but is unhealthy (no key, graph fails to compile) | health `NOT_SERVING` | under 1 s through the pre-flight check | rule-based proposal, flagged "agent unhealthy" |

The fallback itself is sound (GCP-7, ADR-0004). The reliability risk is **silent degradation**: `RulePolicy` always succeeds, so the agent could stay broken for weeks without anyone noticing.

Startup ordering: `app` must **not** declare `depends_on: allocation-agent: condition: service_healthy`. That makes the core wait for the agent and defeats the fallback ([Compose services](https://docs.docker.com/reference/compose-file/services/): `service_healthy` means a dependency "is expected to be healthy ... before starting a dependent service"). Either omit the dependency, or use `required: false` ("Compose only warns you when the dependency service isn't started or available", same page). C13.

### Health checks (`grpc.health.v1`)

- The agent serves the standard health service `grpc.health.v1.Health` through `grpcio-health-checking` ([grpc.io health checking](https://grpc.io/docs/guides/health-checking/), [health.proto](https://github.com/grpc/grpc-proto/blob/master/grpc/health/v1/health.proto)).
- Status for `""` (whole server) and for `timesheetbot.allocation.v1.PolicyService`: `SERVING` only after the graph is compiled and `ANTHROPIC_API_KEY` is present (not after a test model call, so health does not cost tokens or depend on Anthropic). `NOT_SERVING` on shutdown; the guide asks servers to "inform the health check library about server shutdown".
- The compose `healthcheck` on the agent calls `Health/Check` with a deadline ("Clients should set a deadline when calling Check", health.proto) from a small Python probe inside the image, so no extra binary is added.
- The Go core makes one pre-flight `Check` with a 2 s deadline before `Allocate`. Its only purpose is to record "agent down" separately from "agent slow". Client-side health watching through the service config is for load balancing and is not needed for one backend.

### Timeouts (one table, all from config)

| Timeout | Value to start | Owner |
|---|---|---|
| Pre-flight health `Check` | 2 s | Go |
| One `Allocate` attempt | 120 s | Go (`agent.attempt_timeout`) |
| Whole policy chain | 300 s | Go (`agent.chain_timeout`) |
| One Anthropic call | `min(60 s, time_remaining() - 5 s)` | agent |
| Graph steps | `recursion_limit` 12 (tune) | agent |
| Graceful stop | 10 s | agent |

### What to observe (C5)

At this scale, structured logs plus the `agent_runs` row and the Slack proposal message are enough. No metrics stack is required ([SRE book, monitoring distributed systems](https://sre.google/sre-book/monitoring-distributed-systems/)).

- `agent_runs` row per attempt: `request_id`, `week_id`, `user_account_id`, policy name, attempt number, gRPC status code, latency, input and output tokens, outcome (`accepted`, `rejected_validation`, `transport_error`, `fallback`).
- The same `request_id` in gRPC metadata and in the request message, logged by the agent as JSON on stdout, so the logs of both processes join.
- The weekly proposal DM states which policy produced it and, on fallback, why (for example "agent unavailable: DEADLINE_EXCEEDED").
- Two consecutive weeks on fallback trigger a separate owner DM. This is the "stop after N consecutive failures and surface the error" counter from `architecture-references`.
- SLIs: proposal freshness is unaffected because of the fallback. Add one tracking ratio to `docs/operations/slo.md`, the share of weeks accepted from the agent, as a signal and not an SLO.
- Add the failure rows above to `docs/operations/failure-modes.md`, plus a runbook `docs/operations/runbooks/agent-down.md`.

## 6. Finding S1: repository layout has no ADR

**Recommendation: option (b), a separate ADR-0008.**

Reasons:

- The layout is already in place and is independent of the gRPC and LangGraph choices. If ADR-0007 were rejected or later superseded (for example, a Go agent without LangGraph), a layout recorded inside it would lose its authority, although `services/timesheet-core` and the delegating root `Makefile` would still exist.
- ADR-0007 already bundles three decisions (contracts first, in-process by default, a Python agent). A fourth, unrelated one makes it harder to supersede any part of it cleanly. Lightweight ADRs record one decision each ([Thoughtworks Radar: lightweight ADRs](https://www.thoughtworks.com/en-us/radar/techniques/lightweight-architecture-decision-records)).
- `CLAUDE.md` and the root `Makefile` comment then cite ADR-0008 for the layout and ADR-0007 only for `api/proto` and `allocation-agent`.

Draft for the owner (not written into `docs/adr/`; the owner decides):

```markdown
# ADR-0008: Organise the repository as a monorepo of services with orchestration-only root

- Status: Proposed
- Date: 2026-10-09
- Deciders: owner
- Related: [ADR-0006](0006-personal-tool-scale-ready-aws-later.md) (scale-ready seams), [ADR-0007](0007-grpc-contracts-now-split-later.md) (gRPC contracts and allocation-agent; this ADR does not depend on its outcome)

## Context

The repository started as one Go module at the root. ADR-0007 proposes a second service in another language (Python), and ADR-0006 expects bounded contexts to become services later. A root that holds one language's tooling (`go.mod`, `.golangci.yml`) makes a second toolchain a special case and couples every service's CI to the first one. The move to `services/` has already been made; this ADR records it.

## Decision

1. Each deployable service lives in `services/<name>/` and owns its whole toolchain there: language manifest and lockfile (`go.mod` and `go.sum`; the Python lockfile chosen in the Phase 3 plan), linter config, `Dockerfile`, and a `Makefile` with at least the `ci` target.
2. The root holds orchestration only: a delegating `Makefile` (every target runs in each `services/*/` that has a `Makefile`), `docker-compose.yml`, `.github/`, `docs/`, and `api/proto/` for shared contracts. No language manifest or language tooling lives at the root.
3. Shared contracts live only in `api/proto/`. Generated code goes to the service that uses it. Services never import each other's source.
4. A service's paths in documentation are relative to its own folder (for Go: `services/timesheet-core/`).
5. `make ci` at the root is the single local and CI entry point, and it fails if any service fails.

## Alternatives considered

- Single Go module at the root (the original layout): rejected; a second language becomes a special case and root tooling leaks into every service.
- One repository per service: rejected; for one owner it multiplies CI, versioning and contract publishing with no benefit, and contracts would need a registry.
- Go workspace (`go.work`) across several Go modules at the root: not needed while there is one Go service; revisit when a second Go service exists.

## Consequences

- Positive: a new service or language is a new folder; each service can be built, scanned and released on its own; the AWS move can deploy services independently.
- Negative: commands must run in the right folder (Go commands outside `services/timesheet-core/` see no `go.mod`); CI and Dependabot need one entry per service and ecosystem.
- Follow-up: `CLAUDE.md` and the root `Makefile` comment cite ADR-0008 for the layout; `.github/dependabot.yml` and `ci.yml` gain an entry for each new service; `docs/architecture/project-structure.md` links this ADR.
```

## 7. Conditions for the Phase 3 plan

Each condition must appear as a task in a Phase 3 work package (`tech-selection`: a plan is final only when every condition has a task).

| ID | Condition | From |
|---|---|---|
| C1 | `AllocateRequest` carries `user_account_id`, `request_id` (UUID4, AIP-155), `week_id`, structured activities (untrusted text in its own fields, never a pre-built prompt), channel defaults, caps and feedback. `Allocate` declares `option idempotency_level = NO_SIDE_EFFECTS`. Mapping code exists in both directions in `grpcpolicy`. | Seam 1, GCP-8, DDD-4, security 3 |
| C2 | The Go core runs no gRPC server in Phase 3 (no `grpc.NewServer` in `services/timesheet-core`), checked by archtest or a lint rule. | Open question 2 |
| C3 | Deadlines: every `Allocate` call has a per-attempt deadline and the chain has a run deadline, both from config resolved in `wire.go`; the agent reads `time_remaining()` and sizes its model calls to fit. | GCP-6, networking |
| C4 | One retry layer: no gRPC `retryPolicy`; Anthropic SDK `max_retries=1` and an explicit `timeout`; the chain's second agent attempt runs only after a validation rejection, not after a transport or deadline error. | AWS-1, AZ-2, GCP-3, GCP-4 |
| C5 | Observability: an `agent_runs` row per attempt (fields as in section 5), a shared `request_id` in both processes' logs, the policy name and fallback reason in the proposal DM, an owner DM after 2 consecutive fallback weeks, and logs scrubbed of Slack text beyond what is needed. | AWS-10, GCP-7, SRE |
| C6 | Compose: separate `core` (app, db) and `agent` (app, allocation-agent) networks; the agent cannot reach `db`; the agent publishes no port. Verify with `docker compose config`. | Security 1 |
| C7 | `grpcpolicy` validates the proto `Proposal` before mapping: maximum entry count, issue keys only from the request, days inside the week, durations as kernel values, and length caps on description and reasoning. Reasoning is rendered to Slack as plain text with links and images stripped. Fuzz the mapper. | LLM01:2026, LLM10:2026, AZ-5, SEC-9 |
| C8 | The agent places request content in provenance-labelled blocks and has no tools except the final `submit_proposal` structured output; injected-instruction fixtures for Slack text and Jira fields are part of the agent's tests. | LLM01:2026, LLM03:2026, SEC-4, SEC-11 |
| C9 | Agent container least privilege: only `ANTHROPIC_API_KEY` in `environment`, no `env_file`, non-root user, `read_only: true`, `cap_drop: [ALL]`, `no-new-privileges`, workspace mounted read-only. | LLM03:2026, SEC-6 |
| C10 | gRPC plaintext only when the target host is `allocation-agent`; config validation rejects insecure credentials for any other host; no per-RPC credentials on this channel; the AWS move ADR must require TLS with mutual authentication. | TLS ruling |
| C11 | Update `docs/security/threat-model.md` with T4 (extended), T6 (extended), T9 and T10. | SEC-1 |
| C12 | Client settings: `grpc.NewClient` per run and `Close` at the end; `grpc.WithDefaultServiceConfig` (timeout, `waitForReady: false`, no retry) and `grpc.WithDisableServiceConfig()`; 4 MiB send and receive limits set explicitly on both sides; request capped before send. No keepalive settings. | Networking |
| C13 | The agent serves `grpc.health.v1` (`""` and `timesheetbot.allocation.v1.PolicyService`), sets `NOT_SERVING` before graceful stop, and has a compose healthcheck calling `Check` with a deadline; `app` does not depend on the agent being healthy. | SRE |
| C14 | Tests prove the fallback: agent stopped, agent hung past the deadline, agent returning `UNAVAILABLE`, and agent returning an invalid proposal twice; each ends with a `RulePolicy` proposal and the right `agent_runs` outcome. | AWS-11, GCP-9 |
| C15 | Supply chain: one Python lock tool (pip-tools, or uv as CONDITIONAL) with every transitive dependency hash-pinned; CI and the Docker build install with `--require-hashes` (pip) or `--locked` (uv); add a `pip` ecosystem entry for `services/allocation-agent` to `.github/dependabot.yml`; a Python vulnerability scan runs in CI (tool to be chosen with tech-selection evidence); the base image is pinned by digest; buf is pinned to v1.73.0 and verified against `sha256.txt`. | AWS-9, LLM04:2026, open question 3 |
| C16 | Phase 3 writes only `allocation/v1/policy.proto`. `activity/v1/source.proto` waits until a consumer or a split is planned (or ships as `v1alpha1`, which `buf breaking` does not freeze in the same way; to be confirmed when it is written). | AZ-1 |
| C17 | Archtest rule: only `allocation/infrastructure/grpcpolicy` (and `cmd/timesheet`) may import generated protobuf packages; domain and application never do. | DDD-1, AZ-11 |
| C18 | LangSmith tracing stays off: no `LANGSMITH_*` variables in compose, and the agent refuses to start if `LANGSMITH_TRACING` is set to true. | LLM02:2026 |
| C19 | No persistent LangGraph checkpointer or store; `langgraph-checkpoint-postgres` and `langgraph-checkpoint-sqlite` are not dependencies; the agent is stateless between requests. | Seam 2, langgraph advisories |
| C20 | `langgraph>=1.2.14` and `langgraph-checkpoint>=4.1.1` in the lockfile; an explicit `recursion_limit` (start at 12) on every invocation, with `GraphRecursionError` mapped to `RESOURCE_EXHAUSTED`. | GHSA-3fx2-cqw9-xr3c, GHSA-fjqc-hq36-qh5p, LLM06:2026 |
| C21 | Agent cost bounds: `max_tokens` per call and a token budget per request, with usage returned in the response and recorded in `agent_runs`. | LLM06:2026, T6 |

## Evidence log

All commands were run on 2026-10-09 from a zsh shell. The deps.dev and proxy forms match the `tech-selection` evidence commands.

Go module proxy (latest version):

```bash
curl -sS https://proxy.golang.org/google.golang.org/grpc/@latest
curl -sS https://proxy.golang.org/google.golang.org/protobuf/@latest
curl -sS https://proxy.golang.org/github.com/bufbuild/buf/@latest
curl -sS https://proxy.golang.org/google.golang.org/grpc/@v/v1.84.0.mod | head -5
curl -sS https://proxy.golang.org/google.golang.org/protobuf/@v/v1.36.12.mod | head -4
```

PyPI JSON:

```bash
for p in langgraph grpcio grpcio-tools protobuf anthropic grpcio-health-checking; do
  curl -sS "https://pypi.org/pypi/$p/json" | jq -c '{v: .info.version, lic: .info.license, licexpr: .info.license_expression, req: .info.requires_python, up: .urls[0].upload_time_iso_8601}'
done
curl -sS https://pypi.org/pypi/langgraph/1.2.14/json | jq -c '.info.requires_dist'
curl -sS https://pypi.org/pypi/grpcio-tools/1.84.0/json | jq -c '.info.requires_dist'
for p in uv poetry pip-tools pip-audit; do curl -sS "https://pypi.org/pypi/$p/json" | jq -c '{v: .info.version, lic: (.info.license_expression // .info.license), up: .urls[0].upload_time_iso_8601}'; done
```

deps.dev (version metadata, then project Scorecard):

```bash
curl -sS "https://api.deps.dev/v3/systems/go/packages/google.golang.org%2Fgrpc/versions/v1.84.0" | jq '{licenses, advisoryKeys, publishedAt}'
curl -sS "https://api.deps.dev/v3/systems/go/packages/google.golang.org%2Fprotobuf/versions/v1.36.12" | jq '{licenses, advisoryKeys, publishedAt}'
curl -sS "https://api.deps.dev/v3/systems/go/packages/github.com%2Fbufbuild%2Fbuf/versions/v1.73.0" | jq '{licenses, advisoryKeys, publishedAt}'
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/langgraph/versions/1.2.14" | jq '{licenses, advisoryKeys, publishedAt}'
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/langgraph/versions/1.2.14:dependencies" | jq '.nodes | length'
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/langgraph-checkpoint/versions/4.1.0" | jq '{advisoryKeys}'
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/grpcio/versions/1.84.0" | jq '{licenses, advisoryKeys, publishedAt}'
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/protobuf/versions/7.36.2" | jq '{licenses, advisoryKeys, publishedAt}'
curl -sS "https://api.deps.dev/v3/systems/pypi/packages/anthropic/versions/1.12.1" | jq '{licenses, advisoryKeys, publishedAt}'
curl -sS "https://api.deps.dev/v3/projects/github.com%2Fgrpc%2Fgrpc-go" | jq '{scorecard: .scorecard.overallScore, openIssuesCount}'
# same project query for: protocolbuffers/protobuf-go, bufbuild/buf, langchain-ai/langgraph, grpc/grpc,
# protocolbuffers/protobuf, anthropics/anthropic-sdk-python, astral-sh/uv, python-poetry/poetry, jazzband/pip-tools, pypa/pip-audit
```

GitHub (releases, licence, security policy, advisories):

```bash
curl -sS https://api.github.com/repos/bufbuild/buf/releases/latest | jq '{tag_name, published_at, prerelease}'
curl -sS https://api.github.com/repos/bufbuild/buf/releases/latest | jq -r '.assets[].name' | grep -E 'sha256|minisig'
curl -sS "https://api.github.com/repos/langchain-ai/langgraph/releases?per_page=15" | jq -c '.[] | {tag_name, published_at, prerelease}'
curl -sS https://api.github.com/repos/langchain-ai/langgraph | jq '{license: .license.spdx_id, archived, pushed_at}'
curl -sS https://api.github.com/repos/langchain-ai/langgraph/community/profile | jq '.health_percentage'
curl -sS -o /dev/null -w '%{http_code}\n' https://api.github.com/repos/langchain-ai/langgraph/contents/SECURITY.md
curl -sS https://raw.githubusercontent.com/langchain-ai/.github/main/SECURITY.md
curl -sS "https://api.github.com/repos/langchain-ai/langgraph/security-advisories?per_page=20" | jq -c '.[] | {ghsa_id, severity, published_at, summary, patched: [.vulnerabilities[].patched_versions]}'
curl -sS https://api.github.com/repos/anthropics/anthropic-sdk-python/releases/latest | jq '{tag_name, published_at}'
curl -sS https://raw.githubusercontent.com/grpc/grpc-go/v1.84.0/clientconn.go | grep -n -E 'defaultClientMax(Receive|Send)MessageSize *=|^func NewClient\('
curl -sS https://raw.githubusercontent.com/grpc/grpc-go/v1.84.0/server.go | grep -n -E 'defaultServerMax(Receive|Send)MessageSize *='
curl -sS https://raw.githubusercontent.com/grpc/grpc-go/v1.84.0/dialoptions.go | grep -n -E '^func (WithDisableServiceConfig|WithDefaultServiceConfig|WithKeepaliveParams)\('
curl -sS https://raw.githubusercontent.com/grpc/grpc/v1.84.0/include/grpc/impl/channel_arg_names.h | grep -n -B2 'max_receive_message_length'
curl -sS https://raw.githubusercontent.com/protocolbuffers/protobuf/main/src/google/protobuf/descriptor.proto | grep -n -A5 'enum IdempotencyLevel'
curl -sS "https://api.github.com/repos/GenAI-Security-Project/GenAI-LLM-Top10/contents/2026/final" | jq -r '.[].name'
curl -sS https://raw.githubusercontent.com/GenAI-Security-Project/GenAI-LLM-Top10/main/2026/final/LLM01_PromptInjection.md
```

Documentation pages fetched (all returned HTTP 200; checked with `curl -sS -L -o /dev/null -w '%{http_code}' <url>`, then read):

- grpc.io: https://grpc.io/docs/guides/deadlines/, https://grpc.io/docs/guides/retry/, https://grpc.io/docs/guides/service-config/, https://grpc.io/docs/guides/keepalive/, https://grpc.io/docs/guides/health-checking/, https://grpc.io/docs/guides/auth/, https://grpc.io/docs/guides/wait-for-ready/, https://grpc.io/docs/guides/server-graceful-stop/, https://grpc.io/docs/guides/status-codes/, https://grpc.io/docs/guides/error/, https://grpc.io/docs/guides/cancellation/, https://grpc.io/docs/guides/performance/, https://grpc.io/docs/languages/go/, https://grpc.github.io/grpc/python/grpc.html, https://github.com/grpc/grpc-proto/blob/master/grpc/health/v1/health.proto
- protobuf.dev: https://protobuf.dev/programming-guides/style/, https://protobuf.dev/best-practices/dos-donts/, https://protobuf.dev/programming-guides/proto3/
- buf.build: https://buf.build/docs/lint/rules/ ("STANDARD is the default if buf.yaml has no lint section"), https://buf.build/docs/breaking/rules/ ("FILE: Default"), https://buf.build/docs/generate/, https://buf.build/docs/installation/ (redirects to `../cli/installation/`)
- LangGraph: https://docs.langchain.com/oss/python/langgraph/overview, https://docs.langchain.com/oss/python/langgraph/durable-execution, https://docs.langchain.com/oss/python/langgraph/graph-api, https://reference.langchain.com/python/langgraph/
- Anthropic: https://platform.claude.com/docs/en/api/sdks/python, https://raw.githubusercontent.com/anthropics/anthropic-sdk-python/v1.12.1/README.md
- OWASP: https://genai.owasp.org/llm-top-10/ (2025), https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/tree/main/2026/final (2026), https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html
- Docker: https://docs.docker.com/compose/how-tos/networking/, https://docs.docker.com/reference/compose-file/networks/, https://docs.docker.com/reference/compose-file/services/
- Python packaging: https://github.com/jazzband/pip-tools, https://pip.pypa.io/en/stable/topics/secure-installs/, https://docs.astral.sh/uv/concepts/projects/sync/, https://packaging.python.org/en/latest/specifications/pylock-toml/
- Architecture and SRE (reachability re-checked): https://sre.google/sre-book/addressing-cascading-failures/, https://sre.google/sre-book/handling-overload/, https://sre.google/sre-book/monitoring-distributed-systems/, https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/rel_mitigate_interaction_failure_limit_retries.html, https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults, https://learn.microsoft.com/en-us/azure/well-architected/reliability/principles, https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/acl.html, https://google.aip.dev/155, https://www.thoughtworks.com/en-us/radar/techniques/lightweight-architecture-decision-records

UNVERIFIED (not fetched in this review): `pylock.toml` tool support; whether `buf breaking` treats `v1alpha1` packages differently (C16 says to confirm when written); a Python vulnerability scanner choice (C15 leaves it to the plan). Architecture rule references from `architecture-references` that are not quoted above were verified on 2026-10-09 by that skill's registry and were not re-read here.

Shell notes for `docs/commands/`: in zsh, `path` is tied to `PATH`, so `for path in ...` breaks every later command in the loop (use another variable name). zsh also does not split an unquoted variable into words, so `G="python3 -I x.py"; $G` fails with "no such file or directory".
