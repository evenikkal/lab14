package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func WriteJSONL(accidents []Accident, workerID int, logger *zap.Logger) error {
	dir := "../data"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	path := filepath.Join(dir, fmt.Sprintf("shard_%d.jsonl", workerID))

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output file %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, acc := range accidents {
		line, err := json.Marshal(acc)
		if err != nil {
			return fmt.Errorf("marshal accident %s: %w", acc.ID, err)
		}
		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush writer: %w", err)
	}

	logger.Info("Output written",
		zap.String("file", path),
		zap.Int("records", len(accidents)),
	)
	return nil
}
