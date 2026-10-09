"""SlackMcpFetcher against a fake SDK client. Nothing here calls Claude or Slack."""

import asyncio

import pytest
from claude_agent_sdk import ClaudeAgentOptions
from claude_agent_sdk.types import AssistantMessage, ResultMessage, TextBlock

from allocation_agent.activity import slack_mcp
from allocation_agent.activity.slack_mcp import CollectError, SlackMcpFetcher
from allocation_agent.activity.week import IsoWeek

# Tool names are assumptions: the connector was 'needs-auth' during the smoke test, so its
# real tool list could not be read (see ADR-0011).
SLACK_TOOLS = [
    {"name": "slack_search_public_and_private", "annotations": {"readOnly": True}},
    {"name": "slack_search_users", "annotations": {"readOnly": True}},
    {"name": "slack_read_thread", "annotations": {"readOnly": True}},
    {"name": "slack_send_message", "annotations": {"readOnly": False}},
    {"name": "slack_schedule_message"},
    {"name": "slack_create_canvas"},
    {"name": "slack_add_reaction"},
    {"name": "slack_get_file_upload_url"},
    {"name": "slack_read_something_odd", "annotations": {"destructive": True}},
]


def _server(status="connected", tools=SLACK_TOOLS):
    return {"name": "claude.ai Slack", "status": status, "scope": "claudeai", "tools": tools}


def _result(structured=None, *, is_error=False, subtype="success"):
    return ResultMessage(
        subtype=subtype, duration_ms=1, duration_api_ms=1, is_error=is_error, num_turns=2,
        session_id="s", structured_output=structured,
    )  # fmt: skip


class FakeClient:
    def __init__(self, options, statuses, messages, log):
        self.options, self._statuses, self._messages, self._log = options, statuses, messages, log
        self.prompts = []
        log.append(self)

    async def __aenter__(self):
        return self

    async def __aexit__(self, *exc):
        return False

    async def get_mcp_status(self):
        servers = self._statuses.pop(0) if len(self._statuses) > 1 else self._statuses[0]
        return {"mcpServers": servers}

    async def query(self, prompt):
        self.prompts.append(prompt)

    async def receive_response(self):
        for m in self._messages:
            yield m


def _fetcher(statuses, messages=(), **kw):
    log = []

    def factory(options):
        return FakeClient(options, list(statuses), list(messages), log)

    return SlackMcpFetcher(factory, poll_interval=0, **kw), log


GOOD_OUTPUT = {
    "complete": True,
    "messages": [
        {
            "ts": "1759629600.000100", "channel_id": "C0123", "channel_name": "proj-x",
            "text": "t", "permalink": None, "thread_ts": None, "user_id": "U0123ABCD",
            "is_bot": False, "subtype": None,
        }
    ],
}  # fmt: skip


def test_fetch_happy_path_options_and_prompt():
    fetcher, log = _fetcher(
        [[{"name": "other", "status": "connected"}, _server()]],
        [AssistantMessage(content=[TextBlock(text="working")], model="m"), _result(GOOD_OUTPUT)],
    )
    hits = asyncio.run(fetcher.fetch(IsoWeek.parse("2026-W41"), "U0123ABCD"))

    assert [h.ts for h in hits] == ["1759629600.000100"]
    pre, run = log
    assert pre.prompts == []  # the preflight sends no prompt
    assert isinstance(run.options, ClaudeAgentOptions)

    opts = run.options
    p = "mcp__claude_ai_Slack__"
    assert opts.allowed_tools == [
        p + "slack_read_thread", p + "slack_search_public_and_private", p + "slack_search_users",
    ]  # fmt: skip
    for write in ("send_message", "schedule_message", "create_canvas", "add_reaction",
                  "get_file_upload_url", "read_something_odd"):  # fmt: skip
        assert p + "slack_" + write in opts.disallowed_tools
    assert "Bash" in opts.disallowed_tools and "WebFetch" in opts.disallowed_tools
    assert opts.tools == []
    assert opts.permission_mode == "dontAsk"
    assert opts.setting_sources == []
    assert opts.strict_mcp_config is False
    assert opts.output_format["type"] == "json_schema"
    assert run.prompts == [slack_mcp.build_prompt(IsoWeek.parse("2026-W41"), "U0123ABCD")]
    assert "from:<@U0123ABCD> after:2026-10-04 before:2026-10-10" in run.prompts[0]


def test_preflight_waits_for_pending_then_connected():
    fetcher, log = _fetcher([[_server("pending")], [_server("pending")], [_server()]])
    policy = asyncio.run(fetcher.preflight())
    assert policy.allowed
    assert len(log) == 1


@pytest.mark.parametrize(
    ("servers", "match"),
    [
        ([], "not found"),
        ([_server("needs-auth")], "needs-auth"),
        ([_server("failed")], "failed"),
        ([_server(tools=[{"name": "slack_send_message"}])], "no read/search tool"),
    ],
)
def test_preflight_failures_never_send_a_prompt(servers, match):
    fetcher, log = _fetcher([servers], [_result(GOOD_OUTPUT)], preflight_wait=0)
    with pytest.raises(CollectError, match=match):
        asyncio.run(fetcher.fetch(IsoWeek.parse("2026-W41"), "U0123ABCD"))
    assert len(log) == 1 and log[0].prompts == []


@pytest.mark.parametrize(
    ("result", "match"),
    [
        (None, "without a result"),
        (_result(GOOD_OUTPUT, is_error=True, subtype="error_max_turns"), "error_max_turns"),
        (_result(None, subtype="error_max_structured_output_retries"), "no structured output"),
        (_result({"complete": True, "messages": [{"ts": "bad"}]}), "validation"),
        (_result({**GOOD_OUTPUT, "complete": False}), "every page"),
        (_result("not json"), "not JSON"),
    ],
)
def test_parse_result_errors(result, match):
    with pytest.raises(CollectError, match=match):
        slack_mcp.parse_result(result)


def test_validation_error_does_not_leak_message_text():
    bad = {"complete": True, "messages": [{"ts": "x", "channel_id": "C1", "text": "SECRET"}]}
    with pytest.raises(CollectError) as e:
        slack_mcp.parse_result(_result(bad))
    assert "SECRET" not in str(e.value)


def test_parse_result_accepts_json_string():
    import json

    hits = slack_mcp.parse_result(_result(json.dumps(GOOD_OUTPUT)))
    assert hits[0].channel_id == "C0123"


@pytest.mark.parametrize(
    ("server", "tool", "full"),
    [
        ("claude.ai Slack", "slack_x", "mcp__claude_ai_Slack__slack_x"),
        ("claude.ai Slack", "mcp__claude_ai_Slack__slack_x", "mcp__claude_ai_Slack__slack_x"),
    ],
)
def test_full_tool_name(server, tool, full):
    assert slack_mcp.full_tool_name(server, tool) == full
