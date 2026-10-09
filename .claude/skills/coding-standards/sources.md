# Coding standards source registry

Maintained by the `standards-curator` agent (`/update-standards`). Every rule in `SKILL.md` cites a URL from this table. Only official publications of the organization are listed; employee blogs and third-party summaries are not.

Re-verify all rows when the oldest `Verified` date is more than 30 days old (`/update-standards verify`).

| ID | Organization | Source | URL | Authoritative for | Status | Verified |
|---|---|---|---|---|---|---|
| GO-CRC | Go team | Go Code Review Comments | https://go.dev/wiki/CodeReviewComments | Go naming, errors, contexts, interfaces, tests | live | 2026-10-09 |
| GO-EFF | Go team | Effective Go (frozen 2009, baseline) | https://go.dev/doc/effective_go | Go idioms baseline | live | 2026-10-09 |
| GO-LAYOUT | Go team | Organizing a Go module | https://go.dev/doc/modules/layout | `cmd/` and `internal/` layout (N12); shows `*_test.go` beside the code (T5 deviation) | live; no anchors on page | 2026-10-09 |
| GO-CMD-TEST | Go team | `go help test` (cmd/go, Test packages) | https://pkg.go.dev/cmd/go#hdr-Test_packages | `*_test.go` files in the package directory, `_test` packages, `testdata` (T5 deviation) | read locally with Go 1.26.9 | 2026-10-09 |
| GO-TESTING | Go team | Package testing | https://pkg.go.dev/testing | Black-box `_test` packages see only exported identifiers; fuzz seeds in `testdata/fuzz/<Name>` (T5) | live | 2026-10-09 |
| GO-REPO-TEST | Go team | golang/go `test/README.md` | https://github.com/golang/go/blob/master/test/README.md | The Go repo's top-level `test/` holds toolchain tests only; standard library tests stay in their packages (T5 deviation) | live | 2026-10-09 |
| GO-ERR113 | Go team | Blog: Working with Errors in Go 1.13 (2019-10-17) | https://go.dev/blog/go1.13-errors | When to wrap with `%w` (E2) | live; go.dev/blog/errors is 404 | 2026-10-09 |
| GO-DOC | Go team | Go Doc Comments | https://go.dev/doc/comment | Doc comment content (W4) | live | 2026-10-09 |
| GO-SEC | Go team | Security best practices | https://go.dev/doc/security/best-practices | govulncheck, race detector, vet, updates (S1) | live; no anchors on page | 2026-10-09 |
| GOOG-GO | Google | Go Style Guide (guide canonical, decisions normative, best-practices advisory) | https://google.github.io/styleguide/go/ | Go style, naming, tests | live (updated 2025-04-02) | 2026-10-09 |
| GOOG-SHELL | Google | Shell Style Guide | https://google.github.io/styleguide/shellguide.html | Hook scripts (D7) | live; does not mandate `set -e` | 2026-10-09 |
| GOOG-ENGPRAC | Google | Engineering Practices: code review | https://google.github.io/eng-practices/ | Change size, tests in change, descriptions, review standard (V1-V4) | live; repo archived 2024-09 (stable) | 2026-10-09 |
| GOOG-AIP | Google | API Improvement Proposals (AIP-155, AIP-194) | https://google.aip.dev/ | Own gRPC/proto API design; authority list only | live | 2026-10-09 |
| UBER-GO | Uber | Uber Go Style Guide | https://github.com/uber-go/guide/blob/master/style.md | Go gaps: error naming, globals, exit, goroutines | live | 2026-10-09 |
| GITLAB-GO | GitLab | Go standards and style guidelines | https://docs.gitlab.com/development/go_guide/ | Error wording (E1), dependencies (S2), no ORM (D6) | live; tooling section pins golangci-lint v1.56.2 (stale) | 2026-10-09 |
| GITLAB-SEC | GitLab | Secure coding guidelines | https://docs.gitlab.com/development/secure_coding_guidelines/ | Credentials and log injection (R7, S4), TLS minimum | live; mostly Ruby examples | 2026-10-09 |
| K8S-CODE | Kubernetes (CNCF) | Coding conventions | https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md | Async tests (T4) | live | 2026-10-09 |
| MS-AZSDK-GO | Microsoft | Azure SDK for Go guidelines | https://azure.github.io/azure-sdk/golang_introduction.html | API client design | live (design page golang_design.html is dead) | 2026-10-09 |
| MS-AZSDK-IMPL | Microsoft | Azure SDK for Go implementation | https://azure.github.io/azure-sdk/golang_implementation.html | Client logging redaction, tracing | live | 2026-10-09 |
| MS-AZCORE | Microsoft | Azure SDK general azure-core | https://azure.github.io/azure-sdk/general_azurecore.html | Retry defaults, User-Agent | live | 2026-10-09 |
| MS-WAF-TRANSIENT | Microsoft | Well-Architected: handle transient faults | https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults | Layered retries, Retry-After, retry log level (R4, R8, R9) | live (updated 2026-08-17) | 2026-10-09 |
| MS-AAC-RETRY | Microsoft | Architecture Center: Retry pattern | https://learn.microsoft.com/en-us/azure/architecture/patterns/retry | Retry shape, idempotency check; reference only | live (updated 2025-12-09) | 2026-10-09 |
| MS-REST | Microsoft | REST API Guidelines (Azure) | https://github.com/microsoft/api-guidelines/blob/vNext/azure/Guidelines.md | Own REST API design, Repeatability headers; authority list only | live (last revision 2025-03-28) | 2026-10-09 |
| AWS-BL-RETRY | Amazon | Builders' Library: timeouts, retries and backoff with jitter | https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter | Timeouts, retries, jitter | moved from aws.amazon.com/builders-library (301) | 2026-10-09 |
| AWS-BL-IDEMP | Amazon | Builders' Library: Making retries safe with idempotent APIs | https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/ | Client request tokens reused across retries (K1) | live at old host (200); expect a move to builder.aws.com | 2026-10-09 |
| AWS-BL-FALLBACK | Amazon | Builders' Library: Avoiding fallback in distributed systems | https://builder.aws.com/content/3EuS9Sakq7L3VLQIF3qzfMfke1Y/avoiding-fallback-in-distributed-systems | Fail fast over fallback (R6) | moved from aws.amazon.com/builders-library (301) | 2026-10-09 |
| AWS-WA-REL05 | Amazon | Well-Architected REL05-BP03 | https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/rel_mitigate_interaction_failure_limit_retries.html | Retry limits, idempotency | live | 2026-10-09 |
| AWS-WA-SEC02 | Amazon | Well-Architected SEC02-BP03 Store and use secrets securely | https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/sec_identities_secrets.html | Secret storage and rotation (deviation note for D3) | live | 2026-10-09 |
| AWS-WA-SEC03 | Amazon | Well-Architected SEC03-BP02 Grant least privilege access | https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/sec_permissions_least_privileges.html | Least privilege for machine identities (S5) | live | 2026-10-09 |
| AWS-SDKGO | Amazon | AWS SDK for Go v2 retries and timeouts | https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html | Retry defaults reference | live | 2026-10-09 |
| NFLX-HYSTRIX | Netflix | Hystrix wiki: How it Works (project in maintenance mode) | https://github.com/Netflix/Hystrix/wiki/How-it-Works | Fallbacks, bulkheads, timeouts | live | 2026-10-09 |
| GOOG-SRE-CASCADE | Google | SRE book: Addressing Cascading Failures | https://sre.google/sre-book/addressing-cascading-failures/ | Deadlines, retry budgets | live | 2026-10-09 |
| GOOG-SRE-OVERLOAD | Google | SRE book: Handling Overload | https://sre.google/sre-book/handling-overload/ | Retry budgets, client throttling | live | 2026-10-09 |
| STRIPE-IDEMP | Stripe | Idempotent requests | https://docs.stripe.com/api/idempotent_requests | Idempotency-key semantics (K2, K3) | live; no page date | 2026-10-09 |
| STRIPE-ERR | Stripe | Advanced error handling | https://docs.stripe.com/error-low-level | Indeterminate 500s, same-key retries (K1, K2) | live; no page date | 2026-10-09 |
| OWASP-LOG | OWASP | Logging Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html | Data to exclude, log injection (R7, S4) | live | 2026-10-09 |
| OWASP-REST | OWASP | REST Security Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html | Credentials not in URLs (S3) | live | 2026-10-09 |
| OWASP-SECRETS | OWASP | Secrets Management Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html | Least privilege per secret, no ENV/ARG secrets (S5, D3) | live | 2026-10-09 |
| OWASP-LLM-CS | OWASP | LLM Prompt Injection Prevention Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/LLM_Prompt_Injection_Prevention_Cheat_Sheet.html | Least privilege, human approval, injection tests (L4, L6) | live | 2026-10-09 |
| OWASP-LLM01 | OWASP | GenAI LLM Top 10: LLM01:2026 Prompt Injection | https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM01_PromptInjection.md | Separate provenance-labelled channel, schema validation in code (L2, L3) | live; replaces the 2025-edition page https://genai.owasp.org/llmrisk/llm01-prompt-injection/ (2025 edition archived; 2026 edition published 2026-08-03) | 2026-10-09 |
| ANTH-INJECT | Anthropic | Mitigate jailbreaks and prompt injections | https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks | Untrusted content in `tool_result`, labelling, red-teaming (L1, L2, L6) | live | 2026-10-09 |
| ANTH-ERR | Anthropic | Claude API errors | https://platform.claude.com/docs/en/api/errors | SDK retries, `*anthropic.Error`, request IDs (R4, L7) | live | 2026-10-09 |
| ANTH-RATE | Anthropic | Rate limits | https://platform.claude.com/docs/en/api/rate-limits | Spend-cap 429 is not retryable (R8) | live | 2026-10-09 |
| ANTH-TOOLS | Anthropic | Tool use overview | https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview | `strict: true` tool schemas (L3) | live | 2026-10-09 |
| ANTH-AGENTS | Anthropic | Engineering: Building effective agents (2024-12-19) | https://www.anthropic.com/engineering/building-effective-agents | Stopping conditions (L5) | live; page notes tooling has changed since | 2026-10-09 |
| ANTH-CONTAIN | Anthropic | Engineering: How we contain Claude across products (2026-05-25) | https://www.anthropic.com/engineering/how-we-contain-claude | Enforced access boundaries, least privilege (L4) | live | 2026-10-09 |
| DOCKER-BP | Docker | Building best practices | https://docs.docker.com/build/building/best-practices/ | Dockerfile | live | 2026-10-09 |
| 12F | Heroku/Salesforce | The Twelve-Factor App (last updated 2017) | https://12factor.net/ | Config, logs, parity | live | 2026-10-09 |
| PG-IDENT | PostgreSQL | Lexical structure: identifiers | https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS | SQL identifiers | live; no official naming guide exists | 2026-10-09 |
| OTEL-NAMING | CNCF OpenTelemetry | Semantic conventions: naming | https://opentelemetry.io/docs/specs/semconv/general/naming/ | Attribute and log key naming | live | 2026-10-09 |
| GOSEC-RULES | securego | gosec rules list | https://github.com/securego/gosec/blob/master/RULES.md | G101, G201, G202, G402 in the linter note | live | 2026-10-09 |
| SLOGLINT | go-simpler | sloglint README | https://github.com/go-simpler/sloglint | `static-msg`, `key-naming-case` in the linter note | live | 2026-10-09 |
| GOOG-DOCSTYLE | Google | Developer documentation style guide | https://developers.google.com/style/highlights | Docs writing | live | 2026-10-09 |
| MS-STYLE | Microsoft | Writing Style Guide | https://learn.microsoft.com/en-us/style-guide/welcome/ | Docs writing | live (updated 2026-07) | 2026-10-09 |

