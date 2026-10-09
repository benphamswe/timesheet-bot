"""`collect --week 2026-W41 [--out-dir data/activity] [--dry-run]`."""

from __future__ import annotations

import argparse
import asyncio
import sys
from collections.abc import Sequence
from datetime import UTC, datetime
from pathlib import Path

from allocation_agent.activity.collector import Fetch, collect_week
from allocation_agent.activity.config import ConfigError, load_settings
from allocation_agent.activity.normalize import Normalizer, normalize_messages
from allocation_agent.activity.slack_mcp import CollectError, SlackMcpFetcher
from allocation_agent.activity.store import write_atomic
from allocation_agent.activity.summary import render_summary
from allocation_agent.activity.week import IsoWeek

EXIT_OK, EXIT_USAGE, EXIT_FAILED, EXIT_NOT_IMPLEMENTED = 0, 2, 1, 3


def _parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="collect",
        description="Collect your own Slack messages for one ISO week through Claude's "
        "claude.ai Slack connector and write <out-dir>/<week>.json.",
    )
    p.add_argument("--week", required=True, help="ISO week, e.g. 2026-W41")
    p.add_argument("--out-dir", type=Path, help="default: <repo root>/data/activity")
    p.add_argument("--dry-run", action="store_true", help="print counts only; write nothing")
    p.add_argument(
        "--config", type=Path, help="default: services/timesheet-core/config/config.yaml"
    )
    p.add_argument("--slack-user-id", help="overrides SLACK_USER_ID and config.yaml")
    p.add_argument("--timezone", help="overrides TIMEZONE and config.yaml")
    p.add_argument("--model", help="Claude model for the collection session (default: CLI default)")
    p.add_argument("--max-turns", type=int, default=40)
    return p


def main(
    argv: Sequence[str] | None = None,
    *,
    fetch: Fetch | None = None,
    normalize: Normalizer = normalize_messages,
    clock=lambda: datetime.now(UTC),
    env: dict[str, str] | None = None,
) -> int:
    args = _parser().parse_args(argv)
    try:
        week = IsoWeek.parse(args.week)
        settings = load_settings(
            config_path=args.config,
            slack_user_id=args.slack_user_id,
            timezone=args.timezone,
            env=env,
        )
    except (ValueError, ConfigError) as e:
        print(f"collect: {e}", file=sys.stderr)
        return EXIT_USAGE

    if fetch is None:
        fetch = SlackMcpFetcher(max_turns=args.max_turns, model=args.model).fetch

    try:
        doc = asyncio.run(
            collect_week(
                week,
                slack_user_id=settings.slack_user_id,
                tz=settings.timezone,
                fetch=fetch,
                clock=clock,
                normalize=normalize,
            )
        )
    except NotImplementedError as e:
        print(f"collect: {e}; see normalize.py", file=sys.stderr)
        return EXIT_NOT_IMPLEMENTED
    except CollectError as e:
        print(f"collect: {e}", file=sys.stderr)
        return EXIT_FAILED

    if args.dry_run:
        sys.stdout.write(render_summary(doc, week, settings.timezone))
        return EXIT_OK

    out_dir = args.out_dir or settings.repo_root / "data" / "activity"
    path = write_atomic(doc, out_dir)
    print(f"wrote {path} ({len(doc.messages)} messages)")
    return EXIT_OK
