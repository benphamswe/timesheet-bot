# Timesheet Bot - Design Spec

Date: 2026-10-09
Status: Draft, awaiting review

## 1. Goal

Every Friday, collect the work the user did that week from Slack, let an AI agent turn it into Jira Tempo worklogs, send the proposed timesheet to the user as a Slack DM for approval, and on approval write the worklogs to Tempo. The agent improves over time from the user's corrections.

Stack: Go, PostgreSQL, Docker (docker compose). Architecture: Clean Architecture with DDD, package-by-bounded-context.

## 2. Scope

### In scope

- Single user (the owner of the Slack user token).
- Period per run: the current ISO week, Monday to Friday.
- Source: the user's own Slack messages and threads they participated in.
- Target: Jira Tempo Cloud, REST API v4.
- AI allocation with Claude via the Anthropic API, rule-based fallback.
- Approval via Slack DM (Socket Mode), corrections via thread reply.
- Self-learning: lessons from user feedback, agent-drafted skills that require approval.

### Out of scope

- Multiple users / multi-tenant.
- Weekend entries.
- Any web UI.
- Editing or deleting existing Tempo worklogs that the bot did not create.

## 3. Runtime and hosting

- Runs on the user's laptop: Docker on WSL, `docker compose` with `restart: unless-stopped`.
- One long-running process `timesheet serve`: scheduler, Socket Mode listener, outbox relay.
- Slack Socket Mode: outbound WebSocket, no public URL needed.
- Laptop may be off on Friday: an hourly catch-up tick proposes any past week that has no timesheet yet.
- Timezone: `Asia/Ho_Chi_Minh` (configurable).

## 4. Bounded contexts

| Context       | Type       | Responsibility                                                                                              |
| ------------- | ---------- | ----------------------------------------------------------------------------------------------------------- |
| Timesheet     | Core       | `Timesheet` aggregate per (user, ISO week), its entries, lifecycle, submission to Tempo                     |
| Allocation    | Core       | Turn a week of activities into a proposal of entries: `AgentPolicy` (Claude) and `RulePolicy` (45m/key/day) |
| Activity      | Supporting | Collect `WorkActivity` from Slack, extract issue keys, channel to project mapping                           |
| Learning      | Supporting | Capture `Feedback`, distill `Lesson`s, manage agent workspace (skills, memory)                              |
| Shared kernel | -          | `WeekID`, `WorkDay`, `IssueKey`, `Duration`, `Clock`, `DomainEvent`                                         |

Contexts call each other only through application-layer interfaces. No context imports another context's `domain` package.

## 5. Domain model

### 5.1 Shared kernel value objects

- `WeekID`: ISO year + week number, e.g. `2026-W41`. Provides `Days()` returning Monday to Friday.
- `WorkDay`: a calendar date; constructor rejects Saturday and Sunday.
- `IssueKey`: matches `^[A-Z][A-Z0-9]+-\d+$`.
- `Duration`: minutes, must be a positive multiple of 15.

### 5.2 Timesheet aggregate (root)

Fields: `ID`, `UserAccountID`, `Week`, `Status`, `Entries []Entry`, `Policy` (agent or rule), `Version`, `ProposalRevision`.

`Entry` (entity): `ID`, `IssueKey`, `IssueID` (Jira numeric id, resolved at proposal time), `Day`, `Duration`, `Description`, `TempoWorklogID` (nil until submitted), `SubmitAttempts`, `LastError`.

Status lifecycle:

```text
Draft -> AwaitingApproval -> Approved -> Submitted
                |                 |
                v                 v
             Rejected        (entries retried until all submitted
                              or 3 failures -> user notified)
AwaitingApproval -> Draft   (on revision from a thread correction)
```

Invariants:

- Entries only on `Week.Days()`.
- Sum of durations per day <= `daily_cap` (default 8h).
- At most one entry per (issue, day).
- Entries can be added or changed only in `Draft`.
- Only `Approved` can be submitted; only `AwaitingApproval` can be approved or rejected.
- One timesheet per (user, week) - enforced by a unique constraint as well.

Domain events: `TimesheetProposed`, `TimesheetRevised`, `TimesheetApproved`, `TimesheetRejected`, `TimesheetSubmitted`, `EntrySubmissionFailed`.

### 5.3 Allocation