## Authority by organization and topic

Within a topic, a higher row wins. "-" means the organization is not used for that topic.

| Topic | 1st | 2nd | 3rd | 4th | Not authoritative here |
|---|---|---|---|---|---|
| Go style and naming | Go team (CRC, go.dev docs, Effective Go baseline) | Google Go Style Guide | Uber (gaps only) | GitLab Go guide, Kubernetes coding conventions | Azure SDK naming (`Get` verbs), AIP, Microsoft REST |
| API clients and idempotency | Vendor contract (Slack, Jira, Tempo, Anthropic) | Stripe, AWS Builders' Library | Microsoft REST guidelines | Google AIP (own gRPC APIs) | Kubernetes API conventions (declarative APIs only) |
| Resilience | AWS Builders' Library and Well-Architected, Google SRE | Azure Architecture Center and Well-Architected | Netflix Hystrix (concepts, maintenance mode) | - | AIP-194 retry codes for REST vendors |
| Security | OWASP cheat sheets | Go security best practices | GitLab secure coding | AWS Well-Architected security pillar | Microsoft SDL (process only); OWASP Go-SCP chapters not read (stale since 2024-05) |
| LLM and agent safety | Anthropic docs and engineering posts | OWASP LLM prompt-injection cheat sheet, OWASP LLM Top 10 2026 (cite `LLMnn:2026`) | - | - | OWASP LLM Top 10 2025 numbering (Excessive Agency was LLM06, now LLM03) |
| Code review process | Google eng-practices | - | - | - | - |
| Containers and config | Docker docs | Twelve-Factor | OWASP Secrets (secret handling) | AWS SEC02-BP03 | - |
| SQL | PostgreSQL docs | GitLab Go guide (pgx, no ORM) | - | - | Google (no SQL style guide exists) |
| Docs | go.dev/doc/comment (Go comments) | Google developer style | Microsoft Writing Style Guide | - | Cloudflare style guide (content only, not used) |

