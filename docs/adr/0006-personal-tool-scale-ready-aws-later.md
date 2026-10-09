# ADR-0006: Build a single-user tool with scale-ready seams; AWS is the future SaaS target

- Status: Accepted
- Date: 2026-10-09
- Deciders: owner

## Context

The owner's long-term ambition is a product for very many users. The product serves people who use both Slack and Jira Tempo, and today it has one user running it on a laptop. Building multi-tenant, multi-region infrastructure now would add large cost and complexity for users who do not exist yet.

## Decision

- Keep the approved single-user, laptop-hosted design (spec, ADR-0001 to ADR-0005).
- Treat **scale-readiness** as a hard review criterion: a change that would block a later move to a multi-tenant cloud service is a defect. The seams that must hold:
  1. Every aggregate and table is keyed by an owner id (`user_account_id`); there is no implicit "the user".
  2. No mutable global state; handlers are stateless and all state lives in Postgres.
  3. External systems sit behind ports (`ActivitySource`, `IssueDirectory`, `WorklogGateway`, and a secrets port before a second user exists).
  4. Writes to external systems are idempotent, and side effects leave through the outbox, so a queue or event bus can replace the relay.
  5. Config and credentials are resolved per owner at the composition root, never deep inside packages.
- If the product becomes a SaaS, the target cloud is **AWS**. That move gets its own ADRs, and every AWS service choice goes through the production-readiness review with official-source verification.

## Alternatives considered

- Staged multi-tenant SaaS now: rejected by the owner for now; the seams above keep it possible.
- Full billion-user design now: rejected; the cost and complexity far exceed the current need.

## Consequences

- Positive: low cost now; the domain and ports carry over to a SaaS unchanged.
- Negative: hosting, tenancy, OAuth and data partitioning will need real work when the move happens.
- Follow-up: `architecture-reviewer` and `technical-architect` (mode review) check the five seams on every change.
