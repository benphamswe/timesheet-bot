"""Where the owner's timesheet reports live, and how each place is searched.

The owner writes timesheet summaries in two places: one summary channel and their own self-DM
(the "you" conversation). The collector gathers only the owner's messages from those two
places, never every message the owner wrote everywhere.

Self-DM method: Slack search with `in:<@U...>`. The search.messages reference says: "To
specifically search within a channel, group, or DM, add `in:channel_name`, `in:group_name`,
or `in:<@UserID>`" (https://docs.slack.dev/reference/methods/search.messages, read
2026-10-10). The self-DM is the DM whose other member is the owner, so the modifier is
`in:<@OWNER>`. It is unverified through the claude.ai connector, which was `needs-auth`, so
`merge_sources` fails closed if the DM search returns anything that is not a DM (`D...` id).
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from typing import Literal

from allocation_agent.activity.contract import RawSearchHit
from allocation_agent.activity.week import IsoWeek

SELF_DM_NAME = "self-dm"
# Slack channel names: lowercase letters, digits, hyphens, underscores, periods, at most 80.
# Validated so a configured name can never add search modifiers (spaces, colons, quotes).
_CHANNEL_NAME = re.compile(r"^[a-z0-9][a-z0-9._-]{0,79}$")

Source = Literal["channel", "self_dm"]


class ScopeError(ValueError):
    """The report scope is missing or invalid."""


@dataclass(frozen=True)
class ReportScope:
    """`report.channel` and `report.include_self_dm` from config.yaml, after overrides."""

    channel: str | None
    include_self_dm: bool

    @classmethod
    def build(cls, channel: str | None, include_self_dm: bool) -> ReportScope:
        name = (channel or "").strip().removeprefix("#") or None
        if name is not None and not _CHANNEL_NAME.match(name):
            raise ScopeError(
                f"report channel {name!r} is not a Slack channel name "
                "(lowercase letters, digits, '-', '_', '.')"
            )
        if name is None and not include_self_dm:
            raise ScopeError(
                "no report source: set report.channel and/or report.include_self_dm in "
                "config.yaml, REPORT_CHANNEL, --report-channel or --include-self-dm"
            )
        return cls(channel=name, include_self_dm=include_self_dm)

    def sources(self) -> list[Source]:
        out: list[Source] = []
        if self.channel:
            out.append("channel")
        if self.include_self_dm:
            out.append("self_dm")
        return out


def window(week: IsoWeek) -> str:
    """`after:<Sunday> before:<Saturday>`, exclusive, as in the Go collector."""
    return f"after:{week.search_after().isoformat()} before:{week.search_before().isoformat()}"


def channel_query(week: IsoWeek, slack_user_id: str, channel: str) -> str:
    return f"in:#{channel} from:<@{slack_user_id}> {window(week)}"


def self_dm_query(week: IsoWeek, slack_user_id: str) -> str:
    return f"in:<@{slack_user_id}> from:<@{slack_user_id}> {window(week)}"


def queries(scope: ReportScope, week: IsoWeek, slack_user_id: str) -> dict[Source, str]:
    out: dict[Source, str] = {}
    if scope.channel:
        out["channel"] = channel_query(week, slack_user_id, scope.channel)
    if scope.include_self_dm:
        out["self_dm"] = self_dm_query(week, slack_user_id)
    return out


class SourceIntegrityError(Exception):
    """A search returned results from the wrong place. Never holds message text."""


def merge_sources(
    by_source: dict[Source, list[RawSearchHit]], scope: ReportScope
) -> list[RawSearchHit]:
    """Tags, checks and merges the per-source hits; dedups by (channel_id, ts) only.

    - Channel hits get `channel_name` = the configured channel; self-DM hits get "self-dm",
      so the Go core can tell the sources apart.
    - Self-DM hits must come from a DM (`D...` id), and channel hits from one channel id.
      Anything else means the search modifier did not do what we assume, so this fails
      closed instead of mixing in other conversations.
    - The first hit for a (channel_id, ts) wins. Content-level duplicates across the two
      sources are left for the Go core.
    """
    merged: list[RawSearchHit] = []
    seen: set[tuple[str, str]] = set()
    for source in scope.sources():
        hits = by_source.get(source, [])
        if source == "self_dm":
            if any(not h.channel_id.startswith("D") for h in hits):
                raise SourceIntegrityError(
                    "the self-DM search returned messages outside a DM; "
                    "the in:<@user> modifier may not work through the connector"
                )
            name = SELF_DM_NAME
        else:
            if len({h.channel_id for h in hits}) > 1:
                raise SourceIntegrityError(
                    f"the #{scope.channel} search returned messages from several channels"
                )
            name = scope.channel or ""
        for h in hits:
            key = (h.channel_id, h.ts)
            if key in seen:
                continue
            seen.add(key)
            merged.append(h.model_copy(update={"channel_name": name, "source": source}))
    return merged