## Known deviations and resolved conflicts

| Topic | Sources in conflict | Resolution |
|---|---|---|
| Secrets in env vars | 12factor config vs OWASP Secrets 5.1 ("not recommended unless the other methods are not possible") and AWS SEC02-BP03 (secrets service, rotation) | D3 keeps env vars for this single-user service as an accepted deviation; needs the owner's confirmation. Docker `ENV`/`ARG` secrets are banned by both sides. |
| Assertion libraries | GitLab suggests `testify/require` vs Google `use-package-testing` | Google wins by precedence: T2 stays (standard `testing` only). |
| Test placement (D-TESTS) | Go team: GO-CMD-TEST, GO-LAYOUT and GO-REPO-TEST put a package's tests beside its code, in the same directory | **Documented project deviation**, decided by the owner on 2026-10-09 in [ADR-0009](../../../docs/adr/0009-tests-in-separate-tree.md): every test lives in `services/timesheet-core/test/`, black-box only (T5). Accepted costs: only the exported API is testable, three exported seams (`slack.Searcher`, `slack.NewCollectorWithSearcher`, `httpx.ErrNoRewind`), and coverage needs `-coverpkg=./internal/...,./cmd/...`. The official sources stay cited; the Go convention is not changed by this row. |
| Retryable codes | Google AIP-194 (retry only `UNAVAILABLE`) vs vendor contracts (Anthropic SDK retries 429, 5xx, connection errors) | Vendor contract wins for consumed REST APIs; AIP applies only to own gRPC APIs. |
| Fallback | AWS "avoid fallback" vs Azure (degrade or fall back) | R6: fail fast; the only fallback is local and pure (`RulePolicy`). |
| Retry log level | Azure Retry pattern (early failures informational) vs Azure WAF (transient faults as warnings) | R9 uses `Warn` (WAF is the newer design guide); both agree only the final failure is `Error`. |
| Layered retries | `httpx.RetryTransport` plus the Anthropic SDK default of 2 retries | R4: one layer; set `option.WithMaxRetries(0)` if the transport retries. |
| Idempotency mechanism | AIP-155 `request_id`, Microsoft `Repeatability-*` headers, Stripe `Idempotency-Key`, AWS client token | K3: use the vendor's mechanism; semantics agree (same key returns the first outcome, a changed payload errors). |
| Wire naming, pagination, error shapes | AIP vs Microsoft Azure vs Graph vs Kubernetes | Not applicable to consumed APIs; JSON tags follow each vendor's contract. Go identifiers follow Go style. |

