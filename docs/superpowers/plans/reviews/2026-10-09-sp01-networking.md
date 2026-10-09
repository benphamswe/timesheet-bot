# SP-01 hardening plan: networking review

Reviewer: `networking-engineer`, mode `tech-review`. Date: 2026-10-09.

Plan reviewed: `docs/superpowers/plans/2026-10-09-sp01-hardening.md` (`status: draft`). The plan was not edited.

## Summary

The redesigned retry transport is correct in its core rule. It retries 429 for every method, retries 502, 503 and 504 only for idempotent methods or a `WithReadOnly` ctx, never retries a POST on 5xx, uses full jitter, makes 3 attempts and has a ctx-aware wait. RFC 9110, the AWS Builders' Library and the Atlassian guidance all support that rule. The deadline cap works against the 30s `Client.Timeout`, not only against caller deadlines: Go 1.26.9 copies the client timeout into `req.Context()` (verified in source, below).

The plan cannot go final yet. It needs these changes:

- the transport still breaks on `http.NoBody`;
- `Retry-After` has no upper bound when there is no deadline;
- the Tempo reconcile lookup advances the offset by a constant;
- the lookup relies on undocumented date-filter semantics;
- the F12 test asserts something that cannot be true;
- the Postgres pool has no connect timeout.

Each fix is small.

## Verdicts

| WP | Scope reviewed | Verdict |
|---|---|---|
| WP-2 | Retry transport, `WithReadOnly`, Slack opt-in | CONDITIONAL (N1 to N6) |
| WP-6 | Tempo `ListByAuthor`, classification, bounded decode and drain; Jira `*APIError` | CONDITIONAL (N7 to N10) |
| WP-7 | Task 7.3 session timeouts; pool connect | CONDITIONAL (N11, N12) |
| WP-5 | Reconcile flow, saves on `WithoutCancel` with 10s, Ctrl-C | APPROVED (depends on N8 landing in WP-6) |
| WP-8 | One shared `httpx.NewClient(30s)`, retry log fields | APPROVED (see R4) |
| WP-4 | F16 https-only base URLs, hostnames from config | APPROVED |
| WP-9 | Digests, cache mounts; no port or network change (`127.0.0.1:5432:5432` kept) | APPROVED |
| WP-11 | Task 11.3 `network.md` update | CONDITIONAL (N13) |
| WP-1, WP-3, WP-10 | No network surface | Not applicable |

Technology decisions table, Networking column:

| Row | Verdict |
|---|---|
| Tempo `GET /4/worklogs/user/{accountId}` | CONDITIONAL (N8, N9) |
| Retry policy: RFC 9110 9.2.2 plus R2/R3 | CONDITIONAL (N1 to N5) |
| `log/slog` JSON handler on stderr | APPROVED (logs host, never URL query or headers) |
| `CHECK ... NOT VALID` on `outbox.user_account_id` | Not applicable |
| Session `statement_timeout=5s`, `idle_in_transaction_session_timeout=10s` | CONDITIONAL (N11, N12) |
| Image digests | APPROVED |

## Findings and required plan changes

### WP-2: retry transport

**N1. `http.NoBody` requests cannot be retried (existing bug; the plan does not fix it).**

- Evidence: `internal/shared/httpx/retry_test.go:57` already has `t.Skip("BUG: RetryTransport returns errNoRewind for a request whose Body is http.NoBody (GetBody is nil)")`.
- Cause: `http.NewRequestWithContext` sets `GetBody` only for `*bytes.Buffer`, `*bytes.Reader` and `*strings.Reader` (Go 1.26.9 `net/http/request.go`). A caller that passes `http.NoBody` gets a non-nil `Body` and a nil `GetBody`, so `retry.go` returns `errNoRewind`.
- Failure scenario: a Tempo `ListByAuthor` GET or a Jira GET written with `http.NoBody` gets a 503. The retry fails with "request body cannot be replayed" instead of retrying, so `submit` ends before any POST.
- Plan change: Task 2.1 un-skips `TestRetryReplaysNoBodyRequest`. The rule treats `req.Body == nil || req.Body == http.NoBody` as "no body", and sends the clone with `http.NoBody`.

