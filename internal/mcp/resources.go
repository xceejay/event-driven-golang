package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"event-engine-starter/internal/model"
	"event-engine-starter/internal/service"
)

func registerResources(server *mcp.Server, lifecycle *service.EventLifecycleManagementService, processing *service.EventProcessingService, config *service.EventConfigurationService) {
	// event://{eventId} — single event lookup
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "event://{eventId}",
		Name:        "Event Details",
		Description: "Load a single event by ID including its current state",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		// Extract eventId from URI: "event://<uuid>"
		uri := req.Params.URI
		idStr := strings.TrimPrefix(uri, "event://")
		eventID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid event ID in URI: %w", err)
		}

		event, err := processing.Load(ctx, eventID)
		if err != nil {
			return nil, fmt.Errorf("load event: %w", err)
		}

		data, _ := json.MarshalIndent(event, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: uri, MIMEType: "application/json", Text: string(data)},
			},
		}, nil
	})

	// events://pending/{flowType} — list pending events
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "events://pending/{flowType}",
		Name:        "Pending Events",
		Description: "List pending events for a flow type",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		flowStr := strings.TrimPrefix(uri, "events://pending/")
		flowType := model.FlowType(flowStr)

		events, err := lifecycle.FindPendingEvents(ctx, flowType, 100)
		if err != nil {
			return nil, fmt.Errorf("find pending events: %w", err)
		}

		data, _ := json.MarshalIndent(events, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: uri, MIMEType: "application/json", Text: string(data)},
			},
		}, nil
	})

	// config://{eventType} — lifecycle config for an event type
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "config://{eventType}",
		Name:        "Lifecycle Config",
		Description: "Get the retry and lifecycle configuration for an event type",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		eventType := strings.TrimPrefix(uri, "config://")

		cfg, err := config.LoadConfig(ctx, eventType)
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}

		data, _ := json.MarshalIndent(cfg, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: uri, MIMEType: "application/json", Text: string(data)},
			},
		}, nil
	})

	// metrics://queues — queue sizes
	server.AddResource(&mcp.Resource{
		URI:         "metrics://queues",
		Name:        "Queue Metrics",
		Description: "Queue sizes by flow type and status",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		flowTypes := []model.FlowType{model.FlowTypeA, model.FlowTypeB}
		statuses := []model.EventStatus{
			model.StatusAwaitingProcessing,
			model.StatusDispatched,
			model.StatusBeingProcessed,
			model.StatusSuspended,
		}

		type queueEntry struct {
			FlowType string `json:"flow_type"`
			Status   string `json:"status"`
			Size     int64  `json:"size"`
		}

		var entries []queueEntry
		for _, ft := range flowTypes {
			for _, st := range statuses {
				size, err := lifecycle.GetQueueSize(ctx, ft, st)
				if err != nil {
					continue
				}
				entries = append(entries, queueEntry{
					FlowType: string(ft),
					Status:   string(st),
					Size:     size,
				})
			}
		}

		data, _ := json.MarshalIndent(entries, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{URI: req.Params.URI, MIMEType: "application/json", Text: string(data)},
			},
		}, nil
	})
}
