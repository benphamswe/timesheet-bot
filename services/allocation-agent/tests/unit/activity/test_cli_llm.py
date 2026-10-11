"""`collect` with report.parser=llm: FakeLLMClient through the registry; no real API.

`report.parser` comes from config.yaml only, so each run writes a config file.
"""

import json
from datetime import UTC, datetime

import jsonschema
import pytest

from allocation_agent.activity import cli
from allocation_agent.llm import LLMAuthError, LLMResponseError
from llm_fakes import FAKE_MODEL, FAKE_PROVIDER, FakeLLMClient, FakeRegistration
from support import REPORT_CHANNEL, SCHEMA_V2_PATH, SECRET_TEXT, USER, passthrough

ENV = {"SLACK_USER_ID": USER, "TIMEZONE": "Asia/Ho_Chi_Minh", "REPORT_CHANNEL": REPORT_CHANNEL}
NOTES = "PRIVATE-NOTES"
LLM = ["--llm-provider", FAKE_PROVIDER]


def _clock():
    return datetime(2026, 10, 10, 3, 0, tzinfo=UTC)


def _fetch(hits, calls):
    async def fetch(week, user, scope):
        calls.append(str(week))
        return hits

    return fetch


def _run(tmp_path, argv, hits, calls, env=ENV, parser="llm"):
    cfg = tmp_path / "config.yaml"
    cfg.write_text(f"report:\n  parser: {parser}\n", encoding="utf-8")
    return cli.main(
        [*argv, "--config", str(cfg)],
        fetch=_fetch(hits, calls),
        normalize=passthrough,
        clock=_clock,
        env=env,
    )


def _answer(user):
    """One line per message, dated on the posted day."""
    posted = json.loads(user.split("\n", 1)[1])["posted_date"].split()[0]
    return {"lines": [{"date": posted, "project": "proj-x", "minutes": 120, "notes": NOTES}]}


def test_writes_v2_with_report_lines(tmp_path, raw_hits, capsys):
    out_dir = tmp_path / "out"
    calls: list[str] = []
    with FakeRegistration(FakeLLMClient(_answer)) as reg:
        code = _run(
            tmp_path, ["--week", "2026-W41", "--out-dir", str(out_dir), *LLM], raw_hits, calls
        )

    assert code == cli.EXIT_OK
    assert len(reg.client.calls) == 3  # one per message
    assert reg.settings[0].provider == FAKE_PROVIDER
    body = json.loads((out_dir / "2026-W41.json").read_text(encoding="utf-8"))
    schema = json.loads(SCHEMA_V2_PATH.read_text(encoding="utf-8"))
    jsonschema.Draft202012Validator(
        schema, format_checker=jsonschema.Draft202012Validator.FORMAT_CHECKER
    ).validate(body)
    assert body["schema_version"] == 2
    assert body["parser"] == {"kind": "llm", "provider": FAKE_PROVIDER, "model": FAKE_MODEL}
    assert [line["date"] for line in body["report_lines"]] == [
        "2026-10-05", "2026-10-05", "2026-10-07",
    ]  # fmt: skip
    assert {line["source_ts"] for line in body["report_lines"]} == {
        m["ts"] for m in body["messages"]
    }
    out = capsys.readouterr().out
    assert "3 report lines, schema v2" in out
    assert SECRET_TEXT not in out and NOTES not in out


def test_dry_run_calls_the_llm_and_prints_minutes_only(tmp_path, raw_hits, capsys):
    out_dir = tmp_path / "out"
    calls: list[str] = []
    argv = ["--week", "2026-W41", "--dry-run", "--out-dir", str(out_dir), *LLM]
    with FakeRegistration(FakeLLMClient(_answer)) as reg:
        code = _run(tmp_path, argv, raw_hits, calls)
    captured = capsys.readouterr()

    assert code == cli.EXIT_OK
    assert len(reg.client.calls) == 3
    assert "messages: 3" in captured.out
    assert "2026-10-05 Mon  total  240  (4h00m)" in captured.out
    assert "    proj-x  240" in captured.out
    assert "week total: 360 (6h00m)" in captured.out
    for leaked in (SECRET_TEXT, "GCM-12", NOTES, "https://"):
        assert leaked not in captured.out
        assert leaked not in captured.err
    assert not out_dir.exists()