**N2. Put an upper bound on `Retry-After` when there is no deadline.**

- The plan caps the wait only by the ctx deadline. `NewClient` always sets one, but `RetryTransport` is exported and is built directly elsewhere: WP-11 builds it without a client timeout. A `Retry-After: 3600` would then sleep for an hour.
- Plan change, in the section 2 rule: "When the `Retry-After` wait exceeds `MaxDelay`, return the response as-is without sleeping, as for the deadline case."
- Test: `TestRetryAfterAboveMaxDelayReturnsResponse` (no ctx deadline, `Retry-After: 120`, `MaxDelay` 30s; one call, `Wait` never called).

**N3. Parse `Retry-After` strictly, and handle past HTTP-dates.**

- RFC 9110 10.2.3: `delay-seconds = 1*DIGIT`, "a non-negative decimal integer". `strconv.Atoi`, used now, also accepts `+7` and `-0`, and `time.Duration(s) * time.Second` overflows for very large values.
- Plan change:
  - Accept only ASCII digits, with at most 10 digits (anything larger is above `MaxDelay` anyway).
  - When an HTTP-date is at or before `Now()`, or the value is invalid, use the jittered backoff for that attempt. Do not use a zero wait.
- Test: add these rows to `TestRetryAfterSecondsAndHTTPDate`: `+7`, `-1`, `7.5`, a past date, and an RFC 850 date. RFC 9110 5.6.7 says a recipient "MUST accept all three HTTP-date formats", and `http.ParseTime` tries `TimeFormat`, `time.RFC850` and `time.ANSIC`.

**N4. Lock in that `Client.Timeout` drives the deadline cap.**

- Verified in the Go 1.26.9 source: `net/http/client.go:363-379` `setRequestCancel` sets `req.ctx, cancelCtx = context.WithDeadline(oldCtx, deadline)` in both branches, including the branch for an unknown `RoundTripper` such as `RetryTransport`. The 30s client timeout is therefore visible to `RetryTransport` as `req.Context().Deadline()`.
- That behaviour is an implementation detail, so a test should pin it.
- Plan change: add `TestRetryAfterBeyondClientTimeoutReturnsResponse` (`&http.Client{Timeout: 2 * time.Second, Transport: rt}`, `Retry-After: 10`; one call, the 429 is returned, `Wait` never called).
- Consequence for the operator: a Slack 429 with a `Retry-After` longer than the remaining 30s budget is returned at once. slack-go turns it into `*slack.RateLimitedError` (`misc.go:559-564`). This is acceptable, but WP-8 should print it as "Slack rate limited, retry after Ns" (R2 below).

**N5. Name the existing tests that change, and fix the test name.**

- The plan removes the `Sleep` field and changes defaults. These existing tests encode the old behaviour and must be rewritten or deleted in Task 2.1:
  - `TestRetryAfterHTTPDateFallsBackToBackoff` (now inverted);
  - `TestRetryGivesUpAfterMaxAttemptsWithExponentialBackoff` (5 attempts, any 5xx);
  - `TestRetryStopsWhenContextCanceledDuringBackoff`;
  - `TestRetryReturnsTransportErrorWithoutRetry` (both use `Sleep`);
  - `TestRetryDoesNotRetryClientErrors`.
- The plan calls the kept body-resend test `TestRetryPostOn429ResendsBody`; in the file it is `TestRetryResendsBody`.
- Plan change: list these tests in Task 2.1. Without the list, the go-coder will meet test failures that no step explains.

**N6. Keep one retry layer: no idempotency headers and no slack-go retries.**

- `go doc net/http.Transport`: "HTTP requests are considered idempotent if they have HTTP methods GET, HEAD, OPTIONS, or TRACE; or if their Header map contains an "Idempotency-Key" or "X-Idempotency-Key" entry." A Tempo POST with such a header would be re-sent by the stdlib after a read failure on a reused connection, which is the duplicate-worklog case.
- slack-go v0.30.1 has its own retry layer, off by default (`retry.go`: "Retries are off by default; use OptionRetry or OptionRetryConfig to turn them on"). `wire.go` does not enable it today.
- Plan change:
  - WP-6 Task 6.1 adds an assertion to the classification test that the POST carries neither header.
  - WP-8 states "do not pass `slack.OptionRetry` or `slack.OptionRetryConfig` (R4)".
