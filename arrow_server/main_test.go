package main

import (
	"testing"

	"github.com/apache/arrow/go/v17/arrow/array"
)

func TestBuildAccidentRecord_RowCount(t *testing.T) {
	rec := buildAccidentRecord()
	defer rec.Release()

	const wantRows = 200
	if rec.NumRows() != wantRows {
		t.Errorf("NumRows: want %d, got %d", wantRows, rec.NumRows())
	}
}

func TestBuildAccidentRecord_SchemaFields(t *testing.T) {
	rec := buildAccidentRecord()
	defer rec.Release()

	schema := rec.Schema()
	wantFields := []string{"id", "date", "region", "type", "injured", "dead", "collected_at"}
	for _, name := range wantFields {
		if len(schema.FieldIndices(name)) == 0 {
			t.Errorf("schema missing field %q", name)
		}
	}
}

func TestBuildAccidentRecord_DeadLeInjured(t *testing.T) {
	rec := buildAccidentRecord()
	defer rec.Release()

	schema := rec.Schema()
	injuredIdx := schema.FieldIndices("injured")[0]
	deadIdx := schema.FieldIndices("dead")[0]

	injuredCol := rec.Column(injuredIdx).(*array.Int32)
	deadCol := rec.Column(deadIdx).(*array.Int32)

	for i := 0; i < int(rec.NumRows()); i++ {
		injured := injuredCol.Value(i)
		dead := deadCol.Value(i)
		if dead > injured {
			t.Errorf("row %d: dead=%d > injured=%d", i, dead, injured)
		}
	}
}

func TestBuildAccidentRecord_IDsNonEmpty(t *testing.T) {
	rec := buildAccidentRecord()
	defer rec.Release()

	schema := rec.Schema()
	idIdx := schema.FieldIndices("id")[0]
	idCol := rec.Column(idIdx).(*array.String)

	for i := 0; i < int(rec.NumRows()); i++ {
		if idCol.Value(i) == "" {
			t.Errorf("row %d: id is empty", i)
		}
	}
}
