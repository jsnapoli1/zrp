package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestDebugEmptySearch(t *testing.T) {
	origDB := db
	defer func() { db = origDB }()
	db = setupSearchTestDB(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/search?q=", nil)
	w := httptest.NewRecorder()

	handleGlobalSearch(w, req)

	t.Logf("Response body: %s", w.Body.String())
	
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Response: %+v", response)
}
