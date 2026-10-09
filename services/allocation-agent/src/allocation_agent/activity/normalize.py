"""Which Slack search results count as work activity.

`normalize_messages` is the owner's contribution point. Everything else in the collector is
built and tested around it; until it is written, `collect` stops with a clear error.
"""

from __future__ import annotations

from collections.abc import Callable
from datetime import tzinfo

from allocation_agent.activity.contract import Message, RawSearchHit
from allocation_agent.activity.week import IsoWeek

Normalizer = Callable[[list[RawSearchHit], IsoWeek, tzinfo], list[Message]]


def normalize_messages(raw: list[RawSearchHit], week: IsoWeek, tz: tzinfo) -> list[Message]:
    """Turns the agent's raw search hits into the messages written to the activity file.

    Args:
        raw: the hits the agent returned, in the order it returned them. Each hit has already
            passed structural validation (`RawSearchHit`), but nothing else is guaranteed: the
            agent may return a message twice (two search pages, or a page re-read after a
            retry), and the connector may include bot or system messages, thread replies and
            messages just outside the week.
        week: the ISO week being collected. `week.contains(moment, tz)` is True for Monday to
            Friday of the week in tz; `week.workdays()` lists those dates.
        tz: the configured timezone (`timezone` in `config.yaml`). Calendar days are decided
            here, once, as the Go core does at its Slack edge.

    Returns:
        The messages to write, as contract `Message` objects. Build each one with
        `contract.message_from_hit(hit, tz)`, which sets `posted_at` in tz and fills empty
        strings for a missing `channel_name` or `permalink`. Order does not matter: the
        caller sorts by `ts`.

    Decisions to make, and their trade-offs:

    - Deduplication by `ts`. Slack identifies a message by channel and `ts`, so the strict key
      is `(channel_id, ts)`; `ts` alone is almost always unique for one user's messages but
      not guaranteed across channels. A duplicate kept twice doubles that message's weight in
      the rule policy (45 minutes per issue key per day), so it inflates hours silently.
    - Bot and system messages (`is_bot`, `subtype` such as `channel_join`, `bot_message`).
      The search is `from:<@you>`, so these should be rare; dropping them removes noise like
      join notices, but a workflow or app that posts on your behalf may be real work.
    - Thread replies (`thread_ts` set and different from `ts`). Keeping them counts
      discussion work, which is often where the issue key appears; dropping them keeps only
      top-level posts and undercounts reviews and support. The Go collector keeps them.
      A parent message may report `thread_ts == ts`; decide whether that is a reply.
    - Clamping to the week. Slack's `after:`/`before:` dates are evaluated in the workspace
      user's timezone, not necessarily in tz, so hits near midnight can fall on the Sunday or
      Saturday in tz. The Go collector drops anything outside Monday to Friday in tz; keeping
      them would put weekend minutes on no valid day.
    - Empty text (file uploads without a comment). It has no issue key, so it only counts
      through `channel_defaults`; keeping it may still be useful for the allocation agent.

    Raises:
        NotImplementedError: until the owner implements it.
    """
    # TODO(owner): implement message normalization.
    raise NotImplementedError("normalize_messages is not implemented yet (TODO(owner))")
