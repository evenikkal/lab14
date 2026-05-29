# Журнал промптов — Лабораторная работа №14

**ФИО:** Никишина Евгения Александровна  
**Группа:** 221131  
**Вариант:** 11  
**Сложность:** Повышенная  

---

## П1 — Распределённый сборщик (Go)
**Задача:** Реализация распределённого сборщика на Go с использованием etcd для координации.
**Промпт:** 
> Implement a distributed data collector in Go for accident records. Use etcd for leader election and sharding logic to distribute data processing across multiple nodes.

---

## П2 — Скользящее окно (Tumbling Window)
**Задача:** Агрегация данных во временных окнах.
**Промпт:**
> Add tumbling window support to the Go collector. Aggregate accident data into 10-second windows and calculate summary statistics before storage.

---

## П3 — Arrow Flight сервер
**Задача:** Реализация высокопроизводительного RPC сервера.
**Промпт:**
> Create an Apache Arrow Flight server in Go to serve collected accident data. Implement the DoGet method to allow clients to stream Arrow record batches.

---

## П4 — Валидация на Rust (PyO3)
**Задача:** Создание FFI-модуля для быстрой валидации.
**Промпт:**
> Develop a high-performance validation library in Rust using PyO3. It should validate accident record fields like coordinates, severity levels, and vehicle types, and be callable from Python.

---

## П5 — Docker, K8s и HPA
**Задача:** Контейнеризация и оркестрация.
**Промпт:**
> Containerize the application components using Docker. Create Kubernetes manifests including Deployment, Service, and Horizontal Pod Autoscaler (HPA) for the collector.

---

## П6 — Asyncio Benchmark
**Задача:** Сравнение производительности Go и Python.
**Промпт:**
> Write a benchmark script in Python using asyncio to compare the performance of the Go collector and a Python-based collector when processing large volumes of Arrow data.

---

## П7 — NATS Streaming
**Задача:** Интеграция шины сообщений.
**Промпт:**
> Integrate NATS JetStream into the system. The Go collector should publish accident events to a NATS subject, and a Python consumer should process these events in real-time.

---

## П8 — Streamlit Dashboard
**Задача:** Визуализация данных.
**Промпт:**
> Build a Streamlit dashboard to visualize accident statistics. Connect it to the Arrow Flight server for data retrieval and use the Rust validator for on-the-fly data verification.

---

## Тестирование
**Задача:** Покрытие кода тестами.
**Промпт:**
> Write unit tests for the Go collector and Arrow Flight server. Implement a comprehensive Python test suite using pytest to verify end-to-end data flow and validation logic.

---

## Документация
**Задача:** Финализация README и лога.
**Промпт:**
> Finalize the project documentation. Create a detailed README.md with architecture diagrams, run instructions, and benchmark results. Complete the PROMPT_LOG.md with all task summaries.
