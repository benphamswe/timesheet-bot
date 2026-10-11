from datetime import date

import pytest
from pydantic import ValidationError

from allocation_agent.report.models import ParserInfo, ReportLine
from allocation_agent.report.summary import render_report_summary

BASE = {
    "date": "2026-10-05",
    "project": " proj-x ",
    "minutes": 60,
    "notes": "PRIVATE-NOTES",
    "source_ts": "1791135000.000100",
    "source_channel_id": "C0123",
}


def _line(**kw):
    return ReportLine.model_validate({**BASE, **kw})


def test_project_is_trimmed():
    assert _line().project == "proj-x"


@pytest.mark.parametrize(
    "bad",
    [
        {"minutes": 0},
        {"minutes": 50},
        {"project": "  "},
        {"source_ts": "abc"},
        {"source_channel_id": ""},
        {"extra": 1},
    ],
)
def test_report_line_rules(bad):
    with pytest.raises(ValidationError):
        _line(**bad)


def test_week_check_runs_with_context_only(week41):
    ReportLine.model_validate({**BASE, "date": "2026-10-10"})  # no week: not checked
    with pytest.raises(ValidationError, match="not a Monday-Friday of 2026-W41"):
        ReportLine.model_validate({**BASE, "date": "2026-10-10"}, context={"week": week41})


def test_parser_info_kind():
    with pytest.raises(ValidationError):
        ParserInfo(kind="regex")  # type: ignore[arg-type]


def test_dry_run_table_has_minutes_and_never_notes(week41):
    lines = [
        _line(minutes=240),
        _line(project="proj-y", minutes=90),
        _line(minutes=60),
        _line(date="2026-10-07", minutes=15),
    ]
    out = render_report_summary(lines, week41, ParserInfo(kind="llm", provider="p", model="m"))

    assert "report lines: 4  parser: llm p m" in out
    assert "2026-10-05 Mon  total  390  (6h30m)" in out
    assert "    proj-x  300" in out and "    proj-y  90" in out
    assert "2026-10-06 Tue  total    0  (0h00m)" in out
    assert "2026-10-07 Wed  total   15  (0h15m)" in out
    assert "week total: 405 (6h45m)" in out
    assert "PRIVATE-NOTES" not in out
    assert out.index("proj-x  300") < out.index("proj-y  90")  # biggest first
    assert date(2026, 10, 5).isoformat() in out
