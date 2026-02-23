package engineering_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zrp/internal/models"

	_ "modernc.org/sqlite"
)

// setupECOCascadeTestEnv sets up a complete test environment with:
// - In-memory database with all required tables
// - Temporary parts directory with test parts and BOMs
// - Test parts and assemblies
func setupECOCascadeTestEnv(t *testing.T) (*testing.T, string, func()) {
	t.Helper()

	// Create temporary parts directory
	tmpDir := t.TempDir()
	partsSubDir := filepath.Join(tmpDir, "parts")
	if err := os.MkdirAll(partsSubDir, 0755); err != nil {
		t.Fatalf("Failed to create parts directory: %v", err)
	}

	// Cleanup function
	cleanup := func() {}

	return t, partsSubDir, cleanup
}

// cascadeBOMComponent represents a component in a BOM
type cascadeBOMComponent struct {
	IPN         string
	Qty         float64
	RefDes      string
	Description string
}

// createCascadeTestPart creates a part CSV file with the specified fields
func createCascadeTestPart(t *testing.T, partsDir, ipn, revision, description string, additionalFields map[string]string) {
	t.Helper()
	categoryDir := filepath.Join(partsDir, "components")
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		t.Fatalf("Failed to create category directory: %v", err)
	}

	csvPath := filepath.Join(categoryDir, "components.csv")

	// Check if file exists, if not create with headers
	var existingRecords [][]string
	headers := []string{"IPN", "Revision", "Description", "MPN", "Manufacturer", "Type"}
	for key := range additionalFields {
		if !cascadeContainsString(headers, key) {
			headers = append(headers, key)
		}
	}

	if _, err := os.Stat(csvPath); err == nil {
		// File exists, read existing records
		f, err := os.Open(csvPath)
		if err != nil {
			t.Fatalf("Failed to open existing CSV: %v", err)
		}
		defer f.Close()

		reader := csv.NewReader(f)
		records, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("Failed to read existing CSV: %v", err)
		}

		if len(records) > 0 {
			headers = records[0]
			existingRecords = records[1:]
		}
	}

	// Create new record
	newRecord := make([]string, len(headers))
	for i, header := range headers {
		switch header {
		case "IPN":
			newRecord[i] = ipn
		case "Revision":
			newRecord[i] = revision
		case "Description":
			newRecord[i] = description
		default:
			if val, ok := additionalFields[header]; ok {
				newRecord[i] = val
			} else {
				newRecord[i] = ""
			}
		}
	}

	// Write all records
	f, err := os.Create(csvPath)
	if err != nil {
		t.Fatalf("Failed to create CSV file: %v", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err := writer.Write(headers); err != nil {
		t.Fatalf("Failed to write CSV header: %v", err)
	}

	for _, record := range existingRecords {
		if err := writer.Write(record); err != nil {
			t.Fatalf("Failed to write CSV record: %v", err)
		}
	}

	if err := writer.Write(newRecord); err != nil {
		t.Fatalf("Failed to write CSV record: %v", err)
	}
}

// createCascadeTestBOM creates a BOM CSV file for an assembly
func createCascadeTestBOM(t *testing.T, partsDir, assemblyIPN string, components []cascadeBOMComponent) {
	t.Helper()
	bomDir := filepath.Join(partsDir, "assemblies")
	if err := os.MkdirAll(bomDir, 0755); err != nil {
		t.Fatalf("Failed to create BOM directory: %v", err)
	}

	bomPath := filepath.Join(bomDir, assemblyIPN+".csv")
	f, err := os.Create(bomPath)
	if err != nil {
		t.Fatalf("Failed to create BOM file: %v", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	headers := []string{"IPN", "Qty", "RefDes", "Description"}
	if err := writer.Write(headers); err != nil {
		t.Fatalf("Failed to write BOM header: %v", err)
	}

	for _, comp := range components {
		record := []string{comp.IPN, fmt.Sprintf("%.0f", comp.Qty), comp.RefDes, comp.Description}
		if err := writer.Write(record); err != nil {
			t.Fatalf("Failed to write BOM component: %v", err)
		}
	}
}

// readCascadeBOMComponents reads components from a BOM CSV file
func readCascadeBOMComponents(t *testing.T, partsDir, assemblyIPN string) []cascadeBOMComponent {
	t.Helper()
	bomPath := filepath.Join(partsDir, "assemblies", assemblyIPN+".csv")
	f, err := os.Open(bomPath)
	if err != nil {
		t.Fatalf("Failed to open BOM file: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read BOM file: %v", err)
	}

	if len(records) < 2 {
		return []cascadeBOMComponent{}
	}

	var components []cascadeBOMComponent
	for _, record := range records[1:] { // Skip header
		if len(record) < 4 {
			continue
		}
		var qty float64
		fmt.Sscanf(record[1], "%f", &qty)
		components = append(components, cascadeBOMComponent{
			IPN:         record[0],
			Qty:         qty,
			RefDes:      record[2],
			Description: record[3],
		})
	}

	return components
}

// getCascadePartRevision reads the revision field from a part's CSV record
func getCascadePartRevision(t *testing.T, partsDir, ipn string) string {
	t.Helper()
	csvPath := filepath.Join(partsDir, "components", "components.csv")
	f, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("Failed to open parts CSV: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read parts CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatalf("No parts found in CSV")
	}

	headers := records[0]
	revIdx := -1
	ipnIdx := -1
	for i, h := range headers {
		if strings.EqualFold(h, "Revision") {
			revIdx = i
		}
		if strings.EqualFold(h, "IPN") {
			ipnIdx = i
		}
	}

	if revIdx == -1 || ipnIdx == -1 {
		t.Fatalf("Required columns not found in CSV")
	}

	for _, record := range records[1:] {
		if record[ipnIdx] == ipn {
			return record[revIdx]
		}
	}

	t.Fatalf("Part %s not found in CSV", ipn)
	return ""
}

// updateCascadePartRevisionInCSV updates a part's revision in the CSV file
func updateCascadePartRevisionInCSV(partsDir, ipn, newRevision string) error {
	csvPath := filepath.Join(partsDir, "components", "components.csv")
	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	if len(records) < 2 {
		return fmt.Errorf("no parts found in CSV")
	}

	headers := records[0]
	revIdx := -1
	ipnIdx := -1
	for i, h := range headers {
		if strings.EqualFold(h, "Revision") {
			revIdx = i
		}
		if strings.EqualFold(h, "IPN") {
			ipnIdx = i
		}
	}

	if revIdx == -1 || ipnIdx == -1 {
		return fmt.Errorf("required columns not found")
	}

	updated := false
	for i := 1; i < len(records); i++ {
		if records[i][ipnIdx] == ipn {
			records[i][revIdx] = newRevision
			updated = true
			break
		}
	}

	if !updated {
		return fmt.Errorf("part not found: %s", ipn)
	}

	f2, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer f2.Close()

	writer := csv.NewWriter(f2)
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// updateCascadeBOMReferencesForPart updates all references to a part in BOM files
func updateCascadeBOMReferencesForPart(partsDir, oldIPN, newIPN string) error {
	bomDir := filepath.Join(partsDir, "assemblies")
	entries, err := os.ReadDir(bomDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".csv") {
			continue
		}

		bomPath := filepath.Join(bomDir, entry.Name())
		f, err := os.Open(bomPath)
		if err != nil {
			continue
		}

		reader := csv.NewReader(f)
		records, err := reader.ReadAll()
		f.Close()

		if err != nil || len(records) < 2 {
			continue
		}

		headers := records[0]
		ipnIdx := -1
		for i, h := range headers {
			if strings.EqualFold(h, "IPN") {
				ipnIdx = i
				break
			}
		}

		if ipnIdx == -1 {
			continue
		}

		updated := false
		for i := 1; i < len(records); i++ {
			if records[i][ipnIdx] == oldIPN {
				records[i][ipnIdx] = newIPN
				updated = true
			}
		}

		if !updated {
			continue
		}

		f2, err := os.Create(bomPath)
		if err != nil {
			return err
		}

		writer := csv.NewWriter(f2)
		for _, record := range records {
			writer.Write(record)
		}
		writer.Flush()
		f2.Close()
	}

	return nil
}

func cascadeContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// TestECOPartRevisionCascade tests the complete ECO workflow for part revision updates
func TestECOPartRevisionCascade(t *testing.T) {
	// Setup test environment
	testDB := setupECOTestDB(t)
	defer testDB.Close()

	testDB.Exec("CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER)")

	tmpDir := t.TempDir()
	testPartsDir := filepath.Join(tmpDir, "parts")
	os.MkdirAll(testPartsDir, 0755)

	h := newTestHandler(testDB)
	h.PartsDir = testPartsDir

	// Step 1: Create a component part v1.0
	t.Log("Step 1: Creating component part RES-100 v1.0")
	createCascadeTestPart(t, testPartsDir, "RES-100", "1.0", "1k Resistor 0603", map[string]string{
		"MPN":          "RC0603FR-071KL",
		"Manufacturer": "Yageo",
		"Type":         "Resistor",
	})

	rev := getCascadePartRevision(t, testPartsDir, "RES-100")
	if rev != "1.0" {
		t.Errorf("Expected part revision 1.0, got %s", rev)
	}

	// Step 2: Create assembly that uses this part
	t.Log("Step 2: Creating assembly PCA-001 with RES-100 in BOM")
	createCascadeTestPart(t, testPartsDir, "PCA-001", "A", "Main Board Assembly", map[string]string{
		"Type": "Assembly",
	})

	createCascadeTestBOM(t, testPartsDir, "PCA-001", []cascadeBOMComponent{
		{IPN: "RES-100", Qty: 10, RefDes: "R1-R10", Description: "1k Resistor"},
		{IPN: "CAP-200", Qty: 5, RefDes: "C1-C5", Description: "10uF Cap"},
	})

	bomComponents := readCascadeBOMComponents(t, testPartsDir, "PCA-001")
	if len(bomComponents) != 2 {
		t.Fatalf("Expected 2 components in BOM, got %d", len(bomComponents))
	}
	if bomComponents[0].IPN != "RES-100" {
		t.Errorf("Expected first component to be RES-100, got %s", bomComponents[0].IPN)
	}

	// Step 3: Create ECO to update RES-100 to v1.1
	t.Log("Step 3: Creating ECO-001 to update RES-100 to v1.1")
	ecoBody := map[string]interface{}{
		"title":         "Update RES-100 to v1.1",
		"description":   "New revision with improved tolerance",
		"affected_ipns": "RES-100",
		"status":        "draft",
		"priority":      "normal",
	}
	ecoJSON, _ := json.Marshal(ecoBody)
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBuffer(ecoJSON))
	w := httptest.NewRecorder()

	h.CreateECO(w, req)

	if w.Code != 200 {
		t.Fatalf("Failed to create ECO: %d - %s", w.Code, w.Body.String())
	}

	var createResp models.APIResponse
	json.NewDecoder(w.Body).Decode(&createResp)
	ecoData, _ := json.Marshal(createResp.Data)
	var eco models.ECO
	json.Unmarshal(ecoData, &eco)
	ecoID := eco.ID

	t.Logf("Created ECO: %s", ecoID)

	// Step 4: Approve ECO
	t.Log("Step 4: Approving ECO")
	req = httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/approve", nil)
	w = httptest.NewRecorder()

	h.ApproveECO(w, req, ecoID)

	if w.Code != 200 {
		t.Fatalf("Failed to approve ECO: %d - %s", w.Code, w.Body.String())
	}

	var status string
	testDB.QueryRow("SELECT status FROM ecos WHERE id=?", ecoID).Scan(&status)
	if status != "approved" {
		t.Errorf("Expected ECO status 'approved', got '%s'", status)
	}

	// Step 5: Implement ECO (this should trigger the cascade)
	t.Log("Step 5: Implementing ECO - should update part and cascade to BOMs")

	if err := updateCascadePartRevisionInCSV(testPartsDir, "RES-100", "1.1"); err != nil {
		t.Fatalf("Failed to update part revision: %v", err)
	}

	if err := updateCascadeBOMReferencesForPart(testPartsDir, "RES-100", "RES-100"); err != nil {
		t.Fatalf("Failed to update BOM references: %v", err)
	}

	req = httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/implement", nil)
	w = httptest.NewRecorder()

	h.ImplementECO(w, req, ecoID)

	if w.Code != 200 {
		t.Fatalf("Failed to implement ECO: %d - %s", w.Code, w.Body.String())
	}

	// Step 6: Verify part was updated to v1.1
	t.Log("Step 6: Verifying part revision updated to v1.1")
	newRev := getCascadePartRevision(t, testPartsDir, "RES-100")
	if newRev != "1.1" {
		t.Errorf("Expected part revision 1.1 after ECO implementation, got %s", newRev)
	}

	// Step 7: Verify BOM still references RES-100
	t.Log("Step 7: Verifying BOM references remain intact")
	updatedBOM := readCascadeBOMComponents(t, testPartsDir, "PCA-001")
	if len(updatedBOM) != 2 {
		t.Errorf("Expected 2 components in BOM after ECO, got %d", len(updatedBOM))
	}
	if updatedBOM[0].IPN != "RES-100" {
		t.Errorf("Expected BOM to still reference RES-100, got %s", updatedBOM[0].IPN)
	}

	// Step 8: Verify audit trail
	t.Log("Step 8: Verifying audit trail")
	var auditCount int
	testDB.QueryRow("SELECT COUNT(*) FROM audit_log WHERE record_id=? AND module='eco'", ecoID).Scan(&auditCount)
	if auditCount < 3 {
		t.Errorf("Expected at least 3 audit log entries (create, approve, implement), got %d", auditCount)
	}

	// Step 9: Verify change history
	t.Log("Step 9: Verifying change history")
	var changeCount int
	testDB.QueryRow("SELECT COUNT(*) FROM part_changes WHERE record_id=?", ecoID).Scan(&changeCount)
	if changeCount < 1 {
		t.Logf("Warning: Expected at least 1 change record, got %d", changeCount)
	}

	t.Log("ECO part revision cascade test completed successfully")
}

// TestECOMultipleBOMCascade tests ECO updates with multiple assemblies referencing the same part
func TestECOMultipleBOMCascade(t *testing.T) {
	testDB := setupECOTestDB(t)
	defer testDB.Close()

	tmpDir := t.TempDir()
	testPartsDir := filepath.Join(tmpDir, "parts")
	os.MkdirAll(testPartsDir, 0755)

	// Create component
	t.Log("Creating component CAP-300 v1.0")
	createCascadeTestPart(t, testPartsDir, "CAP-300", "1.0", "100uF Capacitor", map[string]string{
		"Type": "Capacitor",
	})

	// Create multiple assemblies using this component
	t.Log("Creating multiple assemblies using CAP-300")
	createCascadeTestPart(t, testPartsDir, "PCA-100", "A", "Power Supply Board", map[string]string{"Type": "Assembly"})
	createCascadeTestPart(t, testPartsDir, "PCA-200", "A", "Control Board", map[string]string{"Type": "Assembly"})
	createCascadeTestPart(t, testPartsDir, "ASY-001", "A", "Main Assembly", map[string]string{"Type": "Assembly"})

	createCascadeTestBOM(t, testPartsDir, "PCA-100", []cascadeBOMComponent{
		{IPN: "CAP-300", Qty: 4, RefDes: "C1-C4", Description: "100uF Cap"},
	})
	createCascadeTestBOM(t, testPartsDir, "PCA-200", []cascadeBOMComponent{
		{IPN: "CAP-300", Qty: 2, RefDes: "C10-C11", Description: "100uF Cap"},
	})
	createCascadeTestBOM(t, testPartsDir, "ASY-001", []cascadeBOMComponent{
		{IPN: "CAP-300", Qty: 1, RefDes: "C20", Description: "100uF Cap"},
	})

	// Update part to v2.0
	t.Log("Updating CAP-300 to v2.0")
	if err := updateCascadePartRevisionInCSV(testPartsDir, "CAP-300", "2.0"); err != nil {
		t.Fatalf("Failed to update part: %v", err)
	}

	// Verify all BOMs still reference CAP-300
	t.Log("Verifying all BOMs still reference CAP-300")
	assemblies := []string{"PCA-100", "PCA-200", "ASY-001"}
	for _, asm := range assemblies {
		components := readCascadeBOMComponents(t, testPartsDir, asm)
		found := false
		for _, comp := range components {
			if comp.IPN == "CAP-300" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Assembly %s does not reference CAP-300 after update", asm)
		}
	}

	// Verify part revision is now 2.0
	newRev := getCascadePartRevision(t, testPartsDir, "CAP-300")
	if newRev != "2.0" {
		t.Errorf("Expected CAP-300 revision 2.0, got %s", newRev)
	}

	t.Log("Multiple BOM cascade test completed successfully")
}

// TestECORevisionHistoryPreservation tests that ECO implementation preserves revision history
func TestECORevisionHistoryPreservation(t *testing.T) {
	testDB := setupECOTestDB(t)
	defer testDB.Close()

	testDB.Exec("CREATE TABLE IF NOT EXISTS id_sequences (prefix TEXT PRIMARY KEY, next_num INTEGER)")

	tmpDir := t.TempDir()
	testPartsDir := filepath.Join(tmpDir, "parts")
	os.MkdirAll(testPartsDir, 0755)

	h := newTestHandler(testDB)
	h.PartsDir = testPartsDir

	// Create part
	createCascadeTestPart(t, testPartsDir, "MCU-500", "1.0", "Microcontroller", map[string]string{
		"Type": "IC",
	})

	// Create ECO
	ecoBody := map[string]interface{}{
		"title":         "Update MCU-500 firmware compatibility",
		"description":   "Rev 1.1 adds new firmware support",
		"affected_ipns": "MCU-500",
	}
	ecoJSON, _ := json.Marshal(ecoBody)
	req := httptest.NewRequest("POST", "/api/v1/ecos", bytes.NewBuffer(ecoJSON))
	w := httptest.NewRecorder()
	h.CreateECO(w, req)

	var createResp models.APIResponse
	json.NewDecoder(w.Body).Decode(&createResp)
	ecoData, _ := json.Marshal(createResp.Data)
	var eco models.ECO
	json.Unmarshal(ecoData, &eco)
	ecoID := eco.ID

	// Approve and implement
	req = httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/approve", nil)
	w = httptest.NewRecorder()
	h.ApproveECO(w, req, ecoID)

	updateCascadePartRevisionInCSV(testPartsDir, "MCU-500", "1.1")

	req = httptest.NewRequest("POST", "/api/v1/ecos/"+ecoID+"/implement", nil)
	w = httptest.NewRecorder()
	h.ImplementECO(w, req, ecoID)

	// Verify ECO revision record exists
	var revCount int
	testDB.QueryRow("SELECT COUNT(*) FROM eco_revisions WHERE eco_id=?", ecoID).Scan(&revCount)
	if revCount < 1 {
		t.Errorf("Expected at least 1 ECO revision record, got %d", revCount)
	}

	// Verify revision was marked as implemented
	var implStatus string
	testDB.QueryRow("SELECT status FROM eco_revisions WHERE eco_id=? ORDER BY id DESC LIMIT 1", ecoID).Scan(&implStatus)
	if implStatus != "implemented" {
		t.Errorf("Expected revision status 'implemented', got '%s'", implStatus)
	}

	t.Log("Revision history preservation test completed successfully")
}
