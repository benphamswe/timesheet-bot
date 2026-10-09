# ADR-0001: Organise code by bounded context with Clean Architecture layers

- Status: Accepted
- Date: 2026-10-09

## Context

The bot integrates four external systems (Slack, Jira, Tempo, Anthropic) and will grow an AI agent and a learning loop. The domain rules (15-minute steps, daily cap, one entry per issue and day, approval lifecycle) must stay correct no matter which system or policy produces the data (NFR-2, NFR-7, NFR-11).

## Decision

- Packages are grouped per bounded context: `internal/<context>/{domain,application,infrastructure}`; entry points live in `internal/interfaces`, wiring only in `cmd/timesheet/wire.go`.
- Domain packages import only the stdlib and `internal/shared/kernel`. A domain package never imports another context's domain.
- Types shared between contexts (the published language, such as `WorkActivity`) live in the kernel.
- Ports are defined by the consuming application package.

## Alternatives considered

- Package by layer (`domain/`, `usecase/`, `adapter/`): Go visibility could not enforce context boundaries; rejected.
- Flat package: fastest to start, but AI output and adapters would reach domain state directly; rejected.

## Consequences

- Positive: boundaries are checkable with `go list -deps`; adapters are swappable and testable with fakes.
- Negative: more packages and some mapping code (snapshots, adapter DTOs).