- Verified as safe and kept: the stdlib's own retry of a POST happens only on `nothingWrittenError` over a reused connection (`net/http/transport.go:728-732`, "We never wrote anything, so it's safe to retry"). RFC 9110 9.2.2 allows this: "some means to detect that the original request was never applied".

Also confirmed for WP-2:

- Drain before waiting. Task 2.1 already drains with `io.Copy(io.Discard, io.LimitReader(body, 64<<10))` and then closes. Keep that order (drain and close, then `Wait`), so the connection goes back to the pool during the backoff. `Client.Do`: "If the Body is not both read to EOF and closed, the Client's underlying RoundTripper (typically Transport) may not be able to re-use a persistent TCP (or TLS) connection." A body above 64 KB is closed without being reused, which is correct.
- `WithReadOnly` for Slack is justified. `search.messages` is a read; slack-go sends it as a form POST (`misc.go:505-515`) and sets `GetBody`, so a body resend works. The Slack method page lists GET, so the POST is a library choice, not a protocol requirement.
- Full jitter, `rand.N(d+1)`, 3 attempts and a single layer match the AWS Builders' Library and coding-standards R2.

### WP-6: Tempo and Jira adapters

**N7. `TestDecodeIsBounded` asserts something that cannot happen.**

- The plan decodes through `io.LimitReader(resp.Body, 1<<20)`, then drains at most 64 KB. With a 5 MB body, about 3.9 MB stays unread, so `Close` drops the connection. The server's write then fails (reset or broken pipe), and "the handler's write completes" is false.
- Plan change: split the test in two.
  - `TestDecodeIsBounded`: a 5 MB success body gives `ErrCreateIndeterminate` within the client timeout. The handler returns, and a write error there is expected. The test does not wait for the full write.
  - `TestSmallTrailingBodyReusesConnection`: a 200 with valid JSON plus 10 KB of trailing whitespace, then a second request. `httptrace.ClientTrace.GotConn` reports `Reused == true` on the second request. This is the property F12 needs.

**N8. Do not rely on undocumented date-filter semantics in `ListByAuthor`.**

- The spec (`/4/worklogs/user/{accountId}`, Last-Modified 2026-10-07 22:26:27 GMT) declares `from` ("Retrieve results starting with this date") and `to` ("up to and including this date") as plain `type: string`, with no `format`. It does not say whether they filter on the local `startDate` or on `startDateTimeUtc`.
- `Worklog.startDateTimeUtc` is converted "using the worker's browser timezone at the time of worklog creation, when available, otherwise the worker's Atlassian profile timezone". The local `startDate` and `startTime` are the values the bot sent.
- Failure scenario: Tempo filters on UTC and the owner's Atlassian time zone is far enough east, for example UTC+13 or more. A 09:00 Monday worklog is then on Sunday in UTC, the Monday-to-Friday query misses it, reconcile finds nothing, and `submit` POSTs a duplicate. This is the exact case H2 exists to prevent.
- Plan change in Task 6.1:
  - Query `from = from-1 day` and `to = to+1 day`.
  - Keep the existing local skip of weekend `startDate` values; it already drops Saturday and Sunday.
  - Filter results to `from..to` by `startDate`.
  - Match only on `startDate` and `startTime`, never on `startDateTimeUtc`.
  - The port contract stays "from..to inclusive by `startDate`", so WP-5 does not change.
  - Update the expected query in `TestListByAuthorPaginates` to `from=2026-10-04&to=2026-10-10`, and add a `startDate` of `2026-10-04` to the fixture to show it is dropped.

**N9. Advance the offset by the results returned, not by a constant 50.**

- The spec declares no `maximum` for `limit` on `getWorklogsByUser` (other operations declare 1000 or 5000). `PageableMetadata` requires `count`, `limit` and `offset`; `next` is optional ("if applicable").
- If the server returns fewer than requested, a fixed `offset += 50` skips worklogs, and reconcile then misses them, which again means a duplicate POST.
- Plan change:
  - Set `offset += len(results)`.
  - Stop when `next` is empty, when `results` is empty, or when `len(results) < limit`.
  - Keep the 20-page cap, as an error before any POST.
