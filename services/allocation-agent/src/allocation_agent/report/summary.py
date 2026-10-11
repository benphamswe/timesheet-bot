"""The `--dry-run` table for report.parser=llm: minutes per day and project.

Prints project names, dates and totals only, never notes or message text.
"""

from __future__ import annotations

from collections import defaultdict
from datetime import date

from allocation_agent.activity.week import IsoWeek
from allocation_agent.report.models import ParserInfo, ReportLine


def _hm(minutes: int) -> str:
    h, m = divmod(minutes, 60)
    return f"{h}h{m:02d}m"


def render_report_summary(lines: list[ReportLine], week: IsoWeek, parser: ParserInfo) -> str:
    per: dict[date, dict[str, int]] = defaultdict(lambda: defaultdict(int))
    for line in lines:
        per[line.date][line.project] += line.minutes

    out = [
        f"report lines: {len(lines)}  parser: {parser.kind} {parser.provider} {parser.model}",
        "minutes per day and project:",
    ]
    for d in week.workdays():
        projects = per.get(d, {})
        total = sum(projects.values())
        out.append(f"  {d.isoformat()} {d.strftime('%a')}  total {total:>4}  ({_hm(total)})")
        for name, minutes in sorted(projects.items(), key=lambda kv: (-kv[1], kv[0])):
            out.append(f"    {name}  {minutes}")
    week_total = sum(line.minutes for line in lines)
    out.append(f"week total: {week_total} ({_hm(week_total)})")
    return "\n".join(out) + "\n"