- `AllocationPolicy` interface: `Allocate(ctx, week, activities, context) (Proposal, error)`.
- `Proposal` value object: list of `{IssueKey, Day, Duration, Description}` plus per-day reasoning text and the policy name.
- `RulePolicy`: 45 minutes per distinct issue key per day it appears; activity in a mapped channel with no key goes to that channel's default issue (45 minutes per day). Days over the cap are scaled down proportionally and rounded to 15 minutes.
- `AgentPolicy`: see section 7.

### 5.4 Activity

- `WorkActivity` value object: `Day`, `ChannelID`, `ChannelName`, `Text`, `Permalink`, `ThreadTS`, `IssueKeys []IssueKey`.
- `ActivitySource` interface: `Collect(ctx, week) ([]WorkActivity, error)`.
- Issue key extraction by regex, then verified against Jira; unknown keys are dropped.

### 5.5 Learning

- `Feedback` entity: `Source` (reject_reason, thread_correction, tempo_edit), `Week`, `Text`, `Diff` (for tempo_edit), `CreatedAt`.
- `Lesson` aggregate: `Text`, `Category` (preference, channel_map, correction), `SourceFeedbackIDs`, `Status` (Active, Retired), `CreatedAt`.
- `SkillProposal` entity: `Name`, `Content`, `Status` (Proposed, Approved, Rejected).
- Lessons are learned only from the user's own feedback signals, never from raw Slack message content.
- Lessons activate automatically. Skill proposals require approval in Slack.
- Caps: max 50 active lessons; oldest lessons in a category are retired first when the cap is hit.

## 6. Weekly flow

1. Trigger: cron Friday 16:00 local time, plus hourly catch-up tick. Both issue `ProposeTimesheet(week)`. Idempotent: no-op if a timesheet for that week exists.
2. Collect: Slack `search.messages` with `from:@me after:<Sunday> before:<Saturday>` (user token, paginated), plus `conversations.replies` for threads the user joined.
3. Allocate: `AgentPolicy` first.
4. Validate: translate the proposal into `AddEntry` commands on a `Draft` aggregate. On any invariant failure, retry the agent once with the error list. If it fails again, or the agent errors or times out, use `RulePolicy`.
5. Propose: status `AwaitingApproval`; `TimesheetProposed` is written to the outbox in the same transaction; relay sends a Block Kit DM: Mon-Fri table, per-day totals, policy used, short reasoning, buttons Approve and Reject.
6. Correct (optional): a reply in the DM thread is stored as `Feedback`; `ReviseTimesheet` moves the timesheet back to `Draft`, re-runs the agent with the correction, and posts a new DM version (the old message is updated to "superseded").
7. Reject: a modal asks for an optional reason (stored as `Feedback`); status `Rejected`. The user can rerun with `timesheet plan --week <id> --force`.
8. Approve: status `Approved`; `TimesheetApproved` to outbox; relay calls Tempo `POST /4/worklogs` per entry and stores `tempoWorklogId` per entry. Entries already holding a `tempoWorklogId` are skipped on retry. When all entries are submitted: status `Submitted`, confirmation DM.
9. Monday learning job: diff last week's submitted entries against current Tempo worklogs (by `tempoWorklogId`). Each difference becomes `Feedback`. Then `DistillLessons` runs.

## 7. AI agent

> **Replaced by [ADR-0007](../../adr/0007-grpc-contracts-now-split-later.md) (Accepted 2026-10-09).** The in-process Go `claudeagent` described in this section is not built. The AI agent is the Python LangGraph service `services/allocation-agent/`, which serves `PolicyService` over gRPC; the Go core reaches it through `internal/allocation/infrastructure/grpcpolicy`, and its proposals still pass the `Timesheet` aggregate and the policy fallback chain. This section is kept as the original design record.

### 7.1 Model and limits

From `agent/agent.yaml`:

- `model`: default `claude-sonnet-5-5`, configurable.
- `max_tool_turns`: 15.
- `max_tokens_per_run`: budget enforced by the loop.
- `timeout`: 90s.
- `daily_cap_minutes`: 480.
- `enabled_skills`: list of skill names.

### 7.2 Prompt assembly

System prompt = `prompts/system.md` + skill index (name and description of each enabled skill) + active lessons rendered from `memory/` + week metadata. The activities are passed in the first user message in a compact form (day, channel, keys, truncated text, permalink id).

### 7.3 Tools

