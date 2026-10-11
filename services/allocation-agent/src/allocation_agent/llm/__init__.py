"""LLM provider port, registry and adapters.

Callers use `create(settings)` and the `LLMClient` / `LLMError` types only, never an adapter.
Importing this package registers the built-in providers (`providers.py`).
"""

from allocation_agent.llm import providers  # noqa: F401  (registers the built-in providers)
from allocation_agent.llm.port import (
    LLMAuthError,
    LLMClient,
    LLMError,
    LLMRateLimitError,
    LLMResponseError,
    LLMUnavailableError,
    UnknownProviderError,
)
from allocation_agent.llm.registry import create, register, registered
from allocation_agent.llm.settings import LLMSettings

__all__ = [
    "LLMAuthError",
    "LLMClient",
    "LLMError",
    "LLMRateLimitError",
    "LLMResponseError",
    "LLMSettings",
    "LLMUnavailableError",
    "UnknownProviderError",
    "create",
    "register",
    "registered",
]
