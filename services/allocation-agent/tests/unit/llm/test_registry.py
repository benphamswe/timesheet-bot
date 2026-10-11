import pytest

from allocation_agent.llm import LLMClient, LLMSettings, UnknownProviderError, create, registered
from allocation_agent.llm.registry import register
from llm_fakes import FAKE_PROVIDER, FakeLLMClient, FakeRegistration


def test_deepseek_is_registered_on_import():
    assert "deepseek" in registered()


def test_unknown_provider_lists_registered_names():
    with pytest.raises(UnknownProviderError) as ei:
        create(LLMSettings(provider="nope"))
    msg = str(ei.value)
    assert "'nope'" in msg
    assert "registered providers: deepseek" in msg


def test_create_calls_the_registered_factory_with_the_settings():
    fake = FakeLLMClient()
    settings = LLMSettings(provider=FAKE_PROVIDER, model="m1", timeout_seconds=5, max_retries=0)
    with FakeRegistration(fake) as reg:
        assert FAKE_PROVIDER in registered()
        client = create(settings)
    assert client is fake
    assert isinstance(client, LLMClient)
    assert reg.settings == [settings]
    assert FAKE_PROVIDER not in registered()


@pytest.mark.parametrize("name", ["", "Fake", " fake", "fake "])
def test_register_rejects_bad_names(name):
    with pytest.raises(ValueError, match="lowercase"):
        register(name, lambda s: FakeLLMClient())
