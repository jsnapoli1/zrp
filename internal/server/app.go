package server

import (
	"database/sql"

	"zrp/internal/auth"
	"zrp/internal/websocket"
)

// ContextKey is the type used for request context keys.
type ContextKey string

const (
	CtxUserID   ContextKey = "userID"
	CtxUsername ContextKey = "username"
	CtxRole     ContextKey = "role"
)

// App holds shared dependencies for the application.
type App struct {
	DB        *sql.DB
	Hub       *websocket.Hub
	PermCache *auth.PermCache
}
