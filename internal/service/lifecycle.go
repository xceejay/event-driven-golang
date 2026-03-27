package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"event-engine-starter/internal/model"
	"event-engine-starter/internal/repository"
)

const (
	maxRetries   = 10
	retryDelayMs = 15
)

// PublishRequest carries the data needed to publish a new event.
type PublishRequest struct {
	EventType       string
	FlowType        model.FlowType
	FlowID          string
	Payload         string
	OnFailEventType *string
}

// SpinoffEvent describes a follow-up event created on successful processing.
type SpinoffEvent struct {
	EventType       string
	Payload         string
	OnFailEventType *string
}

// ProcessingContext is returned when a processing permit is acquired.
type ProcessingContext struct {
	Event   *model.Event
	Payload string
}

// EventLifecycleManagementService is the high-level orchestrator for the event
// lifecycle. It coordinates publishing, dispatching, processing permits, retries,
// spinoff events, and on-failure chaining.
type EventLifecycleManagementService struct {
	processing *EventProcessingService
	config     *EventConfigurationService
	dispatcher *EventDispatcher
	events     repository.EventRepository
	logger     *log.Logger

	// OnEventStateChange is an optional callback invoked after each state
	// transition, useful for broadcasting to WebSocket clients.
	OnEventStateChange func(event *model.Event)
}

