# ADR-0014: LLM provider port with DeepSeek for report parsing

- Status: Proposed
- Date: 2026-10-10
- Deciders: owner
- Related: [ADR-0011](0011-collect-slack-activity-through-claude-slack-mcp-connector.md) (activity week files), [ADR-0012](0012-report-mode-slack-summaries-to-mavenlink-without-jira.md) (report mode), [ADR-0013](0013-per-service-env-files.md) (per-service `.env`)

## Context

- In report mode (ADR-0012), each line of the owner's Slack timesheet summaries becomes one Mavenlink entry.
- The summaries are free text: day headings, `4h` / `2.5h` / `1h30` / `30m`, several projects per message. A hand-written parser (`reportparser.Parser.Parse`, `TODO(owner)`) has to anticipate every format.
- The owner decided to have an LLM parse each summary into structured lines, and chose DeepSeek.
- The owner also wants another provider (Anthropic, OpenAI, ...) to be a one-adapter change.
- The collector (`services/allocation-agent`, `collect`) already holds the message text for one week. It is the natural place to parse, so the Go core receives validated lines and never calls an LLM.

## Decision

1. **Port (Strategy):** `allocation_agent.llm.port.LLMClient` has one method, `complete_json(system, user, *, schema_name) -> dict`. Every failure is an `LLMError` subclass:
   - `LLMAuthError`: the key is missing, empty or rejected (401), or the balance is insufficient (402);
   - `LLMRateLimitError`: 429 after the retries;
   - `LLMUnavailableError`: 5xx after the retries, a timeout, or a connection error;
   - `LLMResponseError`: another 4xx, or the answer is empty, cut off (`finish_reason` other than `stop`), not JSON, or not an object;
   - `UnknownProviderError`: `llm.provider` names no registered provider.
2. **Factory and registry:** `llm.registry.register(name, factory)` and `create(settings)`. Adapters register themselves when imported, and `llm/providers.py` imports every built-in adapter. That file is the one marked place to add a provider. An unknown name raises an error that lists the registered names.
3. **DeepSeek adapter** (`llm/deepseek.py`), over httpx (pinned `httpx>=0.28,<1`, locked at 0.28.1):
   - `POST https://api.deepseek.com/chat/completions` with a Bearer `DEEPSEEK_API_KEY`;
   - `response_format: {"type": "json_object"}`, `thinking: {"type": "disabled"}`, `temperature: 0`, `max_tokens: 8192`, no streaming;
   - default model `deepseek-flash`;
   - `llm.timeout_seconds` per request, and up to `llm.max_retries` retries on 429 and 5xx only, with `Retry-After` or exponential backoff capped at 30 s;
   - the key is read from the environment only. A missing or empty key raises an error that names `DEEPSEEK_API_KEY` and nothing else.
   - httpx was chosen over the `openai` SDK because:
     - the adapter needs one endpoint;
     - httpx gives exact control of the retry policy and of what an error contains;
     - `httpx.MockTransport` tests the request shape offline;
     - the SDK would add its own retry layer and more transitive packages, and the `thinking` field would still need `extra_body`.
4. **Extractor** (`report/extractor.py`): `ReportExtractor(llm).extract(messages, week, tz)` makes one call per message.
   - The input is the message text, its posted date, the week's Monday-Friday dates and the timezone.
   - The answer must be `{"lines": [{"date", "project", "minutes", "notes"}]}`. It is validated strictly: no extra keys, integer minutes, real dates.
   - Every line is checked with the Go core's rules: minutes are a positive multiple of 15, the date is a Monday-Friday of the week, and the project is not blank.
   - The extractor depends only on the port.
