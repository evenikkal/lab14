package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

const (
	shardsKey   = "/lab14/shards"
	lockPrefix  = "/lab14/lock/"
	electionKey = "/lab14/election"
)

var Regions = []string{
	"Центральный",
	"Северный",
	"Южный",
	"Восточный",
	"Западный",
	"Северо-Западный",
	"Приволжский",
	"Уральский",
	"Сибирский",
	"Дальневосточный",
}

func RunLeader(ctx context.Context, client *clientv3.Client, logger *zap.Logger) error {
	sess, err := concurrency.NewSession(client, concurrency.WithTTL(15))
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer sess.Close()

	election := concurrency.NewElection(sess, electionKey)
	logger.Info("Campaigning for leadership...")

	if err := election.Campaign(ctx, "leader"); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("campaign: %w", err)
	}

	defer func() {
		resignCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = election.Resign(resignCtx)
	}()

	logger.Info("Won leadership, publishing shard list")

	data, err := json.Marshal(Regions)
	if err != nil {
		return fmt.Errorf("marshal regions: %w", err)
	}

	if _, err := client.Put(ctx, shardsKey, string(data)); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("put shards key: %w", err)
	}

	logger.Info("Shard list published",
		zap.Int("shards", len(Regions)),
		zap.String("key", shardsKey),
	)

	<-ctx.Done()
	logger.Info("Leader shutting down")
	return nil
}

func GetShards(ctx context.Context, client *clientv3.Client) ([]string, error) {
	resp, err := client.Get(ctx, shardsKey)
	if err != nil {
		return nil, fmt.Errorf("etcd get %s: %w", shardsKey, err)
	}
	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("key %q not found — is the leader running?", shardsKey)
	}

	var shards []string
	if err := json.Unmarshal(resp.Kvs[0].Value, &shards); err != nil {
		return nil, fmt.Errorf("unmarshal shards: %w", err)
	}
	return shards, nil
}

func TryLockShard(ctx context.Context, sess *concurrency.Session, region string) (func(), error) {
	mu := concurrency.NewMutex(sess, lockPrefix+region)

	lockCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := mu.TryLock(lockCtx); err != nil {
		return nil, err
	}

	return func() {
		unlockCtx, ucancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer ucancel()
		_ = mu.Unlock(unlockCtx)
	}, nil
}
