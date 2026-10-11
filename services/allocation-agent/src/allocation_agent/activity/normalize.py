"""Which of the owner's report-source messages count as work activity."""

from __future__ import annotations

from collections.abc import Callable
from datetime import tzinfo
from decimal import Decimal

from allocation_agent.activity.contract import Message, RawSearchHit, message_from_hit
from allocation_agent.activity.week import IsoWeek

Normalizer = Callable[[list[RawSearchHit], IsoWeek, tzinfo], list[Message]]

# Subtypes that are still the owner's own written message. Any other subtype (channel_join,
# channel_leave, channel_topic, channel_purpose, channel_name, bot_message, ...) is dropped.
_KEPT_SUBTYPES = frozenset({"thread_broadcast", "file_share"})


def normalize_messages(raw: list[RawSearchHit], week: IsoWeek, tz: tzinfo) -> list[Message]:
    """Decides which of the owner's report-source messages are written to the activity file.

    Pure: no I/O, no logging, and `raw` is not modified.

    Policy, applied in this order:

    1. Window: keep a message only if its posted time in tz falls on Monday 00:00 to
       Saturday 00:00 of `week` (`IsoWeek.contains`), the same days as the search's
       `after:<Sunday> before:<Saturday>`. Anything outside is dropped, not an error.
       Why: Slack evaluates `after:`/`before:` in the account's timezone, so hits near
       midnight can land on the Sunday or Saturday in tz; the Go core clamps the same way.
    2. Dedup by `(channel_id, ts)`: keep the hit with the latest `edited_ts`; if no
       duplicate is later edited, keep the first one seen.
       Why: the latest edit carries the final wording. Same-report-in-both-sources
       duplicates have different `channel_id`s and are left for the Go core.
    3. Drop bot messages (`is_bot`), any `subtype` other than none, `thread_broadcast` or
       `file_share`, and messages whose text is empty or only whitespace.
       Why: none of these is a report the owner wrote; a broadcast or file share is.
    4. Thread replies are kept.
       Why: a report can continue in a thread; the LLM returns no lines for non-report text.
    5. Edits: the message's current text, as search returns it, is used unchanged.
    6. Output is sorted by `(posted_at, channel_id, ts)`, so it is deterministic.

    Args:
        raw: the merged hits from the report sources (`sources.merge_sources`).
        week: the ISO week being collected.
        tz: the configured timezone (`timezone` in `config.yaml`).

    Returns:
        The kept messages as contract `Message` objects, `posted_at` in tz.
    """
    by_key: dict[tuple[str, str], RawSearchHit] = {}
    for h in raw:
        key = (h.channel_id, h.ts)
        kept = by_key.get(key)
        if kept is None or _edited_after(h, kept):
            by_key[key] = h

    out: list[Message] = []
    for h in by_key.values():
        if not _is_owner_text(h):
            continue
        m = message_from_hit(h, tz)
        if week.contains(m.posted_at, tz):
            out.append(m)
    out.sort(key=lambda m: (m.posted_at, m.channel_id, Decimal(m.ts)))
    return out


def _edited_after(candidate: RawSearchHit, kept: RawSearchHit) -> bool:
    """True if candidate carries a strictly later edit than kept (no edit counts as oldest)."""
    if not candidate.edited_ts:
        return False
    if not kept.edited_ts:
        return True
    return _ts_value(candidate.edited_ts) > _ts_value(kept.edited_ts)


def _ts_value(ts: str) -> Decimal:
    try:
        return Decimal(ts)
    except ArithmeticError:
        return Decimal(-1)


def _is_owner_text(h: RawSearchHit) -> bool:
    if h.is_bot:
        return False
    if h.subtype and h.subtype not in _KEPT_SUBTYPES:
        return False
    return bool(h.text.strip())
