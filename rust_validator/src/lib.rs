use pyo3::prelude::*;
use pyo3::types::PyDict;
use chrono::{DateTime, NaiveDate, Utc};

const VALID_REGIONS: &[&str] = &[
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
];

#[pyclass]
struct ValidationResult {
    #[pyo3(get)]
    valid: bool,
    #[pyo3(get)]
    errors: Vec<String>,
}

#[pymethods]
impl ValidationResult {
    #[new]
    fn new(valid: bool, errors: Vec<String>) -> Self {
        Self { valid, errors }
    }
}

#[pyfunction]
fn validate(record: &Bound<'_, PyDict>) -> PyResult<ValidationResult> {
    let mut errors = Vec::new();

    let injured: i64 = record
        .get_item("injured")?
        .ok_or_else(|| pyo3::exceptions::PyKeyError::new_err("injured"))?
        .extract()?;

    let dead: i64 = record
        .get_item("dead")?
        .ok_or_else(|| pyo3::exceptions::PyKeyError::new_err("dead"))?
        .extract()?;

    let date: String = record
        .get_item("date")?
        .ok_or_else(|| pyo3::exceptions::PyKeyError::new_err("date"))?
        .extract()?;

    let region: String = record
        .get_item("region")?
        .ok_or_else(|| pyo3::exceptions::PyKeyError::new_err("region"))?
        .extract()?;

    if injured < 0 {
        errors.push("injured must be >= 0".to_string());
    }
    if dead < 0 {
        errors.push("dead must be >= 0".to_string());
    }
    if dead > injured {
        errors.push("dead cannot exceed injured".to_string());
    }

    let min_date = NaiveDate::from_ymd_opt(2000, 1, 1).expect("hardcoded valid date");
    let today = Utc::now().date_naive();

    match DateTime::parse_from_rfc3339(&date) {
        Ok(parsed) => {
            let parsed_date = parsed.date_naive();
            if parsed_date < min_date || parsed_date > today {
                errors.push("date out of valid range".to_string());
            }
        }
        Err(_) => errors.push("date out of valid range".to_string()),
    }

    if !VALID_REGIONS.contains(&region.as_str()) {
        errors.push(format!("unknown region: {region}"));
    }

    Ok(ValidationResult {
        valid: errors.is_empty(),
        errors,
    })
}

#[pymodule]
fn rust_validator(m: &Bound<'_, PyModule>) -> PyResult<()> {
    m.add_class::<ValidationResult>()?;
    m.add_function(wrap_pyfunction!(validate, m)?)?;
    Ok(())
}
