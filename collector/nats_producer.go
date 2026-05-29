package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// RunNATSProducer connects to NATS and publishes mock Accident events to the
// "accidents" subject at a rate of one event per 100 ms until ctx is cancelled.
func RunNATSProducer(ctx context.Context, natsURL string, logger *zap.Logger) error {
	nc, err := nats.Connect(natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info("NATS reconnected")
		}),
	)
	if err != nil {
		return fmt.Errorf("nats connect %s: %w", natsURL, err)
	}
	defer func() {
		nc.Drain() //nolint:errcheck
		logger.Info("NATS producer stopped")
	}()

	logger.Info("NATS producer started", zap.String("url", natsURL))

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	regionIdx := 0
	published := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			region := Regions[regionIdx%len(Regions)]
			regionIdx++

			accidents := GenerateMockAccidents(region, 1)
			acc := accidents[0]

			data, err := json.Marshal(acc)
			if err != nil {
				logger.Error("marshal accident", zap.Error(err))
				continue
			}

			if err := nc.Publish("accidents", data); err != nil {
				logger.Warn("NATS publish error", zap.Error(err))
				continue
			}

			published++
			if published%10 == 0 {
				logger.Info("NATS published events",
					zap.Int("total", published),
					zap.String("region", acc.Region),
				)
			} else {
				logger.Debug("NATS published event",
					zap.String("id", acc.ID),
					zap.String("region", acc.Region),
					zap.Int("injured", acc.Injured),
					zap.Int("dead", acc.Dead),
				)
			}
		}
	}
}
