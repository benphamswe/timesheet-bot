# technical-architect memory

## Verified versions (2026-10-09)

- Go 1.26 (required by slack-go v0.30.1 and goose v3.28). pgx v5.11, cobra v1.10, golangci-lint v2.14.
- No sqlc: hand-written pgx queries. Integration tests use `TEST_DATABASE_URL` + build tag, not testcontainers.

## API facts that bit or nearly bit

- Tempo POST /4/worklogs accepts only numeric `issueId`; response `tempoWorklogId` int64; regional base URLs.
- Jira issue `id` is a string. Slack search dates are exclusive; `res.Page` is ambiguous in slack-go.

## Planning lessons

- Put shared contracts (ports, kernel types) in the first work package; later packages code against them.
- Budget a sequential work package for `go.mod` and `cmd/timesheet/wire.go` changes.