| Tool                                  | Backed by              | Purpose                                             |
| ------------------------------------- | ---------------------- | --------------------------------------------------- |
| `load_skill(name)`                    | Learning workspace     | Return full `SKILL.md` content                      |
| `get_thread(permalink_id)`            | ActivitySource         | Full thread text for context                        |
| `lookup_issue(key)`                   | Jira                   | Summary, status, project, id                        |
| `search_issues(project, text)`        | Jira                   | Find an issue for keyless activity                  |
| `list_tempo_worklogs(week)`           | Tempo                  | Avoid double-logging time already in Tempo          |
| `get_leave_and_holidays(week)`        | Config + calendar file | Skip or fix-allocate leave days and public holidays |
| `submit_proposal(entries, reasoning)` | Loop terminator        | Structured final output; ends the loop              |

The agent has no tool that writes to Tempo, Slack, or the workspace. All writes happen in application command handlers.

### 7.4 Anti-corruption layer

`claudeagent` translates the `submit_proposal` input into the `Proposal` value object. Unknown issue keys, non-15-minute durations, weekend days, and over-cap days are returned as validation errors to the retry step. Agent output never reaches the aggregate without going through domain constructors.

### 7.5 Reflect run (learning)

`DistillLessons` runs the agent with the `reflect` skill and tools `list_feedback(since)`, `list_active_lessons()`, and terminator `submit_lessons(lessons, retire_ids, skill_proposals)`. The handler writes lessons (active immediately), retires the requested ones, and stores skill proposals as `Proposed` in Postgres and in `agent/skills/_proposed/<name>/SKILL.md`. A DM lists new skill proposals with Approve and Reject buttons. Approval moves the folder into `agent/skills/<name>/` and adds it to `enabled_skills`.

## 8. Agent workspace (mounted volume)

```text
agent/
├── agent.yaml
├── prompts/
│   └── system.md
├── skills/
│   ├── allocate-hours/SKILL.md
│   ├── map-channel-to-issue/SKILL.md
│   ├── write-worklog-description/SKILL.md
│   ├── detect-meetings-and-leave/SKILL.md
│   ├── reflect/SKILL.md
│   └── _proposed/
└── memory/
    ├── preferences.md
    ├── channel-map.md
    ├── corrections.md
    └── _pending/
```

- `skills/` and `prompts/` are written by the user; the bot only reads them, except when moving an approved proposal out of `_proposed/`.
- `memory/` files are exported from Postgres after each learning run; Postgres is the source of truth. Manual edits to `memory/` are overwritten - edit lessons via CLI (`timesheet lessons list|retire|add`).
- `_pending/` is reserved for future use if lesson approval is turned on (`learning.lessons_require_approval: false` by default).

## 9. Project layout

> **Note:** [`docs/architecture/project-structure.md`](../../architecture/project-structure.md) supersedes the layout below ([ADR-0008](../../adr/0008-monorepo-service-layout.md)). The Go paths shown here now live in `services/timesheet-core/`.

```text
timesheet-bot/
├── cmd/timesheet/main.go               # composition root, cobra commands
├── internal/
│   ├── shared/kernel/
│   ├── timesheet/
│   │   ├── domain/
│   │   ├── application/                # Propose, Revise, Approve, Reject, Submit, CatchUpMissedWeeks
│   │   │   └── ports.go                # WorklogGateway, ApprovalNotifier, IssueDirectory
│   │   └── infrastructure/
│   │       ├── postgres/
│   │       ├── tempo/
│   │       └── jira/
│   ├── allocation/
│   │   ├── domain/                     # AllocationPolicy, Proposal, RulePolicy
│   │   └── infrastructure/claudeagent/
│   ├── activity/
│   │   ├── domain/
│   │   └── infrastructure/slack/
│   ├── learning/
│   │   ├── domain/
│   │   ├── application/                # CaptureFeedback, DistillLessons, ApproveSkill, ExportMemory
│   │   └── infrastructure/
│   │       ├── postgres/
│   │       └── fsworkspace/
│   └── interfaces/
│       ├── scheduler/
│       ├── slackbot/                   # Socket Mode: buttons, modals, thread replies
│       ├── outbox/                     # relay: outbox rows -> event handlers
│       └── cli/
├── agent/                              # default workspace content (section 8)
├── config/config.example.yaml          # channel map, timezone, cron, no secrets
├── migrations/                         # goose SQL
├── Dockerfile                          # multi-stage, distroless static, non-root
└── docker-compose.yml                  # app + postgres:17, named volumes
```

