package web

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"event-engine-starter/internal/model"
	"event-engine-starter/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewRouter creates the HTTP router with all routes registered.
func NewRouter(
	mcpServer *mcp.Server,
	hub *Hub,
	lifecycle *service.EventLifecycleManagementService,
	processing *service.EventProcessingService,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// WebSocket for live event updates
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.Register(conn)

		// Read loop — keep connection alive, handle client close.
		go func() {
			defer hub.Unregister(conn)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		}()
	})

	// REST API for the HTML frontend
	r.Route("/api", func(r chi.Router) {
		r.Post("/events", publishHandler(lifecycle))
		r.Post("/events/{eventID}/acquire", acquireHandler(lifecycle))
		r.Post("/events/{eventID}/success", successHandler(lifecycle))
		r.Post("/events/{eventID}/failure", failureHandler(lifecycle))
		r.Get("/events/{eventID}", getEventHandler(processing))
		r.Get("/events/pending/{flowType}", pendingHandler(lifecycle))
		r.Get("/counts", countsHandler(lifecycle))
		r.Post("/event-types/{eventType}/suspend", suspendHandler(lifecycle))
		r.Post("/event-types/{eventType}/unsuspend", unsuspendHandler(lifecycle))
	})

	// MCP SSE endpoint
	sseHandler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)
	r.Handle("/mcp/*", sseHandler)
	r.Handle("/mcp", sseHandler)

	// Static files — serve HTML frontend
	r.Handle("/*", http.FileServer(http.Dir("web")))

	return r
}

func publishHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EventType       string `json:"event_type"`
			FlowType        string `json:"flow_type"`
			FlowID          string `json:"flow_id"`
			Payload         string `json:"payload"`
			OnFailEventType string `json:"on_fail_event_type,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		var onFail *string
		if req.OnFailEventType != "" {
			onFail = &req.OnFailEventType
		}

		eventID, err := lifecycle.Publish(r.Context(), service.PublishRequest{
			EventType:       req.EventType,
			FlowType:        model.FlowType(req.FlowType),
			FlowID:          req.FlowID,
			Payload:         req.Payload,
			OnFailEventType: onFail,
		})
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]string{"event_id": eventID.String()})
	}
}

func acquireHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, err := parseEventID(r)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}

		pctx, err := lifecycle.TryAcquireProcessingPermit(r.Context(), eventID)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if pctx == nil {
			jsonResponse(w, map[string]any{"acquired": false})
			return
		}

		jsonResponse(w, map[string]any{
			"acquired": true,
			"event_id": pctx.Event.ID.String(),
			"payload":  pctx.Payload,
			"flow_id":  pctx.Event.FlowID,
		})
	}
}

func successHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, err := parseEventID(r)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Parse optional spinoff events from request body.
		var req struct {
			Spinoffs []struct {
				EventType string `json:"event_type"`
				Payload   string `json:"payload"`
			} `json:"spinoffs"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var spinoffs []service.SpinoffEvent
		for _, s := range req.Spinoffs {
			if s.EventType != "" {
				spinoffs = append(spinoffs, service.SpinoffEvent{
					EventType: s.EventType,
					Payload:   s.Payload,
				})
			}
		}

		if err := lifecycle.ReportSuccess(r.Context(), eventID, spinoffs); err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]bool{"acknowledged": true})
	}
}

func failureHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, err := parseEventID(r)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req struct {
			ErrorMessage string `json:"error_message"`
			StackTrace   string `json:"stack_trace,omitempty"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var st *string
		if req.StackTrace != "" {
			st = &req.StackTrace
		}

		if err := lifecycle.ReportFailure(r.Context(), eventID, req.ErrorMessage, st); err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse(w, map[string]bool{"acknowledged": true})
	}
}

func getEventHandler(processing *service.EventProcessingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, err := parseEventID(r)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := processing.Load(r.Context(), eventID)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}

		jsonResponse(w, event)
	}
}

func pendingHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flowType := model.FlowType(chi.URLParam(r, "flowType"))
		events, err := lifecycle.FindPendingEvents(r.Context(), flowType, 100)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, events)
	}
}

func suspendHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventType := chi.URLParam(r, "eventType")
		if err := lifecycle.SuspendEventType(r.Context(), eventType); err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, map[string]bool{"acknowledged": true})
	}
}

func unsuspendHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventType := chi.URLParam(r, "eventType")
		if err := lifecycle.UnsuspendEventType(r.Context(), eventType); err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, map[string]bool{"acknowledged": true})
	}
}

func countsHandler(lifecycle *service.EventLifecycleManagementService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses := []model.EventStatus{
			model.StatusAwaitingProcessing,
			model.StatusDispatched,
			model.StatusBeingProcessed,
			model.StatusProcessed,
			model.StatusFailed,
			model.StatusSuspended,
		}
		flows := []model.FlowType{model.FlowTypeA, model.FlowTypeB}

		counts := make(map[string]int64)
		for _, s := range statuses {
			counts[string(s)] = 0
		}

		for _, ft := range flows {
			for _, s := range statuses {
				n, err := lifecycle.GetQueueSize(r.Context(), ft, s)
				if err != nil {
					continue
				}
				counts[string(s)] += n
			}
		}

		jsonResponse(w, counts)
	}
}

func parseEventID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "eventID")
	return uuid.Parse(raw)
}

func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
