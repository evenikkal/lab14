import mock_generator


def test_generate_returns_requested_count():
    records = mock_generator.generate(42)
    assert len(records) == 42


def test_generate_empty():
    records = mock_generator.generate(0)
    assert records == []


def test_generate_dead_le_injured():
    records = mock_generator.generate(200, seed=0)
    for rec in records:
        assert rec["dead"] <= rec["injured"]


def test_generate_non_negative_counts():
    records = mock_generator.generate(100, seed=1)
    for rec in records:
        assert rec["injured"] >= 0
        assert rec["dead"] >= 0


def test_generate_required_fields_present():
    required = {"id", "date", "region", "type", "injured", "dead", "collected_at"}
    records = mock_generator.generate(10, seed=2)
    for rec in records:
        assert required <= rec.keys()


def test_generate_non_empty_string_fields():
    records = mock_generator.generate(20, seed=3)
    for rec in records:
        assert rec["id"] != ""
        assert rec["date"] != ""
        assert rec["region"] != ""
        assert rec["type"] != ""
        assert rec["collected_at"] != ""


def test_generate_region_from_known_list():
    records = mock_generator.generate(50, seed=4)
    for rec in records:
        assert rec["region"] in mock_generator.REGIONS


def test_generate_type_from_known_list():
    records = mock_generator.generate(50, seed=5)
    for rec in records:
        assert rec["type"] in mock_generator.ACCIDENT_TYPES


def test_generate_deterministic_with_seed():
    r1 = mock_generator.generate(30, seed=99)
    r2 = mock_generator.generate(30, seed=99)
    assert r1 == r2


def test_generate_different_seeds_differ():
    r1 = mock_generator.generate(10, seed=0)
    r2 = mock_generator.generate(10, seed=1)
    assert r1 != r2
