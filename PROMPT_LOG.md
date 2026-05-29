# Журнал промптов — Лабораторная работа №14

**ФИО:** Никишина Евгения Александровна  
**Группа:** 221131  
**Вариант:** 11  
**Сложность:** Повышенная  

---

## П0 — Бутстрап репозитория

**Дата:** 2026-05-29  
**Задача:** Создание структуры репозитория, базовых файлов и генератора мок-данных.

**Промпт:**
```
Bootstrap the empty repository for lab14. Create the full directory structure,
.gitignore, base README.md, PROMPT_LOG.md, and a shared mock data generator
(data/mock_generator.py) that will be reused by all subsequent tasks.
Include docker-compose.yml with etcd v3.5 and NATS JetStream 2.10 services.
```

**Ключевые решения:**
- Все компоненты в отдельных директориях с чёткой ответственностью
- Общий `mock_generator.py` с фиксированными регионами РФ (10 федеральных округов)
- `docker-compose.yml` с healthcheck для etcd и NATS

**Результат:**
- Созданы директории: `collector/`, `collector_py/`, `analyzer/`, `dashboard/`, `arrow_server/`, `rust_validator/`, `k8s/`, `data/`, `charts/`, `tests/`
- Созданы: `.gitignore`, `README.md`, `PROMPT_LOG.md`, `docker-compose.yml`, `data/mock_generator.py`

---

## П1 — Распределённый Go-сборщик (`collector/`)

**Дата:** 2026-05-29  
**Задача:** Реализация распределённого сборщика данных об авариях на Go с использованием etcd для координации лидер-выборов и распределения шардов.

**Промпт:**
```
Implement a distributed data collector in Go for accident records.
Use etcd v3 client for leader election (concurrency.NewElection) and
shard distribution (/lab14/shards key). Workers read the shard list
and try to acquire per-region mutexes (TryLock) before collecting.
Generate mock accident data (6 types, 10 Russian regions, 2022-2024 range).
Support four run modes via --mode flag: leader, worker, window, nats-producer.
```

**Ключевые решения:**
- `concurrency.NewElection` для etcd leader election (ключ `/lab14/election`)
- Шарды = список регионов, публикуется лидером в `/lab14/shards`
- Worker делает `TryLock` на ключ `/lab14/lock/<region>`, чтобы исключить дублирование
- `GenerateMockAccidents` с детерминированным seed на основе имени региона
- Взвешенная случайность для `injured`/`dead` (реалистичное распределение)
- Graceful shutdown через `context.WithCancel` + `SIGINT/SIGTERM`

**Сложности:**
- Правильный порядок `Campaign → Put → Wait` без гонок контекста
- `TryLock` с таймаутом вместо блокирующего `Lock` (чтобы worker не висел)

**Результат:**
- Файлы: `main.go`, `collector.go`, `shard.go`, `accident.go`, `writer.go`
- Все 4 режима запуска работают
- `go test ./...` — OK

---

## П2 — Tumbling Window агрегация

**Дата:** 2026-05-29  
**Задача:** Реализация tumbling window агрегации потока событий Accident с записью в Apache Arrow IPC.

**Промпт:**
```
Add tumbling window support to the Go collector (window.go, window_writer.go).
Window flushes on either 100 records or 30 seconds (whichever comes first).
WindowBatch struct must contain: WindowStart, WindowEnd, Count, SumDead,
AvgInjured, MinDate, MaxDate. Write batches to Apache Arrow IPC files.
Use TumblingWindow(in <-chan Accident) <-chan WindowBatch channel pattern.
```

**Ключевые решения:**
- Канальная архитектура: `TumblingWindow` принимает `<-chan Accident` и возвращает `<-chan WindowBatch`
- Двойной триггер: `len(buf) >= 100` (flush + reset timer) и `time.NewTimer(30s)`
- Сброс таймера через `Stop() + drain + Reset()` для предотвращения двойного flush
- Arrow IPC запись в `window_writer.go` через `arrow/go/v17`

