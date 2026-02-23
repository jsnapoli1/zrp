package parts_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreatePartIPNValidation tests IPN validation when creating parts.
func TestCreatePartIPNValidation(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	tests := []struct {
		name    string
		ipn     string
		wantErr bool
	}{
		{"Empty IPN", "", true},
		{"Valid IPN", "ANA-NEW", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"ipn":      tt.ipn,
				"category": "z-ana",
				"fields":   map[string]string{"description": "Test"},
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreatePart(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestCreatePartCategoryValidation tests category validation when creating parts.
func TestCreatePartCategoryValidation(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	tests := []struct {
		name     string
		category string
		wantErr  bool
	}{
		{"Empty category", "", true},
		{"Valid category", "z-ana", false},
		{"Nonexistent category", "nonexistent", true},
	}

	ipnCounter := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipnCounter++
			payload := map[string]interface{}{
				"ipn":      strings.Repeat("V", 1) + "-" + strings.Repeat("0", ipnCounter),
				"category": tt.category,
				"fields":   map[string]string{"description": "Test"},
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreatePart(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestCreateCategoryValidation tests validation when creating categories.
func TestCreateCategoryValidation(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	tests := []struct {
		name    string
		title   string
		prefix  string
		wantErr bool
	}{
		{"Empty title", "", "xxx", true},
		{"Empty prefix", "Title", "", true},
		{"Both empty", "", "", true},
		{"Valid", "New Category", "new", false},
		{"Duplicate prefix", "Analog Again", "ANA", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]interface{}{
				"title":  tt.title,
				"prefix": tt.prefix,
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest("POST", "/api/v1/parts/categories", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.CreateCategory(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestCheckIPNValidation tests IPN query parameter validation.
func TestCheckIPNValidation(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	tests := []struct {
		name    string
		ipn     string
		wantErr bool
	}{
		{"Empty IPN", "", true},
		{"Existing IPN", "ANA-001", false},
		{"Non-existing IPN", "NOPE-999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts/check-ipn?ipn="+tt.ipn, nil)
			w := httptest.NewRecorder()

			h.CheckIPN(w, req)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success", tt.name)
			}
			if !tt.wantErr && w.Code != 200 {
				t.Errorf("Expected success for %s, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestPartBOMValidation tests BOM endpoint validation for non-assembly parts.
func TestPartBOMValidation(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	tests := []struct {
		name    string
		ipn     string
		wantErr bool
	}{
		{"Non-assembly IPN", "ANA-001", true},
		{"PCA prefix (assembly)", "PCA-TEST", false},
		{"ASY prefix (assembly)", "ASY-TEST", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/parts/"+tt.ipn+"/bom", nil)
			w := httptest.NewRecorder()

			h.PartBOM(w, req, tt.ipn)

			if tt.wantErr && w.Code == 200 {
				t.Errorf("Expected error for %s, got success (status %d)", tt.name, w.Code)
			}
			if !tt.wantErr && w.Code == 400 {
				t.Errorf("Did not expect 400 error for %s", tt.name)
			}
		})
	}
}

// TestCreatePartInvalidBody tests that malformed JSON body is rejected.
func TestCreatePartInvalidBody(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreatePart(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON body, got %d", w.Code)
	}
}

// TestCreateCategoryInvalidBody tests that malformed JSON body is rejected.
func TestCreateCategoryInvalidBody(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	req := httptest.NewRequest("POST", "/api/v1/parts/categories", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateCategory(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON body, got %d", w.Code)
	}
}
