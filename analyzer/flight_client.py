import importlib
import pathlib
from typing import Any


def main() -> None:
    pl: Any = importlib.import_module("polars")
    flight: Any = importlib.import_module("pyarrow.flight")

    client = flight.connect("grpc://localhost:50051")

    reader = client.do_get(flight.Ticket(b"accidents"))

    table = reader.read_all()

    df = pl.from_arrow(table)

    print(f"Shape: {df.shape}")
    print("\nFirst 5 rows:")
    print(df.head(5))

    print("\nColumn dtypes:")
    for name, dtype in zip(df.columns, df.dtypes):
        print(f"  {name}: {dtype}")

    out_path = pathlib.Path(__file__).parent.parent / "data" / "accidents_arrow.parquet"
    out_path.parent.mkdir(parents=True, exist_ok=True)
    df.write_parquet(out_path)
    print(f"\nSaved to {out_path}")


if __name__ == "__main__":
    main()
