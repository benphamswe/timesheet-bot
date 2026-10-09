---
name: learn
description: Ask anything about a timesheet-bot feature or the concepts behind it (DDD, clean architecture, Go, retries, Slack, LLM agents); the learning-coach answers from this codebase and verified official sources and saves the answer into docs/learn/ as study material.
argument-hint: "<question> [--feature FE-nnn]"
context: fork
agent: learning-coach
background: false
---

Mode: `ask`

Question: $ARGUMENTS

Working directory: the timesheet-bot repository root.

If `--feature FE-nnn` is given, save into that feature's `qa.md`. Otherwise decide: a question about a specific feature goes to that feature's `qa.md`, and a general question goes to a concept page.

Return the full answer in the shape your instructions define, followed by the saved paths.
