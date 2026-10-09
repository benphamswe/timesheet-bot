# Backlog

The backlog is the source of truth for what gets built. GitHub Issues is a mirror, created by `/backlog-sync` only after the owner approves a draft.

## Hierarchy

```text
Epic  EP-nnn      a business outcome; spans several sprints or releases
└─ Feature  FE-nnn     a shippable capability; may take one or more sprints
   └─ User story  US-nnn   one user-visible slice; fits in one sprint
      └─ Task  US-nnn-Tn   one unit of engineering work, in hours
```

The levels follow the Azure DevOps Agile process (Epic, Feature, User Story, Task). Its sizing guidance: backlog items such as stories and tasks complete within a sprint, while features and epics "might take one or more sprints to complete". [Azure DevOps: define features and epics](https://learn.microsoft.com/en-us/azure/devops/boards/backlogs/define-features-epics), [choose a process](https://learn.microsoft.com/en-us/azure/devops/boards/work-items/guidance/choose-process)

## Layout

```text
docs/backlog/
├── epics/EP-001-<slug>/
│   ├── epic.md
│   └── features/FE-001-<slug>/
│       ├── feature.md
│       └── stories/US-001-<slug>.md     story, Gherkin AC, task table
├── sprints/SP-01.md                      goal, commitment, progress, review, retro
├── templates/                            epic, feature, story, sprint
└── .sync/                                GitHub mirror drafts and publish log
```

Ids are numbered across the whole repository and never reused. Each file's frontmatter holds `id`, `status`, `parent`, and `github` (the mirrored issue number, set by `backlog-manager`).

## States

`New` -> `Active` -> `Resolved` -> `Closed` (or `Removed`), matching the Azure DevOps Agile states. For a user story: Active means it is committed in a sprint, Resolved means all tasks are done and tests pass, and Closed means the owner agrees it meets its acceptance criteria. [Agile process workflow](https://learn.microsoft.com/en-us/azure/devops/boards/work-items/guidance/agile-process-workflow)

## Sprints

- A sprint has a fixed length of one month or less and a single Sprint Goal. This repository uses **2-week** sprints. [Scrum Guide 2020](https://scrumguides.org/scrum-guide.html#the-sprint)
- An increment is born when an item meets the Definition of Done. [Scrum Guide 2020: Increment](https://scrumguides.org/scrum-guide.html#increment)

## Team conventions (not prescribed by the Scrum Guide)

The Scrum Guide prescribes no user stories, story points, velocity or Definition of Ready; it asks only that the developers size items. These are this repository's choices:

- **Story points**: 1, 2, 3, 5, 8. A story above 8 is split. Tasks are estimated in hours.
- **Acceptance criteria**: Gherkin, using `Given`/`When`/`Then`/`And`/`But`, with ids `AC-nnn.m`. [Gherkin reference](https://cucumber.io/docs/gherkin/reference)

### Definition of Ready (story)

- [ ] Written as `As <role>, I want <capability>, so that <outcome>`
- [ ] Acceptance criteria in Gherkin, including at least one failure path
- [ ] Sized at 8 points or fewer, with dependencies named
- [ ] Linked to its feature and its `FR-*` ids

### Definition of Done (story)

- [ ] Every `AC-nnn.m` has a passing test that carries its id
- [ ] `make ci` passes (lint, unit, race, govulncheck, build) and integration tests pass; tests skipped with a `TODO(owner)` reason are allowed
- [ ] Code review, TA conformance, security and SRE findings resolved or explicitly accepted
- [ ] Feature doc, CHANGELOG entry and learning guide updated
- [ ] The owner accepted it (status `Closed`)

## GitHub mirror

`/backlog-sync` maps epics, features and stories to issues with `type:*` labels and sub-issues (`gh issue create --parent`), tasks to checklist items, and sprints to milestones. GitHub supports up to 100 sub-issues per parent and eight levels of nesting. [Adding sub-issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/adding-sub-issues), [gh issue create](https://cli.github.com/manual/gh_issue_create), [milestones](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/about-milestones)
