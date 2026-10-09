# Runbook: rotate a token

Rotate immediately if a token was ever pasted into chat, a log, a file or a commit.

| Token | Where to rotate |
|---|---|
| `SLACK_USER_TOKEN` / `SLACK_BOT_TOKEN` / `SLACK_APP_TOKEN` | api.slack.com/apps > your app > OAuth & Permissions (reinstall) / Basic Information > App-Level Tokens |
| `TEMPO_API_TOKEN` | Tempo > Settings > API integration > revoke and create |
| `MAVENLINK_API_TOKEN` / `MAVENLINK_SESSION_COOKIE` | See [Mavenlink credentials](mavenlink-credentials.md#rotate): revoke on the registered app's page / sign out of Mavenlink and copy a fresh `Cookie` header |
| `JIRA_API_TOKEN` | id.atlassian.com > Security > API tokens > revoke and create |
| `ANTHROPIC_API_KEY` | console.anthropic.com > API keys |

Steps:

1. Create the new token, then revoke the old one.
2. Update the token wherever it lives: the line in the repository-root `.env` (owner-only, gitignored), or the `export` line in `~/.zshrc`. Then open a new shell. If it is in both, update both, because the shell wins over `.env`.
3. Recreate the container so it picks up the new environment: `docker compose up -d --force-recreate app` (P2) or simply run the next `docker compose run`.
4. Verify: `docker compose run --rm app plan --dry-run`.
