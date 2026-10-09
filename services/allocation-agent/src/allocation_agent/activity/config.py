"""Settings for the collector: the Go core's config.yaml, overridden by env and flags.

One source of truth: `slack_user_id` and `timezone` are read from
`services/timesheet-core/config/config.yaml`, the file the Go core reads. The environment
variables `SLACK_USER_ID` and `TIMEZONE` override it, and the CLI flags override both.
Neither value is a secret.
"""

from __future__ import annotations

import os
import re
from collections.abc import Mapping
from dataclasses import dataclass
from pathlib import Path
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

import yaml

DEFAULT_TIMEZONE = "Asia/Ho_Chi_Minh"  # same default as the Go core
_SLACK_USER = re.compile(r"^[UW][A-Z0-9]+$")


class ConfigError(Exception):
    """A setting is missing or invalid. The message names the setting, never a secret."""


@dataclass(frozen=True)
class Settings:
    slack_user_id: str
    timezone: ZoneInfo
    repo_root: Path


def find_repo_root(start: Path | None = None) -> Path:
    """Walks up from start (default: this file) to the folder that holds `services/`."""
    here = (start or Path(__file__)).resolve()
    for p in [here, *here.parents]:
        if (p / "services" / "timesheet-core").is_dir() and (p / "Makefile").is_file():
            return p
    raise ConfigError("cannot find the repository root (a folder with services/timesheet-core)")


def default_config_path(repo_root: Path) -> Path:
    return repo_root / "services" / "timesheet-core" / "config" / "config.yaml"


def _read_yaml(path: Path) -> Mapping[str, object]:
    if not path.is_file():
        return {}
    data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    if not isinstance(data, Mapping):
        raise ConfigError(f"{path} is not a YAML mapping")
    return data


def load_settings(
    *,
    config_path: Path | None = None,
    slack_user_id: str | None = None,
    timezone: str | None = None,
    env: Mapping[str, str] | None = None,
    repo_root: Path | None = None,
) -> Settings:
    """Resolves settings with the precedence flag > env > config.yaml > default."""
    env = os.environ if env is None else env
    root = repo_root or find_repo_root()
    yml = _read_yaml(config_path or default_config_path(root))

    user = (
        slack_user_id or env.get("SLACK_USER_ID") or str(yml.get("slack_user_id") or "")
    ).strip()
    if not user:
        raise ConfigError(
            "slack_user_id is not set: set it in config.yaml, SLACK_USER_ID or --slack-user-id"
        )
    if not _SLACK_USER.match(user):
        raise ConfigError("slack_user_id must be a Slack member id such as U0123ABCD")

    tz_name = (
        timezone or env.get("TIMEZONE") or str(yml.get("timezone") or "") or DEFAULT_TIMEZONE
    ).strip()
    try:
        tz = ZoneInfo(tz_name)
    except (ZoneInfoNotFoundError, ValueError):
        raise ConfigError(f"unknown timezone {tz_name!r}") from None

    return Settings(slack_user_id=user, timezone=tz, repo_root=root)
