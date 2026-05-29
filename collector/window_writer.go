package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
)

func WriteWindowsJSONL(batches []WindowBatch, logger *zap.Logger) error {
	dir := "../data"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	path := dir + "/windows.jsonl"

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output file %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, batch := range batches {
		line, err := json.Marshal(batch)
		if err != nil {
			return fmt.Errorf("marshal window batch: %w", err)
		}
		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush writer: %w", err)
	}

	logger.Info("Windows written",
		zap.String("file", path),
		zap.Int("batches", len(batches)),
	)
	return nil
}
