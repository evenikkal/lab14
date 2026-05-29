package main

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestGenerateMockAccidents_Count(t *testing.T) {
	const want = 73
	accs := GenerateMockAccidents("Центральный", want)
	if len(accs) != want {
		t.Errorf("want %d accidents, got %d", want, len(accs))
	}
}

func TestGenerateMockAccidents_FixedRegion(t *testing.T) {
	region := "Тестовый"
	accs := GenerateMockAccidents(region, 50)
	for i, a := range accs {
		if a.Region != region {
			t.Errorf("accs[%d].Region = %q, want %q", i, a.Region, region)
		}
	}
}

func TestGenerateMockAccidents_DeadLeInjured(t *testing.T) {
	accs := GenerateMockAccidents("Южный", 300)
	for i, a := range accs {
		if a.Dead > a.Injured {
			t.Errorf("accs[%d]: dead=%d > injured=%d violates invariant", i, a.Dead, a.Injured)
		}
	}
}

func TestGenerateMockAccidents_NonEmptyFields(t *testing.T) {
	accs := GenerateMockAccidents("Северный", 20)
	for i, a := range accs {
		if a.ID == "" {
			t.Errorf("accs[%d].ID is empty", i)
		}
		if a.Type == "" {
			t.Errorf("accs[%d].Type is empty", i)
		}
		if a.Date.IsZero() {
			t.Errorf("accs[%d].Date is zero", i)
		}
		if a.CollectedAt.IsZero() {
			t.Errorf("accs[%d].CollectedAt is zero", i)
		}
	}
}

func TestGenerateMockAccidents_InjuredNonNegative(t *testing.T) {
	accs := GenerateMockAccidents("Восточный", 100)
	for i, a := range accs {
		if a.Injured < 0 {
			t.Errorf("accs[%d].Injured = %d, want >= 0", i, a.Injured)
		}
		if a.Dead < 0 {
			t.Errorf("accs[%d].Dead = %d, want >= 0", i, a.Dead)
		}
	}
}

func TestAggregateWindow_Count(t *testing.T) {
	buf := makeTestBuf()
	wb := aggregateWindow(buf, time.Now(), time.Now())
	if wb.Count != len(buf) {
		t.Errorf("Count: want %d, got %d", len(buf), wb.Count)
	}
}

func TestAggregateWindow_SumDead(t *testing.T) {
	buf := makeTestBuf()
	wb := aggregateWindow(buf, time.Now(), time.Now())
	wantSumDead := 1 + 0 + 2
	if wb.SumDead != wantSumDead {
		t.Errorf("SumDead: want %d, got %d", wantSumDead, wb.SumDead)
	}
}

func TestAggregateWindow_AvgInjured(t *testing.T) {
	buf := makeTestBuf()
	wb := aggregateWindow(buf, time.Now(), time.Now())
	wantAvg := float64(4+2+6) / float64(len(buf))
	if math.Abs(wb.AvgInjured-wantAvg) > 1e-9 {
		t.Errorf("AvgInjured: want %f, got %f", wantAvg, wb.AvgInjured)
	}
}

func TestAggregateWindow_MinMaxDate(t *testing.T) {
	earliest := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	middle := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	latest := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)

	buf := []Accident{
		{Date: middle, Injured: 1, Dead: 0},
		{Date: earliest, Injured: 1, Dead: 0},
		{Date: latest, Injured: 1, Dead: 0},
	}
	wb := aggregateWindow(buf, time.Now(), time.Now())

	if !wb.MinDate.Equal(earliest) {
		t.Errorf("MinDate: want %v, got %v", earliest, wb.MinDate)
	}
	if !wb.MaxDate.Equal(latest) {
		t.Errorf("MaxDate: want %v, got %v", latest, wb.MaxDate)
	}
}

func TestAggregateWindow_WindowBounds(t *testing.T) {
	start := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Second)
	wb := aggregateWindow(makeTestBuf(), start, end)
	if !wb.WindowStart.Equal(start) {
		t.Errorf("WindowStart: want %v, got %v", start, wb.WindowStart)
	}
	if !wb.WindowEnd.Equal(end) {
		t.Errorf("WindowEnd: want %v, got %v", end, wb.WindowEnd)
	}
}

func TestWriteJSONL_ValidJSONL(t *testing.T) {
	const workerID = 99997
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("logger init: %v", err)
	}

	accs := GenerateMockAccidents("Западный", 7)

	if err := WriteJSONL(accs, workerID, logger); err != nil {
		t.Fatalf("WriteJSONL: %v", err)
	}

	path := filepath.Join("..", "data", "shard_99997.jsonl")
	t.Cleanup(func() { os.Remove(path) })

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()

	count := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var a Accident
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			t.Errorf("line %d: invalid JSON: %v", count+1, err)
		}
		count++
	}
	if sc.Err() != nil {
		t.Fatalf("scanner error: %v", sc.Err())
	}
	if count != len(accs) {
		t.Errorf("want %d lines in JSONL, got %d", len(accs), count)
	}
}

func makeTestBuf() []Accident {
	d1 := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)
	return []Accident{
		{Date: d1, Injured: 4, Dead: 1},
		{Date: d2, Injured: 2, Dead: 0},
		{Date: d3, Injured: 6, Dead: 2},
	}
}
