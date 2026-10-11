# ADR-0010: Add Mavenlink as a second submit target through its REST API

- Status: Proposed
- Date: 2026-10-09
- Deciders: owner
- Related: [ADR-0002](0002-postgres-with-transactional-outbox.md) (per-entry submit idempotency), [ADR-0006](0006-personal-tool-scale-ready-aws-later.md) (single-user tool)

## Context

The owner's company tracks time in Mavenlink (Kantata OX) at `https://27global.mavenlink.com/timesheets`, in addition to or instead of Jira Tempo. `submit` must write the approved week there and send it for approval, without breaking the existing Tempo path.

Facts from the official Kantata OX OpenAPI spec (`developer.kantata.com/_bundle/Kantata/specification.yaml`) and developer docs, read 2026-10-09. They were not yet confirmed by `official-source-verifier`.

- **API:**
  - Base URL `https://api.mavenlink.com/api/v1`, with `Authorization: Bearer <token>`.
  - `POST /time_entries.json` takes `{"time_entry": {...}}`. It requires `workspace_id`, `date_performed` and `time_in_minutes`. `story_id`, `role_id`, `notes`, `billable` and an `external_reference` object are optional.
  - `POST /timesheet_submissions.json` takes `{"timesheet_submission": {"workspace_id", "line_item_ids", "title"}}`. "A TimesheetSubmission can be associated with only one project."
  - `GET /time_entries.json` filters by `date_performed_between` and `external_reference_service_name`.
  - With `include=external_references,workspace,active_submission`, the list returns `workspace_id`, `active_submission_id` and `external_reference_ids` per entry.
- **Tokens:**
  - Only an account administrator can register the OAuth app that issues a token. The token for your own user is on the app's page, under "Show your OAuth Token".
  - Tokens do not expire.
- **Idempotency:** the API has none for creates.

The owner cannot get an OAuth app registered. Their browser session works, and they shared a DevTools request with the coordinator; only the header names were used here.

## Decision

1. **Config:**
   - `submit_targets` in `config.yaml` lists `tempo`, `mavenlink`, or both.
   - The default is `[tempo]`, so existing configs behave as before.
2. **Domain:**
   - The domain knows the targets (`domain.Target`).
   - An entry stores one id per target (`TempoWorklogID`, `MavenlinkEntryID`; migration `00002`).
   - `MarkEntrySubmitted` records an id for one target only.
   - The new `CompleteSubmission(targets)` moves the timesheet to `submitted` only when every entry is in every configured target.
   - A failure in one target never marks the entry as sent to the other.
3. **Idempotency:**
   - Every Mavenlink time entry carries the external reference `timesheet-bot` / `timesheet_entry` / `<entry ID>`.
   - Each run first lists the week's entries with that reference and adopts them instead of creating them again.
   - Creates are never re-sent by the retry transport: the client clears `GetBody`. A 429, a 5xx or a timeout fails the entry, and the next run adopts the entry if Mavenlink stored it.
4. **Approval:**
   - Once every entry is in Mavenlink and `mavenlink.submit_for_approval` is true, `submit` sends one timesheet submission per Mavenlink project, titled `Week <ISO week>`.
   - Entries already in an active submission are skipped.
   - The timesheet becomes `submitted` only after the submissions succeed, so a failed submission can be retried.
5. **Mapping:**
   - Rules in `mavenlink.mappings` map a Jira issue key or Jira project to a Mavenlink project (`workspace_id`), with an optional task, role and billable flag.
   - Defaults catch the rest.
   - `MavenlinkMapping.Resolve` follows the owner's policy (2026-10-10, implemented on the owner's request):
     - **Matching:**
       - `issue_key` compares the whole key, ignoring case.
       - `jira_project` compares the key's prefix before the first `-`.
       - `project` (report mode, [ADR-0012](0012-report-mode-slack-summaries-to-mavenlink-without-jira.md)) compares names after `NormalizeReportText`: case ignored, white space trimmed and collapsed, the same normalization dedup uses.
     - **Precedence:** the most specific rule wins (`issue_key`, then `project`, then `jira_project`); within one kind, the first rule in config order wins.
     - **Config validation:** `config.Load` rejects two rules with the same `issue_key` or the same normalized `project`, naming both indexes, and a rule that combines `project` with a Jira field.
     - **No match:**
       - With `default_workspace_id` set, the entry goes to the default project, flagged as a fallback.
       - Without it, the error is `no Mavenlink mapping for project "<name>": add a mavenlink.mappings rule or set default_workspace_id` (or `for issue "<key>"`). Notes are never included.
     - **Role:** the rule's `role_id`, else `default_role_id`.
     - **Task:** the rule's `story_id`, else `default_story_id` only when the entry lands in the default project, because a task belongs to one project.
     - **Billable:** the rule's flag, else Mavenlink's default.
   - `plan` and `plan --dry-run` resolve every entry when Mavenlink is a submit target:
     - a `MAVENLINK` column shows the project and task;
     - a `warning:` names each project or issue that fell back to the default;
     - a resolution error fails the plan, before anything is saved.
