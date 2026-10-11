"""Shared test helpers. Fake values only: U0123ABCD, C0123, proj-x, week 2026-W41."""

from __future__ import annotations

from datetime import datetime, tzinfo
from pathlib import Path
from zoneinfo import ZoneInfo

from allocation_agent.activity.contract import Message, RawSearchHit, message_from_hit
from allocation_agent.activity.week import IsoWeek

REPO_ROOT = Path(__file__).resolve().parents[3]
SCHEMA_PATH = REPO_ROOT / "api" / "activity" / "activity-week.v1.schema.json"
SCHEMA_V2_PATH = REPO_ROOT / "api" / "activity" / "activity-week.v2.schema.json"
TZ = ZoneInfo("Asia/Ho_Chi_Minh")
USER = "U0123ABCD"
SECRET_TEXT = "GCM-12 confidential design note"
REPORT_CHANNEL = "timesheet-reports"
SELF_DM_ID = "D0SELF01"


def passthrough(raw: list[RawSearchHit], week: IsoWeek, tz: tzinfo) -> list[Message]:
    """Test-only stand-in for normalize_messages: keeps every hit, decides nothing."""
    return [message_from_hit(h, tz) for h in raw]


def slack_ts(local: datetime, micros: int = 100) -> str:
    return f"{int(local.timestamp())}.{micros:06d}"


def hit(local: datetime, **kw: object) -> RawSearchHit:
    base: dict[str, object] = {
        "ts": slack_ts(local),
        "channel_id": "C0123",
        "channel_name": REPORT_CHANNEL,
        "text": SECRET_TEXT,
        "permalink": "https://example.slack.com/archives/C0123/p1",
        "thread_ts": None,
    }
    base.update(kw)
    return RawSearchHit.model_validate(base)


def dm_hit(local: datetime, **kw: object) -> RawSearchHit:
    """A self-DM hit as merged: D... channel id, channel_name "self-dm", source self_dm."""
    base: dict[str, object] = {
        "channel_id": SELF_DM_ID,
        "channel_name": "self-dm",
        "source": "self_dm",
        "permalink": f"https://example.slack.com/archives/{SELF_DM_ID}/p1",
    }
    base.update(kw)
    return hit(local, **base)
