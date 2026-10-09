---
paths:
  - "services/timesheet-core/internal/*/domain/**/*.go"
  - "services/timesheet-core/internal/shared/kernel/**/*.go"
---

# Domain and kernel packages

Keep these packages pure: they encode the business rules every other layer relies on.

**This rule overrides any shortcut that adds an import or a setter to make a change compile faster.**

## Never

- Import anything except the stdlib, `internal/shared/kernel`, and the package's own context. No SDKs, no `database/sql`, no `net/http`, no other context's `domain`.
- Call `time.Now()`. Take `now time.Time` or a `kernel.Clock`.
- Add an exported setter or a field write path that skips a status check.
- Export a struct field of a value object.

## Always

- Return a wrapped sentinel error (`fmt.Errorf("%w: ...", ErrX)`) from every rejected state change.
- Record a `domain.Event` on every status transition.
- Add a table-driven test for every new invariant and transition, legal and illegal.
- Verify boundaries with `go list -deps ./internal/<ctx>/domain | grep timesheet-bot`, run inside `services/timesheet-core`, and with `make arch`.
