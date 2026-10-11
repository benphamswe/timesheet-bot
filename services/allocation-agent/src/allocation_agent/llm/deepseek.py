"""DeepSeek adapter for the `LLMClient` port, over httpx.

API facts, checked against DeepSeek's official docs on 2026-10-10:

- Base URL `https://api.deepseek.com`; OpenAI-compatible `POST /chat/completions` with
  `Authorization: Bearer ${DEEPSEEK_API_KEY}` (https://api-docs.deepseek.com/).
- Models: `deepseek-flash` and `deepseek-v4-pro`. `deepseek-chat` and `deepseek-reasoner` are no
  longer listed (https://api-docs.deepseek.com/quick_start/pricing,
  https://api-docs.deepseek.com/api/create-chat-completion).
- `thinking.type` defaults to `enabled`; this adapter sends `disabled`, because extraction needs
  no reasoning and temperature has no effect in thinking mode (same page).
- JSON mode: `response_format={"type": "json_object"}`; the prompt must contain the word "json"
  and should show an example; set `max_tokens` so the JSON is not cut off; the API may return
  empty content (https://api-docs.deepseek.com/guides/json_mode).
- Errors: 400 invalid format, 401 authentication fails, 402 insufficient balance, 422 invalid
  parameters, 429 rate limit, 500 server error, 503 server overloaded
  (https://api-docs.deepseek.com/quick_start/error_codes). While a request waits under the
  concurrency limit the server sends empty lines before the body
  (https://api-docs.deepseek.com/quick_start/rate_limit); JSON parsing ignores them.

Why httpx and not the `openai` SDK: one documented endpoint is all this needs. httpx gives exact
control of the retry policy (429 and 5xx only, bounded) and of what goes into an error, and
`httpx.MockTransport` tests the request shape without a network. The SDK would add its own
retry layer and several transitive packages, and DeepSeek's `thinking` field would still have
to go through `extra_body`.

Retries: a chat completion creates nothing, so 429 and 5xx are retried up to
`max_retries` times with backoff. Timeouts and connection errors are not retried.
"""

from __future__ import annotations

import json
import os
import re
import time
from collections.abc import Callable, Mapping
from typing import Any

import httpx

from allocation_agent.llm.port import (
    LLMAuthError,
    LLMError,
    LLMRateLimitError,
    LLMResponseError,
    LLMUnavailableError,
)
from allocation_agent.llm.registry import register
from allocation_agent.llm.settings import LLMSettings

PROVIDER = "deepseek"
API_KEY_ENV = "DEEPSEEK_API_KEY"
DEFAULT_BASE_URL = "https://api.deepseek.com"
DEFAULT_MODEL = "deepseek-flash"
CURRENT_MODELS = ("deepseek-flash", "deepseek-v4-pro")
DEFAULT_MAX_TOKENS = 8192  # the documented non-thinking default; a week's lines fit easily
MAX_BACKOFF_SECONDS = 30.0

_IDENT = re.compile(r"^[A-Za-z0-9_.-]{1,64}$")


def _error_code(resp: httpx.Response) -> str:
    """The `error.code` or `error.type` identifier of an error body, never its message."""
    try:
        err = resp.json().get("error") or {}
    except (ValueError, AttributeError):
        return ""
    if not isinstance(err, Mapping):
        return ""
    for key in ("code", "type"):
        val = err.get(key)
        if isinstance(val, str) and _IDENT.match(val):
            return f", {key}={val}"
    return ""


def _retry_after(resp: httpx.Response, attempt: int) -> float:
    header = resp.headers.get("retry-after", "")
    try:
        wait = float(header)
    except ValueError:
        wait = float(2**attempt)
    return max(0.0, min(wait, MAX_BACKOFF_SECONDS))


