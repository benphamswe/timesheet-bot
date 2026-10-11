import json
from datetime import date, datetime

import pytest

from allocation_agent.activity.contract import message_from_hit
from allocation_agent.llm import LLMRateLimitError, LLMResponseError
from allocation_agent.report.extractor import ReportExtractionError, ReportExtractor
from allocation_agent.report.models import ReportLine
from allocation_agent.report.prompt import (
    FEW_SHOT_EXAMPLES,
    SCHEMA_NAME,
    SYSTEM_PROMPT,
    system_prompt,
)
from llm_fakes import FakeLLMClient
from support import SECRET_TEXT, TZ, dm_hit, hit

NOTES = "PRIVATE-NOTES"


@pytest.fixture
def messages():
    return [
        message_from_hit(hit(datetime(2026, 10, 5, 18, 0, tzinfo=TZ)), TZ),
        message_from_hit(dm_hit(datetime(2026, 10, 7, 18, 0, tzinfo=TZ)), TZ),
    ]


def _line(**kw):
    base = {"date": "2026-10-05", "project": "proj-x", "minutes": 240, "notes": NOTES}
    base.update(kw)
    return base


def test_one_call_per_message_and_lines_carry_their_source(messages, week41):
    fake = FakeLLMClient(
        {"lines": [_line(), _line(project="  proj-y ", minutes=90, notes="")]},
        {"lines": [_line(date="2026-10-07", minutes=15)]},
    )
    lines = ReportExtractor(fake).extract(messages, week41, TZ)

    assert len(fake.calls) == 2
    assert lines == [
        ReportLine(
            date=date(2026, 10, 5), project="proj-x", minutes=240, notes=NOTES,
            source_ts=messages[0].ts, source_channel_id="C0123",
        ),
        ReportLine(
            date=date(2026, 10, 5), project="proj-y", minutes=90, notes="",
            source_ts=messages[0].ts, source_channel_id="C0123",
        ),
        ReportLine(
            date=date(2026, 10, 7), project="proj-x", minutes=15, notes=NOTES,
            source_ts=messages[1].ts, source_channel_id="D0SELF01",
        ),
    ]  # fmt: skip


def test_prompt_inputs(messages, week41):
    fake = FakeLLMClient()
    ReportExtractor(fake).extract(messages[:1], week41, TZ)
    ((system, user, schema_name),) = fake.calls

    assert schema_name == SCHEMA_NAME
    assert "json" in system  # DeepSeek JSON mode needs the word in the prompt
    assert system == SYSTEM_PROMPT  # no few-shot examples yet
    assert "json" in user
    payload = json.loads(user.split("\n", 1)[1])
    assert payload == {
        "week": "2026-W41",
        "timezone": "Asia/Ho_Chi_Minh",
        "workdays": [
            "2026-10-05 Monday", "2026-10-06 Tuesday", "2026-10-07 Wednesday",
            "2026-10-08 Thursday", "2026-10-09 Friday",
        ],
        "posted_date": "2026-10-05 Monday",
        "message": SECRET_TEXT,
    }  # fmt: skip


def test_empty_lines_are_valid(messages, week41):
    assert ReportExtractor(FakeLLMClient({"lines": []})).extract(messages, week41, TZ) == []


def test_few_shot_examples_are_empty_by_default_and_rendered_when_given():
    assert FEW_SHOT_EXAMPLES == []
    assert system_prompt() == SYSTEM_PROMPT
    rendered = system_prompt([("proj-x 1h", {"lines": [_line(minutes=60)]})])
    assert rendered.startswith(SYSTEM_PROMPT)
    assert '"proj-x 1h"' in rendered and '"minutes": 60' in rendered


@pytest.mark.parametrize(
    ("answer", "where"),
    [
        ({"lines": [_line(minutes=20)]}, "lines.0.minutes"),
        ({"lines": [_line(minutes=0)]}, "lines.0.minutes"),
        ({"lines": [_line(minutes=-15)]}, "lines.0.minutes"),
        ({"lines": [_line(minutes=60.0)]}, "lines.0.minutes"),
        ({"lines": [_line(minutes="60")]}, "lines.0.minutes"),
        ({"lines": [_line(minutes=True)]}, "lines.0.minutes"),
        ({"lines": [_line(date="2026-10-12")]}, "lines.0.date"),  # next week's Monday
        ({"lines": [_line(date="2026-10-10")]}, "lines.0.date"),  # Saturday of the week
        ({"lines": [_line(date="2026-10-04")]}, "lines.0.date"),  # Sunday before
        ({"lines": [_line(date="05/10/2026")]}, "lines.0.date"),
        ({"lines": [_line(date="2026-02-30")]}, "lines.0.date"),
        ({"lines": [_line(project="   ")]}, "lines.0.project"),
        ({"lines": [_line(notes=None)]}, "lines.0.notes"),
        ({"lines": [{k: v for k, v in _line().items() if k != "notes"}]}, "lines.0.notes"),
        ({"lines": [_line(confidence=0.9)]}, "lines.0.confidence"),
        ({"lines": "none"}, "lines"),
        ({"line": []}, "line"),
        ({}, "lines"),
    ],
)
def test_invalid_output_stops_and_names_the_message(messages, week41, answer, where):
    fake = FakeLLMClient(answer)
    with pytest.raises(ReportExtractionError) as ei:
        ReportExtractor(fake).extract(messages, week41, TZ)
    msg = str(ei.value)
    assert f"ts={messages[0].ts}" in msg and "channel=C0123" in msg
    assert where in msg
    assert len(fake.calls) == 1  # stops at the first bad message


def test_no_message_text_or_notes_in_errors(messages, week41):
    """The LLM may echo the message into a bad line; the error still holds ids only."""
    echo = {"lines": [_line(project=SECRET_TEXT, notes=SECRET_TEXT, minutes=20, date="x")]}
    with pytest.raises(ReportExtractionError) as ei:
        ReportExtractor(FakeLLMClient(echo)).extract(messages, week41, TZ)
    text = str(ei.value) + repr(ei.value) + str(ei.value.__cause__)
    assert SECRET_TEXT not in text
    assert "GCM-12" not in text
    assert NOTES not in text
    assert ei.value.__cause__ is None  # the pydantic error (which quotes input) is dropped


def test_llm_errors_stop_with_the_message_ids(messages, week41):
    fake = FakeLLMClient({"lines": []}, LLMRateLimitError("DeepSeek (report_lines): HTTP 429"))
    with pytest.raises(ReportExtractionError) as ei:
        ReportExtractor(fake).extract(messages, week41, TZ)
    assert str(ei.value) == (
        f"message ts={messages[1].ts} channel=D0SELF01: DeepSeek (report_lines): HTTP 429"
    )
    assert isinstance(ei.value.__cause__, LLMRateLimitError)


def test_malformed_json_from_the_provider_stops(messages, week41):
    fake = FakeLLMClient(LLMResponseError("DeepSeek (report_lines): the content is not valid JSON"))
    with pytest.raises(ReportExtractionError, match="not valid JSON"):
        ReportExtractor(fake).extract(messages, week41, TZ)


def test_non_dict_answer_stops(messages, week41):
    fake = FakeLLMClient(lambda user: ["not", "a", "dict"])  # type: ignore[arg-type,return-value]
    with pytest.raises(ReportExtractionError, match="not a JSON object"):
        ReportExtractor(fake).extract(messages, week41, TZ)
