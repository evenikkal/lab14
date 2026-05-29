from __future__ import annotations

import asyncio
import os
import subprocess
import sys
import time
import tracemalloc

import psutil
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

_THIS_DIR = os.path.dirname(__file__)
sys.path.insert(0, _THIS_DIR)

from main import collect_all, write_output, OUTPUT_PATH  # noqa: E402

_REPO_ROOT = os.path.join(_THIS_DIR, "..")
GO_BINARY = os.path.join(_REPO_ROOT, "collector", "collector")
CHARTS_DIR = os.path.join(_REPO_ROOT, "charts")
CHART_PATH = os.path.join(CHARTS_DIR, "benchmark.png")
N_ITERATIONS = 3


def run_go_once() -> tuple[float, float, float]:
    proc = psutil.Process(os.getpid())
    tracemalloc.start()
    proc.cpu_percent(interval=None)

    t0 = time.perf_counter()
    result = subprocess.run(
        [GO_BINARY, "--mode", "window"],
        capture_output=True,
        timeout=60,
        cwd=os.path.dirname(GO_BINARY),
    )
    wall = time.perf_counter() - t0

    _, peak = tracemalloc.get_traced_memory()
    tracemalloc.stop()
    cpu = proc.cpu_percent(interval=None)

    if result.returncode != 0:
        raise RuntimeError(f"Go binary exited {result.returncode}: {result.stderr.decode()}")

    return wall, peak / (1024 * 1024), cpu


def run_python_once() -> tuple[float, float, float]:
    proc = psutil.Process(os.getpid())
    proc.cpu_percent(interval=None)

    t0 = time.perf_counter()
    records, peak_mb, _ = asyncio.run(collect_all())
    wall = time.perf_counter() - t0
    write_output(records, OUTPUT_PATH)

    cpu = proc.cpu_percent(interval=None)
    return wall, peak_mb, cpu


def average(values: list[tuple[float, float, float]]) -> tuple[float, float, float]:
    n = len(values)
    return (
        sum(v[0] for v in values) / n,
        sum(v[1] for v in values) / n,
        sum(v[2] for v in values) / n,
    )


def benchmark_go() -> tuple[float, float, float] | None:
    if not os.path.isfile(GO_BINARY):
        print(f"[WARN] Go binary not found at {GO_BINARY} — using placeholder values")
        return None

    samples: list[tuple[float, float, float]] = []
    for i in range(N_ITERATIONS):
        try:
            sample = run_go_once()
            samples.append(sample)
            print(f"  Go iter {i+1}: wall={sample[0]:.3f}s  mem={sample[1]:.2f}MB  cpu={sample[2]:.1f}%")
        except Exception as exc:
            print(f"  [WARN] Go iter {i+1} failed: {exc} — skipping")

    if not samples:
        return None
    return average(samples)


def benchmark_python() -> tuple[float, float, float]:
    samples: list[tuple[float, float, float]] = []
    for i in range(N_ITERATIONS):
        sample = run_python_once()
        samples.append(sample)
        print(f"  Python iter {i+1}: wall={sample[0]:.3f}s  mem={sample[1]:.2f}MB  cpu={sample[2]:.1f}%")
    return average(samples)


def generate_chart(
    go_avg: tuple[float, float, float] | None,
    py_avg: tuple[float, float, float],
) -> None:
    os.makedirs(CHARTS_DIR, exist_ok=True)

    go_wall, go_mem, go_cpu = go_avg if go_avg else (0.0, 0.0, 0.0)
    py_wall, py_mem, py_cpu = py_avg

    labels = ["Go", "Python"]
    time_vals = [go_wall, py_wall]
    mem_vals = [go_mem, py_mem]
    cpu_vals = [go_cpu, py_cpu]

    fig, axes = plt.subplots(1, 3, figsize=(12, 5))
    fig.suptitle("Go vs Python Collector Benchmark (N=50 sources)", fontsize=13)

    colors = ["#4C9BE8", "#F4845F"]

    for ax, title, vals, unit in zip(
        axes,
        ["Time (s)", "Memory (MB)", "CPU (%)"],
        [time_vals, mem_vals, cpu_vals],
        ["s", "MB", "%"],
    ):
        bars = ax.bar(labels, vals, color=colors, width=0.4)
        ax.set_title(title)
        ax.set_ylabel(unit)
        for bar, val in zip(bars, vals):
            ax.text(
                bar.get_x() + bar.get_width() / 2,
                bar.get_height() + max(vals) * 0.02,
                f"{val:.3f}" if unit == "s" else f"{val:.2f}",
                ha="center",
                va="bottom",
                fontsize=9,
            )

    plt.tight_layout()
    plt.savefig(CHART_PATH, dpi=120)
    plt.close()
    print(f"\nChart saved: {CHART_PATH}")


def print_table(
    go_avg: tuple[float, float, float] | None,
    py_avg: tuple[float, float, float],
) -> None:
    go_wall, go_mem, go_cpu = go_avg if go_avg else (float("nan"),) * 3
    py_wall, py_mem, py_cpu = py_avg

    header = f"{'Metric':<18} {'Go':>10} {'Python':>10}"
    sep = "-" * len(header)
    print(f"\n{header}")
    print(sep)
    print(f"{'Wall time (s)':<18} {go_wall:>10.3f} {py_wall:>10.3f}")
    print(f"{'Peak mem (MB)':<18} {go_mem:>10.2f} {py_mem:>10.2f}")
    print(f"{'CPU (%)':<18} {go_cpu:>10.1f} {py_cpu:>10.1f}")
    print(sep)


if __name__ == "__main__":
    print(f"=== Benchmark: Go vs Python (N=50 sources, {N_ITERATIONS} iterations) ===\n")

    print("[Go collector]")
    go_avg = benchmark_go()

    print("\n[Python collector]")
    py_avg = benchmark_python()

    print_table(go_avg, py_avg)
    generate_chart(go_avg, py_avg)
