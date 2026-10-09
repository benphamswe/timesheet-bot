# ADR-0004: Allocate through an ordered policy chain validated by the aggregate

- Status: Accepted
- Date: 2026-10-09

## Context

The user wants an AI agent to fully allocate hours [P3], with the 45-minute rule as a safety net. AI output is untrusted and may break invariants (FR-ALC-4, FR-ALC-5, NFR-7).

## Decision

- `ProposeHandler` takes `[]allocation.Policy`. Each proposal is replayed into a fresh `Timesheet` aggregate through `AddEntry`; the first one accepted wins.
- Validation errors of a rejected proposal are appended to `Input.Feedback` for the next policy, so `[agent, agent, rule]` means "retry the agent once with its errors, then fall back".
- The agent has read-only tools and ends by calling a `submit_proposal` tool whose schema is the proposal (structured output).

## Alternatives considered

- Trusting the agent and validating only before the Tempo call: errors would surface after approval.
- Rule only: does not meet the user's request.

## Consequences

- Positive: one validation path for every policy; the fallback is configuration, not code.
- Negative: an invalid agent proposal costs one extra model call before falling back.