- Test: add a page where the server returns 30 of the requested 50, and assert that the next request is `offset=30`.

**N10. Jira has nothing blocking.**

- Jira calls are GETs, so 429, 502, 503 and 504 retries are correct.
- Atlassian (page "Last updated Oct 9, 2026"):
  - `Retry-After` is in seconds and "Indicates how many seconds to wait before retrying".
  - "Some transient 5xx responses (such as 503) may also include a `Retry-After` header."
  - The recommended algorithm uses `Retry-After` as a minimum, doubles the delay with jitter, and caps the number of retries.
  - Retry only "if the API is idempotent".
- The plan follows all of this.
- Plan change: none for retries. Task 6.2 adds a case to `TestErrorsAreAPIError`: 429 after the transport's attempts gives `*APIError` with `StatusCode` 429, not `ErrIssueNotFound`, so a rate-limited plan run says so.

### WP-7: Postgres

**N11. Set a connect timeout on the pool.**

- pgx v5.11.0 dials with `&net.Dialer{}`, which has no timeout (`pgconn/config.go:1085-1087`), unless `ConnectTimeout` is set (`pgconn/pgconn.go:254-259`). `db.Open` runs on the signal ctx, which has no deadline.
- Failure scenario: on WSL after a network change, a blackholed address makes `plan` or `submit` hang at `Ping` for the kernel's TCP connect timeout. On Linux that is about two minutes; it is not a fast failure.
- Plan change in Task 7.3: when the DSN gives no `connect_timeout`, `db.Config` sets `cfg.ConnConfig.ConnectTimeout = 5 * time.Second`. `TestRuntimeParams` asserts it, and asserts that `?connect_timeout=2` is kept.

**N12. State the scope of the session timeouts.**

- `RuntimeParams` are "Run-time parameters to set on connection as session default values" (pgconn docs, v5.11.0). They apply only to the pgx pool.
- `Migrate` opens a separate `database/sql` connection (`db.go:32`), so the 5s `statement_timeout` does not apply to goose. That is the right behaviour, because the 00002 backfill `UPDATE` may run longer than 5s.
- Plan change in Task 7.3: "`Migrate` does not use `db.Config` and keeps no `statement_timeout`."
- Also state the precedence: the code always sets both values, overriding any value in the DSN, or the DSN wins. Then test that rule.
- Verified and kept:
  - `statement_timeout` is "taken as milliseconds" without units and aborts the statement.
  - `idle_in_transaction_session_timeout` terminates the session. pgxpool then discards the dead connection.
  - Postgres advises against setting `statement_timeout` in `postgresql.conf` "because it would affect all sessions". A per-session value is the recommended form.
  - 5s and 10s fit inside the 10s save context from WP-5.
  - No transaction in `postgres/repository.go` spans an HTTP call (`pgx.BeginFunc` at `:88` wraps SQL only).

### WP-11

**N13. Task 11.3: `network.md` belongs to `networking-engineer`.**

- Plan change: move the `docs/architecture/network.md` bullet to `networking-engineer`, or require that agent to review it.
- The content must include:
  - the egress row for `GET /4/worklogs/user/{accountId}` (HTTPS 443, bearer, retried on 429, 502, 503 and 504);
  - the retry rule, `WithReadOnly` and full jitter;
  - the `Retry-After` caps (deadline and `MaxDelay`);
  - the `Client.Timeout` deadline propagation;
  - the Postgres session timeouts and the connect timeout;
  - the removal of the "retries non-idempotent POSTs on 5xx" known gap.

### Recommendations (not blocking)

