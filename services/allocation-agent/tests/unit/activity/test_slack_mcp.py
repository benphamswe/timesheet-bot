"""SlackMcpFetcher against a fake SDK client. Nothing here calls Claude or Slack."""

import asyncio
import json

import pytest
from claude_agent_sdk import ClaudeAgentOptions
from claude_agent_sdk.types import AssistantMessage, ResultMessage, TextBlock

from allocation_agent.activity import slack_mcp
from allocation_agent.activity.slack_mcp import CollectError, SlackMcpFetcher
from allocation_agent.activity.sources import ReportScope
from allocation_agent.activity.week import IsoWeek

W41 = IsoWeek.parse("2026-W41")
BOTH = ReportScope.build("timesheet-reports", True)
DM_ONLY = ReportScope.build(None, True)

# Tool names are assumptions: the connector was 'needs-auth' during the smoke test, so its
# real tool list could not be read (see ADR-0011).
SLACK_TOOLS = [
    {"name": "slack_search_public_and_private", "annotations": {"readOnly": True}},
    {"name": "slack_search_users", "annotations": {"readOnly": True}},
    {"name": "slack_read_thread", "annotations": {"readOnly": True}},
    {"name": "slack_read_channel", "annotations": {"readOnly": True}},
    {"name": "slack_send_message", "annotations": {"readOnly": False}},
    {"name": "slack_schedule_message"},
    {"name": "slack_create_canvas"},
    {"name": "slack_add_reaction"},
    {"name": "slack_get_file_upload_url"},
    {"name": "slack_search_odd", "annotations": {"destructive": True}},
]
P = "mcp__claude_ai_Slack__"


def _server(status="connected", tools=SLACK_TOOLS):
    return {"name": "claude.ai Slack", "status": status, "scope": "claudeai", "tools": tools}


def _result(structured=None, *, is_error=False, subtype="success"):
    return ResultMessage(
        subtype=subtype, duration_ms=1, duration_api_ms=1, is_error=is_error, num_turns=2,
        session_id="s", structured_output=structured,
    )  # fmt: skip


def _msg(ts, channel_id, **kw):
    base = {
        "ts": ts, "channel_id": channel_id, "channel_name": None, "text": "t",
        "permalink": None, "thread_ts": None, "edited_ts": None, "user_id": "U0123ABCD",
        "is_bot": False, "subtype": None,
    }  # fmt: skip
    base.update(kw)
    return base


def _section(*msgs, complete=True):
    return {"complete": complete, "messages": list(msgs)}


GOOD_OUTPUT = {
    "channel": _section(_msg("1759629600.000100", "C0123", channel_name="timesheet-reports")),
    "self_dm": _section(
        _msg("1759629600.000100", "D0SELF01", channel_name="U0123ABCD"),
        _msg("1759629600.000100", "D0SELF01"),  # duplicate (channel_id, ts)
    ),
}


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


def test_fetch_happy_path_options_prompt_and_merge():
    fetcher, log = _fetcher(
        [[{"name": "other", "status": "connected"}, _server()]],
        [AssistantMessage(content=[TextBlock(text="working")], model="m"), _result(GOOD_OUTPUT)],
    )
    hits = asyncio.run(fetcher.fetch(W41, "U0123ABCD", BOTH))

    assert [(h.channel_id, h.channel_name, h.source) for h in hits] == [
        ("C0123", "timesheet-reports", "channel"),
        ("D0SELF01", "self-dm", "self_dm"),
    ]
    pre, run = log
    assert pre.prompts == []  # the preflight sends no prompt
    opts = run.options
    assert isinstance(opts, ClaudeAgentOptions)
    assert opts.allowed_tools == [P + "slack_search_public_and_private", P + "slack_search_users"]
    denied_tools = (
        "read_thread", "read_channel", "send_message", "schedule_message",
        "create_canvas", "add_reaction", "get_file_upload_url", "search_odd",
    )  # fmt: skip
    for denied in denied_tools:
        assert P + "slack_" + denied in opts.disallowed_tools
    assert "Bash" in opts.disallowed_tools and "WebFetch" in opts.disallowed_tools
    assert opts.tools == []
    assert opts.permission_mode == "dontAsk"
    assert opts.setting_sources == []
    assert opts.strict_mcp_config is False
    assert opts.output_format["type"] == "json_schema"
    assert set(opts.output_format["schema"]["required"]) == {"channel", "self_dm"}

    [prompt] = run.prompts
    assert prompt == slack_mcp.build_prompt(W41, "U0123ABCD", BOTH)
    assert (
        "channel: in:#timesheet-reports from:<@U0123ABCD> after:2026-10-04 before:2026-10-10"
        in prompt
    )
    assert "self_dm: in:<@U0123ABCD> from:<@U0123ABCD> after:2026-10-04 before:2026-10-10" in prompt


