package mcpserver

import (
	"event-engine-starter/internal/service"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates and configures the MCP server with all event engine tools
// and resources registered.
func NewServer(lifecycle *service.EventLifecycleManagementService, config *service.EventConfigurationService, processing *service.EventProcessingService) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "event-engine",
			Version: "1.0.0",
		},
		&mcp.ServerOptions{
			Instructions: "Event lifecycle engine — publish events, acquire processing permits, report results, and manage lifecycle configuration.",
		},
	)

	registerTools(server, lifecycle, config)
	registerResources(server, lifecycle, processing, config)

	return server
}
