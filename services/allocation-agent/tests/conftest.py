"""Shared fixtures; helpers live in support.py."""

from __future__ import annotations

import json
from datetime import UTC, datetime

import httpx
import pytest

from allocation_agent.activity.contract import RawSearchHit
from allocation_agent.activity.week import IsoWeek
from support import SCHEMA_PATH, TZ, dm_hit, hit, slack_ts


@pytest.fixture(autouse=True)
def _no_real_http(monkeypatch):
    """No test may reach a real API: the default httpx transport refuses every request.

    Tests that exercise an HTTP adapter pass an `httpx.MockTransport`, which is not affected.
    """

    def refuse(self, request):
        raise AssertionError(f"real HTTP call attempted in a test: {request.method} {request.url}")

    monkeypatch.setattr(httpx.HTTPTransport, "handle_request", refuse)


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
        dm_hit(datetime(2026, 10, 7, 14, 0, tzinfo=TZ)),
        hit(datetime(2026, 10, 5, 9, 0, tzinfo=TZ), source="channel"),
        hit(
            datetime(2026, 10, 5, 10, 30, tzinfo=TZ),
            source="channel",
            thread_ts=slack_ts(datetime(2026, 10, 5, 9, 0, tzinfo=TZ)),
        ),
    ]
