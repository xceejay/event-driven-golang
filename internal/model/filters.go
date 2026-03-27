package model

import "time"

// ScheduledBeforeFilter selects events of a given flow and schedule state,
// with attempt_scheduled_at before the current time, limited to Limit results.
type ScheduledBeforeFilter struct {
	FlowType      FlowType
	ScheduleState ScheduleState
	Limit         int
}

// ScheduledBeforeInclusiveFilter selects events of a given flow and schedule
// state with attempt_scheduled_at on or before ScheduledBefore.
type ScheduledBeforeInclusiveFilter struct {
	FlowType       FlowType
	ScheduleState  ScheduleState
	ScheduledBefore time.Time
	Limit          int
}

// CreatedBeforeFilter selects events of a given flow and status that were
// created before a specified time, limited to Limit results.
type CreatedBeforeFilter struct {
	FlowType      FlowType
	Status        EventStatus
	CreatedBefore time.Time
	Limit         int
}
