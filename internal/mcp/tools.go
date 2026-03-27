package mcpserver

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"event-engine-starter/internal/model"
	"event-engine-starter/internal/service"
)

// --- Tool input/output types ---

type publishInput struct {
	EventType       string `json:"event_type" jsonschema:"The event type identifier"`
	FlowType        string `json:"flow_type" jsonschema:"The flow this event belongs to (e.g. FLOW_A)"`
	FlowID          string `json:"flow_id" jsonschema:"Business transaction group ID"`
	Payload         string `json:"payload" jsonschema:"JSON payload for the event"`
	OnFailEventType string `json:"on_fail_event_type,omitempty" jsonschema:"Event type to publish on terminal failure"`
}

type publishOutput struct {
	EventID string `json:"event_id"`
	Message string `json:"message"`
}

type acquirePermitInput struct {
	EventID string `json:"event_id" jsonschema:"UUID of the event to acquire a processing permit for"`
}

type acquirePermitOutput struct {
	Acquired bool   `json:"acquired"`
	EventID  string `json:"event_id,omitempty"`
	Payload  string `json:"payload,omitempty"`
	FlowID   string `json:"flow_id,omitempty"`
	Message  string `json:"message"`
}

type reportSuccessInput struct {
	EventID  string           `json:"event_id" jsonschema:"UUID of the event to mark as successful"`
	Spinoffs []spinoffInput   `json:"spinoff_events,omitempty" jsonschema:"Follow-up events to create"`
}

type spinoffInput struct {
	EventType       string `json:"event_type"`
	Payload         string `json:"payload"`
	OnFailEventType string `json:"on_fail_event_type,omitempty"`
}

type reportFailureInput struct {
	EventID      string `json:"event_id" jsonschema:"UUID of the failed event"`
	ErrorMessage string `json:"error_message" jsonschema:"Description of the failure"`
	StackTrace   string `json:"stack_trace,omitempty" jsonschema:"Optional stack trace"`
}

type ackOutput struct {
	Acknowledged bool   `json:"acknowledged"`
	Message      string `json:"message"`
}

type eventTypeInput struct {
	EventType string `json:"event_type" jsonschema:"The event type to operate on"`
}

type updateConfigInput struct {
	EventType            string                        `json:"event_type" jsonschema:"Event type to configure"`
	MaxAttempts          int                           `json:"max_attempts" jsonschema:"Maximum retry attempts"`
	EventLifespanSeconds int64                         `json:"event_lifespan_seconds" jsonschema:"Total time budget in seconds"`
	AttemptConfigs       []attemptConfigInput          `json:"attempt_configs" jsonschema:"Per-attempt delay configuration"`
}

type attemptConfigInput struct {
	AttemptNumber            int `json:"attempt_number"`
	DelayBeforeAttemptSeconds int `json:"delay_before_attempt_seconds"`
}

