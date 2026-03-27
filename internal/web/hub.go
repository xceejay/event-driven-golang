package web

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"

	"event-engine-starter/internal/model"
)

// Hub manages WebSocket connections and broadcasts event state changes to all
// connected clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
	logger  *log.Logger
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
		logger:  log.New(os.Stdout, "[ws-hub] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Register adds a WebSocket connection to the hub.
func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
	h.logger.Printf("client connected (%d total)", h.count())
}

// Unregister removes a WebSocket connection from the hub.
func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
	h.logger.Printf("client disconnected (%d total)", h.count())
}

// BroadcastEvent sends an event state change to all connected clients.
func (h *Hub) BroadcastEvent(event *model.Event) {
	msg := map[string]any{
		"type":  "event_update",
		"event": event,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Printf("marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			go h.Unregister(conn)
		}
	}
}

func (h *Hub) count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
