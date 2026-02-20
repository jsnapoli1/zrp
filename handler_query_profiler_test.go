package main

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupQueryProfilerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create a simple test table
	_, err = testDB.Exec(`
		CREATE TABLE test_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			value INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create test_data table: %v", err)
	}

	// Create audit_log table - CRITICAL: Used by almost every handler
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			username TEXT,
			action TEXT,
			table_name TEXT,
			record_id TEXT,
			details TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Insert test data
	for i := 1; i <= 10; i++ {
		_, err = testDB.Exec("INSERT INTO test_data (name, value) VALUES (?, ?)", 
			"test"+string(rune(i)), i)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	return testDB
}

func TestQueryProfilerStats_Disabled(t *testing.T) {
	// Save old profiler and restore after test
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	// Disable profiler
	profiler = nil

	req := httptest.NewRequest("GET", "/api/query-profiler/stats", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerStats(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500 when profiler disabled, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["error"] != "Query profiler not initialized" {
		t.Errorf("Expected 'Query profiler not initialized' error, got: %v", resp["error"])
	}

	t.Logf("✓ Query profiler stats returns error when disabled")
}

func TestQueryProfilerStats_Enabled(t *testing.T) {
	// Initialize profiler
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100) // 100ms threshold
	defer profiler.Close()

	// Record some test queries
	profiler.recordQuery("SELECT * FROM test_data WHERE id = ?", 50*time.Millisecond, 1)
	profiler.recordQuery("SELECT * FROM test_data WHERE value > ?", 150*time.Millisecond, 5)
	profiler.recordQuery("INSERT INTO test_data (name, value) VALUES (?, ?)", 30*time.Millisecond, "test", 100)

	req := httptest.NewRequest("GET", "/api/query-profiler/stats", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerStats(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Extract stats from data wrapper
	stats, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response, got: %v", resp)
	}

	// Verify stats structure
	if stats["enabled"] != true {
		t.Errorf("Expected enabled=true, got %v", stats["enabled"])
	}

	totalQueries, ok := stats["total_queries"].(float64)
	if !ok || totalQueries != 3 {
		t.Errorf("Expected total_queries=3, got %v", stats["total_queries"])
	}

	slowQueries, ok := stats["slow_queries"].(float64)
	if !ok || slowQueries != 1 {
		t.Errorf("Expected slow_queries=1, got %v", stats["slow_queries"])
	}

	if _, ok := stats["avg_duration"].(string); !ok {
		t.Error("Expected avg_duration to be string")
	}

	if _, ok := stats["min_duration"].(string); !ok {
		t.Error("Expected min_duration to be string")
	}

	if _, ok := stats["max_duration"].(string); !ok {
		t.Error("Expected max_duration to be string")
	}

	if stats["threshold"] != "100ms" {
		t.Errorf("Expected threshold='100ms', got %v", stats["threshold"])
	}

	t.Logf("✓ Query profiler stats returns correct statistics: %v", stats)
}

func TestQueryProfilerSlowQueries(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100) // 100ms threshold
	defer profiler.Close()

	// Record queries - some slow, some fast
	profiler.recordQuery("SELECT * FROM fast_table", 50*time.Millisecond)
	profiler.recordQuery("SELECT * FROM slow_table WHERE expensive_join", 250*time.Millisecond)
	profiler.recordQuery("UPDATE slow_table SET value = ?", 180*time.Millisecond, 100)
	profiler.recordQuery("SELECT COUNT(*) FROM fast_table", 25*time.Millisecond)

	req := httptest.NewRequest("GET", "/api/query-profiler/slow", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerSlowQueries(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Extract data wrapper
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response, got: %v", resp)
	}

	slowQueries, ok := data["slow_queries"].([]interface{})
	if !ok {
		t.Fatalf("Expected slow_queries to be array, got %T", data["slow_queries"])
	}

	if len(slowQueries) != 2 {
		t.Errorf("Expected 2 slow queries, got %d", len(slowQueries))
	}

	count, ok := data["count"].(float64)
	if !ok || count != 2 {
		t.Errorf("Expected count=2, got %v", data["count"])
	}

	if data["threshold"] != "100ms" {
		t.Errorf("Expected threshold='100ms', got %v", data["threshold"])
	}

	// Verify slow queries contain expected queries
	foundSlowJoin := false
	foundSlowUpdate := false

	for _, q := range slowQueries {
		query := q.(map[string]interface{})
		queryStr := query["query"].(string)
		if queryStr == "SELECT * FROM slow_table WHERE expensive_join" {
			foundSlowJoin = true
		}
		if queryStr == "UPDATE slow_table SET value = ?" {
			foundSlowUpdate = true
		}
	}

	if !foundSlowJoin {
		t.Error("Expected to find slow SELECT query in results")
	}
	if !foundSlowUpdate {
		t.Error("Expected to find slow UPDATE query in results")
	}

	t.Logf("✓ Query profiler correctly identifies %d slow queries", len(slowQueries))
}

func TestQueryProfilerSlowQueries_Disabled(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	profiler = nil

	req := httptest.NewRequest("GET", "/api/query-profiler/slow", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerSlowQueries(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500 when profiler disabled, got %d", w.Code)
	}
}

func TestQueryProfilerAllQueries(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	// Record various queries
	queries := []struct {
		sql      string
		duration time.Duration
		args     []interface{}
	}{
		{"SELECT * FROM table1", 10 * time.Millisecond, nil},
		{"INSERT INTO table2 VALUES (?, ?)", 25 * time.Millisecond, []interface{}{"a", "b"}},
		{"UPDATE table3 SET x = ?", 150 * time.Millisecond, []interface{}{42}},
		{"DELETE FROM table4 WHERE id = ?", 5 * time.Millisecond, []interface{}{123}},
	}

	for _, q := range queries {
		profiler.recordQuery(q.sql, q.duration, q.args...)
	}

	req := httptest.NewRequest("GET", "/api/query-profiler/queries", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerAllQueries(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Extract data wrapper
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response, got: %v", resp)
	}

	allQueries, ok := data["queries"].([]interface{})
	if !ok {
		t.Fatalf("Expected queries to be array, got %T", data["queries"])
	}

	if len(allQueries) != 4 {
		t.Errorf("Expected 4 queries, got %d", len(allQueries))
	}

	count, ok := data["count"].(float64)
	if !ok || count != 4 {
		t.Errorf("Expected count=4, got %v", data["count"])
	}

	// Verify all query types are present
	foundSelect := false
	foundInsert := false
	foundUpdate := false
	foundDelete := false

	for _, q := range allQueries {
		query := q.(map[string]interface{})
		queryStr := query["query"].(string)
		
		if queryStr == "SELECT * FROM table1" {
			foundSelect = true
		}
		if queryStr == "INSERT INTO table2 VALUES (?, ?)" {
			foundInsert = true
		}
		if queryStr == "UPDATE table3 SET x = ?" {
			foundUpdate = true
		}
		if queryStr == "DELETE FROM table4 WHERE id = ?" {
			foundDelete = true
		}

		// Verify structure - duration could be string or number
		if query["duration"] == nil {
			t.Error("Expected duration to be present")
		}
		if query["timestamp"] == nil {
			t.Error("Expected timestamp to be present")
		}
	}

	if !foundSelect || !foundInsert || !foundUpdate || !foundDelete {
		t.Error("Not all query types found in results")
	}

	t.Logf("✓ Query profiler returns all %d queries", len(allQueries))
}

func TestQueryProfilerAllQueries_Disabled(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	profiler = nil

	req := httptest.NewRequest("GET", "/api/query-profiler/queries", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerAllQueries(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500 when profiler disabled, got %d", w.Code)
	}
}

func TestQueryProfilerReset(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	// Record some queries
	profiler.recordQuery("SELECT 1", 10*time.Millisecond)
	profiler.recordQuery("SELECT 2", 20*time.Millisecond)
	profiler.recordQuery("SELECT 3", 30*time.Millisecond)

	// Verify queries exist
	stats := profiler.GetStats()
	if stats["total_queries"] != 3 {
		t.Fatalf("Expected 3 queries before reset, got %v", stats["total_queries"])
	}

	// Reset profiler
	req := httptest.NewRequest("POST", "/api/query-profiler/reset", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerReset(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Extract data wrapper
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response, got: %v", resp)
	}

	if data["message"] != "Profiler reset successfully" {
		t.Errorf("Expected success message, got: %v", data["message"])
	}

	// Verify queries were cleared
	stats = profiler.GetStats()
	if stats["total_queries"] != 0 {
		t.Errorf("Expected 0 queries after reset, got %v", stats["total_queries"])
	}

	t.Logf("✓ Query profiler reset successful")
}

func TestQueryProfilerReset_WrongMethod(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	req := httptest.NewRequest("GET", "/api/query-profiler/reset", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerReset(w, req)

	if w.Code != 405 {
		t.Errorf("Expected 405 for GET request, got %d", w.Code)
	}
}

func TestQueryProfilerReset_Disabled(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	profiler = nil

	req := httptest.NewRequest("POST", "/api/query-profiler/reset", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerReset(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500 when profiler disabled, got %d", w.Code)
	}
}

func TestQueryProfilerCircularBuffer(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	// Set small buffer size for testing
	profiler.maxProfileSize = 5

	// Record more queries than buffer size
	for i := 0; i < 10; i++ {
		profiler.recordQuery("SELECT * FROM test", 10*time.Millisecond)
	}

	stats := profiler.GetStats()
	totalQueries := stats["total_queries"].(int)

	if totalQueries != 5 {
		t.Errorf("Expected circular buffer to limit to 5 queries, got %d", totalQueries)
	}

	t.Logf("✓ Circular buffer correctly limits query storage to %d", totalQueries)
}

func TestQueryProfilerEmptyStats(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	// Get stats without recording any queries
	req := httptest.NewRequest("GET", "/api/query-profiler/stats", nil)
	w := httptest.NewRecorder()

	handleQueryProfilerStats(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Extract data wrapper
	stats, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data field in response, got: %v", resp)
	}

	if stats["total_queries"] != float64(0) {
		t.Errorf("Expected total_queries=0, got %v", stats["total_queries"])
	}

	if stats["enabled"] != true {
		t.Errorf("Expected enabled=true, got %v", stats["enabled"])
	}

	t.Logf("✓ Empty profiler returns valid stats")
}

func TestQueryProfilerThresholdBoundary(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100) // 100ms threshold
	defer profiler.Close()

	// Test queries at threshold boundaries
	profiler.recordQuery("Exactly at threshold", 100*time.Millisecond)
	profiler.recordQuery("Just below threshold", 99*time.Millisecond)
	profiler.recordQuery("Just above threshold", 101*time.Millisecond)

	slowQueries := profiler.GetSlowQueries()

	// Queries >= threshold should be considered slow
	if len(slowQueries) != 2 {
		t.Errorf("Expected 2 slow queries (>= threshold), got %d", len(slowQueries))
	}

	t.Logf("✓ Threshold boundary testing passed")
}

func TestQueryProfilerConcurrentAccess(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	// Simulate concurrent query recording
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				profiler.recordQuery("SELECT * FROM concurrent_test", 10*time.Millisecond, id, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	stats := profiler.GetStats()
	totalQueries := stats["total_queries"].(int)

	// Should have recorded 1000 queries, but circular buffer limits it
	if totalQueries <= 0 || totalQueries > 1000 {
		t.Errorf("Expected reasonable query count, got %d", totalQueries)
	}

	t.Logf("✓ Concurrent access handled correctly: %d queries recorded", totalQueries)
}

func TestQueryProfilerSQLSanitization(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	// Record query with extra whitespace and newlines
	queryWithWhitespace := `
		SELECT   *
		FROM     users
		WHERE    id = ?
			AND  status = ?
	`
	profiler.recordQuery(queryWithWhitespace, 50*time.Millisecond, 1, "active")

	allQueries := profiler.GetAllQueries()
	if len(allQueries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(allQueries))
	}

	sanitized := allQueries[0].Query

	// Verify whitespace is normalized
	if sanitized != "SELECT * FROM users WHERE id = ? AND status = ?" {
		t.Errorf("Expected sanitized query, got: %s", sanitized)
	}

	t.Logf("✓ Query sanitization works correctly")
}

func TestQueryProfilerArgsCapture(t *testing.T) {
	oldProfiler := profiler
	defer func() { profiler = oldProfiler }()

	InitQueryProfiler(true, 100)
	defer profiler.Close()

	// Record query with various argument types
	profiler.recordQuery(
		"INSERT INTO test VALUES (?, ?, ?)",
		50*time.Millisecond,
		42,
		"test string",
		3.14,
	)

	allQueries := profiler.GetAllQueries()
	if len(allQueries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(allQueries))
	}

	args := allQueries[0].Args
	if args == "" {
		t.Error("Expected args to be captured")
	}

	// Verify args contain the values
	if !contains(args, "42") || !contains(args, "test string") || !contains(args, "3.14") {
		t.Errorf("Args don't contain expected values: %s", args)
	}

	t.Logf("✓ Query arguments captured: %s", args)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
