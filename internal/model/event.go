package model

import (
	"time"

	"github.com/google/uuid"
)

// Event is the central domain entity representing a unit of work moving
// through the event lifecycle state machine.
type Event struct {
	ID       uuid.UUID   `db:"id" json:"id"`
	EventType string    `db:"event_type" json:"event_type"`
	FlowType  FlowType  `db:"flow_type" json:"flow_type"`
	FlowID    string    `db:"flow_id" json:"flow_id"`

	Status  EventStatus `db:"status" json:"status"`
	Version int64       `db:"version" json:"version"`

	AttemptsLeft   int `db:"attempts_left" json:"attempts_left"`
	AttemptsFailed int `db:"attempts_failed" json:"attempts_failed"`

	AttemptScheduledAt      *time.Time `db:"attempt_scheduled_at" json:"attempt_scheduled_at,omitempty"`
	AttemptDueDate          *time.Time `db:"attempt_due_date" json:"attempt_due_date,omitempty"`
	EventProcessingDueDate  *time.Time `db:"event_processing_due_date" json:"event_processing_due_date,omitempty"`

	OnFailEventType *string       `db:"on_fail_event_type" json:"on_fail_event_type,omitempty"`
	ScheduleState   ScheduleState `db:"schedule_state" json:"schedule_state"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// EventReference is a lightweight reference to an event, used in spinoff
// chains and cross-event relationships.
type EventReference struct {
	EventID   uuid.UUID `json:"event_id"`
	EventType string    `json:"event_type"`
}
