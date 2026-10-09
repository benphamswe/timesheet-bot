# solution-architect memory

## Decisions made with the owner (2026-10-09 brainstorm)

- Target Jira Tempo Cloud; week = Mon-Fri, proposed every Friday; laptop-hosted Docker on WSL.
- Slack signal: per-project channels plus issue keys in messages. Default 45 minutes per key per day.
- AI agent does full allocation; the rule policy is the fallback. Approval via Slack DM buttons (Socket Mode).
- PostgreSQL chosen over SQLite. Clean Architecture + DDD, package-by-context.
- Self-learning: lessons from the owner's own feedback activate automatically; agent-drafted skills need approval.

## Owner preferences

- No emoji or icons anywhere. No AI attribution in commits, PRs, or docs.
- Secrets only as environment variables in ~/.zshrc.
- Nothing is committed, pushed, or posted without the owner approving the exact content.

## Scale decision (2026-10-09, ADR-0006)

- The owner's long-term ambition is a very large user base. They chose: stay a personal tool now, scale-ready seams enforced, AWS as the future SaaS cloud.
- Do not propose multi-tenant or multi-region infrastructure unless the owner reopens ADR-0006.
