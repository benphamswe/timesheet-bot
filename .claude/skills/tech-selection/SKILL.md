---
name: tech-selection
description: timesheet-bot production-readiness criteria for choosing a library, service, protocol or tool - vulnerabilities, maintenance, license, API stability, supply chain, secure and network defaults - with the commands that produce the evidence. Preloaded into solution-architect, technical-architect, security-engineer, networking-engineer and sre-engineer.
user-invocable: false
---

# Technology selection: production readiness

Go paths (`internal/`, `cmd/`, `migrations/`) are relative to `services/timesheet-core/`. Go tools run there, or from the repo root with `go -C services/timesheet-core`.

Every new dependency, external service, protocol or tool in a plan gets one row in the plan's **Technology decisions** table. It is reviewed by `security-engineer`, `networking-engineer`, `sre-engineer` and `solution-architect` before the plan becomes `status: final`.

## Evidence commands

```bash
# Version metadata: license, advisories, publish date (Google Open Source Insights)
curl -sS "https://api.deps.dev/v3/systems/go/packages/<url-encoded module>/versions/<version>" | jq '{licenses, advisoryKeys, publishedAt}'
# Project health: OpenSSF Scorecard, open issues
curl -sS "https://api.deps.dev/v3/projects/<url-encoded github.com/org/repo>" | jq '{scorecard: .scorecard.overallScore, openIssuesCount}'
# Reachable known vulnerabilities, after adding the dependency on a branch
govulncheck ./...
# Latest version from the Go module proxy
curl -sS https://proxy.golang.org/<module>/@latest | jq .Version
```

Claims about a vendor (an official SDK exists, a deprecation, an API limit) go through `official-source-verifier`.

## Criteria

| ID | Criterion | APPROVED | CONDITIONAL (state the condition) | REJECTED |
|---|---|---|---|---|
| P1 | Known vulnerabilities | `advisoryKeys` empty and govulncheck clean | advisory present but not reachable, with a fixed version planned | reachable vulnerability without a fix |
| P2 | Maintenance | release within 12 months, issues answered | 12-24 months, or a single maintainer | archived, or nothing for 24 months |
| P3 | Supply-chain health | OpenSSF Scorecard of 5.0 or more | under 5.0 with a reason (small, well-known project) | unknown publisher, typosquat risk |
| P4 | License | MIT, BSD-2/3-Clause, Apache-2.0, ISC | MPL-2.0 (file-level copyleft) | GPL, AGPL, SSPL, no license |
| P5 | API stability | v1 or later (semver stable) | v0, pinned exactly, with an upgrade owner | unpublished or pseudo-version only |
| P6 | Provenance | official vendor SDK or Go team (`golang.org/x`) | well-known community library when no official SDK exists (verified) | a fork or a re-upload |
| P7 | Go compatibility | supports the `go` and `toolchain` lines in `go.mod` | requires a toolchain bump (plan it) | requires an unreleased Go |
| P8 | Footprint | few transitive dependencies (deps.dev dependency graph) | heavy, but justified by the value | pulls in a second HTTP, logging or DI framework |
| P9 | Secure defaults | TLS verification on, no credentials in URLs, context support | needs explicit hardening (list the settings) | forces insecure options |
| P10 | Network behaviour | honors `context` deadlines, configurable timeouts, proxy from environment, a single retry layer (R4) | retries or keepalives need wrapping | hidden retries or goroutines that cannot be stopped |
| P11 | Operability | errors expose a status and request ids; logs can be routed to slog | partial | silent failures |
| P12 | Currency | the latest stable release (`proxy.golang.org/<module>/@latest`), and the latest Go patch toolchain (`go.dev/dl/?mode=json`) | one minor behind, with an upgrade task | pre-release, deprecated or end-of-life |

**A plan is final only when every row is APPROVED or CONDITIONAL, and every condition appears as a task in a work package.**

## Decision record

Row format in the plan:

```text
| Tech | Version | Purpose | Alternatives considered | P1..P12 evidence | Security | Networking | SRE | SA fit | Verdict |
```

Choices that are hard to reverse (a datastore, a protocol, a framework) also get an ADR.
