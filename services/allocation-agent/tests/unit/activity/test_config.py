import pytest

from allocation_agent.activity.config import ConfigError, find_repo_root, load_settings


@pytest.fixture
def root(tmp_path):
    (tmp_path / "services" / "timesheet-core" / "config").mkdir(parents=True)
    (tmp_path / "Makefile").write_text("", encoding="utf-8")
    return tmp_path


def _write_cfg(root, text):
    path = root / "services" / "timesheet-core" / "config" / "config.yaml"
    path.write_text(text, encoding="utf-8")
    return path


def test_reads_go_config_yaml(root):
    _write_cfg(root, "timezone: Europe/Berlin\nslack_user_id: U0123ABCD\n")
    s = load_settings(env={}, repo_root=root)
    assert (s.slack_user_id, s.timezone.key, s.repo_root) == ("U0123ABCD", "Europe/Berlin", root)


def test_precedence_flag_over_env_over_yaml(root):
    _write_cfg(root, "timezone: Europe/Berlin\nslack_user_id: U0000YAML\n")
    env = {"SLACK_USER_ID": "U0000ENV", "TIMEZONE": "UTC"}
    s = load_settings(env=env, repo_root=root)
    assert (s.slack_user_id, s.timezone.key) == ("U0000ENV", "UTC")
    s = load_settings(env=env, repo_root=root, slack_user_id="U0000FLAG", timezone="Asia/Tokyo")
    assert (s.slack_user_id, s.timezone.key) == ("U0000FLAG", "Asia/Tokyo")


def test_default_timezone(root):
    s = load_settings(env={"SLACK_USER_ID": "U0123ABCD"}, repo_root=root)
    assert s.timezone.key == "Asia/Ho_Chi_Minh"


@pytest.mark.parametrize(
    ("yaml_text", "match"),
    [
        ("", "slack_user_id is not set"),
        ('slack_user_id: "REPLACE_ME"\n', "Slack member id"),
        ("slack_user_id: U0123ABCD\ntimezone: Mars/Base\n", "unknown timezone"),
    ],
)
def test_errors(root, yaml_text, match):
    _write_cfg(root, yaml_text)
    with pytest.raises(ConfigError, match=match):
        load_settings(env={}, repo_root=root)


def test_find_repo_root_walks_up(root):
    deep = root / "services" / "allocation-agent" / "src"
    deep.mkdir(parents=True)
    assert find_repo_root(deep) == root
