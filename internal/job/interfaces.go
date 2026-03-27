package job

import (
	"context"

	"event-engine-starter/internal/model"
)

// LifecycleService defines the operations jobs need from the lifecycle
// management service. Defined here to avoid circular imports.
type LifecycleService interface {
	FindPendingEvents(ctx context.Context, flowType model.FlowType, batchSize int) ([]model.Event, error)
	FindScheduledEvents(ctx context.Context, flowType model.FlowType, batchSize int) ([]model.Event, error)
	FindExpiredSuspendedEvents(ctx context.Context, flowType model.FlowType, batchSize int) ([]model.Event, error)
	ProcessPendingEvent(ctx context.Context, event *model.Event) error
	ActivateSuspendedEvent(ctx context.Context, event *model.Event) error
	GetQueueSize(ctx context.Context, flowType model.FlowType, status model.EventStatus) (int64, error)
}
