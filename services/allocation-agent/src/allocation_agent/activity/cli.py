"""`collect --week 2026-W41 [--out-dir data/activity] [--dry-run]`.

`report.parser` comes from config.yaml only (the Go core reads the same key). With
report.parser=llm the flow is: collect, normalize, extract report lines with the
configured LLM (one call per message), write contract version 2. Any LLM error or invalid
line stops the run with exit 4 and writes no file; there is no fallback.
"""

from __future__ import annotations

import argparse
import asyncio
import os
import sys
from collections.abc import Sequence
from datetime import UTC, datetime
from pathlib import Path

from allocation_agent.activity.auth_env import AuthEnvError, check_claude_ai_login_env
from allocation_agent.activity.collector import Fetch, collect_week
from allocation_agent.activity.config import ConfigError, load_settings
from allocation_agent.activity.contract import ActivityWeekV2
from allocation_agent.activity.normalize import Normalizer, normalize_messages
from allocation_agent.activity.slack_mcp import CollectError, SlackMcpFetcher
from allocation_agent.activity.store import write_atomic
from allocation_agent.activity.summary import render_summary
from allocation_agent.activity.week import IsoWeek
from allocation_agent.llm import LLMClient, LLMError, UnknownProviderError, create
from allocation_agent.report.extractor import ReportExtractionError, ReportExtractor
from allocation_agent.report.models import ParserInfo
from allocation_agent.report.summary import render_report_summary

EXIT_OK, EXIT_USAGE, EXIT_FAILED = 0, 2, 1
# 3 is reserved (formerly: normalize_messages not implemented); never reuse it.
EXIT_LLM_FAILED = 4  # report.parser=llm: the LLM failed or returned an invalid line


def _parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="collect",
        description="Collect your timesheet reports (summary channel and/or self-DM) for one "
        "ISO week through Claude's claude.ai Slack connector and write <out-dir>/<week>.json.",
    )
    p.add_argument("--week", required=True, help="ISO week, e.g. 2026-W41")
    p.add_argument("--out-dir", type=Path, help="default: <repo root>/data/activity")
    p.add_argument("--dry-run", action="store_true", help="print counts only; write nothing")
    p.add_argument(
        "--config", type=Path, help="default: services/timesheet-core/config/config.yaml"
    )
    p.add_argument("--slack-user-id", help="overrides SLACK_USER_ID and config.yaml")
    p.add_argument("--timezone", help="overrides TIMEZONE and config.yaml")
    p.add_argument(
        "--report-channel", help="summary channel without '#'; overrides REPORT_CHANNEL, config"
    )
    p.add_argument(
        "--include-self-dm",
        action=argparse.BooleanOptionalAction,
        default=None,
        help="also search your self-DM; overrides report.include_self_dm",
    )
    p.add_argument("--llm-provider", help="overrides LLM_PROVIDER and llm.provider (parser llm)")
    p.add_argument("--llm-model", help="overrides LLM_MODEL and llm.model (parser llm)")
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
        # First, before any settings or Claude session: fail fast on a credential that hides the
        # claude.ai Slack connector. Names only, never values.
        check_claude_ai_login_env(os.environ if env is None else env)
    except AuthEnvError as e:
        print(f"collect: {e}", file=sys.stderr)
        return EXIT_USAGE
    try:
        week = IsoWeek.parse(args.week)
        settings = load_settings(
            config_path=args.config,
            slack_user_id=args.slack_user_id,
            timezone=args.timezone,
            report_channel=args.report_channel,
            include_self_dm=args.include_self_dm,
            llm_provider=args.llm_provider,
            llm_model=args.llm_model,
            env=env,
        )
    except (ValueError, ConfigError) as e:
        print(f"collect: {e}", file=sys.stderr)
        return EXIT_USAGE

    # Build the LLM client before any Claude or Slack usage, so a missing key fails fast.
    llm: LLMClient | None = None
    if settings.parser == "llm":
        try:
            llm = create(settings.llm)
        except UnknownProviderError as e:
            print(f"collect: {e}", file=sys.stderr)
            return EXIT_USAGE
        except LLMError as e:
            print(f"collect: llm: {e}", file=sys.stderr)
            return EXIT_LLM_FAILED

    if fetch is None:
        fetch = SlackMcpFetcher(max_turns=args.max_turns, model=args.model).fetch

    try:
        doc = asyncio.run(
            collect_week(
                week,
                slack_user_id=settings.slack_user_id,
                tz=settings.timezone,
                scope=settings.scope,
                fetch=fetch,
                clock=clock,
                normalize=normalize,
            )
        )
    except CollectError as e:
        print(f"collect: {e}", file=sys.stderr)
        return EXIT_FAILED

    if llm is None:
        if args.dry_run:
            sys.stdout.write(render_summary(doc, week, settings.timezone, settings.scope))
            return EXIT_OK
        out_dir = args.out_dir or settings.repo_root / "data" / "activity"
        path = write_atomic(doc, out_dir)
        print(f"wrote {path} ({len(doc.messages)} messages)")
        return EXIT_OK

    try:
        lines = ReportExtractor(llm).extract(doc.messages, week, settings.timezone)
    except ReportExtractionError as e:
        print(f"collect: report extraction stopped: {e}", file=sys.stderr)
        return EXIT_LLM_FAILED
    parser_info = ParserInfo(
        kind="llm",
        provider=settings.llm.provider,
        # The model the adapter actually uses; "default" if a provider does not expose it.
        model=str(getattr(llm, "model", "") or settings.llm.model or "default"),
    )

    if args.dry_run:
        sys.stdout.write(render_summary(doc, week, settings.timezone, settings.scope))
        sys.stdout.write(render_report_summary(lines, week, parser_info))
        return EXIT_OK

    v2 = ActivityWeekV2.from_v1(doc, parser=parser_info, report_lines=lines)
    out_dir = args.out_dir or settings.repo_root / "data" / "activity"
    path = write_atomic(v2, out_dir)
    print(f"wrote {path} ({len(doc.messages)} messages, {len(lines)} report lines, schema v2)")
    return EXIT_OK
