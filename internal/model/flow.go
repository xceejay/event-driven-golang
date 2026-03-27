package model

// FlowType identifies a logical processing flow. Each flow has its own set of
// scheduled jobs and can be extended by adding new constants.
type FlowType string

const (
	// FlowTypeA is the first built-in processing flow.
	FlowTypeA FlowType = "FLOW_A"

	// FlowTypeB is the second built-in processing flow.
	FlowTypeB FlowType = "FLOW_B"
)
