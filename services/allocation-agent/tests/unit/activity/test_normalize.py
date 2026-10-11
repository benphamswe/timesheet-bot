"""Tests for normalize_messages.

Policy: keep Monday 00:00 to Saturday 00:00 in tz; dedup by (channel_id, ts), latest
edited_ts wins; drop bot, system-subtype and empty messages; keep thread replies and edits
with their current text; sort by (posted_at, channel_id, ts).
"""

from datetime import UTC, datetime

import pytest

from allocation_agent.activity.normalize import normalize_messages
from support import REPORT_CHANNEL, TZ, dm_hit, hit, slack_ts

MON_9 = datetime(2026, 10, 5, 9, 0, tzinfo=TZ)
MON_18 = datetime(2026, 10, 5, 18, 0, tzinfo=TZ)


def test_keeps_reports_from_both_sources_with_their_names(week41):
    raw = [hit(MON_18, source="channel"), dm_hit(MON_9)]
    out = normalize_messages(raw, week41, TZ)
    assert sorted(m.channel_name for m in out) == sorted([REPORT_CHANNEL, "self-dm"])


def test_same_report_in_both_sources_is_kept_twice(week41):
    """Content-level duplicates across sources are left for the Go core."""
    out = normalize_messages([hit(MON_18, source="channel"), dm_hit(MON_18)], week41, TZ)
    assert len(out) == 2


def test_keeps_edited_report_with_current_text(week41):
    edited = hit(MON_18, source="channel", text="done: GCM-12 (fixed)", edited_ts=slack_ts(MON_18))
    [m] = normalize_messages([edited], week41, TZ)
    assert m.text == "done: GCM-12 (fixed)"


def test_keeps_thread_replies(week41):
    reply = hit(MON_18, source="channel", thread_ts=slack_ts(MON_9))
    [m] = normalize_messages([reply], week41, TZ)
    assert m.thread_ts == slack_ts(MON_9)


@pytest.mark.parametrize(
    ("local", "kept"),
    [
        (datetime(2026, 10, 4, 23, 59, 59, tzinfo=TZ), False),  # Sunday before Monday
        (datetime(2026, 10, 5, 0, 0, tzinfo=TZ), True),  # Monday 00:00
        (datetime(2026, 10, 9, 23, 59, tzinfo=TZ), True),  # Friday 23:59
        (datetime(2026, 10, 10, 0, 0, tzinfo=TZ), False),  # Saturday 00:00
    ],
)
def test_window_edges_in_tz(week41, local, kept):
    out = normalize_messages([dm_hit(local)], week41, TZ)
    assert len(out) == (1 if kept else 0)


def test_window_is_decided_in_tz_not_utc(week41):
    """Asia/Ho_Chi_Minh is UTC+7: Sunday 18:00 UTC is Monday 01:00 local, Friday 17:00 UTC is
    Saturday 00:00 local."""
    mon_local = datetime(2026, 10, 4, 18, 0, tzinfo=UTC)
    sat_local = datetime(2026, 10, 9, 17, 0, tzinfo=UTC)
    out = normalize_messages([dm_hit(mon_local), dm_hit(sat_local)], week41, TZ)
    assert [m.posted_at.isoformat() for m in out] == ["2026-10-05T01:00:00+07:00"]


def test_dedup_keeps_latest_edit(week41):
    ts = slack_ts(MON_9)
    raw = [
        dm_hit(MON_9, text="first", edited_ts=slack_ts(MON_9, 200)),
        dm_hit(MON_9, text="latest", edited_ts=slack_ts(MON_18)),
        dm_hit(MON_9, text="older", edited_ts=slack_ts(MON_9, 300)),
        dm_hit(MON_9, text="unedited"),
    ]
    [m] = normalize_messages(raw, week41, TZ)
    assert (m.ts, m.text) == (ts, "latest")


def test_dedup_without_edits_keeps_first_seen(week41):
    raw = [dm_hit(MON_9, text="first"), dm_hit(MON_9, text="second")]
    [m] = normalize_messages(raw, week41, TZ)
    assert m.text == "first"


def test_dedup_prefers_edited_over_unedited(week41):
    raw = [dm_hit(MON_9, text="unedited"), dm_hit(MON_9, text="edited", edited_ts=slack_ts(MON_18))]
    [m] = normalize_messages(raw, week41, TZ)
    assert m.text == "edited"


def test_drops_bot_and_system_messages(week41):
    system = [
        "channel_join",
        "channel_leave",
        "channel_topic",
        "channel_purpose",
        "channel_name",
        "bot_message",
        "some_future_subtype",
    ]
    raw = [dm_hit(MON_9, ts=slack_ts(MON_9, 1), is_bot=True)]
    raw += [dm_hit(MON_9, ts=slack_ts(MON_9, 10 + i), subtype=s) for i, s in enumerate(system)]
    raw += [
        dm_hit(MON_9, ts=slack_ts(MON_9, 50)),
        dm_hit(MON_9, ts=slack_ts(MON_9, 51), subtype="thread_broadcast"),
        dm_hit(MON_9, ts=slack_ts(MON_9, 52), subtype="file_share"),
        dm_hit(MON_9, ts=slack_ts(MON_9, 53), is_bot=False),
    ]
    out = normalize_messages(raw, week41, TZ)
    assert [m.ts for m in out] == [slack_ts(MON_9, n) for n in (50, 51, 52, 53)]


@pytest.mark.parametrize("text", ["", " ", "\n\t  \n"])
def test_drops_empty_or_whitespace_text(week41, text):
    assert normalize_messages([dm_hit(MON_9, text=text)], week41, TZ) == []


def test_sorted_by_posted_at_then_channel_then_ts(week41):
    raw = [
        hit(MON_18, source="channel"),
        dm_hit(MON_9, ts=slack_ts(MON_9, 900)),
        dm_hit(MON_9, ts=slack_ts(MON_9, 100)),
        hit(MON_9, source="channel"),
    ]
    out = normalize_messages(raw, week41, TZ)
    assert [(m.channel_id, m.ts) for m in out] == [
        ("C0123", slack_ts(MON_9)),
        ("D0SELF01", slack_ts(MON_9, 100)),
        ("D0SELF01", slack_ts(MON_9, 900)),
        ("C0123", slack_ts(MON_18)),
    ]


def test_does_not_mutate_input(week41):
    raw = [
        dm_hit(MON_9, text="a"),
        dm_hit(MON_9, text="b", edited_ts=slack_ts(MON_18)),
        dm_hit(datetime(2026, 10, 10, 1, 0, tzinfo=TZ)),
        dm_hit(MON_18, text="  "),
    ]
    before = [h.model_dump() for h in raw]
    ids = [id(h) for h in raw]
    normalize_messages(raw, week41, TZ)
    assert [h.model_dump() for h in raw] == before
    assert [id(h) for h in raw] == ids


def test_posted_at_is_in_configured_tz(week41):
    [m] = normalize_messages([dm_hit(MON_9)], week41, TZ)
    assert m.posted_at.isoformat() == "2026-10-05T09:00:00+07:00"
