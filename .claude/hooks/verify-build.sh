#!/usr/bin/env bash
# Stop hook for go-coder, test-engineer and qa-engineer: the agent cannot finish with a broken build.
set -uo pipefail
[[ "$(jq -r '.stop_hook_active // false')" == "true" ]] && exit 0  # avoid loops
cd "${CLAUDE_PROJECT_DIR:-.}/services/timesheet-core" || exit 0
if ! out=$(go build ./... 2>&1 && go vet ./... 2>&1 && go vet -tags=integration ./... 2>&1); then
  printf 'Build or vet failed; fix before finishing:\n%s\n' "$(tail -20 <<<"$out")" >&2
  exit 2
fi
exit 0
