package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSEvent is the payload broadcast to all connected WebSocket clients.
type WSEvent struct {
	Type   string `json:"type"`   // e.g. "eco_updated", "part_created"
	ID     any    `json:"id"`     // resource identifier (int or string)
	Action string `json:"action"` // "create", "update", "delete"
}

// Hub maintains connected WebSocket clients and broadcasts events.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

var wsHub = &Hub{
	clients: make(map[*websocket.Conn]struct{}),
}

func (h *Hub) register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
}

// Broadcast sends an event to all connected clients.
func (h *Hub) Broadcast(evt WSEvent) {
	data, err := json.Marshal(evt)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}
	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			h.unregister(c)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// handleWebSocket upgrades the connection and keeps it alive with pings.
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade error: %v", err)
		return
	}

	wsHub.register(conn)
	log.Printf("ws: client connected (%d total)", len(wsHub.clients))

	// Keep-alive: read loop (handles pongs and detects disconnects)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Ping ticker
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}()

	// Read loop — just discard messages, detect close
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	wsHub.unregister(conn)
	log.Printf("ws: client disconnected")
}

// broadcast is a convenience helper used by handlers.
func broadcast(resourceType, action string, id any) {
	wsHub.Broadcast(WSEvent{
		Type:   resourceType + "_" + action + "d",
		ID:     id,
		Action: action,
	})
}
