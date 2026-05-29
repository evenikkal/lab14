package main

import "time"

type WindowBatch struct {
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	Count       int       `json:"count"`
	SumDead     int       `json:"sum_dead"`
	AvgInjured  float64   `json:"avg_injured"`
	MinDate     time.Time `json:"min_date"`
	MaxDate     time.Time `json:"max_date"`
}

func TumblingWindow(in <-chan Accident) <-chan WindowBatch {
	const (
		maxCount  = 100
		windowDur = 30 * time.Second
	)

	out := make(chan WindowBatch)

	go func() {
		defer close(out)

		windowStart := time.Now()
		buf := make([]Accident, 0, maxCount)

		timer := time.NewTimer(windowDur)
		defer timer.Stop()

		doFlush := func(end time.Time) {
			if len(buf) == 0 {
				return
			}
			out <- aggregateWindow(buf, windowStart, end)
			buf = buf[:0]
			windowStart = end
		}

		for {
			select {
			case acc, ok := <-in:
				if !ok {
					doFlush(time.Now())
					return
				}
				buf = append(buf, acc)
				if len(buf) >= maxCount {
					now := time.Now()
					doFlush(now)
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(windowDur)
				}
			case t := <-timer.C:
				doFlush(t)
				timer.Reset(windowDur)
			}
		}
	}()

	return out
}

func aggregateWindow(buf []Accident, start, end time.Time) WindowBatch {
	sumDead := 0
	sumInjured := 0
	minDate := buf[0].Date
	maxDate := buf[0].Date

	for _, a := range buf {
		sumDead += a.Dead
		sumInjured += a.Injured
		if a.Date.Before(minDate) {
			minDate = a.Date
		}
		if a.Date.After(maxDate) {
			maxDate = a.Date
		}
	}

	return WindowBatch{
		WindowStart: start,
		WindowEnd:   end,
		Count:       len(buf),
		SumDead:     sumDead,
		AvgInjured:  float64(sumInjured) / float64(len(buf)),
		MinDate:     minDate,
		MaxDate:     maxDate,
	}
}