**Результат:**
- `window.go`: `TumblingWindow`, `aggregateWindow`
- `window_writer.go`: запись `WindowBatch` в Arrow IPC формат
- Режим `--mode window` использует оба файла

---

## П3 — Apache Arrow Flight сервер (`arrow_server/`)

**Дата:** 2026-05-29  
**Задача:** Реализация Go Arrow Flight gRPC-сервера для высокопроизводительной передачи данных ДТП клиентам.

**Промпт:**
```
Create an Apache Arrow Flight server in Go using github.com/apache/arrow/go/v17.
Implement BaseFlightServer embedding with DoGet method that returns 200 accident
records as Arrow Record Batch. Schema: id(str), date(str), region(str), type(str),
injured(int32), dead(int32), collected_at(str). Listen on :50051.
Also implement a Python flight_client.py using pyarrow.flight and polars.
```

**Ключевые решения:**
- `flight.BaseFlightServer` embedding + `DoGet` override
- `array.NewRecordBuilder` с явными типами Builder для каждого поля
- `flight.NewRecordWriter(stream, ipc.WithSchema(...))` для стриминга
- Python-клиент через `pyarrow.flight.connect("grpc://localhost:50051")` + Polars
- Сохранение результата в `data/accidents_arrow.parquet` через `df.write_parquet()`

**Сложности:**
- Правильное освобождение памяти: `defer rec.Release()`, `defer b.Release()`
- `w.Close()` через `defer` для корректного завершения gRPC-стрима

**Результат:**
- `arrow_server/main.go`: Flight-сервер + встроенный CLI-клиент
- `analyzer/flight_client.py`: Python Arrow Flight клиент
- `go test ./...` — OK

---

## П4 — Rust PyO3 валидатор (`rust_validator/`)

**Дата:** 2026-05-29  
**Задача:** Реализация высокопроизводительного валидатора записей ДТП на Rust с PyO3-биндингами для вызова из Python.

**Промпт:**
```
Build a Rust library with PyO3 bindings (maturin) for validating accident records.
Function: validate(record: &PyDict) -> PyResult<ValidationResult>.
ValidationResult: { valid: bool, errors: Vec<String> }.
Checks: injured >= 0, dead >= 0, dead <= injured, date in RFC3339 in range
2000-01-01..today (chrono crate), region in list of 10 Russian federal districts.
Package name: rust_validator. Use pyproject.toml for maturin build config.
```

**Ключевые решения:**
- `#[pyclass]` + `#[pymethods]` для `ValidationResult` с `#[pyo3(get)]` полями
- `#[pyfunction]` для `validate`, принимает `&Bound<'_, PyDict>`
- `chrono::DateTime::parse_from_rfc3339` для парсинга даты
- `VALID_REGIONS` — статический массив `&[&str]` из 10 федеральных округов
- `pyproject.toml` + `maturin develop` для сборки wheel в текущий venv

**Сложности:**
- Lifetime аннотации PyO3 (`Bound<'_, PyDict>`)
- Цепочка `get_item → ok_or_else → extract` для каждого поля словаря

**Результат:**
- `rust_validator/src/lib.rs`: 96 строк, полная реализация
- Импортируется в Python как `import rust_validator; rust_validator.validate({...})`

---

## П5 — Docker / Kubernetes / HPA (`k8s/`)

**Дата:** 2026-05-29  
**Задача:** Контейнеризация Go-сборщика и подготовка Kubernetes-манифестов с HPA для автоматического масштабирования.

**Промпт:**
```
Containerize the Go collector with a Dockerfile (multi-stage: builder + alpine).
Create Kubernetes manifests:
- deployment.yaml: replicas=2, image=lab14-collector:latest, env ETCD_URL and
  WORKER_ID (from metadata.name fieldRef), resources cpu 100m-500m, mem 64Mi-256Mi.
- service.yaml: ClusterIP, port 8080.
- hpa.yaml: autoscaling/v2, minReplicas=2, maxReplicas=10, cpu averageUtilization=60.
Update docker-compose.yml to include collector service with NATS/etcd deps.
```

