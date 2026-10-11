# ADR-0012: Report mode: Slack summaries to Mavenlink without Jira

- Status: Proposed
- Date: 2026-10-10
- Deciders: owner
- Related: [ADR-0004](0004-allocation-policy-fallback-chain.md) (policy chain), [ADR-0010](0010-add-mavenlink-as-a-second-submit-target.md) (Mavenlink target), [ADR-0011](0011-collect-slack-activity-through-claude-slack-mcp-connector.md) (activity week files)

## Context

The owner has no Jira and no Tempo. Their real flow is Slack to Mavenlink only.

- They write their own timesheet summaries in Slack, in two places:
  - a dedicated summary channel;
  - their self-DM.
- Each summary names a project, the time and notes.
- The allocation-agent collector writes those messages to the activity week file (ADR-0011, contract v1 unchanged).

The existing `jira_keys` flow cannot serve this owner:

- It credits 45 minutes per Jira key found in any message.
- It resolves each key through Jira.
- It scales days over the cap (`scaleDay`, `TODO(owner)`).
- Its entries need a Jira issue, as Tempo does.

## Decision

1. **Mode switch:** `allocation.mode: jira_keys | report`. The default is `jira_keys`, so existing configs behave as before.
2. **Config validation in report mode:**
   - `JIRA_*` and `jira_account_id` are not required, and no Jira client is built.
   - `slack_user_id` owns the timesheets (`Config.OwnerID`).
   - `submit_targets` must not contain `tempo`: Tempo logs time on Jira issues only.
   - At least one of `report.channel` and `report.include_self_dm` must be set. These are the keys the collector reads, with the same rules on both sides:
     - an absent `include_self_dm` is `false`, so a config with no report source is an error;
     - `report.channel` loses a leading `#` and must match `[a-z0-9._-]+`; a bare `#` means no channel.
   - `activity.source` must be `file`. Slack search would feed every message, not only the summaries.
3. **Parsing:**
   - The port is `allocation/domain.ReportParser`: `Parse(msg kernel.WorkActivity, tz *time.Location) ([]ReportLine, error)`.
   - `ReportLine` is `{Date, Project string, Minutes int, Notes string}`.
   - It takes the shared-kernel activity rather than `activity/domain.Message`, because a domain package must not import another context's domain (architecture tests).
   - The adapter is `allocation/infrastructure/reportparser`. Its `Parse` is `TODO(owner)`: the owner decides the format (day headings, hour formats, separators, lines without hours, weekly or daily messages).
4. **Allocation:**
   - `ReportPolicy` makes each report line one entry, with its date, minutes, notes and project.
   - It applies no 45-minute rule, no `channel_defaults` and no scaling.
   - Lines on a weekend or outside the week are skipped with a warning.
   - A malformed date, an empty project, or minutes that are not a positive multiple of 15 fail the proposal, naming the line. Minutes must fit `kernel.Duration`, so the parser decides whether to round.
5. **Dedup policy** (`allocation/domain.DedupReportLines`): the owner posts the same summary in both places, so the same line arrives twice.
   - Exact duplicates are dropped, and the first one is kept.
   - The key (`ReportDedupKey`) is the date, the project after `NormalizeReportText` (lowercase, white space collapsed), the minutes, and the notes after the same normalization.
   - Lines that differ in any of these are all kept, such as two 1h blocks on one project with different notes.
   - To change the policy, change `DedupReportLines`/`ReportDedupKey` and this section.
6. **Over-cap warning:**
   - A day whose report total exceeds `daily_cap_minutes` is kept as reported, never scaled.
   - `plan` and `plan --dry-run` print `warning: <date>: <total> reported, over the daily cap of <cap>; kept as reported, not scaled`.
   - The owner's own report is the source of truth.
7. **Entries:**
   - `domain.Entry` gains `Project`. A report entry has a project and no issue key.
   - `Timesheet.AddReportEntry` accepts several entries per project and day, and does not apply the daily cap.
   - IDs are `<timesheet>/<day>/report-NN`.
   - Migration `00003_report_entry_project`:
     - adds `project`;
     - replaces the one-issue-per-day unique constraint with a unique index on Jira entries only;
     - checks that exactly one of `issue_key` and `project` is set.
