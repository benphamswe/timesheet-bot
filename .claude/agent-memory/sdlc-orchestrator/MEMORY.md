# sdlc-orchestrator memory

## Project state (2026-10-09)

- Phase 1 built (CLI, rule policy, Postgres, Slack/Jira/Tempo adapters). Phases 2-4 planned in the design spec.
- `scaleDay` in `internal/allocation/domain/rule.go` is TODO(owner); `TestRulePolicyScalesOverCapDay` fails by design.

## Known open issues to schedule

- `httpx.RetryTransport` retries non-idempotent POSTs on 5xx (coding-standards R3) and has no jitter (R2); 5 attempts instead of 3. Risk: duplicate Tempo worklogs.
- First `golangci-lint run` reported 67 issues (about 44 missing doc comments, `os.Exit` after defer in main.go, `min`/`max` shadowing, gosec G101 false positives on env var names).
- Dockerfile base images are not pinned to digests (D2).

## How runs go best

- `isolation: worktree` is unusable until Phase 1 is committed: worktrees branch from the last commit.
- Parallel coders must own disjoint packages; `go.mod` and `cmd/timesheet/wire.go` are shared files, so edit them sequentially.
