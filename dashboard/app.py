import time
import json
from pathlib import Path
import random
from datetime import datetime, timedelta, timezone

import pandas as pd
import plotly.express as px
import streamlit as st

st.set_page_config(
    page_title="Аналитика ДТП",
    page_icon="🚗",
    layout="wide",
)

REGIONS = [
    "Центральный", "Северный", "Южный", "Восточный", "Западный",
    "Северо-Западный", "Приволжский", "Уральский", "Сибирский", "Дальневосточный",
]
TYPES = [
    "Столкновение", "Наезд на пешехода", "Опрокидывание",
    "Наезд на препятствие", "Съезд с дороги", "Наезд на велосипедиста",
]

BASE_DIR = Path(__file__).parent.parent
JSONL_PATH = BASE_DIR / "data" / "collector_py_output.jsonl"
PARQUET_PATH = BASE_DIR / "data" / "accidents_arrow.parquet"


@st.cache_data(ttl=30)
def load_data() -> pd.DataFrame:
    if JSONL_PATH.exists():
        records = []
        with JSONL_PATH.open("r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if line:
                    records.append(json.loads(line))
        df = pd.DataFrame(records)

    elif PARQUET_PATH.exists():
        df = pd.read_parquet(PARQUET_PATH)

    else:
        rng = random.Random(42)
        base = datetime(2022, 1, 1, tzinfo=timezone.utc)
        rows = []
        for i in range(500):
            dt = base + timedelta(days=rng.randint(0, 730), hours=rng.randint(0, 23))
            rows.append({
                "id": f"mock-{i:04d}",
                "date": dt.isoformat(),
                "region": rng.choice(REGIONS),
                "type": rng.choice(TYPES),
                "injured": rng.randint(0, 10),
                "dead": rng.randint(0, 5),
                "collected_at": (dt + timedelta(minutes=rng.randint(30, 120))).isoformat(),
            })
        df = pd.DataFrame(rows)

    for col in ("date", "collected_at"):
        if col in df.columns:
            df[col] = pd.to_datetime(df[col], utc=True, errors="coerce")

    for col in ("injured", "dead"):
        if col in df.columns:
            df[col] = pd.to_numeric(df[col], errors="coerce").fillna(0).astype(int)

    return df


def main() -> None:
    st.title("🚗 Аналитика дорожно-транспортных происшествий")

    df_all = load_data()

    st.sidebar.header("Фильтры")

    all_regions = sorted(df_all["region"].dropna().unique().tolist())
    selected_regions = st.sidebar.multiselect(
        "Регион",
        options=all_regions,
        default=all_regions,
    )

    all_types = sorted(df_all["type"].dropna().unique().tolist())
    selected_types = st.sidebar.multiselect(
        "Тип ДТП",
        options=all_types,
        default=all_types,
    )

    min_date = df_all["date"].dt.date.min()
    max_date = df_all["date"].dt.date.max()
    date_from, date_to = st.sidebar.date_input(
        "Период",
        value=(min_date, max_date),
        min_value=min_date,
        max_value=max_date,
    )

    st.sidebar.divider()
    auto_refresh = st.sidebar.checkbox("Авто-обновление (10 сек)", value=False)

    mask = (
        df_all["region"].isin(selected_regions)
        & df_all["type"].isin(selected_types)
        & (df_all["date"].dt.date >= date_from)
        & (df_all["date"].dt.date <= date_to)
    )
    df = df_all[mask].copy()

    if df.empty:
        st.warning("Нет данных для выбранных фильтров.")
        return

    col1, col2, col3 = st.columns(3)
    col1.metric("Всего ДТП", f"{len(df):,}".replace(",", " "))
    col2.metric("Погибших", f"{df['dead'].sum():,}".replace(",", " "))
    col3.metric("Пострадавших", f"{df['injured'].sum():,}".replace(",", " "))

    st.divider()

    df["month"] = df["date"].dt.to_period("M").dt.to_timestamp()
    monthly = df.groupby("month").size().reset_index(name="count")
    fig_ts = px.line(
        monthly,
        x="month",
        y="count",
        markers=True,
        title="Динамика ДТП по месяцам",
        labels={"month": "Месяц", "count": "Количество ДТП"},
    )
    fig_ts.update_layout(hovermode="x unified")
    st.plotly_chart(fig_ts, use_container_width=True)

    top_regions = (
        df.groupby("region")["dead"]
        .sum()
        .nlargest(10)
        .reset_index()
        .rename(columns={"region": "Регион", "dead": "Погибших"})
        .sort_values("Погибших")
    )
    fig_bar = px.bar(
        top_regions,
        x="Погибших",
        y="Регион",
        orientation="h",
        title="Топ-10 регионов по смертности",
        labels={"Погибших": "Количество погибших", "Регион": ""},
        color="Погибших",
        color_continuous_scale="Reds",
    )
    fig_bar.update_layout(coloraxis_showscale=False)
    st.plotly_chart(fig_bar, use_container_width=True)

    st.subheader("Исходные данные")
    page_size = 20
    total_pages = max(1, (len(df) + page_size - 1) // page_size)
    page = st.slider("Страница", min_value=1, max_value=total_pages, value=1)
    start = (page - 1) * page_size
    end = start + page_size

    display_cols = [c for c in ("id", "date", "region", "type", "injured", "dead", "collected_at") if c in df.columns]
    st.dataframe(df[display_cols].iloc[start:end], use_container_width=True)
    st.caption(f"Страница {page} из {total_pages} | Всего записей: {len(df)}")

    if auto_refresh:
        time.sleep(10)
        st.cache_data.clear()
        st.rerun()


if __name__ == "__main__":
    main()
