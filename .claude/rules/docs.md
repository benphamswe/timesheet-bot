---
paths:
  - "docs/**/*.md"
  - "README.md"
  - "CLAUDE.md"
  - "CHANGELOG.md"
  - "services/*/README.md"
  - "api/proto/README.md"
---

# Documentation

## Never

- Edit the decision of an `Accepted` ADR. Write a new ADR that supersedes it, and set the old one's status to `Superseded by ADR-NNNN`.
- Describe a feature as working without its status marker (`[built]` or `[planned: Pn]`).
- Paste token values, real account ids, or personal data.

## Always

- Use sentence-case headings, second person, active voice, and code font for paths and identifiers.
- Name files `kebab-case.md`. Name ADRs `NNNN-imperative-title.md`, based on `docs/adr/0000-template.md`.
- Update `docs/README.md` when you add a folder, and update `docs/requirements/functional.md` status markers when a phase ships.
