package job

import (
	"context"

	"event-engine-starter/internal/model"
)

// NewScheduledEventsJob creates a job that processes events with scheduled
// attempts that are now due for execution.
func NewScheduledEventsJob(flowType model.FlowType, svc LifecycleService, cfg JobConfig) *Job {
	fetch := func(ctx context.Context, batchSize int) ([]model.Event, error) {
		return svc.FindScheduledEvents(ctx, flowType, batchSize)
	}

	process := func(ctx context.Context, event *model.Event) error {
		return svc.ProcessPendingEvent(ctx, event)
	}

	return NewJob(cfg, fetch, process)
}
