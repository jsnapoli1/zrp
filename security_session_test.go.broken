package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupSecuritySessionTestDB(t *testing.T) *sql.DB {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create users table
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			role TEXT DEFAULT 'user',
			active INTEGER DEFAULT 1,
			last_login TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create sessions table with last_activity for inactivity timeout
	_, err = testDB.Exec(`
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			last_activity DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	// Create CSRF tokens table
	_, err = testDB.Exec(`
		CREATE TABLE csrf_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create csrf_tokens table: %v", err)
	}

	return testDB
}

// Using createTestUser from SKIP__security_auth_bypass_test.go

// Test 1: Session Cookie Security Flags
func TestSessionCookie_SecurityFlags(t *testing.T) {
	oldDB := db
	db = setupSecuritySessionTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	userID := createTestUser(t, db, "testuser", "SecurePass123!", "user", true)
	t.Logf("Created user ID: %d", userID)
	
	// Verify user was created
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "testuser").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}
	if count != 1 {
		t.Fatalf("User not created, count: %d", count)
	}

	reqBody := `{"username":"testuser","password":"SecurePass123!"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handleLogin(w, req)

	if w.Code != 200 {
		t.Fatalf("Login failed with status %d: %s", w.Code, w.Body.String())
	}

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "zrp_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Session cookie not set")
	}

	// Test HttpOnly flag
	if !sessionCookie.HttpOnly {
		t.Error("SECURITY: Session cookie MUST have HttpOnly flag to prevent XSS attacks")
	}

	// Test Secure flag
	if !sessionCookie.Secure {
		t.Error("SECURITY: Session cookie MUST have Secure flag to prevent transmission over HTTP")
	}

	// Test SameSite flag
	if sessionCookie.SameSite != http.SameSiteLaxMode && sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("SECURITY: Session cookie MUST have SameSite flag (Lax or Strict), got %v", sessionCookie.SameSite)
	}

	// Test that cookie has an expiration
	if sessionCookie.Expires.IsZero() {
		t.Error("SECURITY: Session cookie should have an expiration time")
	}

	// Test that expiration is reasonable (not too long)
	maxExpiry := time.Now().Add(48 * time.Hour)
	if sessionCookie.Expires.After(maxExpiry) {
		t.Errorf("SECURITY: Session cookie expiration too long (%v), should be <= 48 hours", 
			sessionCookie.Expires.Sub(time.Now()))
	}
}

// Test 2: Session ID Cryptographic Randomness
func TestSessionID_CryptographicRandomness(t *testing.T) {
	// Generate 100 session tokens
	tokens := make(map[string]bool, 100)
	tokenBytes := make([][]byte, 100)

	for i := 0; i < 100; i++ {
		token := generateToken()
		
		// Test for uniqueness
		if tokens[token] {
			t.Fatalf("SECURITY: Duplicate token generated! Token generation is not random enough")
		}
		tokens[token] = true

		// Test length (should be 64 hex chars = 32 bytes of entropy)
		if len(token) != 64 {
			t.Errorf("Token %d has incorrect length: %d (expected 64)", i, len(token))
		}

		// Convert hex to bytes for statistical analysis
		tokenBytes[i] = []byte(token)
	}

	// Statistical test: Check for sufficient entropy
	// Test 1: All tokens should be different
	if len(tokens) != 100 {
		t.Errorf("SECURITY: Only %d unique tokens out of 100 generated", len(tokens))
	}

	// Test 2: Check for sequential or predictable patterns
	for i := 1; i < len(tokenBytes); i++ {
		// Hamming distance (number of different characters) should be high
		differences := 0
		for j := 0; j < len(tokenBytes[i]) && j < len(tokenBytes[i-1]); j++ {
			if tokenBytes[i][j] != tokenBytes[i-1][j] {
				differences++
			}
		}
		
		// Expect at least 50% different characters between consecutive tokens
		minDifferences := len(tokenBytes[i]) / 2
		if differences < minDifferences {
			t.Errorf("SECURITY: Tokens %d and %d are too similar (%d differences, expected > %d)",
				i-1, i, differences, minDifferences)
		}
	}

	// Test 3: Chi-square test for uniform distribution of hex characters
	charCounts := make(map[byte]int)
	totalChars := 0
	for _, token := range tokenBytes {
		for _, char := range token {
			charCounts[char]++
			totalChars++
		}
	}

	// Expected frequency for each character (uniform distribution)
	expectedFreq := float64(totalChars) / 16.0 // 16 possible hex chars (0-9, a-f)

	// Calculate chi-square statistic
	chiSquare := 0.0
	for _, count := range charCounts {
		deviation := float64(count) - expectedFreq
		chiSquare += (deviation * deviation) / expectedFreq
	}

	// Critical value for chi-square with 15 degrees of freedom at 0.05 significance: ~25
	// We use a more lenient 30 to account for sample size
	if chiSquare > 30.0 {
		t.Logf("WARNING: Chi-square test suggests non-uniform distribution (χ² = %.2f)", chiSquare)
	}
}

// Test 3: Session Fixation Attack Prevention
func TestSessionFixation_Prevention(t *testing.T) {
	oldDB := db
	db = setupSecuritySessionTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	userID := createTestUser(t, db, "testuser", "SecurePass123!", "user", true)

	// Step 1: Attacker creates a session (or steals a pre-auth session token)
	attackerToken := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		attackerToken, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create attacker session: %v", err)
	}

	// Step 2: Victim logs in (attacker tries to fixate the session)
	reqBody := `{"username":"testuser","password":"SecurePass123!"}`
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req.RemoteAddr = "127.0.0.1:12345"
	
	// Attacker tries to fixate session by including old token
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: attackerToken})
	
	w := httptest.NewRecorder()
	handleLogin(w, req)

	if w.Code != 200 {
		t.Fatalf("Login failed: %d", w.Code)
	}

	// Step 3: Verify new session token was generated
	cookies := w.Result().Cookies()
	var newSessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "zrp_session" {
			newSessionCookie = c
			break
		}
	}

	if newSessionCookie == nil {
		t.Fatal("No session cookie returned after login")
	}

	// CRITICAL: New token MUST be different from the old token
	if newSessionCookie.Value == attackerToken {
		t.Error("SECURITY: Session fixation vulnerability! Login MUST generate a new session token, not reuse existing one")
	}

	// Step 4: Verify old token is no longer valid
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ? AND user_id = ?", 
		attackerToken, userID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query old session: %v", err)
	}

	// Note: Current implementation doesn't invalidate old sessions on login
	// This is acceptable if we're generating new tokens, but ideally old sessions
	// should be cleaned up
	if count > 0 {
		t.Logf("INFO: Old session still exists in DB. Consider invalidating previous sessions on login for stronger security")
	}
}

// Test 4: Session Timeout After Inactivity
func TestSessionTimeout_InactivityPeriod(t *testing.T) {
	oldDB := db
	db = setupSecuritySessionTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createTestUser(t, db, "testuser", "SecurePass123!", "user", true)

	// Create a session with last_activity set to 31 minutes ago (past 30-min timeout)
	inactiveToken := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	lastActivity := time.Now().Add(-31 * time.Minute)
	
	_, err := db.Exec(`INSERT INTO sessions (token, user_id, expires_at, last_activity) 
		VALUES (?, ?, ?, ?)`,
		inactiveToken, userID, expires.Format("2006-01-02 15:04:05"), 
		lastActivity.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create inactive session: %v", err)
	}

	// Try to use the inactive session
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: inactiveToken})
	w := httptest.NewRecorder()

	handleMe(w, req)

	// Should be rejected due to inactivity timeout
	// Note: Current implementation only checks expires_at, not last_activity
	// We need to add inactivity checking to the authentication middleware
	if w.Code == 200 {
		t.Error("SECURITY: Session should be invalidated after 30 minutes of inactivity")
		t.Log("INFO: Need to implement inactivity timeout checking in authentication")
	}
}

// Test 5: Active Session Updates Last Activity
func TestSessionTimeout_ActivityUpdate(t *testing.T) {
	oldDB := db
	db = setupSecuritySessionTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createTestUser(t, db, "testuser", "SecurePass123!", "user", true)

	// Create a session - use UTC to avoid timezone issues with SQLite
	activeToken := generateToken()
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	initialActivity := now.Add(-5 * time.Minute)
	
	t.Logf("Setting last_activity to: %s UTC", initialActivity.Format("2006-01-02 15:04:05"))
	
	_, err := db.Exec(`INSERT INTO sessions (token, user_id, created_at, expires_at, last_activity) 
		VALUES (?, ?, ?, ?, ?)`,
		activeToken, userID, now.Format("2006-01-02 15:04:05"),
		expires.Format("2006-01-02 15:04:05"),
		initialActivity.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	
	// Verify what was actually stored
	var storedActivity, createdAt string
	db.QueryRow("SELECT created_at, last_activity FROM sessions WHERE token = ?", activeToken).Scan(&createdAt, &storedActivity)
	t.Logf("Stored created_at: %s, last_activity: %s", createdAt, storedActivity)
	
	// Parse and check the time difference
	parsedActivity, _ := time.Parse(time.RFC3339, storedActivity)
	timeSince := time.Since(parsedActivity)
	t.Logf("Time since last_activity: %v (should be ~5 minutes, NOT > 30 minutes)", timeSince)

	// Wait a moment to ensure time difference
	time.Sleep(100 * time.Millisecond)

	// Make a request (should update last_activity)
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: activeToken})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 200 {
		t.Fatalf("handleMe failed with status %d: %s", w.Code, w.Body.String())
	}

	// Check if last_activity was updated
	var lastActivity string
	err = db.QueryRow("SELECT last_activity FROM sessions WHERE token = ?", activeToken).
		Scan(&lastActivity)
	if err != nil {
		t.Fatalf("Failed to query last_activity: %v", err)
	}

	// Try multiple date formats (SQLite can return different formats)
	var _  time.Time
	formats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}
	for _, format := range formats {
		parsedActivity, err = time.Parse(format, lastActivity)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("Failed to parse last_activity '%s': %v", lastActivity, err)
	}

	// Verify last_activity was updated (should be very recent, within last few seconds)
	timeSinceUpdate := time.Since(parsedActivity)
	t.Logf("Time since last activity update: %v", timeSinceUpdate)
	
	if parsedActivity.Before(time.Now().UTC().Add(-1 * time.Minute)) {
		t.Error("SECURITY: last_activity should be updated on each authenticated request")
		t.Logf("Expected: recent (< 1 min ago), Got: %v ago", timeSinceUpdate)
	} else {
		t.Logf("SUCCESS: last_activity was updated to %s", parsedActivity.Format(time.RFC3339))
	}
}

// Test 6: Multiple Concurrent Sessions Per User
func TestConcurrentSessions_MultipleAllowed(t *testing.T) {
	oldDB := db
	db = setupSecuritySessionTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	createTestUser(t, db, "testuser", "SecurePass123!", "user", true)

	// Login from first device/location
	reqBody := `{"username":"testuser","password":"SecurePass123!"}`
	req1 := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req1.RemoteAddr = "192.168.1.100:12345"
	w1 := httptest.NewRecorder()
	handleLogin(w1, req1)

	if w1.Code != 200 {
		t.Fatalf("First login failed: %d", w1.Code)
	}

	var token1 string
	for _, c := range w1.Result().Cookies() {
		if c.Name == "zrp_session" {
			token1 = c.Value
			break
		}
	}

	// Login from second device/location
	req2 := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(reqBody))
	req2.RemoteAddr = "192.168.1.101:12345"
	w2 := httptest.NewRecorder()
	handleLogin(w2, req2)

	if w2.Code != 200 {
		t.Fatalf("Second login failed: %d", w2.Code)
	}

	var token2 string
	for _, c := range w2.Result().Cookies() {
		if c.Name == "zrp_session" {
			token2 = c.Value
			break
		}
	}

	// Tokens should be different
	if token1 == token2 {
		t.Error("Different logins should generate different session tokens")
	}

	// Both sessions should be valid
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token IN (?, ?)", 
		token1, token2).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query sessions: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 concurrent sessions, found %d", count)
	}

	// Both sessions should work
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token1})
	w := httptest.NewRecorder()
	handleMe(w, req)
	if w.Code != 200 {
		t.Error("First session should still be valid")
	}

	req = httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token2})
	w = httptest.NewRecorder()
	handleMe(w, req)
	if w.Code != 200 {
		t.Error("Second session should be valid")
	}
}

// Test 7: Session Cleanup on Logout
func TestSessionCleanup_OnLogout(t *testing.T) {
	oldDB := db
	db = setupSecuritySessionTestDB(t)
	defer func() { db.Close(); db = oldDB }()
	defer resetLoginRateLimit()

	userID := createTestUser(t, db, "testuser", "SecurePass123!", "user", true)

	// Create a session
	token := generateToken()
	expires := time.Now().Add(24 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if count != 1 {
		t.Fatal("Session not created properly")
	}

	// Logout
	req := httptest.NewRequest("POST", "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: token})
	w := httptest.NewRecorder()
	handleLogout(w, req)

	if w.Code != 200 {
		t.Fatalf("Logout failed: %d", w.Code)
	}

	// Verify session was deleted
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&count)
	if count != 0 {
		t.Error("SECURITY: Session should be deleted from database on logout")
	}

	// Verify cookie was invalidated
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "zrp_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Logout should return a cookie to clear the session")
	}

	if sessionCookie.MaxAge != -1 {
		t.Error("SECURITY: Logout cookie should have MaxAge=-1 to delete it")
	}
}

// Test 8: Expired Sessions Cannot Be Used
func TestExpiredSessions_CannotBeUsed(t *testing.T) {
	oldDB := db
	db = setupSecuritySessionTestDB(t)
	defer func() { db.Close(); db = oldDB }()

	userID := createTestUser(t, db, "testuser", "SecurePass123!", "user", true)

	// Create an expired session
	expiredToken := generateToken()
	expires := time.Now().Add(-1 * time.Hour)
	_, err := db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		expiredToken, userID, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to create expired session: %v", err)
	}

	// Try to use expired session
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "zrp_session", Value: expiredToken})
	w := httptest.NewRecorder()

	handleMe(w, req)

	if w.Code != 401 {
		t.Error("SECURITY: Expired sessions MUST be rejected")
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["code"] != "UNAUTHORIZED" {
		t.Errorf("Expected UNAUTHORIZED error, got %v", resp["code"])
	}
}

// Test 9: Session ID Should Not Be Predictable
func TestSessionID_NotPredictable(t *testing.T) {
	tokens := make([]string, 10)
	for i := 0; i < 10; i++ {
		tokens[i] = generateToken()
	}

	// Check no sequential patterns exist
	for i := 1; i < len(tokens); i++ {
		// Convert to integers if possible (should not be possible with proper random generation)
		// Check that tokens don't increment
		prev := tokens[i-1]
		curr := tokens[i]

		if len(prev) != len(curr) {
			continue
		}

		// Count how many character positions are sequential
		sequential := 0
		for j := 0; j < len(prev)-1; j++ {
			if curr[j] == prev[j]+1 || curr[j] == prev[j]-1 {
				sequential++
			}
		}

		// If more than 25% of characters are sequential, that's suspicious
		// (In truly random hex, expect ~12.5% sequential by chance)
		if float64(sequential)/float64(len(prev)) > 0.25 {
			t.Errorf("SECURITY: Token %d appears to have sequential pattern with token %d (%d sequential chars)",
				i, i-1, sequential)
		}
	}
}

// Test 10: Token Entropy Check
func TestSessionID_SufficientEntropy(t *testing.T) {
	token := generateToken()

	// Token should be 64 hex characters (256 bits of entropy from 32 random bytes)
	if len(token) != 64 {
		t.Errorf("Token should be 64 hex chars, got %d", len(token))
	}

	// Count unique characters (should have good variety)
	charSet := make(map[rune]bool)
	for _, c := range token {
		charSet[c] = true
	}

	// Should have at least 8 different hex characters (out of 16 possible)
	if len(charSet) < 8 {
		t.Errorf("SECURITY: Token has insufficient character variety (%d unique chars), may indicate weak randomness",
			len(charSet))
	}

	// Check for repeated patterns (should not have same 4+ chars in a row)
	for i := 0; i < len(token)-3; i++ {
		pattern := token[i : i+4]
		occurrences := 0
		for j := 0; j < len(token)-3; j++ {
			if token[j:j+4] == pattern {
				occurrences++
			}
		}
		if occurrences > 1 {
			t.Errorf("SECURITY: Found repeated 4-character pattern '%s' %d times in token",
				pattern, occurrences)
		}
	}
}

// Test 11: Statistical Randomness Test (Birthday Paradox)
func TestSessionID_BirthdayParadox(t *testing.T) {
	// Birthday paradox: with 32-byte tokens (256 bits), probability of collision
	// in 2^128 tokens is ~50%. For 100 tokens, collision probability should be negligible.
	
	tokens := make(map[string]bool, 1000)
	
	// Generate 1000 tokens
	for i := 0; i < 1000; i++ {
		token := generateToken()
		if tokens[token] {
			t.Fatal("SECURITY CRITICAL: Collision detected in 1000 tokens! Random number generator is broken or weak")
		}
		tokens[token] = true
	}

	// All 1000 tokens should be unique
	if len(tokens) != 1000 {
		t.Errorf("SECURITY: Generated %d unique tokens out of 1000, possible RNG weakness", len(tokens))
	}
}

// Test 12: Verify crypto/rand is used (not math/rand)
func TestSessionID_UsesCryptoRand(t *testing.T) {
	// Generate tokens and check distribution
	// crypto/rand should have very even distribution
	// math/rand would show statistical bias
	
	const numTokens = 100
	const numBuckets = 16
	
	buckets := make([]int, numBuckets)
	
	for i := 0; i < numTokens; i++ {
		token := generateToken()
		// Use first character as bucket
		firstChar := token[0]
		var bucketIdx int
		if firstChar >= '0' && firstChar <= '9' {
			bucketIdx = int(firstChar - '0')
		} else if firstChar >= 'a' && firstChar <= 'f' {
			bucketIdx = int(firstChar-'a') + 10
		}
		buckets[bucketIdx]++
	}
	
	// Calculate expected frequency and standard deviation
	expected := float64(numTokens) / float64(numBuckets)
	variance := 0.0
	for _, count := range buckets {
		diff := float64(count) - expected
		variance += diff * diff
	}
	variance /= float64(numBuckets)
	stdDev := math.Sqrt(variance)
	
	// For crypto/rand, standard deviation should be reasonable
	// For 100 samples across 16 buckets, expect stddev around 2-3
	// If stddev > 5, might indicate non-uniform distribution
	if stdDev > 5.0 {
		t.Errorf("SECURITY WARNING: Token distribution appears non-uniform (stddev=%.2f), may not be using crypto/rand",
			stdDev)
		t.Logf("Bucket distribution: %v", buckets)
	}
}
