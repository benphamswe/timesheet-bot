"""The activity-week file contract (schema version 1) shared with the Go core.

The JSON Schema is `api/activity/activity-week.v1.schema.json` at the repository root. The Go
reader depends on these exact field names; change both sides and bump `schema_version` together.
"""

from __future__ import annotations

from datetime import datetime, tzinfo
from decimal import Decimal, InvalidOperation
from typing import Annotated, Literal

from pydantic import AfterValidator, BaseModel, ConfigDict, Field

SCHEMA_VERSION = 1
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


class RawSearchHit(BaseModel):
    """One search result as the agent reports it, before normalisation.

    Fields other than `ts`, `channel_id` and `text` are optional because the Slack MCP
    connector's result shape is not documented; the agent sets null when a field is absent.
    """

    model_config = ConfigDict(extra="ignore", frozen=True)

    ts: str = Field(pattern=_SLACK_TS)
    channel_id: str = Field(min_length=1)
    channel_name: str | None = None
    text: str
    permalink: str | None = None
    thread_ts: str | None = None
    user_id: str | None = None
    is_bot: bool | None = None
    subtype: str | None = None


class AgentResult(BaseModel):
    """The structured output the agent must return."""

    model_config = ConfigDict(extra="ignore")

    complete: bool
    messages: list[RawSearchHit]


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
