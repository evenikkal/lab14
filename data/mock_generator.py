from __future__ import annotations

import argparse
import json
import random
import sys
import uuid
from datetime import datetime, timedelta, timezone


REGIONS: list[str] = [
    "Центральный",
    "Северный",
    "Южный",
    "Восточный",
    "Западный",
    "Северо-Западный",
    "Приволжский",
    "Уральский",
    "Сибирский",
    "Дальневосточный",
]

ACCIDENT_TYPES: list[str] = [
    "Столкновение",
    "Наезд на пешехода",
    "Опрокидывание",
    "Наезд на препятствие",
    "Съезд с дороги",
    "Наезд на велосипедиста",
]

_INJURED_WEIGHTS = [40, 30, 15, 10, 4, 1]
_INJURED_RANGES = [(0, 1), (2, 3), (4, 6), (7, 10), (11, 20), (21, 50)]

_DEAD_WEIGHTS = [60, 25, 10, 4, 1]
_DEAD_RANGES = [(0, 0), (1, 1), (2, 2), (3, 4), (5, 10)]

_REFERENCE_START = datetime(2022, 1, 1, tzinfo=timezone.utc)
_REFERENCE_END = datetime(2024, 12, 31, 23, 59, 59, tzinfo=timezone.utc)
_SPAN_SECONDS = int((_REFERENCE_END - _REFERENCE_START).total_seconds())


def _random_datetime(rng: random.Random) -> datetime:
    offset = timedelta(seconds=rng.randint(0, _SPAN_SECONDS))
    return _REFERENCE_START + offset


def _weighted_range(rng: random.Random, weights: list[int], ranges: list[tuple[int, int]]) -> int:
    (lo, hi) = rng.choices(ranges, weights=weights, k=1)[0]
    return rng.randint(lo, hi)


def _make_accident(rng: random.Random) -> dict:
    date = _random_datetime(rng)
    collected_at = date + timedelta(seconds=rng.randint(1, 3600))

    injured = _weighted_range(rng, _INJURED_WEIGHTS, _INJURED_RANGES)
    dead = _weighted_range(rng, _DEAD_WEIGHTS, _DEAD_RANGES)
    dead = min(dead, injured)

    return {
        "id": str(uuid.UUID(int=rng.getrandbits(128), version=4)),
        "date": date.isoformat(),
        "region": rng.choice(REGIONS),
        "type": rng.choice(ACCIDENT_TYPES),
        "injured": injured,
        "dead": dead,
        "collected_at": collected_at.isoformat(),
    }


def generate(count: int, seed: int | None = None) -> list[dict]:
    rng = random.Random(seed)
    return [_make_accident(rng) for _ in range(count)]


def stream(count: int, seed: int | None = None, out=None):
    if out is None:
        out = sys.stdout
    rng = random.Random(seed)
    for _ in range(count):
        out.write(json.dumps(_make_accident(rng), ensure_ascii=False) + "\n")


def write_jsonl(count: int, path: str, seed: int | None = None):
    with open(path, "w", encoding="utf-8") as fh:
        stream(count, seed=seed, out=fh)


def _cli():
    parser = argparse.ArgumentParser(
        prog="mock_generator",
        description="Generate mock Accident records as JSONL",
    )
    parser.add_argument("--count", type=int, default=100, help="Number of records (default: 100)")
    parser.add_argument("--output", type=str, default=None, help="Output file path (default: stdout)")
    parser.add_argument("--seed", type=int, default=None, help="Random seed for reproducibility")
    args = parser.parse_args()

    if args.output:
        write_jsonl(args.count, args.output, seed=args.seed)
        print(f"Generated {args.count} records → {args.output}", file=sys.stderr)
    else:
        stream(args.count, seed=args.seed)


if __name__ == "__main__":
    _cli()
