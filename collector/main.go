package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

func main() {
	workerIDFlag := flag.Int("worker-id", 0, "Worker ID (overridden by WORKER_ID env var)")
	modeFlag := flag.String("mode", "worker", "Run mode: leader or worker")
	flag.Parse()

	workerID := *workerIDFlag
	if v := os.Getenv("WORKER_ID"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			workerID = id
		}
	}

	etcdURL := os.Getenv("ETCD_URL")
	if etcdURL == "" {
		etcdURL = "http://localhost:2379"
	}

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	logger.Info("Collector starting",
		zap.String("mode", *modeFlag),
		zap.Int("worker_id", workerID),
		zap.String("etcd_url", etcdURL),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("Signal received, shutting down", zap.String("signal", sig.String()))
		cancel()
	}()

	if *modeFlag == "window" {
		runWindowMode(ctx, logger)
		return
	}

	if *modeFlag == "nats-producer" {
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			natsURL = "nats://localhost:4222"
		}
		if err := RunNATSProducer(ctx, natsURL, logger); err != nil {
			logger.Fatal("NATS producer error", zap.Error(err))
		}
		return
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdURL},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logger.Fatal("etcd connection failed", zap.Error(err))
	}
	defer client.Close()

	switch *modeFlag {
	case "leader":
		if err := RunLeader(ctx, client, logger); err != nil {
			logger.Fatal("Leader error", zap.Error(err))
		}

	case "worker":
		runWorker(ctx, client, workerID, logger)

	default:
		logger.Fatal("Unknown mode (use leader, worker, window, or nats-producer)",
			zap.String("mode", *modeFlag))
	}
}

func runWorker(ctx context.Context, client *clientv3.Client, workerID int, logger *zap.Logger) {
	sess, err := concurrency.NewSession(client, concurrency.WithTTL(30))
	if err != nil {
		logger.Fatal("Failed to create etcd session", zap.Error(err))
	}
	defer sess.Close()

	logger.Info("Waiting for shard configuration from leader...")
	var shards []string
	for {
		if ctx.Err() != nil {
			return
		}
		shards, err = GetShards(ctx, client)
		if err == nil {
			break
		}
		logger.Warn("Shards not yet available, retrying in 2 s", zap.Error(err))
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}

	logger.Info("Shard list received", zap.Strings("shards", shards))

	type lockedShard struct {
		region string
		unlock func()
	}
	var locked []lockedShard

	for _, region := range shards {
		unlock, err := TryLockShard(ctx, sess, region)
		if err != nil {
			logger.Debug("Shard unavailable (held by another worker)",
				zap.String("region", region))
			continue
		}
		locked = append(locked, lockedShard{region: region, unlock: unlock})
		logger.Info("Acquired shard lock", zap.String("region", region))
	}

	if len(locked) == 0 {
		logger.Info("No shards available for this worker — all locked by peers")
		return
	}

	regions := make([]string, len(locked))
	for i, s := range locked {
		regions[i] = s.region
	}

	accidents, collErr := CollectShards(regions, workerID, logger)

	for _, s := range locked {
		s.unlock()
	}

	if collErr != nil {
		logger.Fatal("Data collection failed", zap.Error(collErr))
	}
	if len(accidents) == 0 {
		logger.Warn("No accidents collected — nothing to write")
		return
	}

	if err := WriteJSONL(accidents, workerID, logger); err != nil {
		logger.Fatal("Write failed", zap.Error(err))
	}

	logger.Info("Worker completed",
		zap.Int("worker_id", workerID),
		zap.Int("regions_processed", len(regions)),
		zap.Int("total_records", len(accidents)),
	)
}

func runWindowMode(ctx context.Context, logger *zap.Logger) {
	in := make(chan Accident, 200)

	go func() {
		defer close(in)
		for _, region := range Regions {
			for _, acc := range GenerateMockAccidents(region, 50) {
				select {
				case in <- acc:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	var batches []WindowBatch
	for batch := range TumblingWindow(in) {
		logger.Info("Window batch flushed",
			zap.Time("window_start", batch.WindowStart),
			zap.Time("window_end", batch.WindowEnd),
			zap.Int("count", batch.Count),
			zap.Int("sum_dead", batch.SumDead),
			zap.Float64("avg_injured", batch.AvgInjured),
		)
		batches = append(batches, batch)
	}

	if err := WriteWindowsJSONL(batches, logger); err != nil {
		logger.Fatal("Write windows failed", zap.Error(err))
	}

	logger.Info("Window mode completed", zap.Int("total_batches", len(batches)))
}
