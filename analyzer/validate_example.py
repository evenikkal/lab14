import rust_validator

result = rust_validator.validate({
    "injured": 2,
    "dead": 1,
    "date": "2024-01-15T10:00:00+00:00",
    "region": "Центральный",
})
print(result.valid, result.errors)
