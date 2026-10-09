import json
from datetime import UTC, datetime

from allocation_agent.activity import cli
from allocation_agent.activity.slack_mcp import CollectError
from support import SECRET_TEXT, USER, passthrough

ENV = {"SLACK_USER_ID": USER, "TIMEZONE": "Asia/Ho_Chi_Minh"}


def _clock():
    return datetime(2026, 10, 10, 3, 0, tzinfo=UTC)


def _fetch_returning(hits, calls=None):
    async def fetch(week, user):
        if calls is not None:
            calls.append((str(week), user))
        return hits

    return fetch


def _run(tmp_path, argv, fetch, **kw):
    cfg = tmp_path / "missing-config.yaml"
    return cli.main([*argv, "--config", str(cfg)], fetch=fetch, clock=_clock, env=ENV, **kw)


def test_dry_run_prints_counts_and_never_text(tmp_path, raw_hits, capsys):
    calls = []
    code = _run(
        tmp_path,
        ["--week", "2026-W41", "--dry-run", "--out-dir", str(tmp_path / "out")],
        _fetch_returning(raw_hits, calls),
        normalize=passthrough,
    )
    out = capsys.readouterr().out

    assert code == 0
    assert calls == [("2026-W41", USER)]
    assert "messages: 3" in out
    assert "2026-10-05 Mon  2" in out
    assert "2026-10-07 Wed  1" in out
    assert "2026-10-09 Fri  0" in out
    assert "proj-x  2" in out and "proj-y  1" in out
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
    assert len(body["messages"]) == 3
    assert SECRET_TEXT not in capsys.readouterr().out


def test_unimplemented_normalizer_exits_3_and_writes_nothing(tmp_path, raw_hits, capsys):
    out_dir = tmp_path / "out"
    argv = ["--week", "2026-W41", "--out-dir", str(out_dir)]
    code = _run(tmp_path, argv, _fetch_returning(raw_hits))
    assert code == cli.EXIT_NOT_IMPLEMENTED
    assert "TODO(owner)" in capsys.readouterr().err
    assert not out_dir.exists()


def test_bad_week_exits_2(tmp_path, capsys):
    assert _run(tmp_path, ["--week", "2027-W53"], _fetch_returning([])) == cli.EXIT_USAGE
    assert "does not exist" in capsys.readouterr().err


def test_collect_error_exits_1(tmp_path, capsys):
    async def fetch(week, user):
        raise CollectError("MCP server \"claude.ai Slack\" is 'needs-auth', not connected.")

    assert _run(tmp_path, ["--week", "2026-W41"], fetch, normalize=passthrough) == cli.EXIT_FAILED
    assert "needs-auth" in capsys.readouterr().err
