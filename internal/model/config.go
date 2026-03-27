package model

// EventLifecycleConfig defines the retry and lifespan rules for a specific
// event type within a flow.
type EventLifecycleConfig struct {
	ID                    int64                  `db:"id" json:"id"`
	EventType             string                 `db:"event_type" json:"event_type"`
	FlowType              FlowType               `db:"flow_type" json:"flow_type"`
	MaxAttempts           int                    `db:"max_attempts" json:"max_attempts"`
	EventLifespanSeconds  int64                  `db:"event_lifespan_seconds" json:"event_lifespan_seconds"`
	IsSuspended           bool                   `db:"is_suspended" json:"is_suspended"`
	AttemptLifecycleConfigs []AttemptLifecycleConfig `json:"attempt_lifecycle_configs"`
}

// AttemptLifecycleConfig defines the delay before a specific retry attempt.
type AttemptLifecycleConfig struct {
	AttemptNumber            int `db:"attempt_number" json:"attempt_number"`
	DelayBeforeAttemptSeconds int `db:"delay_before_attempt_seconds" json:"delay_before_attempt_seconds"`
}