6. **Auth modes:** `mavenlink.auth` picks one. The endpoint code is shared and only the `Auth` strategy differs.
   - **`bearer`** (default, documented): `MAVENLINK_API_TOKEN`, base URL `https://api.mavenlink.com/api/v1`.
   - **`cookie`** (fallback, undocumented and unsupported by Kantata): `MAVENLINK_SESSION_COOKIE` holds the full raw `Cookie` header of a logged-in browser. The base URL defaults to `https://27global.mavenlink.com/api/v1`, because the cookie is valid only on the account's host.
     - Before its first request it reads `<meta name="csrf-token">` from `mavenlink.csrf_page_url` (default `https://27global.mavenlink.com/timesheets`).
     - Every request then sends these headers, like the web client: `X-CSRF-Token`, `X-Requested-With: XMLHttpRequest`, the page as `Referer`, and `Content-Type: application/json;charset=utf-8` on writes.
     - Redirects are never followed.
     - Any of these means the session expired, and the run stops at once without touching any entry: a 401, a redirect to `login.microsoftonline.com` or a `/login` path, or an HTML page where JSON was expected.
     - A 403 or 422 to a request sent without a CSRF token is reported as a CSRF rejection.
     - The cookie and the CSRF token are removed from every error.

## Status of the cookie-mode facts

| Fact | Status |
|---|---|
| The instance path `https://27global.mavenlink.com/api/v1/...` accepts the session cookie | Observed in a browser request (2026-10-09) |
| The cookie names carry an instance suffix (`_mavenlink_session_<n>` and others), so the whole header is copied, never single cookies | Observed |
| The browser sends `x-csrf-token` on GET requests too, so the client sends it on every request | Observed, not verified as required |
| Cookie-authenticated writes are rejected without `X-CSRF-Token` | Assumed from Rails conventions, not verified on this instance |
| An expired session answers with a redirect to Microsoft Entra ID (`login.microsoftonline.com`) or `/login`, or with an HTML page | Inferred from the SSO setup; the exact response is not yet observed |
| Time entries on this instance need a `role_id` (the web UI calls `GET /api/v1/workspaces/{id}/rate?role_id=...`) | Observed in the web client, not verified as required by the API; `role_id` is optional in config |

## Alternatives considered

- **Browser automation (Playwright) of the timesheet grid:** rejected. It is heavier than calling the same JSON API the page uses, and it breaks on UI changes. It would also still need the same SSO session.
- **Replacing Tempo:** rejected. Both systems may be required, and `submit_targets` lets the owner pick either or both.
- **Matching by notes or dates instead of an external reference:** rejected. It is ambiguous; the external reference is exact and filterable.

## Consequences

- **Positive:**
  - One `submit` writes both systems.
  - Retries never duplicate Mavenlink entries.
  - Cookie mode works without an administrator.
- **Negative / accepted risk:**
  - Cookie mode depends on undocumented behavior and stops whenever the SSO session ends. The owner refreshes `MAVENLINK_SESSION_COOKIE` and runs `submit` again.
  - `SubmitAttempts` counts failures across both targets.
  - The timesheet is not split per target in the status: it is `approved` until both are done.
- **Follow-up:**
  - Confirm the cookie-mode assumptions above in DevTools on a real week.
  - Verify the API facts with `official-source-verifier`.
  - Move this ADR to `Accepted`.