5. **Contract v2** (`api/activity/activity-week.v2.schema.json`): version 1 plus `parser: {kind, provider, model}` and `report_lines`. `report.parser: llm` writes v2. `report.parser: manual` keeps writing v1.
6. **Config:** `report.parser` (`manual` | `llm`, default `manual`) and the `llm` block (`provider`, `model`, `base_url`, `timeout_seconds`, `max_retries`) live in `services/timesheet-core/config/config.yaml`.
   - `report.parser` is read from that file only, with no flag and no environment override. The Go core reads the same key to choose between contract v1 and v2, so an override in one service could make the collector write v1 while Go expects v2, or the other way round.
   - The `llm.*` keys follow flag > env > yaml (`--llm-provider`, `--llm-model`, `LLM_*`). Go ignores them.

### Fail-stop policy

The owner's rule is to stop on any LLM error or invalid output.

- Any `LLMError`, or any invalid answer or line for any message, stops `collect`.
- `collect` exits 4 (`EXIT_LLM_FAILED`) and writes no file.
- There is no fallback: no manual parser, no partial week, no retry beyond the bounded 429/5xx retries, and no rounding of minutes.
- A missing key fails before any Claude or Slack usage.
- Errors name the message's `ts` and channel id, the provider, the HTTP status or error code, and the field paths. They never contain message text, notes, prompts, response bodies or the key.

### Data residency

- `report.parser: llm` sends the full text of each timesheet summary to DeepSeek, a third party. DeepSeek's privacy policy says it collects, processes and stores personal data in the People's Republic of China (https://cdn.deepseek.com/policies/en-US/deepseek-privacy-policy.html, last updated 2026-02-10).
- Summaries name clients, projects and work done. **The owner must confirm that company policy allows sending this data to DeepSeek before turning `llm` on.** `manual` remains the default, and with it no data leaves for an LLM.
- If policy forbids it, add another provider adapter (for example Anthropic) and set `llm.provider`. Nothing else changes.

### Cost

Pricing from https://api-docs.deepseek.com/quick_start/pricing, checked 2026-10-10, for `deepseek-flash` per 1M tokens:

| | Off-peak | Peak |
|---|---|---|
| Input, cache hit | $0.003 | $0.006 |
| Input, cache miss | $0.15 | $0.30 |
| Output | $0.60 | $1.20 |

- Peak hours are 01:00-04:00 and 06:00-10:00 UTC, Monday to Friday (08:00-11:00 and 13:00-17:00 in `Asia/Ho_Chi_Minh`). All other hours are off-peak.
- Estimate: about 1.5K input and 300 output tokens per summary. Ten summaries a week (two places, five days) come to about 15K input and 3K output tokens.
- That is under $0.01 a week at peak prices. Retries can at most triple it.

## Alternatives considered

- **The `openai` SDK with `base_url`:** rejected, for the reasons in decision 3.
- **Parse in Go** (`reportparser`, `TODO(owner)`): kept as `report.parser: manual`. It needs no third party, but every format has to be hand-coded.
- **Fall back to manual parsing or skip a bad message:** rejected by the owner. A silent fallback could log wrong hours.
- **One call for the whole week:** rejected. A per-message call keeps each answer small, makes each error point to one message, and lets the line carry its `source_ts`.
- **Claude through the existing claude.ai login:** not chosen by the owner. It remains a one-adapter change.

## Consequences

- **Positive:**
  - Free-text summaries are parsed without hand-written rules.
  - The Go core gets validated lines, and it validates them again.
  - Adding a provider means one module plus one import line.
- **Negative / accepted risk:**
  - Summaries leave the machine for a provider hosted in China (see data residency).
  - A paid account and `DEEPSEEK_API_KEY` are needed.
  - LLM output can be wrong in ways that pass validation (for example a wrong project name). The owner still reviews every timesheet before submit.
  - Model names change. `deepseek-chat` and `deepseek-reasoner` are no longer listed, so a stale `llm.model` fails with HTTP 400/422 and exit 4.
- **Follow-up:**
  - The owner confirms the data policy.
  - The owner adds 2-3 redacted few-shot examples (`report/prompt.py`, `FEW_SHOT_EXAMPLES`, `TODO(owner)`).
  - Move this ADR to `Accepted`.
