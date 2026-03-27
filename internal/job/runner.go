// Package job provides scheduled background jobs for the event lifecycle engine.
// Each job runs in its own goroutine with a time.Ticker, processing batches of
// events with configurable limits for items, failures, and runtime.
package job

import (
	"context"
	"log"
	"os"
	"time"

	"event-engine-starter/internal/model"
)

// FetchFunc retrieves a batch of events to process.
type FetchFunc func(ctx context.Context, batchSize int) ([]model.Event, error)

// ProcessFunc processes a single event.
type ProcessFunc func(ctx context.Context, event *model.Event) error

// JobConfig holds parameters for a scheduled job.
type JobConfig struct {
	Name         string
	Interval     time.Duration
	StartupDelay time.Duration
	BatchSize    int
	MaxItems     int
	MaxFails     int
	MaxRuntime   time.Duration
}

// Job represents a scheduled background job that periodically fetches and
// processes events.
type Job struct {
	config  JobConfig
	fetch   FetchFunc
	process ProcessFunc
	stopCh  chan struct{}
	logger  *log.Logger
}

// NewJob creates a new Job with the given configuration, fetch function, and
// process function.
func NewJob(cfg JobConfig, fetch FetchFunc, process ProcessFunc) *Job {
	logger := log.New(os.Stdout, "["+cfg.Name+"] ", log.LstdFlags|log.Lmsgprefix)
	return &Job{
		config:  cfg,
		fetch:   fetch,
		process: process,
		stopCh:  make(chan struct{}),
		logger:  logger,
	}
}

// Start launches the job in a background goroutine. It waits for the configured
// startup delay, then runs on each tick of the configured interval. The provided
// context is used as the parent for all operations.
func (j *Job) Start(ctx context.Context) {
	go func() {
		// Wait for startup delay.
		select {
		case <-time.After(j.config.StartupDelay):
		case <-j.stopCh:
			return
		case <-ctx.Done():
			return
		}

		ticker := time.NewTicker(j.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				j.execute(ctx)
			case <-j.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop signals the job to stop processing. It is safe to call multiple times.
func (j *Job) Stop() {
	select {
	case <-j.stopCh:
		// Already stopped.
	default:
		close(j.stopCh)
	}
}

// execute runs a single execution cycle: fetch batches and process items until
// limits are reached or no more items are available.
func (j *Job) execute(parentCtx context.Context) {
	startTime := time.Now()

	// Create a context with timeout for the maximum runtime.
	ctx, cancel := context.WithTimeout(parentCtx, j.config.MaxRuntime)
	defer cancel()

	totalProcessed := 0
	totalFailed := 0

	for {
		// Check limits before fetching.
		if totalProcessed >= j.config.MaxItems {
			j.logger.Printf("max items reached (%d), stopping execution", totalProcessed)
			break
		}
		if totalFailed >= j.config.MaxFails {
			j.logger.Printf("max failures reached (%d), stopping execution", totalFailed)
			break
		}
		if ctx.Err() != nil {
			j.logger.Printf("context done (runtime exceeded or canceled), stopping execution")
			break
		}

		// Fetch a batch of events.
		events, err := j.fetch(ctx, j.config.BatchSize)
		if err != nil {
			j.logger.Printf("fetch error: %v", err)
			break
		}
		if len(events) == 0 {
			break
		}

		// Process each event in the batch.
		for i := range events {
			if totalProcessed >= j.config.MaxItems {
				break
			}
			if totalFailed >= j.config.MaxFails {
				break
			}
			if ctx.Err() != nil {
				break
			}

			if err := j.process(ctx, &events[i]); err != nil {
				totalFailed++
				j.logger.Printf("process error for event %s: %v", events[i].ID, err)
			} else {
				totalProcessed++
			}
		}

		// If we got fewer items than the batch size, there are no more to fetch.
		if len(events) < j.config.BatchSize {
			break
		}
	}

	elapsed := time.Since(startTime)
	j.logger.Printf("execution complete: processed=%d failed=%d duration=%s", totalProcessed, totalFailed, elapsed)
}
