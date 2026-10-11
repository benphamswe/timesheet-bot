import json
from datetime import UTC, datetime

import pytest

from allocation_agent.activity import cli
from allocation_agent.activity.slack_mcp import CollectError
from support import REPORT_CHANNEL, SECRET_TEXT, USER, passthrough

ENV = {"SLACK_USER_ID": USER, "TIMEZONE": "Asia/Ho_Chi_Minh", "REPORT_CHANNEL": REPORT_CHANNEL}


def _clock():
    return datetime(2026, 10, 10, 3, 0, tzinfo=UTC)


def _fetch_returning(hits, calls=None):
    async def fetch(week, user, scope):
        if calls is not None:
            calls.append((str(week), user, scope.channel, scope.include_self_dm))
        return hits

    return fetch


def _run(tmp_path, argv, fetch, env=ENV, **kw):
    cfg = tmp_path / "missing-config.yaml"
    return cli.main([*argv, "--config", str(cfg)], fetch=fetch, clock=_clock, env=env, **kw)


def test_dry_run_prints_counts_and_never_text(tmp_path, raw_hits, capsys):
    calls = []
    code = _run(
        tmp_path,
        [
            "--week",
            "2026-W41",
            "--dry-run",
            "--include-self-dm",
            "--out-dir",
            str(tmp_path / "out"),
        ],
        _fetch_returning(raw_hits, calls),
        normalize=passthrough,
    )
    out = capsys.readouterr().out

    assert code == 0
    assert calls == [("2026-W41", USER, REPORT_CHANNEL, True)]
    assert f"sources: #{REPORT_CHANNEL}, self-dm" in out
    assert "messages: 3" in out
    assert "2026-10-05 Mon  2" in out
    assert "2026-10-07 Wed  1" in out
    assert "2026-10-09 Fri  0" in out
    assert f"{REPORT_CHANNEL}  2" in out and "self-dm  1" in out
    assert "after:2026-10-04 before:2026-10-10" in out
    assert SECRET_TEXT not in out
    assert "GCM-12" not in out
    assert "https://" not in out
    assert not (tmp_path / "out").exists()


def test_writes_file(tmp_path, raw_hits, capsys):
    out_dir = tmp_path / "out"
    code = _run(
        tmp_path,
        ["--week", "2026-W41", "--out-dir", str(out_dir)],
        _fetch_returning(raw_hits),
        normalize=passthrough,
    )
    assert code == 0
    body = json.loads((out_dir / "2026-W41.json").read_text(encoding="utf-8"))
    assert body["schema_version"] == 1  # report.parser=manual (the default) keeps contract v1
    assert "report_lines" not in body
    assert len(body["messages"]) == 3
    assert SECRET_TEXT not in capsys.readouterr().out


def test_bad_week_exits_2(tmp_path, capsys):
    assert _run(tmp_path, ["--week", "2027-W53"], _fetch_returning([])) == cli.EXIT_USAGE
    assert "does not exist" in capsys.readouterr().err


def test_collect_error_exits_1(tmp_path, capsys):
    async def fetch(week, user, scope):
        raise CollectError("MCP server \"claude.ai Slack\" is 'needs-auth', not connected.")

    assert _run(tmp_path, ["--week", "2026-W41"], fetch, normalize=passthrough) == cli.EXIT_FAILED
    assert "needs-auth" in capsys.readouterr().err


def test_no_report_source_exits_2_before_fetching(tmp_path, capsys):
    calls = []
    env = {"SLACK_USER_ID": USER}
    code = _run(tmp_path, ["--week", "2026-W41"], _fetch_returning([], calls), env=env)
    assert code == cli.EXIT_USAGE
    assert calls == []
    assert "no report source" in capsys.readouterr().err


def test_report_flags_override_env(tmp_path):
    calls = []
    argv = ["--week", "2026-W41", "--dry-run", "--report-channel", "#other", "--no-include-self-dm"]
    code = _run(tmp_path, argv, _fetch_returning([], calls), normalize=passthrough)
    assert code == 0
    assert calls == [("2026-W41", USER, "other", False)]


def test_self_dm_only(tmp_path):
    calls = []
    env = {"SLACK_USER_ID": USER}
    argv = ["--week", "2026-W41", "--dry-run", "--include-self-dm"]
    code = _run(tmp_path, argv, _fetch_returning([], calls), env=env, normalize=passthrough)
    assert code == 0
    assert calls == [("2026-W41", USER, None, True)]


@pytest.mark.parametrize(
    "name", ["ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN"]
)
def test_connector_blocking_credential_exits_2_before_fetching(tmp_path, capsys, name):
    calls = []
    fake = "fake-value-not-a-credential-123"
    env = {**ENV, name: fake}
    code = _run(tmp_path, ["--week", "2026-W41", "--dry-run"], _fetch_returning([], calls), env=env)
    captured = capsys.readouterr()
    assert code == cli.EXIT_USAGE
    assert calls == []
    assert f"collect: {name} is set." in captured.err
    assert fake not in captured.err + captured.out


def test_deepseek_key_does_not_block_collect(tmp_path):
    calls = []
    env = {**ENV, "DEEPSEEK_API_KEY": "fake-value-not-a-credential-123"}
    argv = ["--week", "2026-W41", "--dry-run"]
    code = _run(tmp_path, argv, _fetch_returning([], calls), env=env, normalize=passthrough)
    assert code == 0
    assert len(calls) == 1


def test_auth_check_reads_process_env_when_env_is_not_injected(tmp_path, capsys, monkeypatch):
    monkeypatch.setenv("ANTHROPIC_API_KEY", "fake-value-not-a-credential-123")
    code = cli.main(["--week", "2026-W41", "--config", str(tmp_path / "x.yaml")], fetch=None)
    assert code == cli.EXIT_USAGE
    assert "ANTHROPIC_API_KEY is set." in capsys.readouterr().err