## 10. Libraries

- `github.com/slack-go/slack` (Web API + `socketmode`)
- `github.com/anthropics/anthropic-sdk-go`
- `github.com/jackc/pgx/v5`, `sqlc` for queries, `github.com/pressly/goose/v3` for migrations
- `github.com/robfig/cron/v3`
- `github.com/spf13/cobra`
- `log/slog` for structured logs
- Tests: standard `testing`, `github.com/testcontainers/testcontainers-go` for Postgres

## 11. Persistence

Tables:

- `timesheets` (id, user_account_id, week_id, status, policy, version, proposal_revision, slack_message_ts, created_at, updated_at; unique (user_account_id, week_id))
- `timesheet_entries` (id, timesheet_id, issue_key, issue_id, day, minutes, description, tempo_worklog_id, submit_attempts, last_error)
- `outbox` (id, aggregate_id, event_type, payload jsonb, created_at, processed_at, attempts)
- `feedback` (id, source, week_id, text, diff jsonb, created_at)
- `lessons` (id, category, text, status, source_feedback_ids, created_at, retired_at)
- `skill_proposals` (id, name, content, status, created_at, decided_at)
- `agent_runs` (id, kind, week_id, model, input_tokens, output_tokens, tool_turns, outcome, created_at) for cost and debugging

Optimistic concurrency on `timesheets.version`.

## 12. Configuration and secrets

Secrets are read from environment variables only (exported in `~/.zshrc`, passed through by compose `environment:` without values in the file):

| Variable                       | Purpose                                                                           |
| ------------------------------ | --------------------------------------------------------------------------------- |
| `SLACK_USER_TOKEN`             | `xoxp-`, scopes `search:read`, `channels:history`, `groups:history`, `im:history` |
| `SLACK_BOT_TOKEN`              | `xoxb-`, scopes `chat:write`, `im:write`                                          |
| `SLACK_APP_TOKEN`              | `xapp-`, `connections:write` for Socket Mode                                      |
| `TEMPO_API_TOKEN`              | Tempo REST v4 bearer token                                                        |
| `JIRA_BASE_URL`                | e.g. `https://<site>.atlassian.net`                                               |
| `JIRA_EMAIL`, `JIRA_API_TOKEN` | Jira Cloud basic auth for issue lookup                                            |
| `ANTHROPIC_API_KEY`            | Claude API                                                                        |
| `DATABASE_URL`                 | Postgres DSN                                                                      |

Non-secret config in `config/config.yaml`: timezone, cron expression, user's Jira `accountId`, Slack user id, channel to default-issue map, daily cap.

## 13. Error handling

- Slack, Tempo, Jira 429: exponential backoff honoring `Retry-After`, max 5 attempts per call.
- Anthropic error or timeout: no retry beyond the single validation retry; fall back to `RulePolicy`.
- Tempo partial failure: timesheet stays `Approved`, failed entries keep `last_error`; relay retries on its tick; after 3 failed attempts for an entry, `EntrySubmissionFailed` sends a DM listing those entries.
- Outbox relay: at-least-once delivery; every handler is idempotent (keyed by timesheet id + revision, or entry id).
- Socket Mode disconnect: the SDK reconnects; button clicks during downtime are lost by Slack, so the DM also shows the CLI fallback `timesheet approve --week <id>`.

## 14. Testing

- Domain: table-driven unit tests for value objects, aggregate invariants, and state transitions.
- Application: command handlers with in-memory fakes for every port.
- `RulePolicy`: golden tests for allocation and cap scaling.
- `claudeagent`: fake HTTP transport replaying recorded Anthropic responses; tests for tool dispatch, turn limit, malformed `submit_proposal`, and fallback.
- Infrastructure: Postgres repositories with testcontainers; Slack, Tempo, Jira adapters against `httptest` servers.
- End-to-end smoke: `timesheet plan --week <id> --dry-run` prints the proposal without DM or DB writes.

## 15. CLI

```text
timesheet serve
timesheet plan    --week 2026-W41 [--dry-run] [--force] [--policy rule|agent]
timesheet approve --week 2026-W41
timesheet submit  --week 2026-W41
timesheet lessons list | add | retire <id>
timesheet skills  list | approve <name> | reject <name>
timesheet migrate up | down
```
