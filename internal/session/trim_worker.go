package session

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

const TrimInterval = time.Hour

type TrimArgs struct{}

func (TrimArgs) Kind() string {
	return "session_trim"
}

type TrimWorker struct {
	river.WorkerDefaults[TrimArgs]
	store  *store
	logger *slog.Logger
}

func NewTrimWorker(store *store, logger *slog.Logger) *TrimWorker {
	return &TrimWorker{
		store:  store,
		logger: logger,
	}
}

func (w *TrimWorker) Work(ctx context.Context, job *river.Job[TrimArgs]) error {
	start := time.Now()

	numTrimmed, err := w.store.trim()
	if err != nil {
		w.logger.Error("error trimming sessions", "error", err)
		return err
	}

	w.logger.Info("successfully completed session trim worker",
		"num_trimmed", numTrimmed,
		"duration_ms", time.Since(start).Milliseconds())

	return nil
}
