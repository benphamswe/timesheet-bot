"""Contract v2: the v2 file validates against the v2 schema; v1 and v2 stay distinct."""

import json
from datetime import date

import jsonschema
import pytest
from pydantic import ValidationError

from allocation_agent.activity.collector import build_activity_week
from allocation_agent.activity.contract import ActivityWeekV2
from allocation_agent.report.models import ParserInfo, ReportLine
from support import REPO_ROOT, SCHEMA_PATH, SCHEMA_V2_PATH, TZ, USER, passthrough

GO_FIXTURES = REPO_ROOT / "services/timesheet-core/test/unit/adapters/file/testdata"
PARSER = ParserInfo(kind="llm", provider="deepseek", model="deepseek-flash")


def _validator(path):
    schema = json.loads(path.read_text(encoding="utf-8"))
    jsonschema.Draft202012Validator.check_schema(schema)
    return jsonschema.Draft202012Validator(
        schema, format_checker=jsonschema.Draft202012Validator.FORMAT_CHECKER
    )


@pytest.fixture
def v1():
    return _validator(SCHEMA_PATH)


@pytest.fixture
def v2():
    return _validator(SCHEMA_V2_PATH)


@pytest.fixture
def doc_v2(raw_hits, week41, fixed_now):
    doc = build_activity_week(
        raw_hits, week=week41, slack_user_id=USER, tz=TZ, now=fixed_now, normalize=passthrough
    )
    first = doc.messages[0]
    lines = [
        ReportLine(
            date=date(2026, 10, 5), project="proj-x", minutes=240, notes="code review",
            source_ts=first.ts, source_channel_id=first.channel_id,
        ),
        ReportLine(
            date=date(2026, 10, 7), project="proj-y", minutes=45, notes="",
            source_ts=first.ts, source_channel_id=first.channel_id,
        ),
    ]  # fmt: skip
    return ActivityWeekV2.from_v1(doc, parser=PARSER, report_lines=lines)


def test_v2_body_shape(doc_v2):
    body = json.loads(doc_v2.to_json())
    assert list(body) == [
        "schema_version", "week", "slack_user_id", "generated_at", "source", "messages",
        "parser", "report_lines",
    ]  # fmt: skip
    assert body["schema_version"] == 2
    assert body["parser"] == {"kind": "llm", "provider": "deepseek", "model": "deepseek-flash"}
    assert body["report_lines"][0] == {
        "date": "2026-10-05",
        "project": "proj-x",
        "minutes": 240,
        "notes": "code review",
        "source_ts": body["messages"][0]["ts"],
        "source_channel_id": "C0123",
    }
    assert len(body["messages"]) == 3


def test_v2_file_validates_against_v2_not_v1(doc_v2, v1, v2):
    body = json.loads(doc_v2.to_json())
    v2.validate(body)
    assert not v1.is_valid(body)


def test_v1_file_is_not_v2(raw_hits, week41, fixed_now, v1, v2):
    doc = build_activity_week(
        raw_hits, week=week41, slack_user_id=USER, tz=TZ, now=fixed_now, normalize=passthrough
    )
    body = json.loads(doc.to_json())
    v1.validate(body)
    assert not v2.is_valid(body)


def test_v2_round_trip(doc_v2):
    assert ActivityWeekV2.model_validate_json(doc_v2.to_json()) == doc_v2


@pytest.mark.parametrize(
    "mutate",
    [
        lambda b: b["report_lines"][0].update(minutes=20),
        lambda b: b["report_lines"][0].update(minutes=0),
        lambda b: b["report_lines"][0].update(minutes="240"),
        lambda b: b["report_lines"][0].update(project="  "),
        lambda b: b["report_lines"][0].update(date="05/10/2026"),
        lambda b: b["report_lines"][0].update(source_ts="x"),
        lambda b: b["report_lines"][0].pop("notes"),
        lambda b: b.update(report_lines=None),
        lambda b: b["parser"].update(kind="regex"),
        lambda b: b["parser"].pop("model"),
        lambda b: b.update(schema_version=1),
    ],
)
def test_v2_schema_and_model_reject_violations(doc_v2, v2, mutate):
    body = json.loads(doc_v2.to_json())
    mutate(body)
    assert not v2.is_valid(body)
    with pytest.raises(ValidationError):
        ActivityWeekV2.model_validate(body)


def test_v2_report_lines_and_parser_are_optional_in_the_schema(doc_v2, v2):
    body = json.loads(doc_v2.to_json())
    del body["report_lines"], body["parser"]
    assert v2.is_valid(body)


# The Go reader's fixtures (read-only). The JSON Schema cannot know which dates are in the
# week, so v2-outside-week and v2-weekend are schema-valid; the collector (ReportLine with the
# week context) and the Go reader (validLine) both reject them.
@pytest.mark.parametrize(
    ("case", "valid"),
    [
        ("v2", True),
        ("v2-empty-lines", True),
        ("v2-outside-week", True),
        ("v2-weekend", True),
        ("v2-bad-date", False),
        ("v2-bad-minutes", False),
        ("v2-zero-minutes", False),
        ("v2-empty-project", False),
        ("v2-no-lines", True),  # report_lines absent: optional; Go's ReportLines() rejects it
        ("schema-v3", False),
        ("happy", False),  # a v1 file
    ],
)
def test_go_reader_fixtures_against_v2_schema(v2, case, valid):
    path = GO_FIXTURES / case / "2026-W41.json"
    assert path.is_file(), f"Go fixture {case} is missing"
    body = json.loads(path.read_text(encoding="utf-8"))
    assert v2.is_valid(body) is valid