## Organizations with no official public standard (checked 2026-10-09)

Do not search these again for Go or service standards unless the scope changes.

| Organization | Finding |
|---|---|
| Meta | No official public company-wide coding standard or Go/backend style guide; only per-project contributing guides (React, PyTorch, Folly). |
| Apple | Nothing for Go or backend. Swift.org API Design Guidelines are Swift-only. |
| Cloudflare | Only a documentation content style guide; no Go or service standard. |
| Shopify | Only a Ruby style guide. |
| Spotify | Only engineering blog posts and product docs, not standards. |
| LinkedIn | No standard; Rest.li is a framework and its repo is archived (2026-07). |
| HashiCorp | No Go style guide; Terraform provider best practices target plugin authors only. |
| Twitter/X | Only Effective Scala (Scala-only, last push 2023-04). |
| Google (SQL) | No SQL or BigQuery style guide in google.github.io/styleguide. |
| Microsoft (SDL) | Practice-level only; no concrete secure-coding rules to cite. |

## Unverifiable

| Organization | Source | Reason |
|---|---|---|
| Netflix | Netflix Tech Blog fault-tolerance posts | netflixtechblog.com returned HTTP 403 to the fetcher on 2026-10-09 |
| Google | AIP-4221 client retry backoff | fetch returned only a redirect stub on 2026-10-09 |
| Netflix | Tech Blog: "Ready for changes with Hexagonal Architecture" | netflixtechblog.com and the Medium mirror returned Cloudflare 403 on 2026-10-09; content and date not verified, not cited |
| OWASP | genai.owasp.org 2025 entries LLM02 and LLM06 | HTTP 403 to every fetcher on 2026-10-09; superseded by the 2026 files on GitHub, which are cited instead |
| OWASP | ASVS 5.0 requirement text | only the chapter index (cheatsheetseries IndexASVS) was read; cite chapter and section IDs, not requirement wording |
| OWASP | LLM Top 10 2026 ordering | taken from the 2026/final file names (LLM01-LLM10); the resource page does not say it supersedes 2025 |
| Thoughtworks | Radar "Dependency drift fitness function", "Architecture drift reduction with LLMs" (vol. 34) | seen in search summaries only, not opened; not cited |
| Google | Follow-up prompt-injection post (2026-04-02) and go.dev/doc/build-cover | not fetched; not cited |

