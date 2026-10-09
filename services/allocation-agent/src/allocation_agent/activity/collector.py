"""Builds one ActivityWeek from fetched hits: normalise, sort, stamp."""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from datetime import datetime, tzinfo

from allocation_agent.activity.contract import ActivityWeek, RawSearchHit, ts_sort_key
from allocation_agent.activity.normalize import Normalizer, normalize_messages
from allocation_agent.activity.week import IsoWeek

Fetch = Callable[[IsoWeek, str], Awaitable[list[RawSearchHit]]]
Clock = Callable[[], datetime]


def build_activity_week(
    raw: list[RawSearchHit],
    *,
    week: IsoWeek,
    slack_user_id: str,
    tz: tzinfo,
    now: datetime,
    normalize: Normalizer = normalize_messages,
) -> ActivityWeek:
    messages = sorted(normalize(raw, week, tz), key=ts_sort_key)
    return ActivityWeek(
        week=str(week),
        slack_user_id=slack_user_id,
        generated_at=now.astimezone(tz).replace(microsecond=0),
        messages=messages,
    )


async def collect_week(
    week: IsoWeek,
    *,
    slack_user_id: str,
    tz: tzinfo,
    fetch: Fetch,
    clock: Clock,
    normalize: Normalizer = normalize_messages,
) -> ActivityWeek:
    raw = await fetch(week, slack_user_id)
    return build_activity_week(
        raw, week=week, slack_user_id=slack_user_id, tz=tz, now=clock(), normalize=normalize
    )
