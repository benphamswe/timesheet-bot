#!/usr/bin/env bash
# PreToolUse(Edit|Write): block writing anything that looks like a real credential.
# Exit 2 blocks the tool call; stderr is shown to Claude. The matched value is never printed.
set -euo pipefail
content=$(jq -r '[.tool_input.content, .tool_input.new_string] | map(select(. != null)) | join("\n")')
[[ -n "$content" ]] || exit 0

patterns=(
  'xox[abposr]-[0-9A-Za-z-]{10,}'          # Slack bot/user/legacy tokens
  'xapp-[0-9]-[0-9A-Za-z-]{10,}'           # Slack app-level token
  'sk-ant-[0-9A-Za-z_-]{20,}'              # Anthropic API key
  'ATATT[0-9A-Za-z_=-]{20,}'               # Atlassian API token
  '-----BEGIN [A-Z ]*PRIVATE KEY-----'
  'postgres(ql)?://[^:/@ ]+:[^@ ]+@'       # DSN with an inline password
)
for p in "${patterns[@]}"; do
  if grep -Eq -- "$p" <<<"$content"; then
    # The local compose database credential is a documented, local-only exception.
    if [[ "$p" == postgres* ]] && ! grep -E -- "$p" <<<"$content" | grep -Evq 'timesheet:timesheet@|t:t@localhost'; then
      continue
    fi
    echo "Blocked: content matches a credential pattern (${p%% *}). Reference the secret by environment variable name instead (see docs/operations/configuration.md)." >&2
    exit 2
  fi
done
exit 0
