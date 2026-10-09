import json
import os
import stat
from pathlib import Path

import pytest

from allocation_agent.activity import store
from allocation_agent.activity.collector import build_activity_week
from support import TZ, USER, passthrough


@pytest.fixture
def doc(raw_hits, week41, fixed_now):
    return build_activity_week(
        raw_hits, week=week41, slack_user_id=USER, tz=TZ, now=fixed_now, normalize=passthrough
    )


def test_writes_week_file_with_0600_and_no_temp_left(tmp_path, doc):
    out = tmp_path / "data" / "activity"
    path = store.write_atomic(doc, out)

    assert path == out / "2026-W41.json"
    assert json.loads(path.read_text(encoding="utf-8"))["week"] == "2026-W41"
    assert stat.S_IMODE(path.stat().st_mode) == 0o600
    assert sorted(p.name for p in out.iterdir()) == ["2026-W41.json"]


def test_replaces_existing_file(tmp_path, doc):
    target = tmp_path / "2026-W41.json"
    target.write_text("old", encoding="utf-8")
    store.write_atomic(doc, tmp_path)
    assert json.loads(target.read_text(encoding="utf-8"))["schema_version"] == 1


def test_failed_rename_keeps_old_file_and_removes_temp(tmp_path, doc, monkeypatch):
    target = tmp_path / "2026-W41.json"
    target.write_text("old", encoding="utf-8")

    def boom(src, dst):
        raise OSError("disk full")

    monkeypatch.setattr(os, "replace", boom)
    with pytest.raises(OSError, match="disk full"):
        store.write_atomic(doc, tmp_path)

    assert target.read_text(encoding="utf-8") == "old"
    assert sorted(p.name for p in tmp_path.iterdir()) == ["2026-W41.json"]


def test_rename_is_the_last_step(tmp_path, doc, monkeypatch):
    """The target only ever appears through os.replace from a temp file in the same folder."""
    calls = []
    real_replace = os.replace

    def spy(src, dst):
        calls.append((os.path.dirname(src), os.path.basename(src), dst))
        assert json.loads(Path(src).read_text(encoding="utf-8"))["week"] == "2026-W41"
        real_replace(src, dst)

    monkeypatch.setattr(os, "replace", spy)
    store.write_atomic(doc, tmp_path)

    [(src_dir, src_name, dst)] = calls
    assert src_dir == str(tmp_path)
    assert src_name.startswith(".2026-W41.") and src_name.endswith(".tmp")
    assert dst == tmp_path / "2026-W41.json"