// NewEventLifecycleManagementService creates a new lifecycle management service.
func NewEventLifecycleManagementService(
	processing *EventProcessingService,
	config *EventConfigurationService,
	dispatcher *EventDispatcher,
	events repository.EventRepository,
) *EventLifecycleManagementService {
	return &EventLifecycleManagementService{
		processing: processing,
		config:     config,
		dispatcher: dispatcher,
		events:     events,
		logger:     log.New(os.Stdout, "[lifecycle] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Publish creates a new event and its payload idempotently.
func (s *EventLifecycleManagementService) Publish(ctx context.Context, req PublishRequest) (uuid.UUID, error) {
	cfg, err := s.config.LoadConfig(ctx, req.EventType)
	if err != nil {
		return uuid.Nil, fmt.Errorf("publish: %w", err)
	}

	now := time.Now().UTC()
	eventID := uuid.New()

	// Calculate scheduling from first attempt config.
	var delay time.Duration
	if len(cfg.AttemptLifecycleConfigs) > 0 {
		delay = time.Duration(cfg.AttemptLifecycleConfigs[0].DelayBeforeAttemptSeconds) * time.Second
	}

	attemptScheduledAt := now.Add(delay)
	eventDueDate := now.Add(time.Duration(cfg.EventLifespanSeconds) * time.Second)

	// First attempt due date = scheduled time + lifespan / maxAttempts (heuristic).
	attemptLifespan := time.Duration(cfg.EventLifespanSeconds/int64(cfg.MaxAttempts)) * time.Second
	attemptDueDate := attemptScheduledAt.Add(attemptLifespan)

	scheduleState := model.ScheduleStateActive
	if delay > 0 {
		scheduleState = model.ScheduleStateSleeping
	}

	cmd := model.EventInitiationCommand{
		EventID:                eventID,
		EventType:              req.EventType,
		FlowType:               req.FlowType,
		FlowID:                 req.FlowID,
		OnFailEventType:        req.OnFailEventType,
		Status:                 model.StatusAwaitingProcessing,
		AttemptsLeft:           cfg.MaxAttempts,
		AttemptScheduledAt:     &attemptScheduledAt,
		AttemptDueDate:         &attemptDueDate,
		EventProcessingDueDate: &eventDueDate,
		ScheduleState:          scheduleState,
	}

	event, err := s.processing.InitiateIdempotently(ctx, cmd, req.Payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("publish event: %w", err)
	}

	s.notifyStateChange(event)
	s.logger.Printf("published event %s type=%s flow=%s", eventID, req.EventType, req.FlowType)

	return eventID, nil
}

// TryAcquireProcessingPermit attempts to lock an event for processing.
// Returns nil if the permit was not granted (event in wrong state or lost race).
func (s *EventLifecycleManagementService) TryAcquireProcessingPermit(ctx context.Context, eventID uuid.UUID) (*ProcessingContext, error) {
	event, err := s.processing.Load(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("acquire permit: load event: %w", err)
	}

	if event.Status != model.StatusDispatched {
		return nil, nil
	}

	now := time.Now().UTC()
	cfg, err := s.config.LoadConfig(ctx, event.EventType)
	if err != nil {
		return nil, fmt.Errorf("acquire permit: load config: %w", err)
	}

	attemptLifespan := time.Duration(cfg.EventLifespanSeconds/int64(cfg.MaxAttempts)) * time.Second
	attemptDueDate := now.Add(attemptLifespan)

	cmd := model.ProcessingPermitAcquisitionCommand{
		EventID:                event.ID,
		EventType:              event.EventType,
		Version:                event.Version,
		AttemptDueDate:         &attemptDueDate,
		EventProcessingDueDate: event.EventProcessingDueDate,
		ScheduleState:          model.ScheduleStateInactive,
	}

	acquired := false
	err = withRetry(func() (bool, error) {
		ok, retryErr := s.processing.TryAcquireProcessingPermit(ctx, cmd)
		if ok {
			acquired = true
		}
		return ok, retryErr
	})
	if err != nil {
		return nil, fmt.Errorf("acquire permit: %w", err)
	}

	if !acquired {
		return nil, nil
	}

	payload, err := s.processing.LoadPayload(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("acquire permit: load payload: %w", err)
	}

	// Re-load event to get updated state.
	event, err = s.processing.Load(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("acquire permit: reload event: %w", err)
	}

	s.notifyStateChange(event)
	return &ProcessingContext{Event: event, Payload: payload}, nil
}

// ReportSuccess marks an event as successfully processed and creates any
// spinoff events using the same flow ID.
func (s *EventLifecycleManagementService) ReportSuccess(ctx context.Context, eventID uuid.UUID, spinoffs []SpinoffEvent) error {
	event, err := s.processing.Load(ctx, eventID)
	if err != nil {
		return fmt.Errorf("report success: load event: %w", err)
	}

	cmd := model.MarkAsSucceededCommand{
		EventID:       event.ID,
		EventType:     event.EventType,
		Version:       event.Version,
		ScheduleState: model.ScheduleStateInactive,
	}

	err = withRetry(func() (bool, error) {
		return s.processing.TryMarkAsSucceeded(ctx, cmd)
	})
	if err != nil {
		return fmt.Errorf("report success: %w", err)
	}

	// Publish spinoff events.
	for _, spinoff := range spinoffs {
		_, err := s.Publish(ctx, PublishRequest{
			EventType:       spinoff.EventType,
			FlowType:        event.FlowType,
			FlowID:          event.FlowID,
			Payload:         spinoff.Payload,
			OnFailEventType: spinoff.OnFailEventType,
		})
		if err != nil {
			s.logger.Printf("failed to publish spinoff event %s for event %s: %v", spinoff.EventType, eventID, err)
		}
	}

	event, _ = s.processing.Load(ctx, eventID)
	s.notifyStateChange(event)
	s.logger.Printf("event %s processed successfully", eventID)

	return nil
}

// ReportFailure handles a failed processing attempt. If retries remain and the
// event has not expired, it schedules the next attempt. Otherwise, it marks the
// event as terminally failed and publishes any on-fail event.
func (s *EventLifecycleManagementService) ReportFailure(ctx context.Context, eventID uuid.UUID, errorMessage string, stackTrace *string) error {
	event, err := s.processing.Load(ctx, eventID)
	if err != nil {
		return fmt.Errorf("report failure: load event: %w", err)
	}

	cfg, err := s.config.LoadConfig(ctx, event.EventType)
	if err != nil {
		return fmt.Errorf("report failure: load config: %w", err)
	}

	// Save error log.
	errLog := model.EventErrorLog{
		ID:           uuid.New(),
		EventID:      eventID,
		ErrorMessage: errorMessage,
		StackTrace:   stackTrace,
		OccurredAt:   time.Now().UTC(),
	}
	if saveErr := s.processing.SaveErrorLog(ctx, errLog); saveErr != nil {
		s.logger.Printf("failed to save error log for event %s: %v", eventID, saveErr)
	}

	now := time.Now().UTC()
	hasRetriesLeft := event.AttemptsLeft > 1
	isNotExpired := event.EventProcessingDueDate != nil && now.Before(*event.EventProcessingDueDate)

	if hasRetriesLeft && isNotExpired {
		return s.scheduleNextAttempt(ctx, event, cfg)
	}

	return s.markAsTerminallyFailed(ctx, event)
}

// scheduleNextAttempt transitions the event back to AWAITING_PROCESSING with
// updated retry counters and scheduling.
func (s *EventLifecycleManagementService) scheduleNextAttempt(ctx context.Context, event *model.Event, cfg *model.EventLifecycleConfig) error {
	now := time.Now().UTC()
	nextAttemptNum := event.AttemptsFailed + 2 // 1-based, next attempt

	var delay time.Duration
	for _, ac := range cfg.AttemptLifecycleConfigs {
		if ac.AttemptNumber == nextAttemptNum {
			delay = time.Duration(ac.DelayBeforeAttemptSeconds) * time.Second
			break
		}
	}

	scheduledAt := now.Add(delay)
	attemptLifespan := time.Duration(cfg.EventLifespanSeconds/int64(cfg.MaxAttempts)) * time.Second
	attemptDueDate := scheduledAt.Add(attemptLifespan)

	scheduleState := model.ScheduleStateActive
	if delay > 0 {
		scheduleState = model.ScheduleStateSleeping
	}

	cmd := model.SwitchToNextAttemptCommand{
		EventID:            event.ID,
		EventType:          event.EventType,
		Version:            event.Version,
		AttemptsLeft:       event.AttemptsLeft - 1,
		AttemptsFailed:     event.AttemptsFailed + 1,
		AttemptScheduledAt: &scheduledAt,
		AttemptDueDate:     &attemptDueDate,
		ScheduleState:      scheduleState,
	}

	err := withRetry(func() (bool, error) {
		return s.processing.TrySwitchToNextAttempt(ctx, cmd)
	})
	if err != nil {
		return fmt.Errorf("schedule next attempt for event %s: %w", event.ID, err)
	}

	event, _ = s.processing.Load(ctx, event.ID)
	s.notifyStateChange(event)
	s.logger.Printf("event %s scheduled for retry (attempts_left=%d)", event.ID, cmd.AttemptsLeft)

	return nil
}

// markAsTerminallyFailed marks the event as FAILED and publishes any on-fail event.
func (s *EventLifecycleManagementService) markAsTerminallyFailed(ctx context.Context, event *model.Event) error {
	cmd := model.MarkAsFailedCommand{
		EventID:        event.ID,
		EventType:      event.EventType,
		Version:        event.Version,
		AttemptsFailed: event.AttemptsFailed + 1,
		ScheduleState:  model.ScheduleStateInactive,
	}

	err := withRetry(func() (bool, error) {
		return s.processing.TryMarkAsFailed(ctx, cmd)
	})
	if err != nil {
		return fmt.Errorf("mark event %s as failed: %w", event.ID, err)
	}

	// Publish on-fail event if configured.
	if event.OnFailEventType != nil {
		payload, _ := s.processing.LoadPayload(ctx, event.ID)
		_, pubErr := s.Publish(ctx, PublishRequest{
			EventType: *event.OnFailEventType,
			FlowType:  event.FlowType,
			FlowID:    event.FlowID,
			Payload:   payload,
		})
		if pubErr != nil {
			s.logger.Printf("failed to publish on-fail event %s for event %s: %v",
				*event.OnFailEventType, event.ID, pubErr)
		}
	}

	event, _ = s.processing.Load(ctx, event.ID)
	s.notifyStateChange(event)
	s.logger.Printf("event %s terminally failed", event.ID)

	return nil
}

// FindPendingEvents returns events in AWAITING_PROCESSING state for a flow type.
func (s *EventLifecycleManagementService) FindPendingEvents(ctx context.Context, flowType model.FlowType, batchSize int) ([]model.Event, error) {
	return s.events.FindByFlowAndCreatedBefore(ctx, model.CreatedBeforeFilter{
		FlowType:      flowType,
		Status:        model.StatusAwaitingProcessing,
		CreatedBefore: time.Now().UTC(),
		Limit:         batchSize,
	})
}

// FindScheduledEvents returns events with scheduled attempts that are due.
func (s *EventLifecycleManagementService) FindScheduledEvents(ctx context.Context, flowType model.FlowType, batchSize int) ([]model.Event, error) {
	return s.events.FindByFlowAndScheduledBefore(ctx, model.ScheduledBeforeInclusiveFilter{
		FlowType:        flowType,
		ScheduleState:   model.ScheduleStateActive,
		ScheduledBefore: time.Now().UTC(),
		Limit:           batchSize,
	})
}

// FindExpiredSuspendedEvents returns suspended events whose due date has passed.
func (s *EventLifecycleManagementService) FindExpiredSuspendedEvents(ctx context.Context, flowType model.FlowType, batchSize int) ([]model.Event, error) {
	return s.events.FindByFlowAndScheduledBefore(ctx, model.ScheduledBeforeInclusiveFilter{
		FlowType:        flowType,
		ScheduleState:   model.ScheduleStateSleeping,
		ScheduledBefore: time.Now().UTC(),
		Limit:           batchSize,
	})
}

// ProcessPendingEvent dispatches a single pending event to the broker.
func (s *EventLifecycleManagementService) ProcessPendingEvent(ctx context.Context, event *model.Event) error {
	cfg, err := s.config.LoadConfig(ctx, event.EventType)
	if err != nil {
		return fmt.Errorf("process pending event: %w", err)
	}

	if cfg.IsSuspended {
		return nil
	}

	cmd := model.MarkAsDispatchedCommand{
		EventID:       event.ID,
		EventType:     event.EventType,
		Version:       event.Version,
		ScheduleState: model.ScheduleStateActive,
	}

	dispatched := false
	err = withRetry(func() (bool, error) {
		ok, retryErr := s.processing.TryMarkAsDispatched(ctx, cmd)
		if ok {
			dispatched = true
		}
		return ok, retryErr
	})
	if err != nil {
		return fmt.Errorf("dispatch event %s: %w", event.ID, err)
	}

	if !dispatched {
		return nil
	}

	payload, err := s.processing.LoadPayload(ctx, event.ID)
	if err != nil {
		s.logger.Printf("failed to load payload for dispatched event %s: %v", event.ID, err)
		payload = "{}"
	}

	if err := s.dispatcher.Dispatch(ctx, event, payload); err != nil {
		s.logger.Printf("failed to dispatch event %s to broker: %v", event.ID, err)
		return err
	}

	event, _ = s.processing.Load(ctx, event.ID)
	s.notifyStateChange(event)

	return nil
}

// ActivateSuspendedEvent reactivates a suspended event.
func (s *EventLifecycleManagementService) ActivateSuspendedEvent(ctx context.Context, event *model.Event) error {
	cmd := model.ActivateSuspendedCommand{
		EventID:       event.ID,
		EventType:     event.EventType,
		Version:       event.Version,
		ScheduleState: model.ScheduleStateActive,
	}

	err := withRetry(func() (bool, error) {
		return s.processing.TryActivateSuspended(ctx, cmd)
	})
	if err != nil {
		return fmt.Errorf("activate suspended event %s: %w", event.ID, err)
	}

	event, _ = s.processing.Load(ctx, event.ID)
	s.notifyStateChange(event)

	return nil
}

// SuspendEventType pauses dispatch of all events of the given type.
func (s *EventLifecycleManagementService) SuspendEventType(ctx context.Context, eventType string) error {
	return s.config.SuspendProcessing(ctx, eventType)
}

// UnsuspendEventType resumes dispatch of events of the given type.
func (s *EventLifecycleManagementService) UnsuspendEventType(ctx context.Context, eventType string) error {
	return s.config.UnsuspendProcessing(ctx, eventType)
}

// GetQueueSize returns the count of events for a flow type and status.
func (s *EventLifecycleManagementService) GetQueueSize(ctx context.Context, flowType model.FlowType, status model.EventStatus) (int64, error) {
	return s.processing.GetQueueSize(ctx, flowType, status)
}

// notifyStateChange invokes the optional callback if set.
func (s *EventLifecycleManagementService) notifyStateChange(event *model.Event) {
	if s.OnEventStateChange != nil && event != nil {
		s.OnEventStateChange(event)
	}
}

// withRetry retries a state transition up to maxRetries times on race condition
// (function returns false). On success (true) or unexpected error, it returns
// immediately.
func withRetry(fn func() (bool, error)) error {
	for i := 0; i < maxRetries; i++ {
		ok, err := fn()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		time.Sleep(retryDelayMs * time.Millisecond)
	}
	return fmt.Errorf("max retries (%d) exceeded for state transition", maxRetries)
}
