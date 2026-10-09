---
name: sdlc
description: Run the full timesheet-bot SDLC for a feature or phase (BA, SA, architecture review board, TA plan, coding, code review, TA conformance review, testing, security, BA acceptance) through the sdlc-orchestrator agent.
argument-hint: "<feature or phase> [--from design|implement|review]"
disable-model-invocation: true
context: fork
agent: sdlc-orchestrator
background: false
---

Run the SDLC for: $ARGUMENTS

Working directory: the timesheet-bot repository root.

Stage selection (the steps are numbered in your instructions):

- `--from design`, or no flag and no `status: final` plan for this scope: run steps 1-8 and stop at GATE 1.
- `--from implement`: the owner approved the plan named in the arguments, or the newest plan in `docs/superpowers/plans/` matching the scope. Run steps 9-19. Refuse if the plan is not `status: final`.
- `--from review`: the code already exists as uncommitted changes (`git status --short`). Run steps 11-19.

Phases defined in the design spec: Phase 2 (Slack approval bot, outbox relay, scheduler), Phase 3 (Claude agent policy and skills), Phase 4 (learning context). Any other text is a new feature.

End with the gate report described in your instructions. Its first line is `GATE: <what the owner must approve>`.
