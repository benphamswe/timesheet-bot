.PHONY: sync lock lint fix fmt test test-race test-int cover vuln build fuzz arch run collect ci

# The root Makefile runs every target in every service (ADR-0008), so this file answers the same
# target names as services/timesheet-core. Targets with no Python meaning yet say so and pass.
# uv is pinned in pyproject.toml ([tool.uv] required-version); installs never re-resolve.
UV ?= uv

# Installs the locked dependencies into .venv; fails if uv.lock is out of date.
sync:
	$(UV) sync --locked

# Re-resolves uv.lock after a pyproject.toml change.
lock:
	$(UV) lock

lint: sync
	$(UV) run --locked ruff check .
	$(UV) run --locked ruff format --check .

fix: sync
	$(UV) run --locked ruff check --fix .
	$(UV) run --locked ruff format .

fmt: fix

# Tests live in tests/ (docs/architecture/testing.md); they never call Claude or Slack.
test: sync
	$(UV) run --locked pytest

test-race: test

# `run` is a no-op here so the root `make run ARGS=...` still reaches the Go CLI.
test-int cover vuln build fuzz arch run:
	@echo "allocation-agent: no '$@' step for the Python service yet"

# LOAD_DOTENV exports this service's own .env (services/allocation-agent/.env, if present;
# gitignored, written by the owner only; ADR-0013) into the recipe's shell, then restores every
# variable already exported in your shell (~/.zshrc), so the shell wins and an empty KEY= never
# clears a shell value. It prints nothing. Same macro as services/timesheet-core/Makefile. The ./
# matters: `. .env` without a slash would search PATH in a POSIX shell.
DOTENV      := ./.env
LOAD_DOTENV  = set -a; if [ -f $(DOTENV) ]; then shell_env=$$(export -p); . $(DOTENV); eval "$$shell_env"; fi; set +a;

# Collect one ISO week through the claude.ai Slack connector, e.g.
#   make collect WEEK=2026-W41            writes <repo root>/data/activity/2026-W41.json
#   make collect WEEK=2026-W41 ARGS=--dry-run
# A relative --out-dir in ARGS is relative to this folder; pass an absolute path.
# Needs a claude.ai login in Claude Code with the Slack connector authenticated (ADR-0011).
# `collect` exits 2 at startup if ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN or
# CLAUDE_CODE_OAUTH_TOKEN is set (from the shell or .env): each hides the connector.
collect: sync
	@test -n "$(WEEK)" || { echo "usage: make collect WEEK=2026-W41 [ARGS=--dry-run]" >&2; exit 2; }
	@$(LOAD_DOTENV) $(UV) run --locked collect --week $(WEEK) $(ARGS)

ci: lint test
