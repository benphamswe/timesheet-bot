"""Settings for the collector: the Go core's config.yaml, overridden by env and flags.

One source of truth: `slack_user_id`, `timezone`, `report.channel` and
`report.include_self_dm` are read from `services/timesheet-core/config/config.yaml`, the file
the Go core reads. The environment variables `SLACK_USER_ID`, `TIMEZONE` and `REPORT_CHANNEL`
override it, and the CLI flags override both. No value is a secret.

`report.include_self_dm` is false when the key is absent, so a config with no `report` block
has no source and is rejected.

`report.parser` (`manual` | `llm`, default `manual`) comes from config.yaml only, with no
env or flag override: the Go core reads the same key from the same file to choose between
contract v1 and v2, so the two sides must never disagree. For `llm`, the `llm` block
(`provider`, `model`, `base_url`, `timeout_seconds`, `max_retries`) comes from the same file,
overridden by `LLM_PROVIDER`, `LLM_MODEL`, `LLM_BASE_URL`, `LLM_TIMEOUT_SECONDS`,
`LLM_MAX_RETRIES` and the flags `--llm-provider`, `--llm-model` (Go ignores `llm.*`). The `llm`
block is read only when the parser is `llm`. API keys are not read here; each provider adapter
reads its own variable.
"""

from __future__ import annotations

import os
import re
from collections.abc import Mapping
from dataclasses import dataclass, field
from pathlib import Path
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError

import yaml

from allocation_agent.activity.sources import ReportScope, ScopeError
from allocation_agent.llm.settings import (
    DEFAULT_MAX_RETRIES,
    DEFAULT_PROVIDER,
    DEFAULT_TIMEOUT_SECONDS,
    LLMSettings,
)

DEFAULT_TIMEZONE = "Asia/Ho_Chi_Minh"  # same default as the Go core
_SLACK_USER = re.compile(r"^[UW][A-Z0-9]+$")
PARSERS = ("manual", "llm")
MAX_LLM_RETRIES = 10


class ConfigError(Exception):
    """A setting is missing or invalid. The message names the setting, never a secret."""


@dataclass(frozen=True)
class Settings:
    slack_user_id: str
    timezone: ZoneInfo
    repo_root: Path
    scope: ReportScope
    parser: str = "manual"
    llm: LLMSettings = field(default_factory=LLMSettings)


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
    report_channel: str | None = None,
    include_self_dm: bool | None = None,
    llm_provider: str | None = None,
    llm_model: str | None = None,
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

    report = yml.get("report") or {}
    if not isinstance(report, Mapping):
        raise ConfigError("report in config.yaml must be a mapping")
    channel = report_channel or env.get("REPORT_CHANNEL") or str(report.get("channel") or "")
    if include_self_dm is None:
        yml_dm = report.get("include_self_dm", False)
        if not isinstance(yml_dm, bool):
            raise ConfigError("report.include_self_dm in config.yaml must be true or false")
        include_self_dm = yml_dm
    try:
        scope = ReportScope.build(channel, include_self_dm)
    except ScopeError as e:
        raise ConfigError(str(e)) from None

    # config.yaml only, never env or a flag: Go reads the same key (see the module docstring).
    parser_name = str(report.get("parser") or "") or "manual"
    parser_name = parser_name.strip().lower()
    if parser_name not in PARSERS:
        raise ConfigError(f"report.parser {parser_name!r} is unknown, want manual or llm")
    llm = LLMSettings()
    if parser_name == "llm":
        llm = _llm_settings(yml, env, provider=llm_provider, model=llm_model)

    return Settings(
        slack_user_id=user,
        timezone=tz,
        repo_root=root,
        scope=scope,
        parser=parser_name,
        llm=llm,
    )


def _number(name: str, env_value: str | None, yml_value: object, default: float) -> float:
    """env > yaml > default; rejects booleans and non-numbers, naming the setting."""
    if env_value is not None and env_value.strip() != "":
        try:
            return float(env_value)
        except ValueError:
            raise ConfigError(f"{name} must be a number") from None
    if yml_value is None or yml_value == "":
        return default
    if isinstance(yml_value, bool) or not isinstance(yml_value, int | float):
        raise ConfigError(f"{name} in config.yaml must be a number")
    return float(yml_value)


def _llm_settings(
    yml: Mapping[str, object],
    env: Mapping[str, str],
    *,
    provider: str | None,
    model: str | None,
) -> LLMSettings:
    block = yml.get("llm") or {}
    if not isinstance(block, Mapping):
        raise ConfigError("llm in config.yaml must be a mapping")
    name = (
        (
            provider
            or env.get("LLM_PROVIDER")
            or str(block.get("provider") or "")
            or DEFAULT_PROVIDER
        )
        .strip()
        .lower()
    )
    model_name = (model or env.get("LLM_MODEL") or str(block.get("model") or "")).strip()
    base_url = (env.get("LLM_BASE_URL") or str(block.get("base_url") or "")).strip()

    timeout = _number(
        "llm.timeout_seconds",
        env.get("LLM_TIMEOUT_SECONDS"),
        block.get("timeout_seconds"),
        DEFAULT_TIMEOUT_SECONDS,
    )
    if timeout <= 0:
        raise ConfigError("llm.timeout_seconds must be greater than 0")
    retries = _number(
        "llm.max_retries", env.get("LLM_MAX_RETRIES"), block.get("max_retries"), DEFAULT_MAX_RETRIES
    )
    if retries != int(retries) or not 0 <= retries <= MAX_LLM_RETRIES:
        raise ConfigError(f"llm.max_retries must be a whole number from 0 to {MAX_LLM_RETRIES}")
    if base_url and not base_url.startswith("https://"):
        raise ConfigError("llm.base_url must be empty or start with https://")
    return LLMSettings(
        provider=name,
        model=model_name,
        base_url=base_url,
        timeout_seconds=timeout,
        max_retries=int(retries),
    )
