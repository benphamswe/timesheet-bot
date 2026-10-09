---
name: coding-standards
description: timesheet-bot coding standards and naming conventions with verified official sources (Go team, Google Go Style Guide, Uber, Google eng-practices, AWS, Azure, Google SRE, Stripe, OWASP, Anthropic, Docker, Twelve-Factor, PostgreSQL, Google/Microsoft doc style). Rule IDs for reviews. Preloaded into go-coder and standards-reviewer.
user-invocable: false
---

# Coding standards

Go paths (`internal/`, `cmd/`, `migrations/`) are relative to `services/timesheet-core/`. Go tools run there, or from the repo root with `go -C services/timesheet-core`.

Sources verified live on 2026-10-09. `golangci-lint` (`.golangci.yml`) already enforces gofmt, goimports, initialisms, receiver names, error-string format, `ErrX` naming, doc comments, `errors.Is`/`%w`, closing bodies and rows, constant `slog` messages and `snake_case` log keys (sloglint), and gosec's hard-coded credential, SQL string-building and TLS-settings checks (G101, G201, G202, G402). The rules below cover what a linter cannot judge.

## Source authority by topic

Within a topic the first source wins. The full organization-by-topic table and the known deviations are in `sources.md`.

- **Go style**: Go team (CRC, go.dev docs; Effective Go is a 2009 baseline) > Google Go Style Guide (`guide` canonical, `decisions` normative, `best-practices` advisory) > Uber, gaps only (not adopted: `_` prefix on unexported globals, 99-character lines) > GitLab, Kubernetes.
- **API clients and idempotency**: the vendor's own contract (Slack, Jira, Tempo, Anthropic) > Stripe and AWS Builders' Library > Microsoft REST guidelines > Google AIP (gRPC-oriented; AIP-194 retry codes don't apply to REST vendors).
- **Resilience**: AWS Builders' Library and Well-Architected, Google SRE > Azure Architecture Center and Well-Architected > Netflix (Hystrix is in maintenance mode; concepts only).
- **Security**: OWASP cheat sheets > Go security best practices > GitLab secure coding > AWS Well-Architected security pillar. Microsoft SDL is process-level only.
- **LLM and agent safety**: Anthropic docs > OWASP LLM Top 10 (2026 edition) and the OWASP LLM prompt-injection cheat sheet.
- **Code review process**: Google eng-practices (archived 2024, stable).
- **Containers and config**: Docker docs > Twelve-Factor. Secret storage in env vars is an accepted deviation from OWASP and AWS (see D3).
- **Test placement**: the Go team (`go help test`, go.dev module layout) puts tests beside the code. The project keeps them in a separate `test/` tree as a documented deviation (T5, ADR-0009).
- **SQL**: PostgreSQL docs > GitLab Go guide.
- **Docs**: go.dev/doc/comment for Go comments; Google developer style > Microsoft style for prose.

Non-Go sources never set Go naming: Azure's `Get` verbs are Azure-SDK-specific, and Go code drops `Get`.

## N - Naming and layout

- **N1** Package names: short, lowercase, one word, no `util`, `common`, `helper`, `model` or `types`. [CRC#package-names](https://go.dev/wiki/CodeReviewComments#package-names), [Google#package-names](https://google.github.io/styleguide/go/decisions#package-names)
- **N2** No stutter at the call site: `tempo.Client`, not `tempo.TempoClient`; `widget.New`, not `widget.NewWidget`. [Google#repetition](https://google.github.io/styleguide/go/decisions#repetition)
- **N3** Getters have no `Get` prefix (`Status()`). Expensive or remote calls use a verb (`Fetch`, `Resolve`). [Effective Go#Getters](https://go.dev/doc/effective_go#Getters), [Google#getters](https://google.github.io/styleguide/go/decisions#getters)
- **N4** Name length tracks scope: one letter for short loops and receivers, descriptive names farther from the declaration. No type in the name (`users`, not `userSlice`). [CRC#variable-names](https://go.dev/wiki/CodeReviewComments#variable-names), [Google#variable-names](https://google.github.io/styleguide/go/decisions#variable-names)
- **N5** Constants use MixedCaps and are named by role, not value: `MaxSubmitAttempts`, never `MAX_ATTEMPTS` or `Three`. [Google#constant-names](https://google.github.io/styleguide/go/decisions#constant-names)
- **N6** One-method interfaces use the `-er` form where natural (`Clock`, `Source` are fine as domain nouns). Interfaces are declared by the consumer. [Effective Go#interface-names](https://go.dev/doc/effective_go#interface-names), [CRC#interfaces](https://go.dev/wiki/CodeReviewComments#interfaces)
- **N7** Error types end in `Error`. Sentinels are `ErrX` when exported and `errX` when unexported. [Uber#error-naming](https://github.com/uber-go/guide/blob/master/style.md#error-naming)
- **N8** Don't shadow the predeclared identifiers `min`, `max`, `len`, `new`, `clear`.
- **N9** SQL identifiers are unquoted `snake_case`, at most 63 bytes. Index names follow `<table>_<purpose>`. [PostgreSQL 4.1.1](https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS)
- **N10** Log attribute keys are lowercase `snake_case`, dot-namespaced only for groups (`timesheet.week_id`). [OTel naming](https://opentelemetry.io/docs/specs/semconv/general/naming/) (adapted for slog)
- **N11** Doc and file names are `kebab-case.md`. ADRs are `NNNN-imperative-title.md`.
- **N12** Commands live in `cmd/<name>/`; all other packages go in `internal/`. The service exports no packages. [Go module layout](https://go.dev/doc/modules/layout)

## E - Errors and control flow

- **E1** Add context once per layer; `%w` goes at the end (`"resolve GCM-1: %w"`), except a sentinel that leads. The context says what was attempted, without `failed`, `error` or `didn't`. [Google best-practices#error-percent-w](https://google.github.io/styleguide/go/best-practices#error-percent-w), [GitLab#adding-context](https://docs.gitlab.com/development/go_guide/#adding-context)
- **E2** Return an error as-is when there is nothing to add. `%w` makes the cause part of your API, so adapters hide driver errors (pgx) behind `%v` or a domain sentinel. [Uber#error-wrapping](https://github.com/uber-go/guide/blob/master/style.md#error-wrapping), [Go blog: errors in 1.13](https://go.dev/blog/go1.13-errors)
- **E3** No in-band errors (`-1`, `""`). Return `(T, error)` or `(T, bool)`. [CRC#in-band-errors](https://go.dev/wiki/CodeReviewComments#in-band-errors)
- **E4** `panic` and `Must*` only at startup and in tests. [Google#dont-panic](https://google.github.io/styleguide/go/decisions#dont-panic)
- **E5** `os.Exit` and `log.Fatal` only in `main`, and only once: put the logic in `run() error`, so defers run. [Uber#exit-in-main](https://github.com/uber-go/guide/blob/master/style.md#exit-in-main), [Uber#exit-once](https://github.com/uber-go/guide/blob/master/style.md#exit-once)
- **E6** Never discard an error with `_` without a comment saying why it is safe. [CRC#handle-errors](https://go.dev/wiki/CodeReviewComments#handle-errors)

## C - Context, concurrency, state

- **C1** `ctx context.Context` is the first parameter of anything that does I/O. Never store it in a struct or an options struct. [CRC#contexts](https://go.dev/wiki/CodeReviewComments#contexts), [Google#contexts](https://google.github.io/styleguide/go/decisions#contexts)
- **C2** Every goroutine has an obvious exit and an owner that waits for it. Prefer synchronous APIs. [CRC#goroutine-lifetimes](https://go.dev/wiki/CodeReviewComments#goroutine-lifetimes), [Uber#wait-for-goroutines-to-exit](https://github.com/uber-go/guide/blob/master/style.md#wait-for-goroutines-to-exit)
- **C3** No mutable package globals and no `init()` with I/O. Inject dependencies through the composition root. [Uber#avoid-mutable-globals](https://github.com/uber-go/guide/blob/master/style.md#avoid-mutable-globals), [Uber#avoid-init](https://github.com/uber-go/guide/blob/master/style.md#avoid-init)
- **C4** Use `time.Duration` for durations in APIs and config, never bare ints, unless the domain unit is fixed and named (`kernel.Duration` minutes). [Uber#use-timeduration-for-periods-of-time](https://github.com/uber-go/guide/blob/master/style.md#use-timeduration-for-periods-of-time)
- **C5** Use comma-ok for every type assertion outside tests. [Uber#handle-type-assertion-failures](https://github.com/uber-go/guide/blob/master/style.md#handle-type-assertion-failures)
- **C6** Marshalled structs have explicit `json`/`yaml` tags. [Uber#use-field-tags-in-marshaled-structs](https://github.com/uber-go/guide/blob/master/style.md#use-field-tags-in-marshaled-structs)

## R - Resilience for outbound calls

- **R1** Every remote call has a timeout. A deadline from the caller's `ctx` is propagated, not invented. [AWS Builders' Library](https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter), [SRE cascading failures](https://sre.google/sre-book/addressing-cascading-failures/)
- **R2** Retries use capped exponential backoff **with jitter**, at most 3 attempts per request. [AWS REL05-BP03](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/rel_mitigate_interaction_failure_limit_retries.html), [SRE handling overload](https://sre.google/sre-book/handling-overload/), [Azure core retry](https://azure.github.io/azure-sdk/general_azurecore.html)
- **R3** Never automatically retry a non-idempotent call (a POST that creates something) on 5xx or a timeout. The request may have succeeded. Retry only on 429, or on an explicit not-processed signal, or make the call idempotent first (K1, K2). [AWS Builders' Library](https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter), [AWS REL05-BP03](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/rel_mitigate_interaction_failure_limit_retries.html)
- **R4** Retry at one layer only. Business-level retries (the submit loop, the outbox relay) must not stack on transport retries for the same failure. An SDK with built-in retries (the Anthropic Go SDK retries twice by default) is either the only layer or gets `option.WithMaxRetries(0)`. [Azure transient faults](https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults), [Anthropic errors](https://platform.claude.com/docs/en/api/errors)
- **R5** Don't retry permanent errors (4xx except 429, validation, auth). Surface them. [Azure transient faults](https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults)
- **R6** Prefer failing fast to the caller over a fallback path. The only fallback allowed makes no network calls and can't fail (`RulePolicy` is local and pure). [AWS avoiding fallback](https://builder.aws.com/content/3EuS9Sakq7L3VLQIF3qzfMfke1Y/avoiding-fallback-in-distributed-systems), [Hystrix How it Works](https://github.com/Netflix/Hystrix/wiki/How-it-Works). Hystrix is in maintenance mode; prefer retry budgets over circuit breakers for this single-user service.
- **R7** Logs and errors never include tokens, `Authorization` headers, database connection strings (`DATABASE_URL`) or request bodies with personal data. Redact them, or log an internal ID instead. [Azure implementation#golang-log-exclude](https://azure.github.io/azure-sdk/golang_implementation.html), [OWASP Logging#data-to-exclude](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html#data-to-exclude), [GitLab secure coding#what-should-not-be-captured-in-the-logs](https://docs.gitlab.com/development/secure_coding_guidelines/#what-should-not-be-captured-in-the-logs)
- **R8** Honor `Retry-After` on 429 and 503. A 429 that signals a hard cap has no `Retry-After` and is permanent: Anthropic's `enforced_spend_limit_reached` is not retried. [Azure transient faults](https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults), [Anthropic rate limits#reaching-your-spend-cap](https://platform.claude.com/docs/en/api/rate-limits#reaching-your-spend-cap)
- **R9** Log a retried transient failure at `Warn`, with whether it was throttling. Only the final failed attempt is `Error`. [Azure transient faults](https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults)

## K - API idempotency

- **K1** Each logical create (a Tempo worklog) has one idempotency key, derived from domain identity (the deterministic entry ID) or generated once and persisted before the first attempt. Every retry reuses it; never mint a key per attempt. [AWS idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/), [Stripe#idempotency](https://docs.stripe.com/error-low-level#idempotency)
- **K2** Treat a 5xx or timeout on a create as indeterminate. If the vendor takes an idempotency key, retry with the same key and the same parameters; if it doesn't (Tempo v4), reconcile by looking the object up before posting again. [Stripe#server-errors](https://docs.stripe.com/error-low-level#server-errors), [Stripe idempotent requests](https://docs.stripe.com/api/idempotent_requests)
- **K3** Use whatever key mechanism each vendor defines (header or body field) and its wire names. Keys never contain personal data. [Stripe idempotent requests](https://docs.stripe.com/api/idempotent_requests)

## S - Security

- **S1** CI runs `govulncheck ./...`, `go vet ./...` and `go test -race ./...`. Stay on the latest Go minor release, and review and test every dependency update. [Go security best practices](https://go.dev/doc/security/best-practices)
- **S2** Keep dependencies to a minimum; justify each new module in the PR description. [GitLab#dependencies](https://docs.gitlab.com/development/go_guide/#dependencies)
- **S3** Tokens and API keys travel in headers, never in URLs or query strings. [OWASP REST#sensitive-information-in-http-requests](https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html#sensitive-information-in-http-requests)
- **S4** Untrusted text (Slack messages, Jira fields, model output) is logged only as a structured attribute value, never concatenated into a message or line, so CR, LF and delimiters can't forge entries. [OWASP Logging#event-collection](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html#event-collection), [GitLab secure coding#protecting-log-files](https://docs.gitlab.com/development/secure_coding_guidelines/#protecting-log-files)
- **S5** Each token has only the scopes its adapter uses (Slack collection: `search:read`); remove unused scopes. [OWASP Secrets#23-access-control](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html#23-access-control), [AWS SEC03-BP02](https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/sec_permissions_least_privileges.html)

## L - LLM and agent safety

- **L1** Third-party content (Slack text, Jira summaries) reaches Claude only inside `tool_result` blocks, JSON-encoded, never in `system` or user text. Our own instructions go in the following user turn, not inside a tool result. [Anthropic: mitigate prompt injection](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks)
- **L2** Each tool description says what its content is and where it came from ("message body from a Slack user"). The system prompt states that tool content is untrusted data that never overrides the system prompt or the user's request. [Anthropic: mitigate prompt injection](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks), [OWASP LLM01:2026](https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM01_PromptInjection.md)
- **L3** Model output is untrusted. Tool definitions use `strict: true`, and every proposal is validated in code by building kernel value objects and replaying it into the `Timesheet` aggregate. [Anthropic tool use](https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview), [OWASP LLM01:2026](https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM01_PromptInjection.md)
- **L4** Agent tools are read-only and get no secrets. Writes to Tempo happen only after human approval, and the executor checks that approval covers the exact entries. [OWASP LLM cheat sheet#least-privilege](https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html#least-privilege), [#human-in-the-loop-hitl-controls](https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html#human-in-the-loop-hitl-controls), [Anthropic: containing Claude](https://www.anthropic.com/engineering/how-we-contain-claude)
- **L5** Agent loops have a hard stop (a maximum number of iterations or tool calls). Reaching it is a rejected proposal, so the policy chain falls through to `RulePolicy`. [Anthropic: building effective agents](https://www.anthropic.com/engineering/building-effective-agents)
- **L6** Tests feed Slack fixtures that contain injected instructions and assert that the proposal ignores them or is rejected. [Anthropic: mitigate prompt injection](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks), [OWASP LLM cheat sheet#smoke-testing-your-defenses](https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html#smoke-testing-your-defenses)
- **L7** Branch on `*anthropic.Error` `StatusCode`, never on message strings, and log the `request-id` header. [Anthropic errors#request-id](https://platform.claude.com/docs/en/api/errors#request-id)

## T - Tests (beyond go-testing-conventions)

- **T1** Failure messages put got before want: `Func(%v) = %v, want %v`. [Google#useful-test-failures](https://google.github.io/styleguide/go/decisions#useful-test-failures), [CRC#useful-test-failures](https://go.dev/wiki/CodeReviewComments#useful-test-failures)
- **T2** Use the standard `testing` package only, no assertion libraries. Compare structs with `cmp.Diff` if needed. [Google#use-package-testing](https://google.github.io/styleguide/go/decisions#use-package-testing)
- **T3** Table rows have a `name` and no conditional logic keyed on row values (split such cases into separate tests). [Google#data-driven-test-cases](https://google.github.io/styleguide/go/decisions#data-driven-test-cases)
- **T4** Async tests never sleep a fixed time and expect a result. Wait and retry against a deadline, or inject the `Clock`. [Kubernetes#testing-conventions](https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md#testing-conventions)
- **T5** Tests live in `services/timesheet-core/test/<kind>/<package path>/` and are black-box (`package x_test`); no `_test.go` under `internal/` or `cmd/`, no `export_test.go`. **Documented project deviation** ([ADR-0009](../../../docs/adr/0009-tests-in-separate-tree.md), owner, 2026-10-09; see D-TESTS in `sources.md`): the official Go convention is a `_test.go` file beside the code, in the same directory. [go help test](https://pkg.go.dev/cmd/go#hdr-Test_packages), [Organizing a Go module](https://go.dev/doc/modules/layout), [testing](https://pkg.go.dev/testing)

## D - Docker, config, database, scripts

- **D1** Multi-stage build, minimal base, non-root `USER`, exec-form `ENTRYPOINT`. [Docker best practices](https://docs.docker.com/build/building/best-practices/)
- **D2** Pin base images to a digest (`image:tag@sha256:...`). Update the pins deliberately. [Docker#pin-base-image-versions](https://docs.docker.com/build/building/best-practices/#pin-base-image-versions)
- **D3** No secrets in `ENV`, `ARG`, image layers or compose files. Config comes from the environment. [12factor config](https://12factor.net/config), [OWASP Secrets#51](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html#51-injection-of-secrets-file-in-memory)
- **D4** Logs go to stdout and stderr as an event stream; the app does not manage log files. [12factor logs](https://12factor.net/logs)
- **D5** Dev uses the same backing services as prod (Postgres everywhere, no SQLite stand-in). [12factor dev-prod-parity](https://12factor.net/dev-prod-parity)
- **D6** Postgres access uses pgx with hand-written, parameterized SQL; no ORM. [GitLab#orm](https://docs.gitlab.com/development/go_guide/#orm)
- **D7** Shell scripts (`.claude/hooks/*.sh`): `#!/bin/bash`, errors to stderr, every expansion quoted, `[[ ]]` and `$(...)`, `local` in functions, `main "$@"` once there is another function, ShellCheck-clean. Past about 100 lines, rewrite in Go. [Google Shell#s1.1](https://google.github.io/styleguide/shellguide.html#s1.1-which-shell-to-use), [#s1.2](https://google.github.io/styleguide/shellguide.html#s1.2-when-to-use-shell), [#s7.7](https://google.github.io/styleguide/shellguide.html#s7.7-main)

## V - Code review process

- **V1** A change does one thing: about 100 lines is reasonable and 1000 is usually too large. Refactorings go in a separate change. [eng-practices#what-is-small](https://google.github.io/eng-practices/review/developer/small-cls.html#what-is-small)
- **V2** A change that adds or changes logic includes the new or updated tests. [eng-practices#test_code](https://google.github.io/eng-practices/review/developer/small-cls.html#test_code)
- **V3** The description's first line is a short, standalone summary. The body gives the context, why this approach, and its shortcomings. [eng-practices#first-line](https://google.github.io/eng-practices/review/developer/cl-descriptions.html#first-line), [#informative](https://google.github.io/eng-practices/review/developer/cl-descriptions.html#informative)
- **V4** Approve once the change definitely improves code health, even if it isn't perfect. Optional comments start with `Nit:`. Technical facts and the style guide overrule preference. [eng-practices: standard of review](https://google.github.io/eng-practices/review/reviewer/standard.html)

## W - Writing docs

- **W1** Sentence-case headings with no trailing period or colon. [Google style capitalization](https://developers.google.com/style/capitalization), [Microsoft top 10](https://learn.microsoft.com/en-us/style-guide/top-10-tips-style-voice)
- **W2** Second person, active voice, present tense. [Google person](https://developers.google.com/style/person), [Google voice](https://developers.google.com/style/voice)
- **W3** Code, commands, paths and identifiers in code font. [Google code in text](https://developers.google.com/style/code-in-text)
- **W4** A boolean func's doc says "reports whether". A type's doc states any concurrency guarantee beyond one goroutine and a non-obvious zero value. Deprecation is a `Deprecated:` paragraph naming the replacement, if any. [Go doc comments#func](https://go.dev/doc/comment#func), [#type](https://go.dev/doc/comment#type), [#syntax](https://go.dev/doc/comment#syntax)
