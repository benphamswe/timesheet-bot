"""The LLM provider port: one Strategy interface every provider adapter implements.

Callers (the report extractor) depend only on `LLMClient` and the `LLMError` family. A provider
adapter (`deepseek.py`, later others) turns its SDK or HTTP errors into these types, so the
caller's fail-stop policy does not change with the provider.

No error in this module, or raised by an adapter, may hold a prompt, a response body, message
text or an API key. Errors name the provider, the schema name, an HTTP status or an error code.
"""

from __future__ import annotations

from typing import Any, Protocol, runtime_checkable


class LLMError(Exception):
    """Any LLM failure. The caller stops on it; there is no fallback."""


class LLMAuthError(LLMError):
    """The key is missing, empty or rejected, or the account cannot be billed (not retried)."""


class LLMRateLimitError(LLMError):
    """The provider kept answering 429 after the bounded retries."""


class LLMUnavailableError(LLMError):
    """The provider kept failing (5xx), timed out or could not be reached."""


class LLMResponseError(LLMError):
    """The request was refused (4xx) or the answer is not usable JSON (empty, cut off, a list)."""


class UnknownProviderError(LLMError):
    """`llm.provider` names no registered provider."""


@runtime_checkable
class LLMClient(Protocol):
    """Completes one chat turn and returns the model's answer as a JSON object."""

    def complete_json(self, system: str, user: str, *, schema_name: str) -> dict[str, Any]:
        """Sends the system and user prompts and returns the parsed JSON object.

        Args:
            system: the system prompt. It must mention JSON (DeepSeek's JSON mode needs it).
            user: the user prompt.
            schema_name: a short name for the expected shape, used only in error messages.

        Raises:
            LLMError: a subclass for every failure; never returns partial data.
        """
        ...
