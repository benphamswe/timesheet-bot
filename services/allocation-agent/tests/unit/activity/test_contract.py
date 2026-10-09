import json
from datetime import UTC, datetime

import jsonschema
import pytest
from pydantic import ValidationError

from allocation_agent.activity.collector import build_activity_week
from allocation_agent.activity.contract import ActivityWeek, Message, ts_to_datetime
from support import REPO_ROOT, TZ, USER, passthrough


def _doc(raw_hits, week41, fixed_now) -> ActivityWeek:
    return build_activity_week(
        raw_hits, week=week41, slack_user_id=USER, tz=TZ, now=fixed_now, normalize=passthrough
    )


def test_serialization_matches_contract(raw_hits, week41, fixed_now):
    body = json.loads(_doc(raw_hits, week41, fixed_now).to_json())

    assert list(body) == [
        "schema_version", "week", "slack_user_id", "generated_at", "source", "messages",
    ]  # fmt: skip
    assert body["schema_version"] == 1
    assert body["week"] == "2026-W41"
    assert body["slack_user_id"] == USER
    assert body["source"] == "slack-mcp"
    assert body["generated_at"] == "2026-10-10T10:00:00+07:00"  # tz offset, whole seconds

    first = body["messages"][0]
    assert list(first) == [
        "ts", "channel_id", "channel_name", "posted_at", "text", "permalink", "thread_ts",
    ]  # fmt: skip
    assert first["posted_at"] == "2026-10-05T09:00:00+07:00"
    assert first["thread_ts"] is None
    assert body["messages"][1]["thread_ts"] == first["ts"]


def test_messages_sorted_by_ts(raw_hits, week41, fixed_now):
    msgs = _doc(raw_hits, week41, fixed_now).messages
    assert [m.posted_at.day for m in msgs] == [5, 5, 7]


def test_output_validates_against_json_schema(raw_hits, week41, fixed_now, schema):
    jsonschema.Draft202012Validator.check_schema(schema)
    body = json.loads(_doc(raw_hits, week41, fixed_now).to_json())
    jsonschema.Draft202012Validator(
        schema, format_checker=jsonschema.Draft202012Validator.FORMAT_CHECKER
    ).validate(body)


def test_empty_week_validates(week41, fixed_now, schema):
    body = json.loads(_doc([], week41, fixed_now).to_json())
    assert body["messages"] == []
    jsonschema.validate(body, schema)


@pytest.mark.parametrize(
    "mutate",
    [
        lambda b: b.update(schema_version=2),
        lambda b: b.update(source="slack-api"),
        lambda b: b.update(week="2026-41"),
        lambda b: b["messages"][0].pop("permalink"),
        lambda b: b["messages"][0].update(posted_at="2026-10-05T09:00:00"),  # no offset
        lambda b: b["messages"][0].update(ts="abc"),
    ],
)
def test_schema_rejects_contract_violations(raw_hits, week41, fixed_now, schema, mutate):
    body = json.loads(_doc(raw_hits, week41, fixed_now).to_json())
    mutate(body)
    with pytest.raises(jsonschema.ValidationError):
        jsonschema.validate(body, schema)
    with pytest.raises(ValidationError):
        ActivityWeek.model_validate(body)


def test_pydantic_round_trip(raw_hits, week41, fixed_now):
    doc = _doc(raw_hits, week41, fixed_now)
    assert ActivityWeek.model_validate_json(doc.to_json()) == doc


def test_message_rejects_naive_posted_at():
    with pytest.raises(ValidationError):
        Message(
            ts="1.0", channel_id="C1", channel_name="", posted_at=datetime(2026, 10, 5),
            text="", permalink="", thread_ts=None,
        )  # fmt: skip


def test_ts_to_datetime_truncates_to_seconds():
    got = ts_to_datetime("1760000000.999999", UTC)
    assert got == datetime.fromtimestamp(1760000000, UTC)


def test_unknown_fields_are_allowed_by_schema_but_not_written(raw_hits, week41, fixed_now, schema):
    """Readers ignore unknown fields (the Go reader does too); our writer never emits them."""
    body = json.loads(_doc(raw_hits, week41, fixed_now).to_json())
    body["future_field"] = {"x": 1}
    body["messages"][0]["reactions"] = []
    jsonschema.validate(body, schema)
    with pytest.raises(ValidationError):
        ActivityWeek.model_validate(body)


GO_FIXTURES = REPO_ROOT / "services/timesheet-core/test/unit/adapters/file/testdata"


@pytest.mark.parametrize(
    ("case", "valid"), [("happy", True), ("bad-posted-at", False), ("schema-v2", False)]
)
def test_go_reader_fixtures_against_schema(schema, case, valid):
    """Cross-checks the Go reader's test files (read-only) against the shared schema."""
    path = GO_FIXTURES / case / "2026-W41.json"
    if not path.is_file():
        pytest.skip(f"Go fixture {case} not present")
    body = json.loads(path.read_text(encoding="utf-8"))
    validator = jsonschema.Draft202012Validator(
        schema, format_checker=jsonschema.Draft202012Validator.FORMAT_CHECKER
    )
    assert validator.is_valid(body) is valid
