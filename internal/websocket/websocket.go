package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

// Event is the payload broadcast to all connected WebSocket clients.
type Event struct {
	Type   string `json:"type"`
	ID     any    `json:"id"`
	Action string `json:"action"`
}

// client wraps a WebSocket connection with a mutex for thread-safe writes.
type client struct {
	conn *ws.Conn
	mu   sync.Mutex
}

// Hub maintains connected WebSocket clients and broadcasts events.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	if c.conn != nil {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ws: close panic: %v", r)
			}
		}()
		_ = c.conn.Close()
	}
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(evt Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}
	h.mu.RLock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.mu.Lock()
		writeErr := func() (writeErr error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("ws: write panic: %v", r)
					writeErr = fmt.Errorf("ws: write panic: %v", r)
				}
			}()
			_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			return c.conn.WriteMessage(ws.TextMessage, data)
		}()
		c.mu.Unlock()

		if writeErr != nil {
			h.unregister(c)
		}
	}
}

// BroadcastChange is a convenience helper for broadcasting resource changes.
func (h *Hub) BroadcastChange(resourceType, action string, id any) {
	h.Broadcast(Event{
		Type:   resourceType + "_" + action + "d",
		ID:     id,
		Action: action,
	})
}

// Upgrader is the default WebSocket upgrader.
var Upgrader = ws.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HandleWebSocket upgrades the connection and keeps it alive with pings.
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade error: %v", err)
		return
	}

	c := &client{conn: conn}
	hub.register(c)

	hub.mu.RLock()
	clientCount := len(hub.clients)
	hub.mu.RUnlock()

	log.Printf("ws: client connected (%d total)", clientCount)

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			c.mu.Lock()
			err := conn.WriteControl(ws.PingMessage, nil, time.Now().Add(5*time.Second))
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	hub.unregister(c)
	log.Printf("ws: client disconnected")
}