- **R1. Response-header timeout.** `network.md` lists "no response-header timeout" as a known gap. The 30s `Client.Timeout` bounds every call, so this is not a blocker. Setting `ResponseHeaderTimeout: 20s` on a cloned `DefaultTransport` would leave budget for a retry after a stalled 503. ("ResponseHeaderTimeout, if non-zero, specifies the amount of time to wait for a server's response headers after fully writing the request.")
- **R2. Slack 429 message.** WP-8 can map `*slack.RateLimitedError` to "Slack rate limited, retry after Ns".
  - Slack returns "HTTP 429 Too Many Requests" with `Retry-After` in seconds. Limits are "per API method per workspace/team per app".
  - `search.messages` is "Tier 2: 20+ per minute", with `count` max 100 and `page` max 100.
  - A week's search is a few pages, so 429 is unlikely.
  - The method page marks `search.messages` as "a legacy method" and recommends `assistant.search.context`. Track this for Phase 2, not SP-01.
  - The 2025 non-Marketplace limits name only `conversations.history` and `conversations.replies`.
- **R3. POST retried on 429.** The plan already lists the risk (section 9). Tempo documents only "the APIs will respond with status code 429". RFC 9110 9.2.2 permits an automatic POST retry only with "some means to detect that the original request was never applied". A 429 is a strong signal of that, not a guarantee. The owner should accept this risk explicitly in Q-list form, or Tempo POSTs should map 429 to indeterminate and reconcile instead of retrying (cost: one GET).
- **R4. WP-8 retry log.** Keep `host` and drop the path for Tempo. The path holds the `accountId` in `/worklogs/user/{accountId}`, which is personal data under the docs rule.
- **R5. Transport errors on GET.** Not retrying transport errors is acceptable for a CLI: a failed reconcile GET ends the run before any POST, and the user re-runs. Revisit for the Phase 2 scheduler.

## Verified sources

All were fetched on 2026-10-09.

