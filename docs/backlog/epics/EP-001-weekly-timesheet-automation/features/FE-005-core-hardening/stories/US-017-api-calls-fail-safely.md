---
id: US-017
title: API calls retry, stop and report safely
status: Active
parent: FE-005
points: 5
sprint: SP-01
github:
fr: [FR-SUB-2, NFR-1, NFR-4]
---

# US-017: API calls retry, stop and report safely

As the timesheet owner, I want calls to Slack, Jira and Tempo to back off politely, to keep a worklog id even when I interrupt a run, and to say which kind of failure happened, so that a bad network day neither duplicates nor loses work and I know what to fix.

Findings: best-practice review F1 and F2-F5 (retry transport), the NoBody bug found by `TestRetryReplaysNoBodyRequest` on 2026-10-09 (AC-017.5: `RetryTransport` returns `ErrNoRewind` for a request whose body is `http.NoBody`, because such a request has no `GetBody`), F11-F14 (cancel-safe submit save, typed API errors). This is a sibling of US-007: US-007 recovers an unknown outcome on the next run, and this story stops one from being created by the transport, by Ctrl-C, or by a misread error. Today the transport makes up to 5 attempts with no jitter (coding standard R2 allows at most 3, with jitter). The per-id wording of F1 to F14 is in the best-practice review, which is not in the repository. The themes here follow the coordinator's grouping.

## Acceptance criteria

```gherkin
# AC-017.1
Scenario: Retries are capped and jittered
  Given Jira answers 503 to every call
  When an issue key is resolved
  Then the client makes at most 3 attempts
  And each wait lies between zero and the capped exponential backoff for that attempt

# AC-017.2
Scenario: An interrupted run still saves the worklog id
  Given submit has just created GCM-1 on Monday in Tempo with worklog id 9001
  When I press Ctrl-C before the id is saved
  Then the GCM-1 entry records worklog id 9001 before the command exits
  And no further entry is sent

# AC-017.3
Scenario: Rejected credentials stop the run and name the variable
  Given Tempo answers 401
  When I submit
  Then submit stops after the first entry
  And the error names TEMPO_API_TOKEN but not its value

# AC-017.4
Scenario: An outage is not mistaken for an unknown issue
  Given Jira answers 500 for GCM-9
  When I run plan for 2026-W41
  Then plan fails with an error saying Jira is unavailable
  And GCM-9 is not dropped as an unknown issue key

# AC-017.5
Scenario: A request with an empty body is retried like any other
  Given a GET request whose body is http.NoBody
  And the server answers 503 and then 200
  When the retry transport sends it
  Then it is sent a second time and returns 200
  And it does not fail with ErrNoRewind
```

## Dependencies

- US-005-T2 (no re-send of a create on 5xx) and US-007 (reconcile).
- Assumption: Ctrl-C cancels the command context; the save after a Tempo success runs on a context that is not cancelled, with its own short timeout.

## Plan

`docs/superpowers/plans/2026-10-09-sp01-hardening.md` (status: draft).

Draft plan coverage, read 2026-10-09: partly covered: WP-2 changes which requests are retried (method rule) but not the attempt count or jitter; WP-6 adds `ErrCreateIndeterminate` as the first typed error.

## Definition of Ready

The Definition of Ready and story points are team conventions, not Scrum Guide 2020 rules (see `docs/backlog/README.md`).

- [x] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Provisional. `technical-architect` aligns them with the plan's work packages.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-017-T1 | Retry transport: at most 3 attempts, capped exponential backoff with full jitter (F1-F5, R2) | - | go-coder | 3 | To Do |
| US-017-T2 | Cancel-safe save after a Tempo success (F11-F12) | - | go-coder | 2 | To Do |
| US-017-T3 | Typed API errors: auth, rate limit, not found, server, indeterminate (F13-F14) | - | go-coder | 3 | To Do |
| US-017-T4 | Treat an `http.NoBody` body as replayable in `RetryTransport`, then remove the skip from `TestRetryReplaysNoBodyRequest` (AC-017.5) | - | go-coder | 1 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-017.1 | planned | |
| AC-017.2 | planned | |
| AC-017.3 | planned | |
| AC-017.4 | planned | |
| AC-017.5 | `TestRetryReplaysNoBodyRequest` | skipped (BUG NoBody) |