## Architecture sources

Cited by `.claude/skills/architecture-references/SKILL.md` (rule IDs AWS-n, AZ-n, GCP-n, DDD-n, SEC-n). That skill also reuses these rows from the main table: AWS-WA-REL05, AWS-WA-SEC02, AWS-BL-RETRY, AWS-BL-IDEMP, MS-WAF-TRANSIENT, MS-AAC-RETRY, GOOG-SRE-OVERLOAD, GOOG-SRE-CASCADE, OWASP-LLM-CS, OWASP-LLM01, ANTH-INJECT, ANTH-AGENTS, GO-LAYOUT. "re-checked" means the page was fetched a second time, independently of the research report, on the same day.

| ID | Organization | Source | URL | Authoritative for | Status | Verified |
|---|---|---|---|---|---|---|
| AWS-WA-RELDP | Amazon | Well-Architected reliability design principles | https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/design-principles.html | Test recovery procedures (AWS-11) | live; undated | 2026-10-09 |
| AWS-WA-OEDP | Amazon | Well-Architected operational excellence design principles | https://docs.aws.amazon.com/wellarchitected/latest/framework/oe-design-principles.html | Small reversible changes, test failure (AWS-11) | live; undated | 2026-10-09 |
| AWS-WA-REL03 | Amazon | Well-Architected REL03-BP02 bounded contexts | https://docs.aws.amazon.com/wellarchitected/latest/framework/rel_service_architecture_business_domains.html | Bounded contexts, ACL for a new context (AWS-5) | live; links to the dead hexagonal guide welcome.html | 2026-10-09 |
| AWS-WA-REL04 | Amazon | Well-Architected REL04-BP04 idempotent operations | https://docs.aws.amazon.com/wellarchitected/latest/framework/rel_prevent_interaction_failure_idempotent.html | Idempotency tokens, state, TTL (AWS-3) | live; says "exactly once", Azure says "effectively-once" (AZ-6 wording preferred) | 2026-10-09 |
| AWS-WA-SEC11 | Amazon | Well-Architected SEC11-BP02 automated security testing | https://docs.aws.amazon.com/wellarchitected/latest/framework/sec_appsec_automate_testing_throughout_lifecycle.html | Security testing in CI, SBOM (AWS-9) | live; undated | 2026-10-09 |
| AWS-WA-OPS04 | Amazon | Well-Architected OPS04-BP02 application telemetry | https://docs.aws.amazon.com/wellarchitected/latest/framework/ops_observability_application_telemetry.html | Logs, metrics, traces, masking (AWS-10) | live; undated | 2026-10-09 |
| AWS-PG-HEX | Amazon | Prescriptive Guidance: hexagonal architecture pattern (added 2024-05-07) | https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/hexagonal-architecture.html | Ports, adapters, when adapters pay off (DDD-1, DDD-3) | live; re-checked | 2026-10-09 |
| AWS-PG-OUTBOX | Amazon | Prescriptive Guidance: transactional outbox (updated 2024-02-23) | https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html | Outbox write, order, duplicates, sagas (AWS-6, scale filter) | live | 2026-10-09 |
| AWS-PG-ACL | Amazon | Prescriptive Guidance: anti-corruption layer | https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/acl.html | ACL intent and lifecycle (AWS-7, DDD-11) | live | 2026-10-09 |
| AWS-PG-CB | Amazon | Prescriptive Guidance: circuit breaker | https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/circuit-breaker.html | Breaker override and logging (scale filter) | live | 2026-10-09 |
| AWS-PG-DECOMP | Amazon | Prescriptive Guidance: decompose by subdomain (updated 2023-04-06) | https://docs.aws.amazon.com/prescriptive-guidance/latest/modernization-decomposing-monoliths/decompose-subdomain.html | "Too many microservices" drawback (scale filter) | live; guide landing is introduction.html | 2026-10-09 |
| MS-WAF-REL | Microsoft | Well-Architected reliability principles | https://learn.microsoft.com/en-us/azure/well-architected/reliability/principles | Keep it simple, critical path (AZ-1) | live (updated 2026-03-17) | 2026-10-09 |
| MS-WAF-SEC | Microsoft | Well-Architected security principles | https://learn.microsoft.com/en-us/azure/well-architected/security/principles | Threat modeling, supply chain, audit trail (AZ-10) | live (updated 2025-10-30) | 2026-10-09 |
| MS-AAC-ACL | Microsoft | Architecture Center: Anti-Corruption Layer | https://learn.microsoft.com/en-us/azure/architecture/patterns/anti-corruption-layer | Translation only, boundary validation (AZ-5) | live (updated 2026-09-26); re-checked | 2026-10-09 |
| MS-AAC-IDEMCONS | Microsoft | Architecture Center: Idempotent Consumer (new 2026-08-13) | https://learn.microsoft.com/en-us/azure/architecture/patterns/idempotent-consumer | Dedup key, atomic marker, in-progress/completed (AZ-6, AZ-7) | live (updated 2026-09-05); re-checked | 2026-10-09 |
| MS-AAC-OUTBOX | Microsoft | Architecture Center: transactional outbox with Cosmos DB | https://learn.microsoft.com/en-us/azure/architecture/databases/guide/transactional-out-box-cosmos | Relay, order, dedup, TTL (AZ-8); Cosmos-specific | live (updated 2026-09-25); no generic /patterns/ page | 2026-10-09 |
| MS-AAC-COMP | Microsoft | Architecture Center: Compensating Transaction | https://learn.microsoft.com/en-us/azure/architecture/patterns/compensating-transaction | Compensation steps (AZ-9) | live (updated 2026-09-29) | 2026-10-09 |
| MS-AAC-CB | Microsoft | Architecture Center: Circuit Breaker | https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker | When a breaker is unsuitable (scale filter) | live (updated 2026-09-26) | 2026-10-09 |
| MS-AAC-TACTICAL | Microsoft | Architecture Center: tactical DDD | https://learn.microsoft.com/en-us/azure/architecture/microservices/model/tactical-domain-driven-design | Aggregates, entities, value objects, services (DDD-5, DDD-7, DDD-8) | live (updated 2026-09-29) | 2026-10-09 |
| MS-AAC-DOMAIN | Microsoft | Architecture Center: domain analysis | https://learn.microsoft.com/en-us/azure/architecture/microservices/model/domain-analysis | Bounded contexts, subdomains, context map (DDD-4, DDD-11) | live (updated 2026-05-30) | 2026-10-09 |
| MS-AAC-EVOLVE | Microsoft | Architecture Center: design for change | https://learn.microsoft.com/en-us/azure/architecture/guide/design-principles/design-for-evolution | Abstract infrastructure, fitness functions (AZ-11) | live (updated 2026-03-06) | 2026-10-09 |
| MS-NET-DDD | Microsoft | .NET e-book: DDD-oriented microservice | https://learn.microsoft.com/en-us/dotnet/architecture/microservices/microservice-ddd-cqrs-patterns/ddd-oriented-microservice | Thin application layer, DDD only for complex services (DDD-3, DDD-8) | live (updated 2022-04-13); concepts only | 2026-10-09 |
| MS-NET-MODEL | Microsoft | .NET e-book: domain model | https://learn.microsoft.com/en-us/dotnet/architecture/microservices/microservice-ddd-cqrs-patterns/microservice-domain-model | Aggregate root entry point (DDD-6) | live (updated 2026-04-28) | 2026-10-09 |
| MS-NET-EVENTS | Microsoft | .NET e-book: domain events | https://learn.microsoft.com/en-us/dotnet/architecture/microservices/microservice-ddd-cqrs-patterns/domain-events-design-implementation | Event naming and dispatch timing (DDD-9) | live (updated 2024-01-03) | 2026-10-09 |
| MS-NET-REPO | Microsoft | .NET e-book: infrastructure persistence layer | https://learn.microsoft.com/en-us/dotnet/architecture/microservices/microservice-ddd-cqrs-patterns/infrastructure-persistence-layer-design | Repository per aggregate, unit of work (DDD-10) | live (updated 2023-02-21) | 2026-10-09 |
| MS-NET-CLEAN | Microsoft | .NET e-book: common web application architectures | https://learn.microsoft.com/en-us/dotnet/architecture/modern-web-apps-azure/common-web-application-architectures | Clean Architecture dependency rule, composition root (DDD-1, DDD-2) | live (updated 2026-07-08) | 2026-10-09 |
| MS-STRIDE | Microsoft | Threat Modeling Tool threats (STRIDE) | https://learn.microsoft.com/en-us/azure/security/develop/threat-modeling-tool-threats | STRIDE categories (SEC-1) | live (updated 2026-03-04) | 2026-10-09 |
| MSRC-XPIA | Microsoft | MSRC blog: defending against indirect prompt injection (2025-07-29) | https://www.microsoft.com/en-us/msrc/blog/2025/07/how-microsoft-defends-against-indirect-prompt-injection-attacks | Exfiltration blocking, human approval (SEC-9) | live via fetch tool; curl got 403 | 2026-10-09 |
| GOOG-WAF-REL | Google | Well-Architected Framework: reliability (updated 2024-12-30) | https://docs.cloud.google.com/architecture/framework/reliability | User-centred targets, recovery tests, postmortems (GCP-1, GCP-9) | live; moved from cloud.google.com (301) | 2026-10-09 |
| GOOG-WAF-SEC | Google | Well-Architected Framework: security (updated 2025-02-05) | https://docs.cloud.google.com/architecture/framework/security | Security by design, shift left (GCP-10) | live; moved from cloud.google.com (301) | 2026-10-09 |
| GOOG-SRE-SLO | Google | SRE Workbook: Implementing SLOs | https://sre.google/workbook/implementing-slos/ | SLIs, windows, no 100 percent (GCP-1, GCP-2) | live | 2026-10-09 |
| GOOG-AIP155 | Google | AIP-155 Request identification | https://google.aip.dev/155 | Request IDs for own APIs (GCP-8) | live (changelog through 2024-01-08) | 2026-10-09 |
| GOOG-GENAI-PI | Google | Google Security blog: mitigating prompt injection (2025-06-13) | https://blog.google/security/mitigating-prompt-injection-attacks/ | Layered defence, URL redaction, confirmation (SEC-9) | live | 2026-10-09 |
| GOOG-SWE | Google | Software Engineering at Google, ch. 11-14 | https://abseil.io/resources/swe-book | Test sizes, unit tests, doubles, larger tests (Testing architecture) | live; chapters at https://abseil.io/resources/swe-book/html/ch11.html, ch12.html, ch13.html, ch14.html; ch. 11 re-checked | 2026-10-09 |
| GO-FUZZ | Go team | Go Fuzzing | https://go.dev/doc/security/fuzz/ | Fuzz target rules (Testing architecture) | live; undated | 2026-10-09 |
| GO-RACE | Go team | Data Race Detector | https://go.dev/doc/articles/race_detector | `-race` limits and requirements (Testing architecture) | live; undated | 2026-10-09 |
| GO-COVER | Go team | Blog: The cover story (2013-12-02) | https://go.dev/blog/cover | Coverage flags (Testing architecture) | live; unit-test coverage only | 2026-10-09 |
| GO-VULN | Go team | Vulnerability management | https://go.dev/doc/security/vuln/ | `govulncheck` reachability (SEC-10) | live; undated | 2026-10-09 |
| GO-TX | Go team | Executing transactions | https://go.dev/doc/database/execute-transactions | `BeginTx`, deferred `Rollback` (DDD-12) | live; undated | 2026-10-09 |
| OWASP-ASVS | OWASP | Application Security Verification Standard 5.0.0 (May 2025) | https://github.com/OWASP/ASVS | Security requirement IDs (SEC-2) | live | 2026-10-09 |
| OWASP-ASVS-IDX | OWASP | Cheat Sheet Series ASVS 5.0 index | https://cheatsheetseries.owasp.org/IndexASVS.html | ASVS 5.0 chapter and section numbers (SEC-2) | live; re-checked (V1-V17) | 2026-10-09 |
| OWASP-LLM-2026 | OWASP | GenAI LLM Top 10 2026, LLM01-LLM10 source files | https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/tree/main/2026/final | LLM01, LLM02, LLM03 rules (SEC-3 to SEC-8, SEC-10, SEC-11) | live; re-checked (folder, LLM01, LLM02, LLM03); files cited: https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM02_SensitiveInformationDisclosure.md, https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM03_ExcessiveAgency.md; resource page https://genai.owasp.org/resource/owasp-genai-llm-top-10-2026/ dated 2026-08-03 | 2026-10-09 |
| OWASP-TM | OWASP | Threat Modeling Cheat Sheet | https://cheatsheetseries.owasp.org/cheatsheets/Threat_Modeling_Cheat_Sheet.html | Four questions, DFD, STRIDE (SEC-1) | live | 2026-10-09 |
| ANTH-PI-RESEARCH | Anthropic | Research: mitigating prompt injections in browser use (2025-11-24) | https://www.anthropic.com/research/prompt-injection-defenses | Adaptive-attacker testing (SEC-11) | live | 2026-10-09 |
| NFLX-HYSTRIX-README | Netflix | Hystrix README (maintenance mode) | https://github.com/Netflix/Hystrix | Breaker stance, adaptive concurrency limits (scale filter) | live | 2026-10-09 |
| TW-FITNESS | Thoughtworks | Technology Radar: architectural fitness function (Trial, 2017-11, updated 2018-05) | https://www.thoughtworks.com/radar/techniques/architectural-fitness-function | Fitness function definition (Fitness functions) | live; no longer on the current Radar | 2026-10-09 |
| TW-ADR | Thoughtworks | Technology Radar: lightweight ADRs (Adopt since 2018-05) | https://www.thoughtworks.com/en-us/radar/techniques/lightweight-architecture-decision-records | ADRs in source control (Fitness functions) | live | 2026-10-09 |

