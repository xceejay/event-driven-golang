// Package model defines the core domain types for the event lifecycle engine.
package model

// EventStatus represents the current processing state of an event.
type EventStatus string

const (
	// StatusAwaitingProcessing indicates the event is queued and waiting to be dispatched.
	StatusAwaitingProcessing EventStatus = "AWAITING_PROCESSING"

	// StatusDispatched indicates the event has been sent to the broker for consumption.
	StatusDispatched EventStatus = "DISPATCHED"

	// StatusBeingProcessed indicates an adapter has acquired a processing permit.
	StatusBeingProcessed EventStatus = "BEING_PROCESSED"

	// StatusProcessed indicates the event was successfully processed (terminal state).
	StatusProcessed EventStatus = "PROCESSED"

	// StatusFailed indicates the event has exhausted all retries (terminal state).
	StatusFailed EventStatus = "FAILED"

	// StatusCanceled indicates the event was canceled (terminal state).
	StatusCanceled EventStatus = "CANCELED"

	// StatusSuspended indicates the event type is suspended and dispatch is paused.
	StatusSuspended EventStatus = "SUSPENDED"
)

// IsFinal returns true if the status is a terminal state from which no further
// transitions are possible.
func (s EventStatus) IsFinal() bool {
	switch s {
	case StatusProcessed, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// IsEligibleForDispatching returns true if the event can be picked up by the
// awaiting-processing scheduler for dispatch.
func (s EventStatus) IsEligibleForDispatching() bool {
	return s == StatusAwaitingProcessing
}

// IsRejectableForProcessing returns true if the event is in a state where a
// processing permit acquisition can be rejected (e.g., already dispatched or
// being processed by another consumer).
func (s EventStatus) IsRejectableForProcessing() bool {
	switch s {
	case StatusAwaitingProcessing, StatusProcessed, StatusFailed, StatusCanceled, StatusSuspended:
		return true
	default:
		return false
	}
}
