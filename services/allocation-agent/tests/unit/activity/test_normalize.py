"""Tests for normalize_messages, the owner's contribution point.

They describe one reasonable rule set (the Go collector's behaviour plus dedup and bot
filtering). Adjust them to the rules you choose, then remove the skip.
"""

from datetime import datetime

import pytest

from allocation_agent.activity.normalize import normalize_messages
from support import TZ, hit, slack_ts

pytestmark = pytest.mark.skip(
    reason="TODO(owner): normalize_messages is the owner's task; remove this skip when it lands"
)

MON_9 = datetime(2026, 10, 5, 9, 0, tzinfo=TZ)


def test_dedups_by_channel_and_ts(week41):
    out = normalize_messages([hit(MON_9), hit(MON_9)], week41, TZ)
    assert len(out) == 1


def test_same_ts_in_two_channels_is_two_messages(week41):
    out = normalize_messages([hit(MON_9), hit(MON_9, channel_id="C0456")], week41, TZ)
    assert len(out) == 2


def test_drops_bot_and_system_messages(week41):
    raw = [
        hit(MON_9, is_bot=True),
        hit(MON_9, ts=slack_ts(MON_9, 2), subtype="channel_join"),
        hit(MON_9),
    ]
    out = normalize_messages(raw, week41, TZ)
    assert [m.ts for m in out] == [slack_ts(MON_9)]


def test_keeps_thread_replies(week41):
    reply = hit(datetime(2026, 10, 5, 10, tzinfo=TZ), thread_ts=slack_ts(MON_9))
    [m] = normalize_messages([reply], week41, TZ)
    assert m.thread_ts == slack_ts(MON_9)


def test_clamps_to_monday_friday_in_tz(week41):
    raw = [
        hit(datetime(2026, 10, 4, 23, 59, tzinfo=TZ)),  # Sunday before
        hit(MON_9),
        hit(datetime(2026, 10, 10, 0, 1, tzinfo=TZ)),  # Saturday after
    ]
    out = normalize_messages(raw, week41, TZ)
    assert [m.posted_at.date().isoformat() for m in out] == ["2026-10-05"]


def test_posted_at_is_in_configured_tz(week41):
    [m] = normalize_messages([hit(MON_9)], week41, TZ)
    assert m.posted_at.isoformat() == "2026-10-05T09:00:00+07:00"
