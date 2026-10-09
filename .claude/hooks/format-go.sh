#!/usr/bin/env bash
# PostToolUse(Edit|Write): format an edited Go file with the repo's formatters
# (gofmt + goimports with the local prefix, from .golangci.yml).
set -euo pipefail
file=$(jq -r '.tool_input.file_path // empty')
[[ "$file" == *.go && -f "$file" ]] || exit 0
cd "${CLAUDE_PROJECT_DIR:-.}/services/timesheet-core" 2>/dev/null || cd "${CLAUDE_PROJECT_DIR:-.}"
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint fmt "$file" >/dev/null 2>&1 || gofmt -w "$file"
else
  gofmt -w "$file"
fi
exit 0