## Moved and dead URLs (checked 2026-10-09)

| Old URL | Status | Use instead |
|---|---|---|
| https://docs.aws.amazon.com/prescriptive-guidance/latest/hexagonal-architectures/welcome.html | dead (404); AWS REL03-BP02 still links to it | https://docs.aws.amazon.com/prescriptive-guidance/latest/hexagonal-architectures/introduction.html |
| https://docs.aws.amazon.com/prescriptive-guidance/latest/modernization-decomposing-monoliths/welcome.html and .../decomposing-patterns.html | dead (404) | https://docs.aws.amazon.com/prescriptive-guidance/latest/modernization-decomposing-monoliths/introduction.html |
| https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/ | moved (301) | https://builder.aws.com/content/3EumjoZascWd1oZiEgL8ORlv3qE/timeouts-retries-and-backoff-with-jitter |
| https://learn.microsoft.com/en-us/azure/well-architected/reliability/handle-transient-faults | moved (redirect) | https://learn.microsoft.com/en-us/azure/well-architected/design-guides/handle-transient-faults |
| https://learn.microsoft.com/en-us/azure/architecture/databases/guide/transactional-outbox-cosmos | moved (redirect) | https://learn.microsoft.com/en-us/azure/architecture/databases/guide/transactional-out-box-cosmos |
| https://learn.microsoft.com/en-us/azure/architecture/patterns/transactional-outbox | dead (404); no generic Microsoft outbox page | MS-AAC-OUTBOX (Cosmos guide) or AWS-PG-OUTBOX |
| Short slug .../microservices/model/tactical-ddd | moved (redirect) | https://learn.microsoft.com/en-us/azure/architecture/microservices/model/tactical-domain-driven-design |
| https://learn.microsoft.com/en-us/security/threat-modeling-tool/threats | dead (404, guessed URL) | https://learn.microsoft.com/en-us/azure/security/develop/threat-modeling-tool-threats |
| https://cloud.google.com/architecture/framework (and sub-pages) | moved (301) | https://docs.cloud.google.com/architecture/framework |
| https://docs.claude.com/... | moved (redirect) | https://platform.claude.com/docs/en/... |
| https://genai.owasp.org/llmrisk/llm01-prompt-injection/ | live but 2025 edition (archived) | https://github.com/GenAI-Security-Project/GenAI-LLM-Top10/blob/main/2026/final/LLM01_PromptInjection.md |
| https://www.uber.com/blog/microservice-architecture/ | 404 to fetch tool, 406 to curl | https://www.uber.com/en/blog/microservice-architecture/ (blog; not cited) |
