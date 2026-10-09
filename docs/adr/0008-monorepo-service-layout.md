# ADR-0008: Organise the repository as a monorepo of services with orchestration-only root

- Status: Accepted (2026-10-09, owner)
- Date: 2026-10-09
- Deciders: owner
- Related: [ADR-0006](0006-personal-tool-scale-ready-aws-later.md) (scale-ready seams), [ADR-0007](0007-grpc-contracts-now-split-later.md) (gRPC contracts and allocation-agent; this ADR does not depend on its outcome)

## Context

The repository started as one Go module at the root. ADR-0007 adds a second service in another language (Python), and ADR-0006 expects bounded contexts to become services later. A root that holds one language's tooling (`go.mod`, `.golangci.yml`) makes a second toolchain a special case and couples every service's CI to the first one. The move to `services/` has already been made; this ADR records it.

## Decision

1. Each deployable service lives in `services/<name>/` and owns its whole toolchain there: language manifest and lockfile (`go.mod` and `go.sum`; the Python lockfile chosen in the Phase 3 plan), linter config, `Dockerfile`, and a `Makefile` with at least the `ci` target.
2. The root holds orchestration only: a delegating `Makefile` (every target runs in each `services/*/` that has a `Makefile`), `docker-compose.yml`, `.github/`, `docs/`, and `api/proto/` for shared contracts. No language manifest or language tooling lives at the root.
3. Shared contracts live only in `api/proto/`. Generated code goes to the service that uses it. Services never import each other's source.
4. A service's paths in documentation are relative to its own folder (for Go: `services/timesheet-core/`).
5. `make ci` at the root is the single local and CI entry point, and it fails if any service fails.

## Alternatives considered

- Single Go module at the root (the original layout): rejected; a second language becomes a special case and root tooling leaks into every service.
- One repository per service: rejected; for one owner it multiplies CI, versioning and contract publishing with no benefit, and contracts would need a registry.
- Go workspace (`go.work`) across several Go modules at the root: not needed while there is one Go service; revisit when a second Go service exists.

## Consequences

- Positive: a new service or language is a new folder; each service can be built, scanned and released on its own; the AWS move can deploy services independently.
- Negative: commands must run in the right folder (Go commands outside `services/timesheet-core/` see no `go.mod`); CI and Dependabot need one entry per service and ecosystem.
- Follow-up: `CLAUDE.md` and the root `Makefile` comment cite ADR-0008 for the layout; `.github/dependabot.yml` and `ci.yml` gain an entry for each new service; `docs/architecture/project-structure.md` links this ADR.

## Reviews

- [Finding S1 in the ADR-0007 production-readiness review, 2026-10-09](reviews/0007-production-readiness.md#6-finding-s1-repository-layout-has-no-adr)
