package model

import (
	"errors"
	"fmt"
)

// Sentinel errors for domain-level error matching.
var (
	// ErrRaceCondition indicates an optimistic locking conflict where the event
	// version has changed between read and update.
	ErrRaceCondition = errors.New("race condition: event version mismatch")

	// ErrEventNotFound indicates that the requested event does not exist.
	ErrEventNotFound = errors.New("event not found")

	// ErrConfigNotFound indicates that no lifecycle configuration exists for
	// the given event type.
	ErrConfigNotFound = errors.New("lifecycle config not found")

	// ErrPayloadNotFound indicates that no payload was found for the given event ID.
	ErrPayloadNotFound = errors.New("payload not found")
)

// RaceConditionError wraps ErrRaceCondition with additional context about the
// conflicting event and version.
type RaceConditionError struct {
	EventID        string
	ExpectedVersion int64
	Message        string
}

// Error returns a human-readable description of the race condition.
func (e *RaceConditionError) Error() string {
	return fmt.Sprintf("race condition on event %s (expected version %d): %s",
		e.EventID, e.ExpectedVersion, e.Message)
}

// Unwrap returns the sentinel ErrRaceCondition so callers can use errors.Is.
func (e *RaceConditionError) Unwrap() error {
	return ErrRaceCondition
}