**Ключевые решения:**
- `WORKER_ID` через `fieldRef: fieldPath: metadata.name` — уникальный ID каждого пода
- HPA `autoscaling/v2` с `resource.cpu.target.averageUtilization: 60`
- `imagePullPolicy: IfNotPresent` для совместимости с локальным minikube
- `emptyDir` volume для `/data` (Arrow IPC файлы)
- docker-compose зависимости с `condition: service_healthy`

**Результат:**
- `k8s/deployment.yaml`, `k8s/service.yaml`, `k8s/hpa.yaml`
- `collector/Dockerfile` с multi-stage сборкой
- Команды для minikube задокументированы в README

---

## П6 — Python asyncio коллектор + бенчмарк (`collector_py/`)

**Дата:** 2026-05-29  
**Задача:** Реализация Python asyncio-сборщика и сравнительного бенчмарка производительности Go vs Python.

**Промпт:**
```
Implement a Python asyncio collector (main.py) with N_SOURCES=50, RECORDS_PER_SOURCE=20.
Use asyncio.gather() for parallel fetch_source() calls. Measure: wall time (perf_counter),
peak memory (tracemalloc), CPU% (psutil). Write output to data/collector_py_output.jsonl.
Add benchmark.py that runs both Go (subprocess, --mode window) and Python collectors
N_ITERATIONS=3 times, averages results, and generates charts/benchmark.png via matplotlib.
```

**Ключевые решения:**
- `asyncio.gather(*tasks)` для параллельного сбора из 50 источников
- `tracemalloc.start() / get_traced_memory()` для точного замера пиковой памяти
- `psutil.Process(os.getpid()).cpu_percent(interval=None)` — сэмплирование до и после
- Go-бинарь запускается через `subprocess.run([GO_BINARY, "--mode", "window"])` с таймаутом 60 с
- Усреднение по 3 итерациям для стабильности результатов
- `matplotlib` с `Agg` backend (без GUI) → `charts/benchmark.png`

**Результаты бенчмарка:**

| Компонент | Wall time | Peak mem | CPU |
|-----------|-----------|----------|-----|
| Go collector (window) | 0.008 s | 0.30 MB | 45.9% |
| Python asyncio | 0.049 s | 0.53 MB | 74.0% |

**Результат:**
- `collector_py/main.py`, `collector_py/benchmark.py`
- `charts/benchmark.png` сгенерирован

---

## П7 — NATS Streaming Consumer (`analyzer/nats_consumer.py`)

**Дата:** 2026-05-29  
**Задача:** Реализация Python asyncio NATS-потребителя со скользящим окном агрегации событий ДТП в реальном времени.

**Промпт:**
```
Implement a Python asyncio NATS consumer (analyzer/nats_consumer.py).
Subscribe to subject "accidents". Maintain a sliding window of 300 seconds
using collections.deque (evict old entries). Every 30 seconds report:
count, sum_dead, avg_injured, time range of window. Auto-reconnect to NATS
(max_reconnect_attempts=-1). Env: NATS_URL (default nats://localhost:4222).
```

**Ключевые решения:**
- `collections.deque` + `_evict_old()` для скользящего окна без блокировок
- `asyncio.create_task(report_loop())` — независимая задача отчётности
- `await asyncio.Event().wait()` для бесконечного ожидания (вместо `while True: sleep`)
- NATS callbacks: `disconnected_cb`, `reconnected_cb`, `error_cb` для observability
- `await nc.drain()` в `finally` для чистого завершения

**Результат:**
- `analyzer/nats_consumer.py`: 92 строки, полная реализация
- Запуск: `NATS_URL=nats://localhost:4222 python3 nats_consumer.py`

---