8. **Mapping:**
   - `MavenlinkMapping.Resolve(key, project)` receives the report project.
   - Mapping rules gain `project`. `issue_key` and `jira_project` stay for `jira_keys` mode.
   - A `project` rule matches after `NormalizeReportText`, the same normalization dedup uses, so a project that dedups as one name also maps as one name.
   - The full policy is in [ADR-0010](0010-add-mavenlink-as-a-second-submit-target.md), decision 5: precedence, fallback, task and role rules, and duplicate-rule validation.
   - `plan` shows each entry's Mavenlink project and warns about every project that fell back to `default_workspace_id`.

## LLM parser path (`report.parser: llm`)

Added 2026-10-10. The owner chose to have an LLM (DeepSeek) in the allocation-agent collector parse the summaries. The LLM decision and its prompt belong to the collector's own ADR.

- **Config:**
  - `report.parser: manual | llm`. The default is `manual`, which keeps the behavior above: the hand-written `reportparser.Parser`, `TODO(owner)`.
  - The `llm` block (`provider`, `model`, `base_url`, `timeout_seconds`, `max_retries`) is read by the collector only. Go loads `config.yaml` with non-strict YAML decoding, so the block is ignored without a struct.
- **Input:**
  - With `llm`, the collector writes activity file **version 2**: version 1 plus `parser` and `report_lines`.
  - The collector validates every line, and on any error writes no file.
  - Go reads `report_lines` through `file.Collector.ReportLines` and never calls the hand-written parser.
  - A version 1 file, or a version 2 file without `report_lines`, fails `plan` with `activity file has no report_lines: re-run the collector with report.parser: llm`.
- **Re-validation:**
  - Go checks every line again: minutes a positive multiple of 15, the date a Monday to Friday of the requested week, and a non-empty project.
  - It fails at the first invalid line, naming its index and `source_ts`, never its notes.
  - Unlike the manual path, a weekend or out-of-week line is an error, not a warning, because the collector promises it never writes one.
- **After the lines are read,** everything is shared with the manual path: `DedupReportLines`, the over-cap warning, `AddReportEntry`, and the Mavenlink mapping. Tests check that the same lines give the same proposal on both paths.
- **The seam** respects the rule that a domain package must not import another context's domain:
  - `ReportLine` moved to the shared kernel (`kernel.ReportLine`, with `SourceTS` and `SourceChannelID`), next to `WorkActivity`. `allocation/domain.ReportLine` is an alias, so the policy and the parser port are unchanged.
  - A consumer-defined port, `timesheet/application.ReportLineSource`, is implemented by the activity file adapter.
  - `ProposeHandler.Reports` fills `alloc.Input.ReportLines`.
  - `ReportPolicy{Precomputed: true}` takes its lines from there instead of calling `Parser`.
  - Rejected alternatives:
    - **A "precomputed" `ReportParser` keyed by message `ts`:** it would need the message `ts` on `WorkActivity`, and it would lose lines whose source message falls outside the working days, such as a summary posted on Saturday about Friday.
    - **Carrying lines on `WorkActivity`:** the same loss.
- **Dedup** ignores `SourceTS` and `SourceChannelID`, so the channel copy and the self-DM copy of a line are still one line.

## Alternatives considered

- **Merge lines of one project and day into one entry:** rejected. It loses the separate notes the owner wrote, and Mavenlink accepts several entries per day.
- **Scale over-cap days like `jira_keys` mode:** rejected. The report is what the owner says they worked; silently changing it would be wrong. A warning lets them fix the report.
- **Fuzzy dedup** (same date and project with different minutes, or similar notes): rejected for now. It could drop real work. Exact duplicates cover the two-places case.
- **Filtering report messages in Go by channel:** rejected. The collector already reads only `report.channel` and the self-DM, and a self-DM cannot be told from other DMs by channel id alone.

## Consequences

- **Positive:**
  - The owner's real flow works without Jira or Tempo.
  - `jira_keys` mode is unchanged.
  - Both modes share the timesheet, approval and Mavenlink submission.
- **Negative / accepted risk:**
  - The daily cap is advisory in report mode.
  - Minutes must be multiples of 15.
  - Switching modes changes the timesheet owner id, so earlier weeks stay under the old id.
  - Migration `00003` down deletes report entries.
- **Follow-up:**
  - The owner implements `reportparser.Parser.Parse` if they keep `report.parser: manual`, and removes its `TODO(owner)` skip. (`MavenlinkMapping.Resolve` was implemented on 2026-10-10 at the owner's request.)
  - Move this ADR to `Accepted`.
