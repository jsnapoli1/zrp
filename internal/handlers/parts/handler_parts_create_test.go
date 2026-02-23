package parts_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"zrp/internal/models"
)

// setupCSVPartsDir creates a temp dir with flat CSV files (the actual ZRP format)
func setupCSVPartsDir(t *testing.T) (string, *testing.T) {
	t.Helper()
	dir := t.TempDir()

	// z-ana.csv - analog parts
	f1, _ := os.Create(filepath.Join(dir, "z-ana.csv"))
	w1 := csv.NewWriter(f1)
	w1.WriteAll([][]string{
		{"IPN", "description", "manufacturer", "value"},
		{"ANA-001", "Op Amp", "TI", "LM358"},
	})
	f1.Close()

	// z-dig.csv - digital parts
	f2, _ := os.Create(filepath.Join(dir, "z-dig.csv"))
	w2 := csv.NewWriter(f2)
	w2.WriteAll([][]string{
		{"IPN", "description", "package"},
		{"DIG-001", "FPGA", "BGA-256"},
	})
	f2.Close()

	return dir, t
}

func TestCreatePart_Success(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	body := `{"ipn":"ANA-002","category":"z-ana","fields":{"description":"New Amp","manufacturer":"AD","value":"AD8421"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreatePart(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it was written to the CSV
	cats, _, _, _ := h.LoadPartsFromDir()
	found := false
	for _, p := range cats["z-ana"] {
		if p.IPN == "ANA-002" {
			found = true
			if p.Fields["description"] != "New Amp" {
				t.Errorf("expected description 'New Amp', got %q", p.Fields["description"])
			}
		}
	}
	if !found {
		t.Error("created part ANA-002 not found in z-ana category")
	}
}

func TestCreatePart_DuplicateIPN(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	body := `{"ipn":"ANA-001","category":"z-ana","fields":{"description":"Duplicate"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreatePart(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409 for duplicate, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePart_DuplicateAcrossCategories(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	// Try to create a part in z-dig with an IPN that exists in z-ana
	body := `{"ipn":"ANA-001","category":"z-dig","fields":{"description":"Cross-cat duplicate"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreatePart(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409 for cross-category duplicate, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreatePart_MissingIPN(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	body := `{"category":"z-ana","fields":{"description":"No IPN"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreatePart(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400 for missing IPN, got %d", w.Code)
	}
}

func TestCreatePart_InvalidCategory(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	body := `{"ipn":"XXX-001","category":"nonexistent","fields":{"description":"Bad cat"}}`
	req := httptest.NewRequest("POST", "/api/v1/parts", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreatePart(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404 for bad category, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCategory_Success(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	body := `{"title":"Connectors","prefix":"CON"}`
	req := httptest.NewRequest("POST", "/api/v1/categories", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateCategory(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the CSV was created
	csvPath := filepath.Join(dir, "z-con.csv")
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Fatal("z-con.csv was not created")
	}

	// Read it and check header
	f, _ := os.Open(csvPath)
	defer f.Close()
	r := csv.NewReader(f)
	r.Comment = '#'
	records, _ := r.ReadAll()
	if len(records) < 1 {
		t.Fatal("CSV has no header row")
	}
	if records[0][0] != "IPN" {
		t.Errorf("expected first column 'IPN', got %q", records[0][0])
	}
}

func TestCreateCategory_DuplicatePrefix(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	body := `{"title":"Analog Again","prefix":"ANA"}`
	req := httptest.NewRequest("POST", "/api/v1/categories", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateCategory(w, req)

	if w.Code != 409 {
		t.Fatalf("expected 409 for duplicate prefix, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCategory_MissingFields(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	body := `{"title":"","prefix":""}`
	req := httptest.NewRequest("POST", "/api/v1/categories", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h.CreateCategory(w, req)

	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCheckIPNExists(t *testing.T) {
	dir, _ := setupCSVPartsDir(t)
	h := newTestHandler(nil, dir)

	req := httptest.NewRequest("GET", "/api/v1/parts/check-ipn?ipn=ANA-001", nil)
	w := httptest.NewRecorder()
	h.CheckIPN(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp struct {
		Data struct {
			Exists bool `json:"exists"`
		} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Data.Exists {
		t.Error("expected ANA-001 to exist")
	}

	// Check non-existent
	req2 := httptest.NewRequest("GET", "/api/v1/parts/check-ipn?ipn=NOPE-999", nil)
	w2 := httptest.NewRecorder()
	h.CheckIPN(w2, req2)
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Data.Exists {
		t.Error("expected NOPE-999 to not exist")
	}
}

// Ensure models import is used
var _ models.Part
