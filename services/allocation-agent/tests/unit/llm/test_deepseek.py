"""DeepSeekClient against httpx.MockTransport: request shape, retries, error mapping."""

import json

import httpx
import pytest

from allocation_agent.llm import (
    LLMAuthError,
    LLMError,
    LLMRateLimitError,
    LLMResponseError,
    LLMSettings,
    LLMUnavailableError,
    create,
)
from allocation_agent.llm.deepseek import API_KEY_ENV, DEFAULT_MODEL, DeepSeekClient

KEY = "test-key-123"  # fake
SYSTEM = "Answer in json."
USER = "GCM-12 confidential design note"


def _completion(content, finish="stop"):
    return {
        "id": "x",
        "object": "chat.completion",
        "choices": [
            {
                "index": 0,
                "finish_reason": finish,
                "message": {"role": "assistant", "content": content},
            }
        ],
        "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
    }


class Server:
    """Answers each request with the next queued (status, body); records requests."""

    def __init__(self, *responses):
        self.responses = list(responses)
        self.requests: list[httpx.Request] = []

    def __call__(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        status, body = self.responses[min(len(self.requests) - 1, len(self.responses) - 1)]
        if isinstance(body, Exception):
            raise body
        if isinstance(body, bytes):
            return httpx.Response(status, content=body)
        return httpx.Response(status, json=body)


def _client(server, sleeps=None, **kw):
    return DeepSeekClient(
        api_key=KEY,
        transport=httpx.MockTransport(server),
        sleep=(sleeps.append if sleeps is not None else lambda s: None),
        **kw,
    )


def test_request_shape_and_parsed_answer():
    server = Server((200, _completion('{"lines": []}')))
    out = _client(server).complete_json(SYSTEM, USER, schema_name="report_lines")

    assert out == {"lines": []}
    (req,) = server.requests
    assert req.method == "POST"
    assert str(req.url) == "https://api.deepseek.com/chat/completions"
    assert req.headers["authorization"] == f"Bearer {KEY}"
    assert req.headers["content-type"] == "application/json"
    body = json.loads(req.content)
    assert body == {
        "model": DEFAULT_MODEL,
        "messages": [
            {"role": "system", "content": SYSTEM},
            {"role": "user", "content": USER},
        ],
        "temperature": 0,
        "response_format": {"type": "json_object"},
        "thinking": {"type": "disabled"},
        "max_tokens": 8192,
        "stream": False,
    }


def test_default_model_is_a_current_deepseek_model():
    assert DEFAULT_MODEL == "deepseek-flash"


def test_settings_model_base_url_and_timeout(monkeypatch):
    server = Server((200, _completion("{}")))
    settings = LLMSettings(
        provider="deepseek",
        model="deepseek-v4-pro",
        base_url="https://proxy.example.com/ds/",
        timeout_seconds=7,
        max_retries=0,
    )
    client = DeepSeekClient.from_settings(
        settings, env={API_KEY_ENV: KEY}, transport=httpx.MockTransport(server)
    )
    client.complete_json(SYSTEM, USER, schema_name="s")
    (req,) = server.requests
    assert str(req.url) == "https://proxy.example.com/ds/chat/completions"
    assert json.loads(req.content)["model"] == "deepseek-v4-pro"
    assert req.extensions["timeout"]["read"] == 7


@pytest.mark.parametrize("env", [{}, {API_KEY_ENV: ""}, {API_KEY_ENV: "   "}])
def test_missing_key_names_the_variable_only(env):
    with pytest.raises(LLMAuthError) as ei:
        DeepSeekClient.from_settings(LLMSettings(), env=env)
    assert str(ei.value) == "DEEPSEEK_API_KEY is not set"


def test_registry_create_reads_the_key_from_the_process_env(monkeypatch):
    monkeypatch.delenv(API_KEY_ENV, raising=False)
    with pytest.raises(LLMAuthError, match=r"^DEEPSEEK_API_KEY is not set$"):
        create(LLMSettings(provider="deepseek"))
    monkeypatch.setenv(API_KEY_ENV, KEY)
    client = create(LLMSettings(provider="deepseek", model="deepseek-v4-pro"))
    assert isinstance(client, DeepSeekClient)
    assert client.model == "deepseek-v4-pro"
    assert KEY not in repr(client)


def test_plain_http_base_url_is_refused():
    with pytest.raises(LLMError, match="https://"):
        DeepSeekClient(api_key=KEY, base_url="http://api.deepseek.com")


@pytest.mark.parametrize(
    ("status", "error", "match"),
    [
        (401, LLMAuthError, "HTTP 401.*DEEPSEEK_API_KEY"),
        (402, LLMAuthError, "insufficient balance"),
        (400, LLMResponseError, "HTTP 400.*llm.model"),
        (422, LLMResponseError, "HTTP 422.*deepseek-flash"),
        (404, LLMResponseError, "HTTP 404"),
    ],
)
def test_non_retryable_errors_are_mapped_and_not_retried(status, error, match):
    server = Server((status, {"error": {"message": USER, "type": "invalid_request_error"}}))
    sleeps: list[float] = []
    with pytest.raises(error, match=match) as ei:
        _client(server, sleeps).complete_json(SYSTEM, USER, schema_name="report_lines")
    assert len(server.requests) == 1
    assert sleeps == []
    assert USER not in str(ei.value)  # the provider's error message is never copied
    assert KEY not in str(ei.value)


def test_error_code_identifier_is_kept():
    server = Server((422, {"error": {"message": "x", "code": "invalid_model"}}))
    with pytest.raises(LLMResponseError, match="code=invalid_model"):
        _client(server).complete_json(SYSTEM, USER, schema_name="s")


def test_429_is_retried_then_raises_rate_limit():
    server = Server((429, {"error": {}}))
    sleeps: list[float] = []
    with pytest.raises(LLMRateLimitError, match="after 3 attempt"):
        _client(server, sleeps, max_retries=2).complete_json(SYSTEM, USER, schema_name="s")
    assert len(server.requests) == 3
    assert sleeps == [1.0, 2.0]


def test_retry_after_header_is_honoured_and_capped():
    def handler(request):
        handler.n += 1
        if handler.n == 1:
            return httpx.Response(429, headers={"Retry-After": "120"})
        return httpx.Response(200, json=_completion('{"ok": true}'))

    handler.n = 0
    sleeps: list[float] = []
    client = DeepSeekClient(
        api_key=KEY, transport=httpx.MockTransport(handler), sleep=sleeps.append
    )
    assert client.complete_json(SYSTEM, USER, schema_name="s") == {"ok": True}
    assert sleeps == [30.0]


def test_5xx_then_success():
    server = Server((503, {}), (500, {}), (200, _completion('{"lines": []}')))
    assert _client(server, max_retries=2).complete_json(SYSTEM, USER, schema_name="s") == {
        "lines": []
    }
    assert len(server.requests) == 3


def test_5xx_after_retries_is_unavailable():
    server = Server((500, {}))
    with pytest.raises(LLMUnavailableError, match="HTTP 500"):
        _client(server, max_retries=1).complete_json(SYSTEM, USER, schema_name="s")
    assert len(server.requests) == 2


def test_zero_retries_means_one_attempt():
    server = Server((503, {}))
    with pytest.raises(LLMUnavailableError):
        _client(server, max_retries=0).complete_json(SYSTEM, USER, schema_name="s")
    assert len(server.requests) == 1


@pytest.mark.parametrize(
    ("exc", "match"),
    [
        (httpx.ReadTimeout("t"), "no answer within"),
        (httpx.ConnectError("c"), "ConnectError"),
    ],
)
def test_transport_errors_are_unavailable_and_not_retried(exc, match):
    server = Server((0, exc))
    with pytest.raises(LLMUnavailableError, match=match):
        _client(server).complete_json(SYSTEM, USER, schema_name="s")
    assert len(server.requests) == 1


@pytest.mark.parametrize(
    ("response", "match"),
    [
        ((200, _completion('{"lines": []}', finish="length")), "finish_reason='length'"),
        ((200, _completion('{"lines": []}', finish="content_filter")), "content_filter"),
        ((200, _completion("")), "empty content"),
        ((200, _completion(None)), "empty content"),
        ((200, _completion("not json " + USER)), "not valid JSON"),
        ((200, _completion("[1, 2]")), "not an object"),
        ((200, {"choices": []}), "not a chat completion"),
        ((200, b"<html>"), "not a chat completion"),
    ],
)
def test_unusable_answers_raise_response_error_without_content(response, match):
    with pytest.raises(LLMResponseError, match=match) as ei:
        _client(Server(response)).complete_json(SYSTEM, USER, schema_name="report_lines")
    assert USER not in str(ei.value)
    assert "report_lines" in str(ei.value)


def test_leading_keep_alive_blank_lines_are_tolerated():
    body = b"\n\n" + json.dumps(_completion('{"lines": []}')).encode()
    assert _client(Server((200, body))).complete_json(SYSTEM, USER, schema_name="s") == {
        "lines": []
    }
