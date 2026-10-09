---
id: US-nnn
title: <short title>
status: New
parent: FE-nnn
points:
sprint:
github:
fr: [FR-XXX-n]
---

# US-nnn: <title>

As <role>, I want <capability>, so that <outcome>.

## Acceptance criteria

```gherkin
# AC-nnn.1
Scenario: <happy path>
  Given <starting state>
  When <action>
  Then <observable outcome>

# AC-nnn.2
Scenario: <failure path>
  Given <starting state>
  When <failing action>
  Then <observable outcome>
```

## Definition of Ready

- [ ] Story form, Gherkin AC with a failure path, at most 8 points, dependencies named, linked to its FR ids

## Tasks

Written by `technical-architect` when the plan is made; status kept by `sdlc-orchestrator`.

| Id | Task | WP | Owner agent | Estimate (h) | Status |
|---|---|---|---|---|---|
| US-nnn-T1 | <task> | WP-1 | go-coder | 2 | To Do |

## Traceability

| AC | Test | Result |
|---|---|---|
| AC-nnn.1 | `TestXxx` | |