| Claim | Status | Source |
|---|---|---|
| Client "SHOULD NOT automatically retry a request with a non-idempotent method unless it has some means to know that the request semantics are actually idempotent ... or some means to detect that the original request was never applied"; PUT, DELETE and safe methods are idempotent | VERIFIED | https://www.rfc-editor.org/rfc/rfc9110#section-9.2.2 |
| `Retry-After = HTTP-date / delay-seconds`; `delay-seconds = 1*DIGIT`, "a non-negative decimal integer"; with 503 it gives the expected unavailability | VERIFIED | https://www.rfc-editor.org/rfc/rfc9110#section-10.2.3 |
| Recipients "MUST accept all three HTTP-date formats" | VERIFIED | https://www.rfc-editor.org/rfc/rfc9110#section-5.6.7 |
| `Client.Timeout` "includes connection time, any redirects, and reading the response body"; body must be read to EOF and closed for reuse; `ParseTime` tries `TimeFormat`, `time.RFC850`, `time.ANSIC` | VERIFIED | https://pkg.go.dev/net/http (page shows go1.27.2); same text in `go doc` for go1.26.9 |
| Transport retries on network error only for idempotent requests, including those with `Idempotency-Key` or `X-Idempotency-Key` headers; `ResponseHeaderTimeout` text; `DefaultTransport` values (30s dial, 10s TLS, 90s idle, 100 idle conns, `ProxyFromEnvironment`) | VERIFIED | `go doc net/http.Transport`, `go doc -src net/http.DefaultTransport` (go1.26.9), https://pkg.go.dev/net/http |
| `Client.Timeout` is applied as a ctx deadline on the request, also for a custom `RoundTripper` | VERIFIED (source) | go1.26.9 `src/net/http/client.go:356-379` |
| Stdlib re-sends a POST only when nothing was written on a reused connection | VERIFIED (source) | go1.26.9 `src/net/http/transport.go:704-736` |
| `NewRequestWithContext` sets `GetBody` only for `bytes.Buffer`, `bytes.Reader`, `strings.Reader` | VERIFIED (source) | go1.26.9 `src/net/http/request.go` |
| "Retries are selfish"; "retry at a single point in the stack"; "APIs with side effects aren't safe to retry unless they provide idempotency"; cap backoff; limit retries; jitter | VERIFIED | https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter (301 from https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/) |
| Slack: 429 with `Retry-After` in seconds; limits per method per workspace per app; tiers (Tier 2 is 20+ per minute) | VERIFIED | https://docs.slack.dev/apis/web-api/rate-limits (302 from https://api.slack.com/docs/rate-limits) |
| `search.messages`: Tier 2, user token, `search:read`, `count` max 100, `page` max 100, "legacy method" | VERIFIED | https://docs.slack.dev/reference/methods/search.messages |
| Tempo: 429 on excess traffic, no `Retry-After` documented; `getWorklogsByUser` `from`/`to` untyped strings, `to` inclusive, `offset` default 0, `limit` default 50 with no maximum; `PageableMetadata` requires `count`, `limit`, `offset`; `Worklog.startDate`, `startTime` (pattern `^([0-1]?[0-9]|2[0-3])(:[0-5][0-9])(:[0-5][0-9])$` on input) and `startDateTimeUtc` (converted with the worker's browser or profile time zone) | VERIFIED | https://apidocs.tempo.io/#tag/Worklogs/operation/getWorklogsByUser ; spec https://apidocs.tempo.io/tempo-openapi.yaml (HTTP 200, 559477 bytes, Last-Modified Wed, 07 Oct 2026 22:26:27 GMT) |
| Atlassian: 429 for all limit types; `Retry-After` in seconds; some 503 carry `Retry-After`; backoff with jitter, `Retry-After` as minimum, capped retries, retry only idempotent APIs | VERIFIED | https://developer.atlassian.com/cloud/jira/platform/rate-limiting/ (Last updated Oct 9, 2026) |
| pgconn v5.11.0: `RuntimeParams` "Run-time parameters to set on connection as session default values"; unknown DSN keys "become runtime parameters or pgx-specific options"; `PGCONNECT_TIMEOUT` recognized; default dialer has no timeout | VERIFIED | https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#Config ; module source `pgconn/config.go:1085`, `pgconn/pgconn.go:254` |
| `statement_timeout` (ms without units, aborts the statement, not recommended in `postgresql.conf`); `idle_in_transaction_session_timeout` terminates the session | VERIFIED | https://www.postgresql.org/docs/17/runtime-config-client.html |

`official-source-verifier` was not available to this agent either, so these facts were checked directly (see plan Q5).

## Commands run

All commands were read-only. Downloads went to their own empty directories under `/tmp`.

```text
cat .claude/agents/networking-engineer.md docs/architecture/network.md
ls docs/superpowers/plans/ docs/superpowers/plans/reviews
cat internal/shared/httpx/*.go
cat -n internal/timesheet/infrastructure/tempo/client.go internal/timesheet/infrastructure/jira/client.go internal/platform/db/db.go cmd/timesheet/wire.go internal/activity/infrastructure/slack/collector.go internal/timesheet/application/submit.go
grep -n ... docs/superpowers/plans/2026-10-09-sp01-hardening.md .claude/skills/coding-standards/*.md internal/timesheet/infrastructure/postgres/repository.go go.mod docker-compose.yml
go version
grep/sed on $(go env GOROOT)/src/net/http/{client.go,transport.go,request.go}
grep/sed on $(go env GOMODCACHE)/github.com/slack-go/slack@v0.30.1/{misc.go,search.go,retry.go,slack.go}
grep/sed on $(go env GOMODCACHE)/github.com/jackc/pgx/v5@v5.11.0/pgconn/{config.go,pgconn.go}
go doc net/http.Transport ; go doc net/http.ParseTime ; go doc -src net/http.DefaultTransport
go doc github.com/jackc/pgx/v5/pgconn.ParseConfig
curl -sS -D headers.txt -o /tmp/nwreview-tempo/tempo-openapi.yaml https://apidocs.tempo.io/tempo-openapi.yaml ; grep/sed/awk on it
curl -sS -o /tmp/nwreview-rfc/rfc9110.txt https://www.rfc-editor.org/rfc/rfc9110.txt ; awk/sed on it
WebFetch: rfc-editor.org rfc9110, pkg.go.dev/net/http, aws.amazon.com builders-library (301) and builder.aws.com, api.slack.com/docs/rate-limits (302) and docs.slack.dev rate-limits, api.slack.com/methods/search.messages (302) and docs.slack.dev, developer.atlassian.com rate-limiting, pkg.go.dev pgconn, postgresql.org runtime-config-client
```
