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


REPORT = "report:\n  channel: timesheet-reports\n  include_self_dm: true\n"
CH = {"REPORT_CHANNEL": "reports"}  # a minimal report source for tests about other keys


def test_reads_go_config_yaml(root):
    _write_cfg(root, "timezone: Europe/Berlin\nslack_user_id: U0123ABCD\n" + REPORT)
    s = load_settings(env={}, repo_root=root)
    assert (s.slack_user_id, s.timezone.key, s.repo_root) == ("U0123ABCD", "Europe/Berlin", root)
    assert (s.scope.channel, s.scope.include_self_dm) == ("timesheet-reports", True)


def test_precedence_flag_over_env_over_yaml(root):
    _write_cfg(root, "timezone: Europe/Berlin\nslack_user_id: U0000YAML\n")
    env = {"SLACK_USER_ID": "U0000ENV", "TIMEZONE": "UTC", **CH}
    s = load_settings(env=env, repo_root=root)
    assert (s.slack_user_id, s.timezone.key) == ("U0000ENV", "UTC")
    s = load_settings(env=env, repo_root=root, slack_user_id="U0000FLAG", timezone="Asia/Tokyo")
    assert (s.slack_user_id, s.timezone.key) == ("U0000FLAG", "Asia/Tokyo")


def test_default_timezone(root):
    s = load_settings(env={"SLACK_USER_ID": "U0123ABCD", **CH}, repo_root=root)
    assert s.timezone.key == "Asia/Ho_Chi_Minh"


@pytest.mark.parametrize(
    ("yaml_text", "match"),
    [
        ("", "slack_user_id is not set"),
        ('slack_user_id: "REPLACE_ME"\n', "Slack member id"),
        ("slack_user_id: U0123ABCD\ntimezone: Mars/Base\n", "unknown timezone"),
        ("slack_user_id: U0123ABCD\n", "no report source"),
        (
            'slack_user_id: U0123ABCD\nreport:\n  channel: ""\n  include_self_dm: false\n',
            "no report source",
        ),
        ("slack_user_id: U0123ABCD\nreport:\n  include_self_dm: yes please\n", "true or false"),
        ("slack_user_id: U0123ABCD\nreport: [x]\n", "must be a mapping"),
        ("slack_user_id: U0123ABCD\nreport:\n  channel: Bad Name\n", "not a Slack channel"),
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


def test_report_precedence_flag_over_env_over_yaml(root):
    _write_cfg(root, "slack_user_id: U0123ABCD\n" + REPORT)
    s = load_settings(env={"REPORT_CHANNEL": "env-reports"}, repo_root=root)
    assert (s.scope.channel, s.scope.include_self_dm) == ("env-reports", True)
    s = load_settings(
        env={"REPORT_CHANNEL": "env-reports"},
        repo_root=root,
        report_channel="#flag-reports",
        include_self_dm=False,
    )
    assert (s.scope.channel, s.scope.include_self_dm) == ("flag-reports", False)


def test_include_self_dm_defaults_to_false_when_absent(root):
    _write_cfg(root, "slack_user_id: U0123ABCD\nreport:\n  channel: reports\n")
    s = load_settings(env={}, repo_root=root)
    assert (s.scope.channel, s.scope.include_self_dm) == ("reports", False)


def test_self_dm_flag_alone_is_a_source(root):
    _write_cfg(root, "slack_user_id: U0123ABCD\n")
    s = load_settings(env={}, repo_root=root, include_self_dm=True)
    assert (s.scope.channel, s.scope.include_self_dm) == (None, True)
