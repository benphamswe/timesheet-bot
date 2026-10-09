#!/usr/bin/env bash
# UserPromptSubmit: classify the owner's request and tell Claude which agents to dispatch
# immediately (CLAUDE.md "Which agent to call"). Stdout is added to Claude's context.
set -uo pipefail
prompt=$(jq -r '.prompt // empty' | tr '[:upper:]' '[:lower:]')
[[ -n "$prompt" ]] || exit 0
[[ "$prompt" == /* ]] && exit 0   # slash commands already route themselves

routes=()
add() { routes+=("$1"); }
m() { grep -Eq -- "$1" <<<"$prompt"; }

m '\bfeatures?\b|phase [0-9]|\bimplement|tính năng|làm chức năng|xây dựng' && add "/sdlc <feature> (sdlc-orchestrator runs the full pipeline)"
m 'bug|fix|broken|fail|lỗi|sửa'                     && add "superpowers:systematic-debugging, then go-coder for the fix and test-engineer for a regression test"
m 'review|kiểm tra code|đánh giá code'              && add "/review-fanout (ddd-reviewer + standards-reviewer, verified findings)"
m '\btests?\b|testing|coverage|\bfuzz|\brace\b|kiểm thử'                && add "test-engineer (unit/integration) and qa-engineer (contract, e2e, fuzz, race, coverage)"
m 'security|vuln|secret|token|bảo mật|cve'          && add "security-engineer"
m 'network|timeout|proxy|\btls\b|websocket|\bdns\b|mạng'    && add "networking-engineer"
m 'slo|reliab|incident|outage|postmortem|runbook|monitor|sự cố' && add "sre-engineer"
m 'architect|design|adr|ddd|clean arch|kiến trúc|thiết kế' && add "solution-architect, then architecture-reviewer (relevant lenses) and technical-architect mode review"
m 'library|dependency|upgrade|version|latest|thư viện|công nghệ|mới nhất' && add "technical-architect (Technology decisions) with security-engineer, networking-engineer and sre-engineer mode tech-review"
m '\bci\b|pipeline|github action|docker|release|changelog' && add "devops-engineer"
m '\bstor(y|ies)\b|\bepics?\b|backlog|sprint|\btasks?\b|requirement|yêu cầu'  && add "business-analyst (and /backlog-sync to mirror to GitHub after approval)"
m '\bdocs?\b|documentation|readme|tài liệu'                              && add "tech-writer"
m '\blearn|explain|what is|\bwhy\b|how does|giải thích|là gì|tại sao|\bhọc\b' && add "/learn <question> (learning-coach saves the answer to docs/learn)"
m 'standard|best practice|convention|naming|chuẩn'   && add "standards-reviewer, and /update-standards when a rule is missing"
m 'verify|official|source|docs chính thức|xác minh'  && add "official-source-verifier"

(( ${#routes[@]} )) || routes=("general-purpose (no specialist matches this request)")
echo "Request routing (CLAUDE.md, which agent to call): act only as the orchestrator. Do not read, search, edit or run anything in the main session, not even a small edit or quick lookup. Dispatch:"
printf -- '- %s\n' "${routes[@]}"
echo "Split independent parts and dispatch one agent per part in a single message so they run concurrently. Keep only the agents' conclusions. Every external claim goes through official-source-verifier."
exit 0
