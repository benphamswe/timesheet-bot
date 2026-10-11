"""The activity-week file contract (schema versions 1 and 2) shared with the Go core.

The JSON Schemas are `api/activity/activity-week.v1.schema.json` and
`activity-week.v2.schema.json` at the repository root. Version 2 is version 1 plus `parser` and
`report_lines` (report.parser=llm); report.parser=manual keeps writing version 1. The Go reader
depends on these exact field names; change both sides and bump `schema_version` together.
"""

from __future__ import annotations

from datetime import datetime, tzinfo
from decimal import Decimal, InvalidOperation
from typing import Annotated, Literal

from pydantic import AfterValidator, BaseModel, ConfigDict, Field

from allocation_agent.report.models import ParserInfo, ReportLine

SCHEMA_VERSION = 1
SCHEMA_VERSION_REPORT_LINES = 2
SOURCE = "slack-mcp"

_SLACK_TS = r"^\d+\.\d+$"
_WEEK = r"^\d{4}-W(0[1-9]|[1-4]\d|5[0-3])$"
_SLACK_USER = r"^[UW][A-Z0-9]+$"


def _require_offset(v: datetime) -> datetime:
    if v.tzinfo is None or v.utcoffset() is None:
        raise ValueError("datetime must carry a UTC offset (RFC 3339)")
    return v


AwareDatetime = Annotated[datetime, AfterValidator(_require_offset)]


class Message(BaseModel):
    """One Slack message that counts as work activity."""

    model_config = ConfigDict(extra="forbid", frozen=True)

    ts: str = Field(pattern=_SLACK_TS)
    channel_id: str = Field(min_length=1)
    channel_name: str
    posted_at: AwareDatetime
    text: str
    permalink: str
    thread_ts: str | None = Field(default=None, pattern=_SLACK_TS)


class ActivityWeek(BaseModel):
    """The whole file: `<out-dir>/<ISO week>.json`."""

    model_config = ConfigDict(extra="forbid")

    schema_version: Literal[1] = SCHEMA_VERSION
    week: str = Field(pattern=_WEEK)
    slack_user_id: str = Field(pattern=_SLACK_USER)
    generated_at: AwareDatetime
    source: Literal["slack-mcp"] = SOURCE
    messages: list[Message]

    def to_json(self) -> str:
        """Serialises the file body: UTF-8 JSON, two-space indent, trailing newline."""
        return self.model_dump_json(indent=2) + "\n"


class ActivityWeekV2(BaseModel):
    """Version 2 of the file: version 1 plus how the lines were parsed and the lines."""

    model_config = ConfigDict(extra="forbid")

    schema_version: Literal[2] = SCHEMA_VERSION_REPORT_LINES
    week: str = Field(pattern=_WEEK)
    slack_user_id: str = Field(pattern=_SLACK_USER)
    generated_at: AwareDatetime
    source: Literal["slack-mcp"] = SOURCE
    messages: list[Message]
    parser: ParserInfo
    report_lines: list[ReportLine]

    @classmethod
    def from_v1(
        cls, doc: ActivityWeek, *, parser: ParserInfo, report_lines: list[ReportLine]
    ) -> ActivityWeekV2:
        return cls(
            week=doc.week,
            slack_user_id=doc.slack_user_id,
            generated_at=doc.generated_at,
            source=doc.source,
            messages=doc.messages,
            parser=parser,
            report_lines=report_lines,
        )

    def to_json(self) -> str:
        """Serialises the file body: UTF-8 JSON, two-space indent, trailing newline."""
        return self.model_dump_json(indent=2) + "\n"


class RawSearchHit(BaseModel):
    """One search result as the agent reports it, before normalisation. Not part of the file.

    Fields other than `ts`, `channel_id` and `text` are optional because the Slack MCP
    connector's result shape is not documented; the agent sets null when a field is absent.
    `source` is set by the collector (`sources.merge_sources`), never by the agent.
    """

    model_config = ConfigDict(extra="ignore", frozen=True)

    ts: str = Field(pattern=_SLACK_TS)
    channel_id: str = Field(min_length=1)
    channel_name: str | None = None
    text: str
    permalink: str | None = None
    thread_ts: str | None = None
    edited_ts: str | None = None
    user_id: str | None = None
    is_bot: bool | None = None
    subtype: str | None = None
    source: Literal["channel", "self_dm"] | None = None


class SourceResult(BaseModel):
    """The agent's answer for one search query."""

    model_config = ConfigDict(extra="ignore")

    complete: bool
    messages: list[RawSearchHit]


class AgentResult(BaseModel):
    """The structured output the agent must return: one section per requested source."""

    model_config = ConfigDict(extra="ignore")

    channel: SourceResult | None
    self_dm: SourceResult | None


def ts_to_datetime(ts: str, tz: tzinfo) -> datetime:
    """Converts a Slack ts ("1760000000.000100") to an aware datetime in tz, whole seconds.

    Seconds only, like the Go collector's `parseTS`.
    """
    try:
        secs = int(Decimal(ts))
    except InvalidOperation:
        raise ValueError(f"bad slack ts {ts!r}") from None
    return datetime.fromtimestamp(secs, tz)


def message_from_hit(hit: RawSearchHit, tz: tzinfo) -> Message:
    """Maps one raw hit to a contract `Message`, with `posted_at` in tz. Filters nothing."""
    return Message(
        ts=hit.ts,
        channel_id=hit.channel_id,
        channel_name=hit.channel_name or "",
        posted_at=ts_to_datetime(hit.ts, tz),
        text=hit.text,
        permalink=hit.permalink or "",
        thread_ts=hit.thread_ts or None,
    )


def ts_sort_key(m: Message) -> Decimal:
    return Decimal(m.ts)
