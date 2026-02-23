package main

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"sync"
	"time"
)

// Rate limiter for login attempts: max 5 per minute per IP
var loginLimiter = struct {
	sync.Mutex
	attempts map[string][]time.Time
}{attempts: make(map[string][]time.Time)}

func checkLoginRateLimit(ip string) bool {
	loginLimiter.Lock()
	defer loginLimiter.Unlock()

	now := time.Now()
	cutoff := now.Add(-1 * time.Minute)

	// Filter to recent attempts
	recent := loginLimiter.attempts[ip][:0]
	for _, t := range loginLimiter.attempts[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	loginLimiter.attempts[ip] = recent

	if len(recent) >= 5 {
		return false
	}

	loginLimiter.attempts[ip] = append(loginLimiter.attempts[ip], now)
	return true
}

func resetLoginRateLimit() {
	loginLimiter.Lock()
	loginLimiter.attempts = make(map[string][]time.Time)
	loginLimiter.Unlock()
}

func getClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleLogin(w, r)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleLogout(w, r)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleMe(w, r)
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleChangePassword(w, r)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateCSRFToken creates a new CSRF token for the given user
func generateCSRFToken(userID int) (string, error) {
	return getAdminHandler().GenerateCSRFToken(userID)
}

// handleGetCSRFToken returns a new CSRF token for the authenticated user
func handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	getAdminHandler().HandleGetCSRFToken(w, r)
}
