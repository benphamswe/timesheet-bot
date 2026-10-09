from datetime import date, datetime
from zoneinfo import ZoneInfo

import pytest

from allocation_agent.activity.week import IsoWeek

HCM = ZoneInfo("Asia/Ho_Chi_Minh")


def test_week41_window_matches_go_query():
    w = IsoWeek.parse("2026-W41")
    assert w.monday() == date(2026, 10, 5)
    assert w.search_after() == date(2026, 10, 4)  # Sunday, exclusive
    assert w.search_before() == date(2026, 10, 10)  # Saturday, exclusive
    assert w.search_query("U0123ABCD") == "from:<@U0123ABCD> after:2026-10-04 before:2026-10-10"
    assert w.workdays() == [date(2026, 10, d) for d in range(5, 10)]


@pytest.mark.parametrize(
    ("week", "after", "before", "first", "last"),
    [
        # 2026 starts on a Thursday, so it has 53 weeks; W53 runs into 2027.
        ("2026-W53", date(2026, 12, 27), date(2027, 1, 2), date(2026, 12, 28), date(2027, 1, 1)),
        ("2027-W01", date(2027, 1, 3), date(2027, 1, 9), date(2027, 1, 4), date(2027, 1, 8)),
        # 2025-W01 starts in 2024.
        ("2025-W01", date(2024, 12, 29), date(2025, 1, 4), date(2024, 12, 30), date(2025, 1, 3)),
        ("2026-W01", date(2025, 12, 28), date(2026, 1, 3), date(2025, 12, 29), date(2026, 1, 2)),
    ],
)
def test_year_boundaries(week, after, before, first, last):
    w = IsoWeek.parse(week)
    assert (w.search_after(), w.search_before()) == (after, before)
    assert (w.workdays()[0], w.workdays()[-1]) == (first, last)
    assert str(w) == week


@pytest.mark.parametrize("bad", ["2027-W53", "2026-W00", "2026-W54", "2026W41", "2026-W4", "x"])
def test_parse_rejects(bad):
    with pytest.raises(ValueError):
        IsoWeek.parse(bad)


def test_of_maps_dates_to_iso_week():
    assert IsoWeek.of(date(2027, 1, 1)) == IsoWeek(2026, 53)
    assert IsoWeek.of(date(2024, 12, 30)) == IsoWeek(2025, 1)


@pytest.mark.parametrize(
    ("moment", "want"),
    [
        (datetime(2026, 10, 5, 0, 0, tzinfo=HCM), True),  # Monday 00:00 local
        (datetime(2026, 10, 9, 23, 59, 59, tzinfo=HCM), True),  # Friday last second
        (datetime(2026, 10, 4, 23, 59, 59, tzinfo=HCM), False),  # Sunday before
        (datetime(2026, 10, 10, 0, 0, tzinfo=HCM), False),  # Saturday after
        # Sunday 18:00 UTC is Monday 01:00 in Ho Chi Minh City: counts.
        (datetime(2026, 10, 4, 18, 0, tzinfo=ZoneInfo("UTC")), True),
        # Friday 18:00 UTC is Saturday 01:00 in Ho Chi Minh City: does not count.
        (datetime(2026, 10, 9, 18, 0, tzinfo=ZoneInfo("UTC")), False),
    ],
)
def test_contains_clamps_to_monday_friday_in_tz(moment, want):
    assert IsoWeek.parse("2026-W41").contains(moment, HCM) is want


def test_contains_rejects_naive():
    with pytest.raises(ValueError):
        IsoWeek.parse("2026-W41").contains(datetime(2026, 10, 5, 9), HCM)


def test_w53_contains_new_years_day_but_not_the_weekend():
    w = IsoWeek.parse("2026-W53")
    assert w.contains(datetime(2027, 1, 1, 12, tzinfo=HCM), HCM)
    assert not w.contains(datetime(2027, 1, 2, 12, tzinfo=HCM), HCM)
    assert not w.contains(datetime(2026, 12, 27, 12, tzinfo=HCM), HCM)
