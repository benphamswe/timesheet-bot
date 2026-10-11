from datetime import datetime

import pytest

from allocation_agent.activity.sources import (
    ReportScope,
    ScopeError,
    SourceIntegrityError,
    channel_query,
    merge_sources,
    queries,
    self_dm_query,
)
from allocation_agent.activity.week import IsoWeek
from support import REPORT_CHANNEL, SELF_DM_ID, TZ, hit, slack_ts

W41 = IsoWeek.parse("2026-W41")
MON_9 = datetime(2026, 10, 5, 9, 0, tzinfo=TZ)
TUE_9 = datetime(2026, 10, 6, 9, 0, tzinfo=TZ)
BOTH = ReportScope.build(REPORT_CHANNEL, True)


def test_channel_query():
    assert channel_query(W41, "U0123ABCD", "timesheet-reports") == (
        "in:#timesheet-reports from:<@U0123ABCD> after:2026-10-04 before:2026-10-10"
    )


def test_self_dm_query():
    assert self_dm_query(W41, "U0123ABCD") == (
        "in:<@U0123ABCD> from:<@U0123ABCD> after:2026-10-04 before:2026-10-10"
    )


def test_queries_follow_scope_and_year_boundary():
    w53 = IsoWeek.parse("2026-W53")
    assert queries(ReportScope.build("r", False), w53, "U1") == {
        "channel": "in:#r from:<@U1> after:2026-12-27 before:2027-01-02"
    }
    assert queries(ReportScope.build(None, True), w53, "U1") == {
        "self_dm": "in:<@U1> from:<@U1> after:2026-12-27 before:2027-01-02"
    }
    assert set(queries(BOTH, W41, "U1")) == {"channel", "self_dm"}


@pytest.mark.parametrize(
    ("channel", "dm", "want"),
    [
        ("#timesheet-reports", False, ("timesheet-reports", False)),
        ("  reports ", True, ("reports", True)),
        ("", True, (None, True)),
        (None, True, (None, True)),
    ],
)
def test_scope_build(channel, dm, want):
    s = ReportScope.build(channel, dm)
    assert (s.channel, s.include_self_dm) == want


@pytest.mark.parametrize(
    ("channel", "dm", "match"),
    [
        ("", False, "no report source"),
        (None, False, "no report source"),
        ("reports from:<@U9>", False, "not a Slack channel name"),
        ("Reports", True, "not a Slack channel name"),
        ("in:#x", True, "not a Slack channel name"),
    ],
)
def test_scope_build_rejects(channel, dm, match):
    with pytest.raises(ScopeError, match=match):
        ReportScope.build(channel, dm)


def test_merge_tags_names_and_dedups_by_channel_and_ts():
    ch = [
        hit(MON_9, channel_name="whatever-the-connector-said"),
        hit(MON_9),  # same (channel_id, ts): dropped
        hit(TUE_9),
    ]
    dm = [
        hit(MON_9, channel_id=SELF_DM_ID, channel_name="U0123ABCD"),  # same ts, other channel
        hit(TUE_9, channel_id=SELF_DM_ID, channel_name=None),
        hit(TUE_9, channel_id=SELF_DM_ID),  # duplicate inside the DM
    ]
    out = merge_sources({"channel": ch, "self_dm": dm}, BOTH)

    assert [(h.channel_id, h.ts, h.channel_name, h.source) for h in out] == [
        ("C0123", slack_ts(MON_9), REPORT_CHANNEL, "channel"),
        ("C0123", slack_ts(TUE_9), REPORT_CHANNEL, "channel"),
        (SELF_DM_ID, slack_ts(MON_9), "self-dm", "self_dm"),
        (SELF_DM_ID, slack_ts(TUE_9), "self-dm", "self_dm"),
    ]


def test_merge_keeps_same_text_in_both_sources():
    """Content-level duplicates are the Go core's job, not this merge's."""
    out = merge_sources(
        {"channel": [hit(MON_9)], "self_dm": [hit(TUE_9, channel_id=SELF_DM_ID)]}, BOTH
    )
    assert len(out) == 2 and out[0].text == out[1].text


def test_merge_ignores_sources_outside_scope():
    scope = ReportScope.build(None, True)
    out = merge_sources({"channel": [hit(MON_9)], "self_dm": []}, scope)
    assert out == []


def test_merge_fails_closed_when_dm_search_leaks_a_channel():
    with pytest.raises(SourceIntegrityError, match="outside a DM"):
        merge_sources({"self_dm": [hit(MON_9, channel_id="C0999")]}, BOTH)


def test_merge_fails_closed_when_channel_search_spans_channels():
    with pytest.raises(SourceIntegrityError, match="several channels"):
        merge_sources({"channel": [hit(MON_9), hit(TUE_9, channel_id="C0999")]}, BOTH)
