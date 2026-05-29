from __future__ import annotations

import asyncio
import json
import os
import sys
import time
import tracemalloc

import psutil

_DATA_DIR = os.path.join(os.path.dirname(__file__), "..", "data")
sys.path.insert(0, os.path.abspath(_DATA_DIR))

import mock_generator  # noqa: E402

REGIONS: list[str] = mock_generator.REGIONS
N_SOURCES: int = 50
RECORDS_PER_SOURCE: int = 20
OUTPUT_PATH = os.path.join(_DATA_DIR, "collector_py_output.jsonl")


async def fetch_source(source_id: int, region: str) -> list[dict]:
    await asyncio.sleep(0.01)
    records = mock_generator.generate(RECORDS_PER_SOURCE, seed=source_id)
    for rec in records:
        rec["region"] = region
    return records


async def collect_all() -> tuple[list[dict], float, float]:
    tracemalloc.start()
    proc = psutil.Process(os.getpid())
    proc.cpu_percent(interval=None)

    tasks = [
        fetch_source(i, REGIONS[i % len(REGIONS)])
        for i in range(N_SOURCES)
    ]
    results = await asyncio.gather(*tasks)

    _, peak = tracemalloc.get_traced_memory()
    tracemalloc.stop()

    cpu = proc.cpu_percent(interval=None)
    peak_mb = peak / (1024 * 1024)

    all_records: list[dict] = []
    for batch in results:
        all_records.extend(batch)

    return all_records, peak_mb, cpu


def write_output(records: list[dict], path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as fh:
        for rec in records:
            fh.write(json.dumps(rec, ensure_ascii=False) + "\n")


def run() -> tuple[float, int, float, float]:
    t0 = time.perf_counter()
    records, peak_mb, cpu = asyncio.run(collect_all())
    wall = time.perf_counter() - t0
    write_output(records, OUTPUT_PATH)
    return wall, len(records), peak_mb, cpu


if __name__ == "__main__":
    wall, count, peak_mb, cpu = run()
    rps = count / wall if wall > 0 else 0.0
    print(f"Collected  : {count} records")
    print(f"Wall time  : {wall:.3f} s")
    print(f"Throughput : {rps:.1f} records/s")
    print(f"Peak mem   : {peak_mb:.2f} MB (tracemalloc)")
    print(f"CPU        : {cpu:.1f} %")
    print(f"Output     : {OUTPUT_PATH}")
