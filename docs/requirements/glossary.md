# Glossary (ubiquitous language)

Maintained by `business-analyst`. Code names must match identifiers in `internal/`.

| Term | Definition | Code name |
|---|---|---|
| Week | ISO-8601 week, Monday to Friday only | `kernel.WeekID` |
| Work day | A Monday-Friday calendar date in the configured timezone | `kernel.WorkDay` |
| Issue key | Jira key such as `GCM-12` | `kernel.IssueKey` |
| Activity | One piece of evidence of work, such as a Slack message | `kernel.WorkActivity` |
| Timesheet | One user's worklog lines for one week, with an approval lifecycle | `timesheet/domain.Timesheet` |
| Entry | Time spent on one issue on one day | `timesheet/domain.Entry` |
| Worklog | An entry as written to Tempo | `application.Worklog` |
| Proposal | A policy's suggested entries, not yet validated | `allocation/domain.Proposal` |
| Policy | A way to turn activity into a proposal (rule, agent) | `allocation/domain.Policy` |
| Daily cap | Maximum minutes per day (default 480) | `DailyCap` |
| Per-key credit | Minutes credited per issue key per day by the rule policy (default 45) | `PerKey` |
| Channel default | Issue credited for keyless activity in a channel | `ChannelDefaults` |
| Owner | The person a timesheet belongs to: their Slack activity, their Tempo week. Every timesheet, event and outbox row carries the owner's id (NFR-12) | `Timesheet.UserAccountID()`, column `user_account_id` |
| Unknown outcome | A Tempo create that may or may not have stored the worklog: a 5xx, a timeout, or a stop before the worklog id was saved. The entry is checked against Tempo before anything is sent again | `application.ErrCreateIndeterminate` (planned, US-007) |
| Outbox | The table of events stored in the same transaction as the change that caused them; a relay delivers them later (Phase 2) | table `outbox` |
| Fitness function | An automated test that checks an architecture rule, such as the dependency rule or no wall clock in the domain | `test/architecture` (in `services/timesheet-core/`) |
| Contract test | A shared test that every implementation of a port must pass, such as both timesheet stores, or a test of an adapter against recorded responses of the real API | `domaintest.RunRepositoryContract` (planned, US-010); adapter `testdata` fixtures (planned, US-023) |
