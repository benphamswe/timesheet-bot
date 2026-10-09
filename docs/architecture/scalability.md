# Scalability and extension

Decision: [ADR-0006](../adr/0006-personal-tool-scale-ready-aws-later.md). The tool stays single-user and laptop-hosted, but must stay **scale-ready**: no change may block a later move to a multi-tenant service on AWS.

The workload is one user, one batch a week, so throughput is not the constraint. The design is built to scale in **scope**: new sources, new policies, new targets and, later, more users, without rewriting the core.

## Extension points

| To add | Implement | Touches |
|---|---|---|
| A new activity source (GitHub, calendar, Teams) | `application.ActivitySource` adapter in `internal/activity/infrastructure/<source>` | adapter + `wire.go` |
| A new allocation strategy | `allocation/domain.Policy` | the policy + its position in the chain in `wire.go` |
| A new time-tracking target (Jira worklogs, Clockify) | `application.WorklogGateway` + `IssueDirectory` adapters | adapters + `wire.go` |
| A new bounded context | `internal/<ctx>/{domain,application,infrastructure}` | archtest rules apply automatically |
| A new event consumer | outbox relay handler [P2] | relay only; producers are unchanged |

None of these changes touch the `Timesheet` aggregate. Its invariants protect every source and policy equally ([ADR-0004](../adr/0004-allocation-policy-fallback-chain.md)).

## Toward multiple users (future SaaS on AWS)

Already in place: the aggregate is keyed by `user_account_id` and Postgres enforces one timesheet per user per week. What would change:

- Per-user tokens: a secrets store instead of environment variables (see the D3 deviation in `.claude/skills/coding-standards/sources.md`).
- A config per user instead of one `config.yaml`.
- An always-on AWS deployment instead of a laptop; each AWS service is chosen through the production-readiness review with official-source verification, and per-user rate limits against Slack and Tempo (Tempo documents about 5 requests per second).

## Limits that apply now

| Limit | Value | Where handled |
|---|---|---|
| Slack search page size | 100 per page | collector paginates |
| Tempo rate limit | about 5 req/s, 429 on excess | `RetryTransport` backoff; submit is sequential |
| Postgres | one small database | no sharding needed |
