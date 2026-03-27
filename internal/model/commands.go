package model

import (
	"time"

	"github.com/google/uuid"
)

// EventInitiationCommand carries all data needed to create a new event.
type EventInitiationCommand struct {
	EventID                uuid.UUID
	EventType              string
	FlowType               FlowType
	FlowID                 string
	OnFailEventType        *string
	Status                 EventStatus
	AttemptsLeft           int
	AttemptScheduledAt     *time.Time
	AttemptDueDate         *time.Time
	EventProcessingDueDate *time.Time
	ScheduleState          ScheduleState
}

// MarkAsDispatchedCommand transitions an event from AWAITING_PROCESSING to DISPATCHED.
type MarkAsDispatchedCommand struct {
	EventID       uuid.UUID
	EventType     string
	Version       int64
	ScheduleState ScheduleState
}

// ProcessingPermitAcquisitionCommand transitions an event from DISPATCHED to BEING_PROCESSED.
type ProcessingPermitAcquisitionCommand struct {
	EventID                uuid.UUID
	EventType              string
	Version                int64
	AttemptDueDate         *time.Time
	EventProcessingDueDate *time.Time
	ScheduleState          ScheduleState
}

// SwitchToNextAttemptCommand transitions a failed attempt back to AWAITING_PROCESSING
// with updated retry counters and scheduling.
type SwitchToNextAttemptCommand struct {
	EventID            uuid.UUID
	EventType          string
	Version            int64
	AttemptsLeft       int
	AttemptsFailed     int
	AttemptScheduledAt *time.Time
	AttemptDueDate     *time.Time
	ScheduleState      ScheduleState
}

// MarkAsSucceededCommand transitions an event to the PROCESSED terminal state.
type MarkAsSucceededCommand struct {
	EventID       uuid.UUID
	EventType     string
	Version       int64
	ScheduleState ScheduleState
}

// MarkAsFailedCommand transitions an event to the FAILED terminal state.
type MarkAsFailedCommand struct {
	EventID        uuid.UUID
	EventType      string
	Version        int64
	AttemptsFailed int
	ScheduleState  ScheduleState
}

// ActivateSuspendedCommand reactivates a suspended event, transitioning it
// back to AWAITING_PROCESSING.
type ActivateSuspendedCommand struct {
	EventID       uuid.UUID
	EventType     string
	Version       int64
	ScheduleState ScheduleState
}
