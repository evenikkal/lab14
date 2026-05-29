import asyncio
import json
import os
import tempfile

import main as collector


def test_fetch_source_returns_expected_count():
    records = asyncio.run(collector.fetch_source(0, "Центральный"))
    assert len(records) == collector.RECORDS_PER_SOURCE


def test_fetch_source_applies_region():
    region = "Уральский"
    records = asyncio.run(collector.fetch_source(1, region))
    for rec in records:
        assert rec["region"] == region


def test_fetch_source_records_have_required_fields():
    required = {"id", "date", "region", "type", "injured", "dead", "collected_at"}
    records = asyncio.run(collector.fetch_source(2, "Южный"))
    for rec in records:
        assert required <= rec.keys()


def test_collect_all_total_record_count():
    records, _peak_mb, _cpu = asyncio.run(collector.collect_all())
    expected = collector.N_SOURCES * collector.RECORDS_PER_SOURCE
    assert len(records) == expected


def test_write_output_creates_valid_jsonl():
    sample = [
        {"id": "abc-1", "dead": 0, "injured": 2},
        {"id": "abc-2", "dead": 1, "injured": 1},
    ]
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".jsonl", delete=False
    ) as fh:
        path = fh.name

    try:
        collector.write_output(sample, path)

        with open(path, encoding="utf-8") as fh:
            lines = [ln for ln in fh.readlines() if ln.strip()]

        assert len(lines) == len(sample)
        for ln in lines:
            data = json.loads(ln)
            assert "id" in data
    finally:
        os.unlink(path)


def test_write_output_line_count_matches_records():
    records = [{"x": i} for i in range(15)]
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=".jsonl", delete=False
    ) as fh:
        path = fh.name

    try:
        collector.write_output(records, path)
        with open(path) as fh:
            count = sum(1 for ln in fh if ln.strip())
        assert count == 15
    finally:
        os.unlink(path)