def test_prompt_skips_unrequested_source():
    prompt = slack_mcp.build_prompt(W41, "U0123ABCD", DM_ONLY)
    assert "channel: (skip; set null)" in prompt
    assert "in:#" not in prompt
    assert "self_dm: in:<@U0123ABCD>" in prompt


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
        ([_server(tools=[{"name": "slack_read_channel"}])], "no search tool"),
    ],
)
def test_preflight_failures_never_send_a_prompt(servers, match):
    fetcher, log = _fetcher([servers], [_result(GOOD_OUTPUT)], preflight_wait=0)
    with pytest.raises(CollectError, match=match):
        asyncio.run(fetcher.fetch(W41, "U0123ABCD", BOTH))
    assert len(log) == 1 and log[0].prompts == []


@pytest.mark.parametrize(
    ("result", "match"),
    [
        (None, "without a result"),
        (_result(GOOD_OUTPUT, is_error=True, subtype="error_max_turns"), "error_max_turns"),
        (_result(None, subtype="error_max_structured_output_retries"), "no structured output"),
        (_result({"channel": {"complete": True, "messages": [{"ts": "bad"}]}}), "validation"),
        (_result({**GOOD_OUTPUT, "self_dm": None}), "no self_dm section"),
        (_result({**GOOD_OUTPUT, "channel": _section(complete=False)}), "every page of channel"),
        (_result({**GOOD_OUTPUT, "self_dm": _section(_msg("1.1", "C0999"))}), "outside a DM"),
        (_result("not json"), "not JSON"),
    ],
)
def test_parse_result_errors(result, match):
    with pytest.raises(CollectError, match=match):
        slack_mcp.parse_result(result, BOTH)


def test_parse_result_ignores_section_outside_scope():
    out = {"channel": None, "self_dm": _section(_msg("1759629600.000100", "D0SELF01"))}
    hits = slack_mcp.parse_result(_result(out), DM_ONLY)
    assert [h.channel_name for h in hits] == ["self-dm"]


def test_validation_error_does_not_leak_message_text():
    bad = {"channel": _section({"ts": "x", "channel_id": "C1", "text": "SECRET"}), "self_dm": None}
    with pytest.raises(CollectError) as e:
        slack_mcp.parse_result(_result(bad), BOTH)
    assert "SECRET" not in str(e.value)


def test_parse_result_accepts_json_string():
    hits = slack_mcp.parse_result(_result(json.dumps(GOOD_OUTPUT)), BOTH)
    assert [h.channel_id for h in hits] == ["C0123", "D0SELF01"]


@pytest.mark.parametrize(
    ("server", "tool", "full"),
    [
        ("claude.ai Slack", "slack_x", "mcp__claude_ai_Slack__slack_x"),
        ("claude.ai Slack", "mcp__claude_ai_Slack__slack_x", "mcp__claude_ai_Slack__slack_x"),
    ],
)
def test_full_tool_name(server, tool, full):
    assert slack_mcp.full_tool_name(server, tool) == full
