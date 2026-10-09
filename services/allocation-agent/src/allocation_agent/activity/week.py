"""ISO-8601 weeks and the Slack search window, mirroring the Go core.

The Go collector (`services/timesheet-core/internal/activity/infrastructure/slack/collector.go`)
searches `from:<@user> after:<Sunday> before:<Saturday>`. Slack's `after:` and `before:` are
exclusive dates, so the search covers Monday to Friday. Messages are then clamped to Monday to
Friday of the ISO week in the configured timezone (`kernel.WeekID.Contains`).
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from datetime import date, datetime, timedelta, tzinfo

_WEEK_RE = re.compile(r"^(\d{4})-W(\d{2})$")


@dataclass(frozen=True, order=True)
class IsoWeek:
    """An ISO-8601 week such as 2026-W41. Build it with :meth:`parse` or :meth:`of`."""

    year: int
    week: int

    @classmethod
    def parse(cls, s: str) -> IsoWeek:
        """Parses the "YYYY-Www" form and rejects weeks that do not exist (2027-W53)."""
        m = _WEEK_RE.match(s)
        if not m:
            raise ValueError(f"invalid week id {s!r}, want YYYY-Www")
        year, week = int(m.group(1)), int(m.group(2))
        try:
            date.fromisocalendar(year, week, 1)
        except ValueError:
            raise ValueError(f"week {s!r} does not exist") from None
        return cls(year, week)

    @classmethod
    def of(cls, d: date) -> IsoWeek:
        """Returns the ISO week that contains the calendar date d."""
        iso = d.isocalendar()
        return cls(iso.year, iso.week)

    def __str__(self) -> str:
        return f"{self.year:04d}-W{self.week:02d}"

    def monday(self) -> date:
        return date.fromisocalendar(self.year, self.week, 1)

    def workdays(self) -> list[date]:
        """Monday to Friday of the week, the days a message can count for."""
        m = self.monday()
        return [m + timedelta(days=i) for i in range(5)]

    def search_after(self) -> date:
        """The Sunday before the week's Monday: Slack `after:` (exclusive)."""
        return self.monday() - timedelta(days=1)

    def search_before(self) -> date:
        """The Saturday after the week's Friday: Slack `before:` (exclusive)."""
        return self.monday() + timedelta(days=5)

    def search_query(self, slack_user_id: str) -> str:
        """The same query string the Go collector builds (`slack.Query`)."""
        return (
            f"from:<@{slack_user_id}> "
            f"after:{self.search_after().isoformat()} before:{self.search_before().isoformat()}"
        )

    def contains_workday(self, d: date) -> bool:
        """Reports whether d is Monday to Friday of this week (`kernel.WeekID.Contains`)."""
        return d.weekday() < 5 and IsoWeek.of(d) == self

    def contains(self, moment: datetime, tz: tzinfo) -> bool:
        """Reports whether an aware datetime falls on Monday to Friday of this week in tz."""
        if moment.tzinfo is None:
            raise ValueError("contains needs an aware datetime")
        return self.contains_workday(moment.astimezone(tz).date())
