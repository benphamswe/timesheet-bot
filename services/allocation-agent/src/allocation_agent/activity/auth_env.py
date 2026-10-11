"""Startup check: `collect` must run on the claude.ai subscription login.

Claude Code fetches claude.ai connectors (the Slack connector this collector uses) only when its
active credential is a claude.ai subscription login. These variables select another credential,
so with any of them set the connector never loads (https://code.claude.com/docs/en/mcp, "Use MCP
servers from claude.ai"; precedence: https://code.claude.com/docs/en/authentication):

- `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` outrank the subscription login;
- `CLAUDE_CODE_OAUTH_TOKEN` (from `claude setup-token`) can only make model requests.

The check reads names only and never prints or returns a value. Other secrets, such as
`DEEPSEEK_API_KEY`, are not checked.
"""

from __future__ import annotations

from collections.abc import Mapping

CONNECTOR_BLOCKING_VARS = ("ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN")


class AuthEnvError(Exception):
    """A variable that hides the claude.ai connectors is set. The message holds names only."""


def blocking_auth_vars(env: Mapping[str, str]) -> list[str]:
    """Names from CONNECTOR_BLOCKING_VARS whose value is non-empty, in that order."""
    return [name for name in CONNECTOR_BLOCKING_VARS if env.get(name)]


def check_claude_ai_login_env(env: Mapping[str, str]) -> None:
    """Raises AuthEnvError if any connector-blocking variable is non-empty."""
    names = blocking_auth_vars(env)
    if not names:
        return
    one = len(names) == 1
    raise AuthEnvError(
        f"{', '.join(names)} {'is' if one else 'are'} set. The claude.ai Slack connector loads "
        "only on a claude.ai subscription login, and "
        f"{'this variable makes' if one else 'these variables make'} Claude Code use another "
        f"credential. Unset {'it' if one else 'them'} in your shell, ~/.zshrc and "
        "services/allocation-agent/.env, then run collect again."
    )
