package main

import (
	"net/http"

	"zrp/internal/websocket"
)

// Type aliases for backward compatibility.
type WSEvent = websocket.Event
type Hub = websocket.Hub

// Global hub instance.
var wsHub = websocket.NewHub()

// handleWebSocket upgrades the HTTP connection to a WebSocket.
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	websocket.HandleWebSocket(wsHub, w, r)
}

// broadcast is a convenience helper used by handlers.
func broadcast(resourceType, action string, id any) {
	wsHub.BroadcastChange(resourceType, action, id)
}
