#!/usr/bin/env bash
# SessionStart: print a short project status; stdout is added to Claude's context.
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
echo "## timesheet-bot status"
echo "- branch: $(git branch --show-current 2>/dev/null || echo none), uncommitted files: $(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')"
owner=$(grep -rn 'TODO(owner)' --include='*.go' services 2>/dev/null | cut -d: -f1,2 | tr '\n' ' ')
[[ -n "$owner" ]] && echo "- owner-only TODOs (do not implement): $owner"
latest_plan=$(ls -1 docs/superpowers/plans/*.md 2>/dev/null | tail -1)
[[ -n "$latest_plan" ]] && echo "- latest plan: $latest_plan"
src=.claude/skills/coding-standards/sources.md
if [[ -f "$src" ]]; then
  oldest=$(grep -Eo '\| [0-9]{4}-[0-9]{2}-[0-9]{2} \|$' "$src" | tr -d '| ' | sort | head -1)
  if [[ -n "$oldest" ]]; then
    age=$(( ( $(date +%s) - $(date -d "$oldest" +%s) ) / 86400 ))
    (( age > 30 )) && echo "- coding-standards sources last verified $age days ago: suggest the user runs /update-standards verify"
  fi
fi
exit 0
