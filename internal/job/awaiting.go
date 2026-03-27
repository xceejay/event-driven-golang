package job

import (
	"context"

	"event-engine-starter/internal/model"
)

// NewAwaitingProcessingJob creates a job that processes events in the
// AWAITING_PROCESSING state by dispatching them for a given flow type.
func NewAwaitingProcessingJob(flowType model.FlowType, svc LifecycleService, cfg JobConfig) *Job {
	fetch := func(ctx context.Context, batchSize int) ([]model.Event, error) {
		return svc.FindPendingEvents(ctx, flowType, batchSize)
	}

	process := func(ctx context.Context, event *model.Event) error {
		return svc.ProcessPendingEvent(ctx, event)
	}

	return NewJob(cfg, fetch, process)
}
