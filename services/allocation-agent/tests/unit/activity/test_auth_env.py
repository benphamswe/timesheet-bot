import pytest

from allocation_agent.activity.auth_env import (
    CONNECTOR_BLOCKING_VARS,
    AuthEnvError,
    blocking_auth_vars,
    check_claude_ai_login_env,
)

FAKE = "fake-value-not-a-credential-123"


def test_clean_env_passes():
    check_claude_ai_login_env({})
    check_claude_ai_login_env({"SLACK_USER_ID": "U0123ABCD", "TIMEZONE": "UTC"})


def test_empty_values_pass():
    check_claude_ai_login_env(dict.fromkeys(CONNECTOR_BLOCKING_VARS, ""))


def test_deepseek_key_is_not_blocked():
    check_claude_ai_login_env({"DEEPSEEK_API_KEY": FAKE})


@pytest.mark.parametrize("name", CONNECTOR_BLOCKING_VARS)
def test_each_blocking_var_fails_with_its_name_and_never_the_value(name):
    with pytest.raises(AuthEnvError) as e:
        check_claude_ai_login_env({name: FAKE})
    msg = str(e.value)
    assert msg.startswith(f"{name} is set.")
    assert FAKE not in msg
    assert "Unset it" in msg


def test_all_set_names_are_listed_in_order():
    env = {
        "CLAUDE_CODE_OAUTH_TOKEN": FAKE,
        "ANTHROPIC_API_KEY": FAKE,
        "DEEPSEEK_API_KEY": FAKE,
    }
    assert blocking_auth_vars(env) == ["ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN"]
    with pytest.raises(AuthEnvError) as e:
        check_claude_ai_login_env(env)
    msg = str(e.value)
    assert msg.startswith("ANTHROPIC_API_KEY, CLAUDE_CODE_OAUTH_TOKEN are set.")
    assert "DEEPSEEK_API_KEY" not in msg
    assert FAKE not in msg


def test_blocking_vars_are_exactly_the_three_credential_names():
    assert CONNECTOR_BLOCKING_VARS == (
        "ANTHROPIC_API_KEY",
        "ANTHROPIC_AUTH_TOKEN",
        "CLAUDE_CODE_OAUTH_TOKEN",
    )
