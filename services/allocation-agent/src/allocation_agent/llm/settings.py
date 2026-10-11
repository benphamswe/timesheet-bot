"""The `llm` block of config.yaml after flag and env overrides. No value is a secret."""

from __future__ import annotations

from dataclasses import dataclass

DEFAULT_PROVIDER = "deepseek"
DEFAULT_TIMEOUT_SECONDS = 60.0
DEFAULT_MAX_RETRIES = 2


@dataclass(frozen=True)
class LLMSettings:
    """Provider-neutral settings. An empty `model` or `base_url` means the provider default."""

    provider: str = DEFAULT_PROVIDER
    model: str = ""
    base_url: str = ""
    timeout_seconds: float = DEFAULT_TIMEOUT_SECONDS
    max_retries: int = DEFAULT_MAX_RETRIES