@pytest.mark.parametrize(
    "answer",
    [
        LLMResponseError("fake (report_lines): the content is not valid JSON"),
        {"lines": [{"date": "2026-10-05", "project": SECRET_TEXT, "minutes": 20, "notes": NOTES}]},
        {"lines": [{"date": "2026-10-12", "project": "proj-x", "minutes": 60, "notes": NOTES}]},
    ],
)
def test_any_llm_failure_exits_4_and_writes_nothing(tmp_path, raw_hits, capsys, answer):
    out_dir = tmp_path / "out"
    calls: list[str] = []
    with FakeRegistration(FakeLLMClient(answer)):
        code = _run(
            tmp_path, ["--week", "2026-W41", "--out-dir", str(out_dir), *LLM], raw_hits, calls
        )
    captured = capsys.readouterr()

    assert code == cli.EXIT_LLM_FAILED == 4
    assert not out_dir.exists()
    assert "report extraction stopped: message ts=" in captured.err
    assert "channel=C0123" in captured.err
    for leaked in (SECRET_TEXT, NOTES):
        assert leaked not in captured.err + captured.out


def test_missing_deepseek_key_exits_4_before_any_collection(
    tmp_path, raw_hits, capsys, monkeypatch
):
    monkeypatch.delenv("DEEPSEEK_API_KEY", raising=False)
    calls: list[str] = []
    argv = ["--week", "2026-W41", "--out-dir", str(tmp_path / "out")]
    code = _run(tmp_path, argv, raw_hits, calls)
    assert code == cli.EXIT_LLM_FAILED
    assert calls == []  # no Claude or Slack usage
    assert capsys.readouterr().err.strip() == "collect: llm: DEEPSEEK_API_KEY is not set"


def test_provider_auth_error_from_factory_exits_4(tmp_path, raw_hits, capsys):
    calls: list[str] = []

    class Reg(FakeRegistration):
        def factory(self, settings):
            raise LLMAuthError("FAKE_API_KEY is not set")

    with Reg(FakeLLMClient()):
        code = _run(tmp_path, ["--week", "2026-W41", *LLM], raw_hits, calls)
    assert code == cli.EXIT_LLM_FAILED
    assert calls == []
    assert "FAKE_API_KEY is not set" in capsys.readouterr().err


def test_unknown_provider_exits_2(tmp_path, raw_hits, capsys):
    calls: list[str] = []
    argv = ["--week", "2026-W41", "--llm-provider", "nope"]
    assert _run(tmp_path, argv, raw_hits, calls) == cli.EXIT_USAGE
    assert calls == []
    assert "registered providers: deepseek" in capsys.readouterr().err


def test_llm_settings_from_env(tmp_path, raw_hits):
    calls: list[str] = []
    env = {**ENV, "LLM_PROVIDER": FAKE_PROVIDER, "LLM_MODEL": "m2"}
    with FakeRegistration(FakeLLMClient()) as reg:
        code = _run(tmp_path, ["--week", "2026-W41", "--dry-run"], raw_hits, calls, env=env)
    assert code == cli.EXIT_OK
    assert reg.settings[0].model == "m2"
    assert len(reg.client.calls) == 3


def test_report_parser_env_is_ignored(tmp_path, raw_hits):
    """config.yaml says manual, so v1 is written whatever REPORT_PARSER says (Go reads yaml)."""
    calls: list[str] = []
    env = {**ENV, "REPORT_PARSER": "llm", "LLM_PROVIDER": FAKE_PROVIDER}
    out_dir = tmp_path / "out"
    with FakeRegistration(FakeLLMClient()) as reg:
        argv = ["--week", "2026-W41", "--out-dir", str(out_dir)]
        code = _run(tmp_path, argv, raw_hits, calls, env=env, parser="manual")
    assert code == cli.EXIT_OK
    assert reg.client.calls == []
    assert json.loads((out_dir / "2026-W41.json").read_text())["schema_version"] == 1


def test_there_is_no_parser_flag(tmp_path, raw_hits, capsys):
    with pytest.raises(SystemExit) as ei:
        _run(tmp_path, ["--week", "2026-W41", "--parser", "llm"], raw_hits, [])
    assert ei.value.code == 2
    assert "unrecognized arguments: --parser" in capsys.readouterr().err
