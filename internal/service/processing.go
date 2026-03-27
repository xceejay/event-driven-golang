package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"event-engine-starter/internal/model"
	"event-engine-starter/internal/repository"
)

// EventProcessingService is a thin layer over repositories that owns all
// event state transitions. It does NOT retry on race conditions — that
// responsibility belongs to the higher-level lifecycle management service.
type EventProcessingService struct {
	events    repository.EventRepository
	payloads  repository.PayloadRepository
	errorLogs repository.ErrorLogRepository
}

// NewEventProcessingService creates a new processing service.
func NewEventProcessingService(
	events repository.EventRepository,
	payloads repository.PayloadRepository,
	errorLogs repository.ErrorLogRepository,
) *EventProcessingService {
	return &EventProcessingService{
		events:    events,
		payloads:  payloads,
		errorLogs: errorLogs,
	}
}

// InitiateIdempotently creates an event and stores its payload. If the event
// already exists (same ID), the existing event is loaded and returned.
func (s *EventProcessingService) InitiateIdempotently(ctx context.Context, cmd model.EventInitiationCommand, payload string) (*model.Event, error) {
	created, err := s.events.Create(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("create event %s: %w", cmd.EventID, err)
	}

	if created {
		if err := s.payloads.Save(ctx, cmd.EventID, payload); err != nil {
			return nil, fmt.Errorf("save payload for event %s: %w", cmd.EventID, err)
		}
	}

	event, err := s.events.Load(ctx, cmd.EventID)
	if err != nil {
		return nil, fmt.Errorf("load event %s after initiation: %w", cmd.EventID, err)
	}

	return event, nil
}

// Load retrieves an event by ID.
func (s *EventProcessingService) Load(ctx context.Context, eventID uuid.UUID) (*model.Event, error) {
	return s.events.Load(ctx, eventID)
}

// LoadPayload retrieves the payload for an event.
func (s *EventProcessingService) LoadPayload(ctx context.Context, eventID uuid.UUID) (string, error) {
	return s.payloads.Load(ctx, eventID)
}

// TryMarkAsDispatched attempts to transition the event to DISPATCHED.
// Returns (true, nil) on success, (false, nil) on race condition, or
// (false, err) on unexpected error.
func (s *EventProcessingService) TryMarkAsDispatched(ctx context.Context, cmd model.MarkAsDispatchedCommand) (bool, error) {
	err := s.events.MarkAsDispatched(ctx, cmd)
	return handleOptimisticLock(err)
}

// TryAcquireProcessingPermit attempts to transition the event to BEING_PROCESSED.
func (s *EventProcessingService) TryAcquireProcessingPermit(ctx context.Context, cmd model.ProcessingPermitAcquisitionCommand) (bool, error) {
	err := s.events.AcquireProcessingPermit(ctx, cmd)
	return handleOptimisticLock(err)
}

// TrySwitchToNextAttempt transitions a failed event back to AWAITING_PROCESSING
// with updated retry counters.
func (s *EventProcessingService) TrySwitchToNextAttempt(ctx context.Context, cmd model.SwitchToNextAttemptCommand) (bool, error) {
	err := s.events.SwitchToNextAttempt(ctx, cmd)
	return handleOptimisticLock(err)
}

// TryMarkAsSucceeded transitions the event to the PROCESSED terminal state.
func (s *EventProcessingService) TryMarkAsSucceeded(ctx context.Context, cmd model.MarkAsSucceededCommand) (bool, error) {
	err := s.events.MarkAsSucceeded(ctx, cmd)
	return handleOptimisticLock(err)
}

// TryMarkAsFailed transitions the event to the FAILED terminal state.
func (s *EventProcessingService) TryMarkAsFailed(ctx context.Context, cmd model.MarkAsFailedCommand) (bool, error) {
	err := s.events.MarkAsFailed(ctx, cmd)
	return handleOptimisticLock(err)
}

// TryActivateSuspended reactivates a suspended event back to AWAITING_PROCESSING.
func (s *EventProcessingService) TryActivateSuspended(ctx context.Context, cmd model.ActivateSuspendedCommand) (bool, error) {
	err := s.events.ActivateSuspended(ctx, cmd)
	return handleOptimisticLock(err)
}

// SaveErrorLog persists an error log entry.
func (s *EventProcessingService) SaveErrorLog(ctx context.Context, errLog model.EventErrorLog) error {
	return s.errorLogs.Save(ctx, errLog)
}

// GetQueueSize returns the count of events for a flow type and status.
func (s *EventProcessingService) GetQueueSize(ctx context.Context, flowType model.FlowType, status model.EventStatus) (int64, error) {
	return s.events.GetQueueSize(ctx, flowType, status)
}

// handleOptimisticLock converts a race condition error into (false, nil).
func handleOptimisticLock(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if errors.Is(err, model.ErrRaceCondition) {
		return false, nil
	}
	return false, err
}
