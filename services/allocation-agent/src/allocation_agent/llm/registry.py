"""Provider registry (Factory + Registry): `llm.provider` in config picks the adapter.

Adapters call `register(name, factory)` when their module is imported; `llm/__init__.py`
imports every built-in adapter. Tests register a fake under their own name.
"""

from __future__ import annotations

from collections.abc import Callable

from allocation_agent.llm.port import LLMClient, UnknownProviderError
from allocation_agent.llm.settings import LLMSettings

Factory = Callable[[LLMSettings], LLMClient]

_FACTORIES: dict[str, Factory] = {}


def register(name: str, factory: Factory) -> None:
    """Registers factory under name, replacing an earlier registration of the same name."""
    if not name or name != name.strip().lower():
        raise ValueError(f"provider name {name!r} must be lowercase with no spaces")
    _FACTORIES[name] = factory


def unregister(name: str) -> None:
    """Removes name if registered. For tests."""
    _FACTORIES.pop(name, None)


def registered() -> list[str]:
    """The registered provider names, sorted."""
    return sorted(_FACTORIES)


def create(settings: LLMSettings) -> LLMClient:
    """Builds the client for `settings.provider`.

    Raises:
        UnknownProviderError: the name is not registered; the message lists the registered ones.
        LLMError: whatever the provider factory raises (for example a missing API key).
    """
    factory = _FACTORIES.get(settings.provider)
    if factory is None:
        known = ", ".join(registered()) or "(none)"
        raise UnknownProviderError(
            f"unknown llm.provider {settings.provider!r}; registered providers: {known}"
        )
    return factory(settings)
