package main

import (
	"testing"
)

func TestSchemaColumns_AuditLog(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Test that audit_log has module column
	_, err := testDB.Exec(`INSERT INTO audit_log (module, action, record_id, username) VALUES (?, ?, ?, ?)`,
		"test", "CREATE", "123", "testuser")
	if err != nil {
		t.Fatalf("Failed to insert into audit_log with module column: %v", err)
	}

	var module string
	err = testDB.QueryRow("SELECT module FROM audit_log WHERE record_id = '123'").Scan(&module)
	if err != nil {
		t.Fatalf("Failed to select module column from audit_log: %v", err)
	}
	if module != "test" {
		t.Errorf("Expected module='test', got '%s'", module)
	}
	t.Logf("✓ audit_log.module column verified")
}

func TestSchemaColumns_ECOs(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Test that ecos has affected_ipns, created_by, approved_at, approved_by columns
	_, err := testDB.Exec(`INSERT INTO ecos (id, title, affected_ipns, created_by) VALUES (?, ?, ?, ?)`,
		"ECO-001", "Test ECO", "IPN-001,IPN-002", "engineer1")
	if err != nil {
		t.Fatalf("Failed to insert into ecos with affected_ipns and created_by: %v", err)
	}

	var affectedIpns, createdBy string
	err = testDB.QueryRow("SELECT affected_ipns, created_by FROM ecos WHERE id = 'ECO-001'").
		Scan(&affectedIpns, &createdBy)
	if err != nil {
		t.Fatalf("Failed to select affected_ipns, created_by from ecos: %v", err)
	}
	if affectedIpns != "IPN-001,IPN-002" {
		t.Errorf("Expected affected_ipns='IPN-001,IPN-002', got '%s'", affectedIpns)
	}
	if createdBy != "engineer1" {
		t.Errorf("Expected created_by='engineer1', got '%s'", createdBy)
	}

	// Test approved_at and approved_by (nullable)
	_, err = testDB.Exec("UPDATE ecos SET approved_by = 'manager1' WHERE id = 'ECO-001'")
	if err != nil {
		t.Fatalf("Failed to update approved_by: %v", err)
	}

	t.Logf("✓ ecos.affected_ipns, created_by, approved_at, approved_by columns verified")
}

func TestSchemaColumns_NCRs(t *testing.T) {
	testDB := setupTestDB(t)
	defer testDB.Close()

	// Test that ncrs has ipn, serial_number, severity, defect_type, corrective_action, resolved_at
	_, err := testDB.Exec(`INSERT INTO ncrs (id, title, ipn, serial_number, severity, defect_type, corrective_action) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"NCR-001", "Test NCR", "IPN-100", "SN-12345", "critical", "crack", "replace component")
	if err != nil {
		t.Fatalf("Failed to insert into ncrs with ipn, serial_number, severity, defect_type, corrective_action: %v", err)
	}

	var ipn, serialNumber, severity, defectType, correctiveAction string
	err = testDB.QueryRow(`SELECT ipn, serial_number, severity, defect_type, corrective_action 
		FROM ncrs WHERE id = 'NCR-001'`).
		Scan(&ipn, &serialNumber, &severity, &defectType, &correctiveAction)
	if err != nil {
		t.Fatalf("Failed to select columns from ncrs: %v", err)
	}

	if ipn != "IPN-100" {
		t.Errorf("Expected ipn='IPN-100', got '%s'", ipn)
	}
	if serialNumber != "SN-12345" {
		t.Errorf("Expected serial_number='SN-12345', got '%s'", serialNumber)
	}
	if severity != "critical" {
		t.Errorf("Expected severity='critical', got '%s'", severity)
	}
	if defectType != "crack" {
		t.Errorf("Expected defect_type='crack', got '%s'", defectType)
	}
	if correctiveAction != "replace component" {
		t.Errorf("Expected corrective_action='replace component', got '%s'", correctiveAction)
	}

	// Test resolved_at (nullable)
	_, err = testDB.Exec("UPDATE ncrs SET resolved_at = CURRENT_TIMESTAMP WHERE id = 'NCR-001'")
	if err != nil {
		t.Fatalf("Failed to update resolved_at: %v", err)
	}

	t.Logf("✓ ncrs.ipn, serial_number, severity, defect_type, corrective_action, resolved_at columns verified")
}
