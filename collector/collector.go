package main

import (
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

var accidentTypes = []string{
	"Столкновение",
	"Наезд на пешехода",
	"Опрокидывание",
	"Наезд на препятствие",
	"Съезд с дороги",
	"Наезд на велосипедиста",
}

var (
	refStart    = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	refEnd      = time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	spanSeconds = int64(refEnd.Sub(refStart).Seconds())
)

func GenerateMockAccidents(region string, count int) []Accident {
	var h int64
	for _, c := range region {
		h = h*31 + int64(c)
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano() ^ h))

	accidents := make([]Accident, count)
	for i := range accidents {
		offset := time.Duration(rng.Int63n(spanSeconds)) * time.Second
		date := refStart.Add(offset)
		collectedAt := date.Add(time.Duration(rng.Intn(3600)+1) * time.Second)

		injured := weightedRandom(rng,
			[]int{40, 30, 15, 10, 4, 1},
			[][2]int{{0, 1}, {2, 3}, {4, 6}, {7, 10}, {11, 20}, {21, 50}},
		)
		dead := weightedRandom(rng,
			[]int{60, 25, 10, 4, 1},
			[][2]int{{0, 0}, {1, 1}, {2, 2}, {3, 4}, {5, 10}},
		)
		if dead > injured {
			dead = injured
		}

		accidents[i] = Accident{
			ID:          newUUID(rng),
			Date:        date,
			Region:      region,
			Type:        accidentTypes[rng.Intn(len(accidentTypes))],
			Injured:     injured,
			Dead:        dead,
			CollectedAt: collectedAt,
		}
	}
	return accidents
}

func weightedRandom(rng *rand.Rand, weights []int, ranges [][2]int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rng.Intn(total)
	cum := 0
	for i, w := range weights {
		cum += w
		if r < cum {
			lo, hi := ranges[i][0], ranges[i][1]
			if lo == hi {
				return lo
			}
			return lo + rng.Intn(hi-lo+1)
		}
	}
	return 0
}

func newUUID(rng *rand.Rand) string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rng.Intn(256))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // RFC 4122 version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant bits
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func CollectShards(shards []string, workerID int, logger *zap.Logger) ([]Accident, error) {
	type result struct {
		region    string
		accidents []Accident
		err       error
	}

	ch := make(chan result, len(shards))

	for _, shard := range shards {
		region := shard
		go func() {
			logger.Info("Collecting region", zap.String("region", region))
			accs := GenerateMockAccidents(region, 50)
			ch <- result{region: region, accidents: accs}
		}()
	}

	var all []Accident
	for range shards {
		r := <-ch
		if r.err != nil {
			logger.Error("Collection error",
				zap.String("region", r.region),
				zap.Error(r.err),
			)
			continue
		}
		logger.Info("Region collected",
			zap.String("region", r.region),
			zap.Int("records", len(r.accidents)),
		)
		all = append(all, r.accidents...)
	}

	return all, nil
}
