#!/usr/bin/env bash
# PreToolUse(Bash): enforce the owner's git rules before the command runs.
set -euo pipefail
cmd=$(jq -r '.tool_input.command // empty')
[[ -n "$cmd" ]] || exit 0

if grep -Eq 'git[[:space:]]+commit' <<<"$cmd"; then
  if grep -Eiq 'co-authored-by|generated with|claude\.com|noreply@anthropic' <<<"$cmd"; then
    echo "Blocked: commit message contains AI attribution. Commits carry only the owner's identity; remove the trailer." >&2
    exit 2
  fi
  if grep -Eq -- '--no-verify|--author' <<<"$cmd"; then
    echo "Blocked: --no-verify and --author are not allowed; commits use the repository identity and run all checks." >&2
    exit 2
  fi
fi
if grep -Eq 'git[[:space:]]+push' <<<"$cmd" && grep -Eq -- '(^|[[:space:]])(-f|--force|--force-with-lease)([[:space:]]|$)' <<<"$cmd"; then
  echo "Blocked: force push rewrites published history; ask the owner first." >&2
  exit 2
fi
exit 0
