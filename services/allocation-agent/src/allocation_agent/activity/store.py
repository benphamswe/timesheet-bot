"""Writes the activity-week file atomically."""

from __future__ import annotations

import contextlib
import os
import tempfile
from pathlib import Path

from allocation_agent.activity.contract import ActivityWeek


def output_path(out_dir: Path, week: str) -> Path:
    return out_dir / f"{week}.json"


def write_atomic(doc: ActivityWeek, out_dir: Path) -> Path:
    """Writes `<out_dir>/<week>.json` so a reader sees the old file or the new one, never half.

    The body goes to a temporary file in the same folder, is flushed and fsynced, then renamed
    over the target with `os.replace` (atomic on POSIX within one file system). The file holds
    message text, so it is created with mode 0600.
    """
    out_dir.mkdir(parents=True, exist_ok=True)
    target = output_path(out_dir, doc.week)
    body = doc.to_json().encode("utf-8")

    fd, tmp_name = tempfile.mkstemp(dir=out_dir, prefix=f".{doc.week}.", suffix=".tmp")
    try:
        with os.fdopen(fd, "wb") as f:
            f.write(body)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(tmp_name, 0o600)
        os.replace(tmp_name, target)
    except BaseException:
        with contextlib.suppress(FileNotFoundError):
            os.unlink(tmp_name)
        raise

    dir_fd = os.open(out_dir, os.O_RDONLY)
    try:
        os.fsync(dir_fd)
    finally:
        os.close(dir_fd)
    return target
