"""Shared fixtures; helpers live in support.py."""

from __future__ import annotations

import json
from datetime import UTC, datetime

import pytest

from allocation_agent.activity.contract import RawSearchHit
from allocation_agent.activity.week import IsoWeek
from support import SCHEMA_PATH, TZ, hit, slack_ts


@pytest.fixture
def schema() -> dict:
    return json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))


@pytest.fixture
def fixed_now() -> datetime:
    return datetime(2026, 10, 10, 3, 0, 0, 123456, tzinfo=UTC)


@pytest.fixture
def week41() -> IsoWeek:
    return IsoWeek.parse("2026-W41")


@pytest.fixture
def raw_hits() -> list[RawSearchHit]:
    return [
        hit(datetime(2026, 10, 7, 14, 0, tzinfo=TZ), channel_id="C0456", channel_name="proj-y"),
        hit(datetime(2026, 10, 5, 9, 0, tzinfo=TZ)),
        hit(
            datetime(2026, 10, 5, 10, 30, tzinfo=TZ),
            thread_ts=slack_ts(datetime(2026, 10, 5, 9, 0, tzinfo=TZ)),
        ),
    ]
