# ADR-0005: Target Go 1.26 and embed migrations and tzdata in a distroless image

- Status: Accepted
- Date: 2026-10-09

## Context

`github.com/slack-go/slack` v0.30.1 and `github.com/pressly/goose/v3` v3.28 require Go 1.26. The runtime image should be small and have no shell (NFR-10).

## Decision

- `go.mod` declares `go 1.26.0`; builds use `golang:1.26-alpine`.
- Migrations are embedded (`migrations/embed.go`) and run by `timesheet migrate`.
- The binary imports `time/tzdata` because `gcr.io/distroless/static-debian12:nonroot` has no zoneinfo.
- Queries are hand-written with pgx (no sqlc) while the schema is small.

## Consequences

- Positive: one 24 MB static image; no external migration tool.
- Negative: older local Go toolchains rely on `GOTOOLCHAIN=auto` to download 1.26.