## П8 — Streamlit дашборд (`dashboard/app.py`)

**Дата:** 2026-05-29  
**Задача:** Реализация интерактивного Streamlit-дашборда для визуализации статистики ДТП с фильтрами и графиками Plotly.

**Промпт:**
```
Build a Streamlit dashboard (dashboard/app.py) for accident analytics.
Data sources (priority): data/collector_py_output.jsonl → data/accidents_arrow.parquet
→ fallback 500 mock records. Sidebar filters: region (multiselect), type (multiselect),
date range. Charts via plotly.express: accidents by region (bar), time series (line),
type distribution (pie), injured/dead comparison. Cache with @st.cache_data(ttl=30).
Page config: layout="wide", page_icon="🚗".
```

**Ключевые решения:**
- Приоритетная цепочка источников данных: JSONL → Parquet → встроенные мок-данные
- `@st.cache_data(ttl=30)` — кэш на 30 секунд без перезагрузки страницы
- `pd.to_datetime(..., utc=True, errors="coerce")` для унификации timezone-aware дат
- `st.sidebar.multiselect` + `date_input` для фильтров
- `plotly.express` (px) для всех графиков с интерактивностью

**Результат:**
- `dashboard/app.py`: полнофункциональный дашборд
- Запуск: `streamlit run dashboard/app.py` → http://localhost:8501

---

## Тестирование

**Дата:** 2026-05-29  
**Задача:** Покрытие кода unit- и интеграционными тестами для Go-компонентов и Python-модулей.

**Промпт:**
```
Write tests for all implemented components:
Go: collector_test.go (GenerateMockAccidents, TumblingWindow, sharding logic),
    arrow_server/main_test.go (Flight server round-trip).
Python pytest: tests/test_mock_generator.py (fields, types, determinism),
    tests/test_collector_py.py (collect_all(), JSONL output),
    tests/test_nats_consumer.py (sliding window eviction, aggregation logic).
All tests must pass: go test ./... OK for both modules, python3 -m pytest: 24 passed.
```

**Ключевые решения:**
- Go: тест `TumblingWindow` через закрытие входного канала и чтение `WindowBatch`
- Go: тест Flight-сервера через `httptest`-подобный подход с in-process сервером
- Python: `pytest.mark.asyncio` для тестирования `collect_all()` с `asyncio.run`
- Python: мок-объекты для NATS-сообщений в `test_nats_consumer.py`
- `conftest.py` с общими фикстурами и настройкой путей

**Результаты:**
- `cd collector && go test ./...` — **OK ✅**
- `cd arrow_server && go test ./...` — **OK ✅**
- `python3 -m pytest` — **24 passed ✅**

---

## Документация

**Дата:** 2026-05-29  
**Задача:** Финализация README.md (полное описание архитектуры, инструкции, бенчмарки) и PROMPT_LOG.md (журнал всех задач).

**Промпт:**
```
Finalize documentation for lab14. README.md must include: full title with student info,
Mermaid architecture diagram, П1-П8 descriptions with files, run instructions for all
components (docker compose, collector modes, arrow_server, rust_validator maturin,
Python asyncio, NATS consumer, streamlit, kubernetes/minikube commands), benchmark
results (Go: 0.008s/0.30MB/45.9%, Python: 0.049s/0.53MB/74.0%), test commands,
git commit plan. PROMPT_LOG.md must have concise summaries for П0-П8, tests, docs.
```

**Ключевые решения:**
- Mermaid `flowchart TD` с subgraph для каждой подсистемы
- Точные команды запуска с env-переменными и флагами
- Бенчмарк-таблица с реальными измеренными значениями
- Примечание о недоступности Docker/K8s в текущей среде
- Git commit plan с `feat(scope):` конвенцией

**Результат:**
- `README.md`: ~220 строк, без placeholder-секций
- `PROMPT_LOG.md`: ~250 строк, все задачи П0–П8 + тесты + документация

---

_Конец журнала_
