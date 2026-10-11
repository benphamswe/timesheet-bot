"""Turns the week's summary messages into report lines through any `LLMClient`.

Depends only on the LLM port. One call per message (each message is one summary). Any LLM
error or any invalid line stops the whole extraction: there is no fallback and no partial
result. Errors name the message's `ts` and channel id, never its text, and nothing here logs.
"""

from __future__ import annotations

from collections.abc import Sequence
from datetime import tzinfo

from pydantic import ValidationError

from allocation_agent.activity.contract import Message
from allocation_agent.activity.week import IsoWeek
from allocation_agent.llm.port import LLMClient, LLMError
from allocation_agent.report.models import LLMOutput, ReportLine
from allocation_agent.report.prompt import SCHEMA_NAME, system_prompt, user_prompt


class ReportExtractionError(Exception):
    """The LLM failed or returned an invalid line. Holds ids and field names, never text."""

    def __init__(self, reason: str, *, source_ts: str, channel_id: str) -> None:
        super().__init__(f"message ts={source_ts} channel={channel_id}: {reason}")
        self.source_ts = source_ts
        self.channel_id = channel_id


def _where(e: ValidationError) -> str:
    """Field paths and error types only: pydantic's own messages may quote the input."""
    return ", ".join(
        sorted({".".join(str(p) for p in err["loc"]) + f" ({err['type']})" for err in e.errors()})
    )


class ReportExtractor:
    """`extract(messages, week, tz)`; inject any `LLMClient` (the registry builds the real one)."""

    def __init__(self, llm: LLMClient) -> None:
        self._llm = llm

    def extract(self, messages: Sequence[Message], week: IsoWeek, tz: tzinfo) -> list[ReportLine]:
        system = system_prompt()
        tz_name = str(getattr(tz, "key", tz))
        out: list[ReportLine] = []
        for m in messages:
            out.extend(self._one(m, week, tz, tz_name, system))
        return out

    def _one(
        self, m: Message, week: IsoWeek, tz: tzinfo, tz_name: str, system: str
    ) -> list[ReportLine]:
        def fail(reason: str) -> ReportExtractionError:
            return ReportExtractionError(reason, source_ts=m.ts, channel_id=m.channel_id)

        user = user_prompt(
            text=m.text,
            posted_date=m.posted_at.astimezone(tz).date(),
            workdays=week.workdays(),
            week=str(week),
            timezone=tz_name,
        )
        try:
            raw = self._llm.complete_json(system, user, schema_name=SCHEMA_NAME)
        except LLMError as e:
            raise fail(str(e)) from e
        if not isinstance(raw, dict):
            raise fail("the LLM answer is not a JSON object")
        try:
            parsed = LLMOutput.model_validate(raw, context={"week": week})
        except ValidationError as e:
            raise fail(f"invalid LLM output at: {_where(e)}") from None
        try:
            return [
                ReportLine.model_validate(
                    {
                        "date": line.date,
                        "project": line.project,
                        "minutes": line.minutes,
                        "notes": line.notes,
                        "source_ts": m.ts,
                        "source_channel_id": m.channel_id,
                    },
                    context={"week": week},
                )
                for line in parsed.lines
            ]
        except ValidationError as e:
            raise fail(f"invalid report line at: {_where(e)}") from None