func registerTools(server *mcp.Server, lifecycle *service.EventLifecycleManagementService, config *service.EventConfigurationService) {
	// publish_event
	mcp.AddTool(server, &mcp.Tool{
		Name:        "publish_event",
		Description: "Publish a new event into the lifecycle engine for processing",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input publishInput) (*mcp.CallToolResult, publishOutput, error) {
		var onFail *string
		if input.OnFailEventType != "" {
			onFail = &input.OnFailEventType
		}

		eventID, err := lifecycle.Publish(ctx, service.PublishRequest{
			EventType:       input.EventType,
			FlowType:        model.FlowType(input.FlowType),
			FlowID:          input.FlowID,
			Payload:         input.Payload,
			OnFailEventType: onFail,
		})
		if err != nil {
			return nil, publishOutput{}, fmt.Errorf("publish failed: %w", err)
		}

		return nil, publishOutput{
			EventID: eventID.String(),
			Message: "Event published successfully",
		}, nil
	})

	// acquire_processing_permit
	mcp.AddTool(server, &mcp.Tool{
		Name:        "acquire_processing_permit",
		Description: "Attempt to acquire a processing permit (lock) for an event",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input acquirePermitInput) (*mcp.CallToolResult, acquirePermitOutput, error) {
		eventID, err := uuid.Parse(input.EventID)
		if err != nil {
			return nil, acquirePermitOutput{}, fmt.Errorf("invalid event ID: %w", err)
		}

		pctx, err := lifecycle.TryAcquireProcessingPermit(ctx, eventID)
		if err != nil {
			return nil, acquirePermitOutput{}, fmt.Errorf("acquire permit failed: %w", err)
		}

		if pctx == nil {
			return nil, acquirePermitOutput{
				Acquired: false,
				Message:  "Processing permit not granted (event in wrong state or lost race)",
			}, nil
		}

		return nil, acquirePermitOutput{
			Acquired: true,
			EventID:  pctx.Event.ID.String(),
			Payload:  pctx.Payload,
			FlowID:   pctx.Event.FlowID,
			Message:  "Processing permit acquired",
		}, nil
	})

	// report_success
	mcp.AddTool(server, &mcp.Tool{
		Name:        "report_success",
		Description: "Report that an event was successfully processed, with optional spinoff events",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input reportSuccessInput) (*mcp.CallToolResult, ackOutput, error) {
		eventID, err := uuid.Parse(input.EventID)
		if err != nil {
			return nil, ackOutput{}, fmt.Errorf("invalid event ID: %w", err)
		}

		var spinoffs []service.SpinoffEvent
		for _, s := range input.Spinoffs {
			var onFail *string
			if s.OnFailEventType != "" {
				onFail = &s.OnFailEventType
			}
			spinoffs = append(spinoffs, service.SpinoffEvent{
				EventType:       s.EventType,
				Payload:         s.Payload,
				OnFailEventType: onFail,
			})
		}

		if err := lifecycle.ReportSuccess(ctx, eventID, spinoffs); err != nil {
			return nil, ackOutput{}, fmt.Errorf("report success failed: %w", err)
		}

		return nil, ackOutput{Acknowledged: true, Message: "Event marked as processed"}, nil
	})

	// report_failure
	mcp.AddTool(server, &mcp.Tool{
		Name:        "report_failure",
		Description: "Report that event processing failed; the engine will retry or mark as terminal failure",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input reportFailureInput) (*mcp.CallToolResult, ackOutput, error) {
		eventID, err := uuid.Parse(input.EventID)
		if err != nil {
			return nil, ackOutput{}, fmt.Errorf("invalid event ID: %w", err)
		}

		var stackTrace *string
		if input.StackTrace != "" {
			stackTrace = &input.StackTrace
		}

		if err := lifecycle.ReportFailure(ctx, eventID, input.ErrorMessage, stackTrace); err != nil {
			return nil, ackOutput{}, fmt.Errorf("report failure failed: %w", err)
		}

		return nil, ackOutput{Acknowledged: true, Message: "Failure reported"}, nil
	})

	// suspend_event_type
	mcp.AddTool(server, &mcp.Tool{
		Name:        "suspend_event_type",
		Description: "Pause dispatch of all events of the given type",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input eventTypeInput) (*mcp.CallToolResult, ackOutput, error) {
		if err := lifecycle.SuspendEventType(ctx, input.EventType); err != nil {
			return nil, ackOutput{}, fmt.Errorf("suspend failed: %w", err)
		}
		return nil, ackOutput{Acknowledged: true, Message: fmt.Sprintf("Event type %s suspended", input.EventType)}, nil
	})

	// unsuspend_event_type
	mcp.AddTool(server, &mcp.Tool{
		Name:        "unsuspend_event_type",
		Description: "Resume dispatch of events of the given type",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input eventTypeInput) (*mcp.CallToolResult, ackOutput, error) {
		if err := lifecycle.UnsuspendEventType(ctx, input.EventType); err != nil {
			return nil, ackOutput{}, fmt.Errorf("unsuspend failed: %w", err)
		}
		return nil, ackOutput{Acknowledged: true, Message: fmt.Sprintf("Event type %s unsuspended", input.EventType)}, nil
	})

	// update_lifecycle_config
	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_lifecycle_config",
		Description: "Update the retry and lifecycle configuration for an event type",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input updateConfigInput) (*mcp.CallToolResult, ackOutput, error) {
		var attempts []model.AttemptLifecycleConfig
		for _, a := range input.AttemptConfigs {
			attempts = append(attempts, model.AttemptLifecycleConfig{
				AttemptNumber:            a.AttemptNumber,
				DelayBeforeAttemptSeconds: a.DelayBeforeAttemptSeconds,
			})
		}

		cfg := model.EventLifecycleConfig{
			EventType:               input.EventType,
			MaxAttempts:             input.MaxAttempts,
			EventLifespanSeconds:    input.EventLifespanSeconds,
			AttemptLifecycleConfigs: attempts,
		}

		if err := config.UpdateConfig(ctx, cfg); err != nil {
			return nil, ackOutput{}, fmt.Errorf("update config failed: %w", err)
		}

		return nil, ackOutput{Acknowledged: true, Message: fmt.Sprintf("Config updated for %s", input.EventType)}, nil
	})
}
