import pytest

from allocation_agent.activity.config import ConfigError, load_settings
from allocation_agent.llm import LLMSettings


@pytest.fixture
def root(tmp_path):
    (tmp_path / "services" / "timesheet-core" / "config").mkdir(parents=True)
    (tmp_path / "Makefile").write_text("", encoding="utf-8")
    return tmp_path


BASE = "slack_user_id: U0123ABCD\n"
REPORT_LLM = "report:\n  channel: reports\n  parser: llm\n"


def _load(root, yaml_text, env=None, **kw):
    path = root / "services" / "timesheet-core" / "config" / "config.yaml"
    path.write_text(BASE + yaml_text, encoding="utf-8")
    return load_settings(env=env or {}, repo_root=root, **kw)


def test_defaults_to_manual(root):
    s = _load(root, "report:\n  channel: reports\n")
    assert s.parser == "manual"
    assert s.llm == LLMSettings()


def test_llm_block_from_yaml(root):
    s = _load(
        root,
        REPORT_LLM + "llm:\n  provider: DeepSeek\n  model: deepseek-v4-pro\n"
        '  base_url: ""\n  timeout_seconds: 30\n  max_retries: 1\n',
    )
    assert s.parser == "llm"
    assert s.llm == LLMSettings(
        provider="deepseek", model="deepseek-v4-pro", base_url="", timeout_seconds=30.0,
        max_retries=1,
    )  # fmt: skip


def test_llm_defaults_when_block_absent(root):
    s = _load(root, REPORT_LLM)
    assert s.llm == LLMSettings(provider="deepseek", model="", base_url="", timeout_seconds=60.0)


def test_precedence_flag_over_env_over_yaml(root):
    yml = REPORT_LLM + "llm:\n  provider: yamlp\n  model: yamlm\n  timeout_seconds: 10\n"
    env = {
        "LLM_PROVIDER": "envp",
        "LLM_MODEL": "envm",
        "LLM_BASE_URL": "https://proxy.example.com",
        "LLM_TIMEOUT_SECONDS": "20",
        "LLM_MAX_RETRIES": "0",
    }
    s = _load(root, yml, env=env)
    assert s.llm == LLMSettings(
        provider="envp", model="envm", base_url="https://proxy.example.com",
        timeout_seconds=20.0, max_retries=0,
    )  # fmt: skip
    s = _load(root, yml, env=env, llm_provider="flagp", llm_model="flagm")
    assert (s.llm.provider, s.llm.model) == ("flagp", "flagm")


def test_parser_comes_from_yaml_only(root):
    yml = "report:\n  channel: reports\n  parser: manual\n"
    assert _load(root, yml, env={"REPORT_PARSER": "llm"}).parser == "manual"
    assert _load(root, REPORT_LLM, env={"REPORT_PARSER": "manual"}).parser == "llm"
    with pytest.raises(TypeError):
        _load(root, yml, parser="llm")  # no override parameter


def test_llm_block_is_ignored_for_manual(root):
    assert _load(root, "report:\n  channel: reports\nllm: [broken]\n").parser == "manual"


@pytest.mark.parametrize(
    ("yaml_text", "env", "match"),
    [
        ("report:\n  channel: r\n  parser: regex\n", {}, "report.parser 'regex' is unknown"),
        (REPORT_LLM + "llm: [x]\n", {}, "llm in config.yaml must be a mapping"),
        (REPORT_LLM + "llm:\n  timeout_seconds: 0\n", {}, "greater than 0"),
        (REPORT_LLM + "llm:\n  timeout_seconds: true\n", {}, "must be a number"),
        (REPORT_LLM + "llm:\n  timeout_seconds: soon\n", {}, "must be a number"),
        (REPORT_LLM, {"LLM_TIMEOUT_SECONDS": "x"}, "llm.timeout_seconds must be a number"),
        (REPORT_LLM + "llm:\n  max_retries: -1\n", {}, "from 0 to 10"),
        (REPORT_LLM + "llm:\n  max_retries: 1.5\n", {}, "whole number"),
        (REPORT_LLM + "llm:\n  max_retries: 11\n", {}, "from 0 to 10"),
        (REPORT_LLM + "llm:\n  base_url: http://x.example.com\n", {}, "https://"),
    ],
)
def test_errors(root, yaml_text, env, match):
    with pytest.raises(ConfigError, match=match):
        _load(root, yaml_text, env=env)
