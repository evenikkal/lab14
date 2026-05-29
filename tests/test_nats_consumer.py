import io
import time
from collections import deque
from contextlib import redirect_stdout

from nats_consumer import WINDOW_SECONDS, _evict_old, _report


def test_evict_old_removes_stale_entries():
    now = 1_000_000.0
    window = deque()
    window.append((now - WINDOW_SECONDS - 10, {"dead": 1}))
    window.append((now - WINDOW_SECONDS + 10, {"dead": 0}))

    _evict_old(window, now)

    assert len(window) == 1
    remaining_ts, _ = window[0]
    assert remaining_ts > now - WINDOW_SECONDS


def test_evict_old_empty_window_is_noop():
    window = deque()
    _evict_old(window, 1_000_000.0)
    assert len(window) == 0


def test_evict_old_all_fresh_entries_remain():
    now = 1_000_000.0
    window = deque()
    window.append((now - 10, {"dead": 0}))
    window.append((now - 5, {"dead": 1}))

    _evict_old(window, now)

    assert len(window) == 2


def test_evict_old_all_stale_entries_removed():
    now = 1_000_000.0
    window = deque()
    window.append((now - WINDOW_SECONDS - 100, {"dead": 0}))
    window.append((now - WINDOW_SECONDS - 1, {"dead": 1}))

    _evict_old(window, now)

    assert len(window) == 0


def test_report_empty_window_prints_empty_message():
    window = deque()
    buf = io.StringIO()
    with redirect_stdout(buf):
        _report(window, time.time())
    assert "empty" in buf.getvalue()


def test_report_correct_count():
    now = 1_000_000.0
    window = deque()
    window.append((now - 10, {"dead": 1, "injured": 3}))
    window.append((now - 5, {"dead": 0, "injured": 2}))

    buf = io.StringIO()
    with redirect_stdout(buf):
        _report(window, now)

    assert "count=2" in buf.getvalue()


def test_report_correct_sum_dead():
    now = 1_000_000.0
    window = deque()
    window.append((now - 10, {"dead": 2, "injured": 4}))
    window.append((now - 5, {"dead": 3, "injured": 5}))

    buf = io.StringIO()
    with redirect_stdout(buf):
        _report(window, now)

    assert "sum_dead=5" in buf.getvalue()


def test_report_evicts_stale_before_counting():
    now = 1_000_000.0
    window = deque()
    window.append((now - WINDOW_SECONDS - 1, {"dead": 10, "injured": 10}))
    window.append((now - 10, {"dead": 1, "injured": 2}))

    buf = io.StringIO()
    with redirect_stdout(buf):
        _report(window, now)

    assert "count=1" in buf.getvalue()
    assert "sum_dead=1" in buf.getvalue()
