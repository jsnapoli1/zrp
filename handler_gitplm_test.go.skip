package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestGetGitPLMConfigDefault(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("GET", "/api/v1/settings/gitplm", "", loginAdmin(t))
	w := httptest.NewRecorder()
	handleGetGitPLMConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data GitPLMConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.BaseURL != "" {
		t.Errorf("expected empty base_url, got %q", resp.Data.BaseURL)
	}
}

func TestUpdateAndGetGitPLMConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Update
	req := authedRequest("PUT", "/api/v1/settings/gitplm", `{"base_url":"https://gitplm.example.com/"}`, cookie)
	w := httptest.NewRecorder()
	handleUpdateGitPLMConfig(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify trailing slash trimmed
	var putResp struct {
		Data GitPLMConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &putResp)
	if putResp.Data.BaseURL != "https://gitplm.example.com" {
		t.Errorf("expected trimmed URL, got %q", putResp.Data.BaseURL)
	}

	// Get
	req = authedRequest("GET", "/api/v1/settings/gitplm", "", cookie)
	w = httptest.NewRecorder()
	handleGetGitPLMConfig(w, req)
	var getResp struct {
		Data GitPLMConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	if getResp.Data.BaseURL != "https://gitplm.example.com" {
		t.Errorf("expected 'https://gitplm.example.com', got %q", getResp.Data.BaseURL)
	}
}

func TestUpdateGitPLMConfigOverwrite(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Set first value
	req := authedRequest("PUT", "/api/v1/settings/gitplm", `{"base_url":"https://old.example.com"}`, cookie)
	w := httptest.NewRecorder()
	handleUpdateGitPLMConfig(w, req)

	// Overwrite
	req = authedRequest("PUT", "/api/v1/settings/gitplm", `{"base_url":"https://new.example.com"}`, cookie)
	w = httptest.NewRecorder()
	handleUpdateGitPLMConfig(w, req)

	// Verify
	req = authedRequest("GET", "/api/v1/settings/gitplm", "", cookie)
	w = httptest.NewRecorder()
	handleGetGitPLMConfig(w, req)
	var resp struct {
		Data GitPLMConfig `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.BaseURL != "https://new.example.com" {
		t.Errorf("expected overwritten URL, got %q", resp.Data.BaseURL)
	}
}

func TestGetGitPLMURLNotConfigured(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	req := authedRequest("GET", "/api/v1/parts/IPN-001/gitplm-url", "", loginAdmin(t))
	w := httptest.NewRecorder()
	handleGetGitPLMURL(w, req, "IPN-001")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data GitPLMURLResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Configured {
		t.Error("expected configured=false")
	}
	if resp.Data.URL != "" {
		t.Errorf("expected empty URL, got %q", resp.Data.URL)
	}
}

func TestGetGitPLMURLConfigured(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	cookie := loginAdmin(t)

	// Configure
	req := authedRequest("PUT", "/api/v1/settings/gitplm", `{"base_url":"https://plm.acme.com"}`, cookie)
	w := httptest.NewRecorder()
	handleUpdateGitPLMConfig(w, req)

	// Get URL
	req = authedRequest("GET", "/api/v1/parts/IPN-001/gitplm-url", "", cookie)
	w = httptest.NewRecorder()
	handleGetGitPLMURL(w, req, "IPN-001")
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data GitPLMURLResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Data.Configured {
		t.Error("expected configured=true")
	}
	if resp.Data.URL != "https://plm.acme.com/parts/IPN-001" {
		t.Errorf("expected 'https://plm.acme.com/parts/IPN-001', got %q", resp.Data.URL)
	}
}
