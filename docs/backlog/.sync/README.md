# Backlog sync

This directory holds the GitHub Issues mirror files that `backlog-manager` and `/backlog-sync` write:

- `draft.md`: every create, update and link operation, with the exact `gh` commands. The owner reviews it and its sha256 before anything is published.
- `log.md`: one entry per publish, with the date, the draft sha256 and the created issue numbers.

Nothing here is published by itself. See [`docs/backlog/README.md`](../README.md).
