# Contracts

gRPC contracts between services live in `api/proto/timesheetbot/<context>/v1/` ([ADR-0007](../adr/0007-grpc-contracts-now-split-later.md)). No `.proto` files or `buf.yaml` exist yet, so everything on this page except the placeholder output is [planned: P3].

## `make proto-lint`

```bash
make proto-lint
```

- **Run from**: the repository root only. It is defined in the root `Makefile`, not in any service.
- **What it does**: if `api/proto` contains a `.proto` file, runs `buf lint` and then `buf breaking --against '.git#branch=main'`. Otherwise prints a placeholder.
- **When to use it**: after editing any `.proto` file, and before a review.
- **Expected today**: `no proto files yet`, exit 0. Confirmed 2026-10-09.
- **Common failure, once protos exist**: `buf: command not found`. Install the buf CLI (ADR-0007 evaluated v1.73.0) and rerun. buf is not installed on the owner's machine as of 2026-10-09.

## `buf lint` [planned: P3]

```bash
buf lint
```

- **Run from**: the repository root (where `buf.yaml` will live).
- **What it does**: checks the protos against the buf style rules (package versioning, naming, comments).
- **When to use it**: through `make proto-lint`.
- **Not run while documenting**: there are no protos and buf is not installed.

## `buf breaking` [planned: P3]

```bash
buf breaking --against '.git#branch=main'
```

- **Run from**: the repository root.
- **What it does**: compares the working tree's protos with those on `main` and fails on any wire- or source-breaking change to a `v1` package. Breaking changes go into a new `v2` package instead.
- **When to use it**: through `make proto-lint`, on every change to a contract.
- **Common failure**: `main` must exist as a local branch for the `.git#branch=main` input. Keep the argument in single quotes.
- **Not run while documenting**: there are no protos and buf is not installed.

## `buf generate` [planned: P3]

ADR-0007 also plans a CI check that `buf generate` produces no diff from the committed generated code. The exact command is added here when the Phase 3 plan defines `buf.gen.yaml`.
