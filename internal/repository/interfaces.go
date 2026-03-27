package repository

import (
	"context"

	"github.com/google/uuid"

	"event-engine-starter/internal/model"
)

// EventRepository manages event persistence with optimistic locking.
type EventRepository interface {
	Create(ctx context.Context, cmd model.EventInitiationCommand) (bool, error)
	Load(ctx context.Context, eventID uuid.UUID) (*model.Event, error)
	FindByFlowAndCreatedBefore(ctx context.Context, filter model.CreatedBeforeFilter) ([]model.Event, error)
	FindByFlowAndScheduledBefore(ctx context.Context, filter model.ScheduledBeforeInclusiveFilter) ([]model.Event, error)
	MarkAsDispatched(ctx context.Context, cmd model.MarkAsDispatchedCommand) error
	AcquireProcessingPermit(ctx context.Context, cmd model.ProcessingPermitAcquisitionCommand) error
	SwitchToNextAttempt(ctx context.Context, cmd model.SwitchToNextAttemptCommand) error
	MarkAsSucceeded(ctx context.Context, cmd model.MarkAsSucceededCommand) error
	MarkAsFailed(ctx context.Context, cmd model.MarkAsFailedCommand) error
	ActivateSuspended(ctx context.Context, cmd model.ActivateSuspendedCommand) error
	GetQueueSize(ctx context.Context, flowType model.FlowType, status model.EventStatus) (int64, error)
}

// PayloadRepository manages event payload storage.
type PayloadRepository interface {
	Save(ctx context.Context, eventID uuid.UUID, payload string) error
	Load(ctx context.Context, eventID uuid.UUID) (string, error)
}

// ConfigRepository manages event lifecycle configuration.
type ConfigRepository interface {
	FindByEventType(ctx context.Context, eventType string) (*model.EventLifecycleConfig, error)
	Save(ctx context.Context, cfg model.EventLifecycleConfig) error
	UpdateSuspensionState(ctx context.Context, eventType string, suspended bool) error
}

// ErrorLogRepository manages event error logs.
type ErrorLogRepository interface {
	Save(ctx context.Context, log model.EventErrorLog) error
	FindByEventID(ctx context.Context, eventID uuid.UUID) ([]model.EventErrorLog, error)
}
