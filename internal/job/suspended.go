package job

import (
	"context"

	"event-engine-starter/internal/model"
)

// NewExpiredSuspendedJob creates a job that reactivates suspended events whose
// attempt due date has passed.
func NewExpiredSuspendedJob(flowType model.FlowType, svc LifecycleService, cfg JobConfig) *Job {
	fetch := func(ctx context.Context, batchSize int) ([]model.Event, error) {
		return svc.FindExpiredSuspendedEvents(ctx, flowType, batchSize)
	}

	process := func(ctx context.Context, event *model.Event) error {
		return svc.ActivateSuspendedEvent(ctx, event)
	}

	return NewJob(cfg, fetch, process)
}
