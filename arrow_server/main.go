package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"syscall"
	"time"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/flight"
	"github.com/apache/arrow/go/v17/arrow/ipc"
	"github.com/apache/arrow/go/v17/arrow/memory"
)

var accSchema = arrow.NewSchema([]arrow.Field{
	{Name: "id", Type: arrow.BinaryTypes.String, Nullable: false},
	{Name: "date", Type: arrow.BinaryTypes.String, Nullable: false},
	{Name: "region", Type: arrow.BinaryTypes.String, Nullable: false},
	{Name: "type", Type: arrow.BinaryTypes.String, Nullable: false},
	{Name: "injured", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
	{Name: "dead", Type: arrow.PrimitiveTypes.Int32, Nullable: false},
	{Name: "collected_at", Type: arrow.BinaryTypes.String, Nullable: false},
}, nil)

var (
	regions = []string{
		"Центральный", "Северный", "Южный", "Восточный", "Западный",
		"Северо-Западный", "Приволжский", "Уральский", "Сибирский", "Дальневосточный",
	}
	accTypes = []string{
		"Столкновение", "Наезд на пешехода", "Опрокидывание",
		"Наезд на препятствие", "Съезд с дороги", "Наезд на велосипедиста",
	}
)

type accidentServer struct {
	flight.BaseFlightServer
}

func (s *accidentServer) DoGet(ticket *flight.Ticket, stream flight.FlightService_DoGetServer) error {
	slog.Info("DoGet called", "ticket", string(ticket.GetTicket()))

	rec := buildAccidentRecord()
	defer rec.Release()

	w := flight.NewRecordWriter(stream, ipc.WithSchema(accSchema))
	defer w.Close()

	return w.Write(rec)
}

func buildAccidentRecord() arrow.Record {
	pool := memory.NewGoAllocator()
	b := array.NewRecordBuilder(pool, accSchema)
	defer b.Release()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now().UTC()

	idB := b.Field(0).(*array.StringBuilder)
	dateB := b.Field(1).(*array.StringBuilder)
	regionB := b.Field(2).(*array.StringBuilder)
	typeB := b.Field(3).(*array.StringBuilder)
	injuredB := b.Field(4).(*array.Int32Builder)
	deadB := b.Field(5).(*array.Int32Builder)
	collectedB := b.Field(6).(*array.StringBuilder)

	for i := 0; i < 200; i++ {
		injured := rng.Intn(10)
		dead := 0
		if injured > 0 {
			dead = rng.Intn(injured + 1)
		}
		accDate := now.Add(-time.Duration(rng.Intn(8760)) * time.Hour)

		idB.Append(fmt.Sprintf("acc-%04d", i+1))
		dateB.Append(accDate.Format(time.RFC3339))
		regionB.Append(regions[rng.Intn(len(regions))])
		typeB.Append(accTypes[rng.Intn(len(accTypes))])
		injuredB.Append(int32(injured))
		deadB.Append(int32(dead))
		collectedB.Append(now.Format(time.RFC3339))
	}

	return b.NewRecord()
}

func main() {
	slog.Info("starting Arrow Flight server", "addr", ":50051")

	srv := flight.NewFlightServer()
	srv.RegisterFlightService(&accidentServer{})
	srv.SetShutdownOnSignals(syscall.SIGINT, syscall.SIGTERM)

	if err := srv.Init(":50051"); err != nil {
		slog.Error("init failed", "err", err)
		os.Exit(1)
	}

	slog.Info("Arrow Flight server ready", "addr", ":50051")
	if err := srv.Serve(); err != nil {
		slog.Error("serve error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
