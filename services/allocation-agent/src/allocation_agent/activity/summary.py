"""The `--dry-run` summary: counts only, never message text or permalinks."""

from __future__ import annotations

from collections import Counter
from datetime import tzinfo

from allocation_agent.activity.contract import ActivityWeek
from allocation_agent.activity.sources import SELF_DM_NAME, ReportScope
from allocation_agent.activity.week import IsoWeek


def render_summary(doc: ActivityWeek, week: IsoWeek, tz: tzinfo, scope: ReportScope) -> str:
    """Counts per working day and per channel. Channel names and ids are the only free text."""
    per_day = Counter(m.posted_at.astimezone(tz).date() for m in doc.messages)
    per_channel = Counter(m.channel_name or m.channel_id for m in doc.messages)

    lines = [
        f"week {doc.week}  user {doc.slack_user_id}  source {doc.source}",
        "sources: "
        + ", ".join(
            [f"#{scope.channel}"] * bool(scope.channel) + [SELF_DM_NAME] * scope.include_self_dm
        ),
        f"search window: after:{week.search_after()} before:{week.search_before()} (exclusive)",
        f"messages: {len(doc.messages)}",
        "per day:",
    ]
    for d in week.workdays():
        lines.append(f"  {d.isoformat()} {d.strftime('%a')}  {per_day.pop(d, 0)}")
    for d, n in sorted(per_day.items()):  # outside Monday-Friday; only if normalisation kept it
        lines.append(f"  {d.isoformat()} {d.strftime('%a')}  {n}  (outside the week)")
    lines.append("per channel:")
    for name, n in sorted(per_channel.items(), key=lambda kv: (-kv[1], kv[0])):
        lines.append(f"  {name}  {n}")
    lines.append("dry run: no file written")
    return "\n".join(lines) + "\n"