class DeepSeekClient:
    """`LLMClient` for DeepSeek's chat completions in JSON mode, temperature 0, no thinking."""

    def __init__(
        self,
        *,
        api_key: str,
        model: str = DEFAULT_MODEL,
        base_url: str = DEFAULT_BASE_URL,
        timeout_seconds: float = 60.0,
        max_retries: int = 2,
        max_tokens: int = DEFAULT_MAX_TOKENS,
        transport: httpx.BaseTransport | None = None,
        sleep: Callable[[float], None] = time.sleep,
    ) -> None:
        if not api_key.strip():
            raise LLMAuthError(f"{API_KEY_ENV} is not set")
        base = (base_url or DEFAULT_BASE_URL).rstrip("/")
        if not base.startswith("https://"):
            raise LLMError("llm.base_url must start with https:// (the API key is sent to it)")
        if timeout_seconds <= 0:
            raise LLMError("llm.timeout_seconds must be greater than 0")
        if max_retries < 0:
            raise LLMError("llm.max_retries must be 0 or more")
        self._key = api_key.strip()
        self.model = model or DEFAULT_MODEL
        self.base_url = base
        self.timeout_seconds = timeout_seconds
        self.max_retries = max_retries
        self.max_tokens = max_tokens
        self._transport = transport
        self._sleep = sleep

    def __repr__(self) -> str:  # never shows the key
        return f"DeepSeekClient(model={self.model!r}, base_url={self.base_url!r})"

    @classmethod
    def from_settings(
        cls,
        settings: LLMSettings,
        *,
        env: Mapping[str, str] | None = None,
        transport: httpx.BaseTransport | None = None,
        sleep: Callable[[float], None] = time.sleep,
    ) -> DeepSeekClient:
        """Builds the client from config; the key comes from DEEPSEEK_API_KEY only."""
        env = os.environ if env is None else env
        key = (env.get(API_KEY_ENV) or "").strip()
        if not key:
            raise LLMAuthError(f"{API_KEY_ENV} is not set")
        return cls(
            api_key=key,
            model=settings.model or DEFAULT_MODEL,
            base_url=settings.base_url or DEFAULT_BASE_URL,
            timeout_seconds=settings.timeout_seconds,
            max_retries=settings.max_retries,
            transport=transport,
            sleep=sleep,
        )

    def request_body(self, system: str, user: str) -> dict[str, Any]:
        return {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
            "temperature": 0,
            "response_format": {"type": "json_object"},
            "thinking": {"type": "disabled"},
            "max_tokens": self.max_tokens,
            "stream": False,
        }

    def complete_json(self, system: str, user: str, *, schema_name: str) -> dict[str, Any]:
        resp = self._post(self.request_body(system, user), schema_name)
        return self._parse(resp, schema_name)

    def _post(self, body: dict[str, Any], schema_name: str) -> httpx.Response:
        headers = {"Authorization": f"Bearer {self._key}", "Content-Type": "application/json"}
        url = f"{self.base_url}/chat/completions"
        where = f"DeepSeek ({schema_name})"
        with httpx.Client(timeout=self.timeout_seconds, transport=self._transport) as http:
            for attempt in range(self.max_retries + 1):
                try:
                    resp = http.post(url, headers=headers, json=body)
                except httpx.TimeoutException:
                    raise LLMUnavailableError(
                        f"{where}: no answer within {self.timeout_seconds:g}s (llm.timeout_seconds)"
                    ) from None
                except httpx.HTTPError as e:
                    raise LLMUnavailableError(
                        f"{where}: request failed ({type(e).__name__})"
                    ) from None
                status = resp.status_code
                retryable = status == 429 or status >= 500
                if retryable and attempt < self.max_retries:
                    self._sleep(_retry_after(resp, attempt))
                    continue
                break
        if status == 200:
            return resp
        tries = f"after {self.max_retries + 1} attempt(s)"
        if status == 401:
            raise LLMAuthError(f"{where}: the API key was rejected (HTTP 401); check {API_KEY_ENV}")
        if status == 402:
            raise LLMAuthError(f"{where}: the account has insufficient balance (HTTP 402)")
        if status == 429:
            raise LLMRateLimitError(f"{where}: rate limited (HTTP 429) {tries}")
        if status >= 500:
            raise LLMUnavailableError(f"{where}: server error (HTTP {status}) {tries}")
        hint = ""
        if status in (400, 404, 422):
            hint = f"; check llm.model (current models: {', '.join(CURRENT_MODELS)})"
        raise LLMResponseError(f"{where}: request refused (HTTP {status}{_error_code(resp)}){hint}")

    def _parse(self, resp: httpx.Response, schema_name: str) -> dict[str, Any]:
        where = f"DeepSeek ({schema_name})"
        try:
            data = resp.json()
            choice = data["choices"][0]
            finish = choice.get("finish_reason")
            content = choice["message"]["content"]
        except (ValueError, KeyError, IndexError, TypeError, AttributeError):
            raise LLMResponseError(f"{where}: the response is not a chat completion") from None
        if finish != "stop":
            extra = " (raise max_tokens)" if finish == "length" else ""
            raise LLMResponseError(f"{where}: finish_reason={finish!r}{extra}")
        if not isinstance(content, str) or not content.strip():
            raise LLMResponseError(f"{where}: empty content")
        try:
            out = json.loads(content)
        except json.JSONDecodeError:
            raise LLMResponseError(f"{where}: the content is not valid JSON") from None
        if not isinstance(out, dict):
            raise LLMResponseError(f"{where}: the content is JSON but not an object")
        return out


register(PROVIDER, DeepSeekClient.from_settings)
