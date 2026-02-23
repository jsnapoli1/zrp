package admin_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"zrp/internal/handlers/admin"

	_ "modernc.org/sqlite"
)

func decodeGeneralSettingsLocal(t *testing.T, w *httptest.ResponseRecorder) admin.GeneralSettings {
	t.Helper()
	var resp struct {
		Data admin.GeneralSettings `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	return resp.Data
}

func TestGetGeneralSettingsDefaults(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	req := httptest.NewRequest("GET", "/api/v1/settings/general", nil)
	w := httptest.NewRecorder()
	h.GetGeneralSettings(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	s := decodeGeneralSettingsLocal(t, w)
	if s.AppName != "ZRP" {
		t.Errorf("expected default app_name ZRP, got %q", s.AppName)
	}
	if s.Currency != "USD" {
		t.Errorf("expected default currency USD, got %q", s.Currency)
	}
	if s.DateFormat != "YYYY-MM-DD" {
		t.Errorf("expected default date_format YYYY-MM-DD, got %q", s.DateFormat)
	}
}

func TestPutGeneralSettings(t *testing.T) {
	db := setupAuthTestDB(t)
	defer db.Close()
	h := newTestHandler(db)

	body := `{"app_name":"MyApp","company_name":"Acme Corp","company_address":"123 Main St","currency":"EUR","date_format":"DD/MM/YYYY"}`
	req := httptest.NewRequest("PUT", "/api/v1/settings/general", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.PutGeneralSettings(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify persisted via GET
	req2 := httptest.NewRequest("GET", "/api/v1/settings/general", nil)
	w2 := httptest.NewRecorder()
	h.GetGeneralSettings(w2, req2)

	s := decodeGeneralSettingsLocal(t, w2)
	if s.AppName != "MyApp" {
		t.Errorf("expected MyApp, got %q", s.AppName)
	}
	if s.CompanyName != "Acme Corp" {
		t.Errorf("expected Acme Corp, got %q", s.CompanyName)
	}
	if s.Currency != "EUR" {
		t.Errorf("expected EUR, got %q", s.Currency)
	}
	if s.DateFormat != "DD/MM/YYYY" {
		t.Errorf("expected DD/MM/YYYY, got %q", s.DateFormat)
	}
}
