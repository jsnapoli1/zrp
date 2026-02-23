package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// QueryProfiler tracks query execution times for performance monitoring
type QueryProfiler struct {
	enabled        bool
	slowThreshold  time.Duration
	queries        []QueryProfile
	mu             sync.RWMutex
	logFile        *os.File
	maxProfileSize int
}

// QueryProfile represents a single query execution profile
type QueryProfile struct {
	Query     string        `json:"query"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Args      string        `json:"args,omitempty"`
}

var profiler *QueryProfiler

// InitQueryProfiler initializes the global query profiler
func InitQueryProfiler(enabled bool, slowThresholdMs int) {
	profiler = &QueryProfiler{
		enabled:        enabled,
		slowThreshold:  time.Duration(slowThresholdMs) * time.Millisecond,
		queries:        make([]QueryProfile, 0, 1000),
		maxProfileSize: 1000,
	}
	
	if enabled {
		// Open log file for slow queries
		f, err := os.OpenFile("slow_queries.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("[QueryProfiler] Failed to open slow query log: %v", err)
		} else {
			profiler.logFile = f
		}
		log.Printf("[QueryProfiler] Enabled with threshold: %dms", slowThresholdMs)
	}
}

// ProfileQuery wraps db.Query with profiling
func ProfileQuery(query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.Query(query, args...)
	duration := time.Since(start)
	
	if profiler != nil && profiler.enabled {
		profiler.recordQuery(query, duration, args...)
	}
	
	return rows, err
}

// ProfileQueryRow wraps db.QueryRow with profiling
func ProfileQueryRow(query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := db.QueryRow(query, args...)
	duration := time.Since(start)
	
	if profiler != nil && profiler.enabled {
		profiler.recordQuery(query, duration, args...)
	}
	
	return row
}

// ProfileExec wraps db.Exec with profiling
func ProfileExec(query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := db.Exec(query, args...)
	duration := time.Since(start)
	
	if profiler != nil && profiler.enabled {
		profiler.recordQuery(query, duration, args...)
	}
	
	return result, err
}

// recordQuery records a query execution
func (p *QueryProfiler) recordQuery(query string, duration time.Duration, args ...interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Sanitize query for logging
	cleanQuery := strings.Join(strings.Fields(query), " ")
	
	profile := QueryProfile{
		Query:     cleanQuery,
		Duration:  duration,
		Timestamp: time.Now(),
		Args:      fmt.Sprintf("%v", args),
	}
	
	// Add to in-memory profiles (circular buffer)
	if len(p.queries) >= p.maxProfileSize {
		p.queries = p.queries[1:]
	}
	p.queries = append(p.queries, profile)
	
	// Log slow queries
	if duration >= p.slowThreshold {
		logMsg := fmt.Sprintf("[%s] SLOW QUERY (%s): %s | Args: %v\n",
			profile.Timestamp.Format("2006-01-02 15:04:05"),
			duration,
			cleanQuery,
			args,
		)
		
		// Console log
		log.Printf("[SLOW QUERY] %s took %v", cleanQuery, duration)
		
		// File log
		if p.logFile != nil {
			p.logFile.WriteString(logMsg)
		}
	}
}

// GetSlowQueries returns queries slower than threshold
func (p *QueryProfiler) GetSlowQueries() []QueryProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var slow []QueryProfile
	for _, q := range p.queries {
		if q.Duration >= p.slowThreshold {
			slow = append(slow, q)
		}
	}
	return slow
}

// GetAllQueries returns all recorded queries
func (p *QueryProfiler) GetAllQueries() []QueryProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return append([]QueryProfile(nil), p.queries...)
}

// GetStats returns profiling statistics
func (p *QueryProfiler) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if len(p.queries) == 0 {
		return map[string]interface{}{
			"total_queries": 0,
			"enabled":       p.enabled,
		}
	}
	
	var totalDuration time.Duration
	var slowCount int
	minDuration := p.queries[0].Duration
	maxDuration := p.queries[0].Duration
	
	for _, q := range p.queries {
		totalDuration += q.Duration
		if q.Duration >= p.slowThreshold {
			slowCount++
		}
		if q.Duration < minDuration {
			minDuration = q.Duration
		}
		if q.Duration > maxDuration {
			maxDuration = q.Duration
		}
	}
	
	avgDuration := totalDuration / time.Duration(len(p.queries))
	
	return map[string]interface{}{
		"enabled":        p.enabled,
		"total_queries":  len(p.queries),
		"slow_queries":   slowCount,
		"avg_duration":   avgDuration.String(),
		"min_duration":   minDuration.String(),
		"max_duration":   maxDuration.String(),
		"threshold":      p.slowThreshold.String(),
	}
}

// SlowThresholdString returns the slow query threshold as a string.
func (p *QueryProfiler) SlowThresholdString() string {
	return p.slowThreshold.String()
}

// GetStatsMap returns stats as map[string]interface{} (satisfies admin.QueryProfiler).
func (p *QueryProfiler) GetStatsMap() map[string]interface{} {
	return p.GetStats()
}

// GetSlowQueriesAny returns slow queries as interface{} (satisfies admin.QueryProfiler).
func (p *QueryProfiler) GetSlowQueriesAny() interface{} {
	return p.GetSlowQueries()
}

// GetAllQueriesAny returns all queries as interface{} (satisfies admin.QueryProfiler).
func (p *QueryProfiler) GetAllQueriesAny() interface{} {
	return p.GetAllQueries()
}

// Reset clears all recorded queries
func (p *QueryProfiler) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.queries = make([]QueryProfile, 0, p.maxProfileSize)
	log.Println("[QueryProfiler] Reset complete")
}

// Close closes the profiler and log file
func (p *QueryProfiler) Close() {
	if p.logFile != nil {
		p.logFile.Close()
	}
}
