import asyncio
import json
import os
import time
from collections import deque

import nats

NATS_URL = os.environ.get("NATS_URL", "nats://localhost:4222")
SUBJECT = "accidents"
WINDOW_SECONDS = 300
REPORT_INTERVAL = 30


def _evict_old(window: deque, now: float) -> None:
    cutoff = now - WINDOW_SECONDS
    while window and window[0][0] < cutoff:
        window.popleft()


def _report(window: deque, now: float) -> None:
    _evict_old(window, now)
    count = len(window)
    if count == 0:
        print(f"[window] empty — no events in last {WINDOW_SECONDS}s")
        return

    total_dead = sum(entry[1]["dead"] for entry in window)
    total_injured = sum(entry[1]["injured"] for entry in window)
    avg_injured = total_injured / count

    oldest_ts = window[0][0]
    newest_ts = window[-1][0]
    oldest_str = time.strftime("%H:%M:%S", time.localtime(oldest_ts))
    newest_str = time.strftime("%H:%M:%S", time.localtime(newest_ts))

    print(
        f"[window] count={count}  sum_dead={total_dead}"
        f"  avg_injured={avg_injured:.2f}"
        f"  range={oldest_str}–{newest_str}"
    )


async def run() -> None:
    window: deque = deque()

    async def message_handler(msg) -> None:
        now = time.time()
        try:
            accident = json.loads(msg.data.decode())
        except (json.JSONDecodeError, UnicodeDecodeError):
            return
        window.append((now, accident))
        _evict_old(window, now)

    async def report_loop() -> None:
        while True:
            await asyncio.sleep(REPORT_INTERVAL)
            _report(window, time.time())

    nc = await nats.connect(
        NATS_URL,
        reconnect_time_wait=2,
        max_reconnect_attempts=-1,
        disconnected_cb=lambda: print("[nats] disconnected"),
        reconnected_cb=lambda: print("[nats] reconnected"),
        error_cb=lambda e: print(f"[nats] error: {e}"),
    )
    print(f"[nats] connected to {NATS_URL}, subscribing to '{SUBJECT}'")

    await nc.subscribe(SUBJECT, cb=message_handler)

    reporter = asyncio.create_task(report_loop())
    try:
        await asyncio.Event().wait()
    except asyncio.CancelledError:
        pass
    finally:
        reporter.cancel()
        await nc.drain()
        print("[nats] consumer stopped")


def main() -> None:
    try:
        asyncio.run(run())
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()
