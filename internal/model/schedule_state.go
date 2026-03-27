package model

// ScheduleState represents the scheduling state of an event, controlling
// whether it is eligible for pickup by scheduled jobs.
type ScheduleState string

const (
	// ScheduleStateActive indicates the event is eligible for scheduled processing.
	ScheduleStateActive ScheduleState = "ACTIVE"

	// ScheduleStateInactive indicates the event should not be picked up by schedulers.
	ScheduleStateInactive ScheduleState = "INACTIVE"

	// ScheduleStateSleeping indicates the event is waiting for a scheduled attempt time.
	ScheduleStateSleeping ScheduleState = "SLEEPING"
)
