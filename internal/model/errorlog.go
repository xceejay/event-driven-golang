package model

import (
	"time"

	"github.com/google/uuid"
)

// EventErrorLog records an error that occurred during event processing,
// providing an audit trail for debugging and operational visibility.
type EventErrorLog struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	EventID      uuid.UUID  `db:"event_id" json:"event_id"`
	ErrorMessage string     `db:"error_message" json:"error_message"`
	StackTrace   *string    `db:"stack_trace" json:"stack_trace,omitempty"`
	OccurredAt   time.Time  `db:"occurred_at" json:"occurred_at"`
}
