"""FakeLLMClient: a hand-written LLMClient that records calls and can be told to fail."""

from __future__ import annotations

from collections.abc import Callable
from typing import Any

from allocation_agent.llm import LLMError, LLMSettings, register
from allocation_agent.llm.registry import unregister

FAKE_PROVIDER = "fake"
FAKE_MODEL = "fake-model"

Answer = dict[str, Any] | LLMError | Callable[[str], dict[str, Any]]


class FakeLLMClient:
    """Returns the queued answers in order; the last one repeats. An LLMError answer is raised.

    A callable answer gets the user prompt and returns the answer, so a test can answer per
    message.
    """

    def __init__(self, *answers: Answer, model: str = FAKE_MODEL) -> None:
        self.answers = list(answers) or [{"lines": []}]
        self.model = model
        self.calls: list[tuple[str, str, str]] = []

    def complete_json(self, system: str, user: str, *, schema_name: str) -> dict[str, Any]:
        self.calls.append((system, user, schema_name))
        idx = min(len(self.calls) - 1, len(self.answers) - 1)
        answer = self.answers[idx]
        if isinstance(answer, LLMError):
            raise answer
        if callable(answer):
            return answer(user)
        return answer


class FakeRegistration:
    """Registers a FakeLLMClient under FAKE_PROVIDER and remembers the settings it got."""

    def __init__(self, client: FakeLLMClient) -> None:
        self.client = client
        self.settings: list[LLMSettings] = []

    def factory(self, settings: LLMSettings) -> FakeLLMClient:
        self.settings.append(settings)
        return self.client

    def __enter__(self) -> FakeRegistration:
        register(FAKE_PROVIDER, self.factory)
        return self

    def __exit__(self, *exc: object) -> None:
        unregister(FAKE_PROVIDER)
