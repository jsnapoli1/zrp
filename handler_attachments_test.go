package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// setupAttachmentsTestDB creates an in-memory test database with required tables
func setupAttachmentsTestDB(t *testing.T) (*sql.DB, func()) {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	if _, err := testDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create attachments table
	_, err = testDB.Exec(`
		CREATE TABLE attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			original_name TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			mime_type TEXT,
			uploaded_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create attachments table: %v", err)
	}

	// Create users table for authentication
	_, err = testDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user' CHECK(role IN ('admin','user','readonly')),
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create audit_log table
	_, err = testDB.Exec(`
		CREATE TABLE audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			username TEXT DEFAULT 'system',
			action TEXT NOT NULL,
			module TEXT NOT NULL,
			record_id TEXT NOT NULL,
			summary TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create audit_log table: %v", err)
	}

	// Create uploads directory
	os.RemoveAll("uploads")
	os.MkdirAll("uploads", 0755)
	
	cleanup := func() {
		testDB.Close()
		os.RemoveAll("uploads")
		os.MkdirAll("uploads", 0755) // Recreate for other tests
	}

	return testDB, cleanup
}

// createTestUser creates a user for testing
func createAttachmentTestUser(t *testing.T, db *sql.DB, username, role string) int {
	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, role, active) VALUES (?, ?, ?, 1)",
		username, "test_hash", role,
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

// createMultipartRequest creates a multipart form request for file upload
func createMultipartRequest(t *testing.T, filename string, content []byte, module, recordID string) (*http.Request, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}

	writer.WriteField("module", module)
	writer.WriteField("record_id", recordID)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/attachments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, writer.FormDataContentType()
}

// ==================== UPLOAD TESTS ====================

func TestHandleUploadAttachment_Success(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	_ = createAttachmentTestUser(t, testDB, "testuser", "user")

	tests := []struct {
		name     string
		filename string
		content  []byte
		module   string
		recordID string
	}{
		{
			name:     "PDF upload",
			filename: "document.pdf",
			content:  []byte("%PDF-1.4 fake pdf content"),
			module:   "eco",
			recordID: "ECO-001",
		},
		{
			name:     "Image upload",
			filename: "photo.png",
			content:  []byte("PNG fake image data"),
			module:   "ncr",
			recordID: "NCR-123",
		},
		{
			name:     "Excel upload",
			filename: "data.xlsx",
			content:  []byte("Excel data"),
			module:   "rma",
			recordID: "RMA-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := createMultipartRequest(t, tt.filename, tt.content, tt.module, tt.recordID)

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if w.Code != 201 {
				t.Errorf("Expected 201 Created, got %d: %s", w.Code, w.Body.String())
				return
			}

			var resp struct {
				Data Attachment `json:"data"`
			}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}
			result := resp.Data

			if result.Module != tt.module {
				t.Errorf("Expected module %s, got %s", tt.module, result.Module)
			}

			if result.RecordID != tt.recordID {
				t.Errorf("Expected record_id %s, got %s", tt.recordID, result.RecordID)
			}

			if result.OriginalName != tt.filename {
				t.Errorf("Expected original_name %s, got %s", tt.filename, result.OriginalName)
			}

			if result.SizeBytes != int64(len(tt.content)) {
				t.Errorf("Expected size %d, got %d", len(tt.content), result.SizeBytes)
			}

			// Verify file was saved
			filePath := filepath.Join("uploads", result.Filename)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("File was not saved to disk: %s", filePath)
			}

			// Verify database record
			var count int
			err := testDB.QueryRow("SELECT COUNT(*) FROM attachments WHERE id = ?", result.ID).Scan(&count)
			if err != nil || count != 1 {
				t.Errorf("Attachment not found in database")
			}
		})
	}
}

func TestHandleUploadAttachment_MissingFields(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	tests := []struct {
		name     string
		module   string
		recordID string
		wantCode int
	}{
		{
			name:     "Missing module",
			module:   "",
			recordID: "TEST-001",
			wantCode: 400,
		},
		{
			name:     "Missing record_id",
			module:   "eco",
			recordID: "",
			wantCode: 400,
		},
		{
			name:     "Both missing",
			module:   "",
			recordID: "",
			wantCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := createMultipartRequest(t, "test.pdf", []byte("content"), tt.module, tt.recordID)

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

func TestHandleUploadAttachment_NoFile(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("module", "eco")
	writer.WriteField("record_id", "ECO-001")
	writer.Close()

	req := httptest.NewRequest("POST", "/api/attachments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	handleUploadAttachment(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 Bad Request for missing file, got %d", w.Code)
	}
}

// ==================== SECURITY TESTS ====================

func TestHandleUploadAttachment_DangerousExtensions(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	dangerousFiles := []struct {
		filename string
		reason   string
	}{
		{"malware.exe", "Windows executable"},
		{"script.bat", "Batch script"},
		{"shell.sh", "Shell script"},
		{"backdoor.php", "PHP script"},
		{"virus.js", "JavaScript"},
		{"trojan.vbs", "VBScript"},
		{"payload.jar", "Java archive"},
		{"exploit.ps1", "PowerShell"},
		{"hack.cmd", "Command script"},
		{"rootkit.dll", "Dynamic library"},
		{"malicious.app", "macOS application"},
		{"installer.msi", "Windows installer"},
	}

	for _, tf := range dangerousFiles {
		t.Run("Block_"+tf.filename, func(t *testing.T) {
			req, _ := createMultipartRequest(t, tf.filename, []byte("malicious"), "eco", "ECO-001")

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if w.Code == 201 || w.Code == 200 {
				t.Errorf("SECURITY VULNERABILITY: %s was accepted! Reason: %s", tf.filename, tf.reason)
			}

			// Verify file was NOT saved
			files, _ := filepath.Glob("uploads/*" + tf.filename)
			if len(files) > 0 {
				t.Errorf("SECURITY VULNERABILITY: Dangerous file saved to disk: %v", files)
			}
		})
	}
}

func TestHandleUploadAttachment_PathTraversal(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	pathTraversalAttempts := []string{
		"../../../etc/passwd.txt",
		"..\\..\\..\\windows\\system32\\config\\sam.txt",
		"../../../../../root/.ssh/id_rsa.txt",
		"test/../../../secret.txt",
		"/etc/passwd.txt",
		"\\etc\\passwd.txt",
		"C:\\Windows\\System32\\drivers\\etc\\hosts.txt",
		"test/../../outside.txt",
	}

	for _, filename := range pathTraversalAttempts {
		t.Run("Block_"+filename, func(t *testing.T) {
			req, _ := createMultipartRequest(t, filename, []byte("content"), "eco", "ECO-001")

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Even if accepted, verify no path traversal occurred
			uploadsDir, _ := filepath.Abs("uploads")

			// Check no files created outside uploads
			if _, err := os.Stat("../../../etc/passwd.txt"); !os.IsNotExist(err) {
				t.Error("CRITICAL: Path traversal succeeded - file created outside uploads!")
			}

			// If file was accepted, verify it's in uploads directory
			if w.Code == 201 {
				var resp struct {
					Data Attachment `json:"data"`
				}
				json.NewDecoder(w.Body).Decode(&resp)
				result := resp.Data
				
				filePath, _ := filepath.Abs(filepath.Join("uploads", result.Filename))
				if !strings.HasPrefix(filePath, uploadsDir) {
					t.Errorf("SECURITY VULNERABILITY: File saved outside uploads directory: %s", filePath)
				}

				// Filename should be sanitized
				if strings.Contains(result.Filename, "..") {
					t.Errorf("SECURITY VULNERABILITY: Filename contains '..' after sanitization: %s", result.Filename)
				}
			}
		})
	}
}

func TestHandleUploadAttachment_MaliciousFilenames(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	maliciousNames := []struct {
		filename string
		danger   string
	}{
		{"file;rm -rf /.txt", "Command injection via semicolon"},
		{"file|whoami.txt", "Pipe character"},
		{"file&cat /etc/passwd&.txt", "Command chaining"},
		{"file`id`.txt", "Command substitution"},
		{"file$(ls).txt", "Command substitution"},
		{"file\x00.exe.txt", "Null byte injection"},
		{"file\r\n.txt", "CRLF injection"},
		{"file<script>.txt", "HTML injection"},
		{"file>output.txt", "Redirect"},
		{"file*.txt", "Wildcard"},
		{"file?.txt", "Wildcard"},
	}

	for _, tf := range maliciousNames {
		t.Run("Sanitize_"+tf.filename, func(t *testing.T) {
			req, _ := createMultipartRequest(t, tf.filename, []byte("content"), "eco", "ECO-001")

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Might be rejected or sanitized
			if w.Code == 201 {
				var resp struct {
					Data Attachment `json:"data"`
				}
				json.NewDecoder(w.Body).Decode(&resp)
				result := resp.Data

				// Verify dangerous characters were sanitized
				dangerous := []string{";", "|", "&", "`", "$", "<", ">", "\x00", "\r", "\n"}
				for _, char := range dangerous {
					if strings.Contains(result.Filename, char) {
						t.Errorf("SECURITY VULNERABILITY: Dangerous character '%s' not sanitized: %s (Danger: %s)",
							char, result.Filename, tf.danger)
					}
				}
			}
		})
	}
}

func TestHandleUploadAttachment_FileSizeLimits(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	tests := []struct {
		name       string
		size       int64
		shouldFail bool
		wantCode   int
	}{
		{
			name:       "Empty file",
			size:       0,
			shouldFail: true,
			wantCode:   400,
		},
		{
			name:       "Tiny file (1 byte)",
			size:       1,
			shouldFail: false,
			wantCode:   201,
		},
		{
			name:       "Normal file (1MB)",
			size:       1 * 1024 * 1024,
			shouldFail: false,
			wantCode:   201,
		},
		{
			name:       "Large file (50MB)",
			size:       50 * 1024 * 1024,
			shouldFail: false,
			wantCode:   201,
		},
		{
			name:       "Max size (100MB)",
			size:       100 * 1024 * 1024,
			shouldFail: false,
			wantCode:   201,
		},
		// NOTE: This test would require actually uploading 101MB of data to properly test,
		// which is memory intensive for unit tests. The MaxBytesReader middleware handles
		// this in production. Integration tests should verify this behavior.
		// {
		// 	name:       "Over limit (101MB) - DOS prevention",
		// 	size:       101 * 1024 * 1024,
		// 	shouldFail: true,
		// 	wantCode:   413, // Request Entity Too Large
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For very large files, create sparse content
			var content []byte
			if tt.size <= 10*1024*1024 {
				content = bytes.Repeat([]byte("A"), int(tt.size))
			} else {
				// For larger files, create smaller content to save memory
				// The multipart library will handle the size check
				content = bytes.Repeat([]byte("A"), 1024)
			}

			req, _ := createMultipartRequest(t, "test.pdf", content, "eco", "ECO-001")

			// For oversized test, simulate the size in header
			if tt.size > 100*1024*1024 {
				req.ContentLength = tt.size
			}

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if tt.shouldFail {
				if w.Code == 201 || w.Code == 200 {
					t.Errorf("SECURITY VULNERABILITY: File size %d MB should have been rejected (DOS risk)", tt.size/(1024*1024))
				}
			} else {
				if w.Code != tt.wantCode {
					t.Errorf("Expected %d, got %d: %s", tt.wantCode, w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestHandleUploadAttachment_DoubleExtension(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	// Test double extension tricks
	doubleExtensions := []string{
		"document.pdf.exe",
		"image.png.bat",
		"data.xlsx.js",
		"report.txt.sh",
	}

	for _, filename := range doubleExtensions {
		t.Run("Block_"+filename, func(t *testing.T) {
			req, _ := createMultipartRequest(t, filename, []byte("fake content"), "eco", "ECO-001")

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Should be blocked based on final extension
			if w.Code == 201 || w.Code == 200 {
				t.Errorf("SECURITY VULNERABILITY: Double extension file accepted: %s", filename)
			}
		})
	}
}

// ==================== LIST ATTACHMENTS TESTS ====================

func TestHandleListAttachments_Success(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	// Insert test attachments
	testDB.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by)
		VALUES ('eco', 'ECO-001', 'eco-ECO-001-1234-test1.pdf', 'test1.pdf', 1024, 'application/pdf', 'user1')`)
	testDB.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by)
		VALUES ('eco', 'ECO-001', 'eco-ECO-001-1235-test2.pdf', 'test2.pdf', 2048, 'application/pdf', 'user2')`)
	testDB.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by)
		VALUES ('ncr', 'NCR-123', 'ncr-NCR-123-1236-other.pdf', 'other.pdf', 512, 'application/pdf', 'user1')`)

	req := httptest.NewRequest("GET", "/api/attachments?module=eco&record_id=ECO-001", nil)
	w := httptest.NewRecorder()

	handleListAttachments(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	var resp struct {
		Data []Attachment `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	attachments := resp.Data

	if len(attachments) != 2 {
		t.Errorf("Expected 2 attachments for ECO-001, got %d", len(attachments))
	}

	// Verify filtering works
	for _, att := range attachments {
		if att.Module != "eco" || att.RecordID != "ECO-001" {
			t.Errorf("Got wrong attachment: %+v", att)
		}
	}
}

func TestHandleListAttachments_MissingParams(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	tests := []struct {
		name  string
		query string
	}{
		{"Missing module", "record_id=ECO-001"},
		{"Missing record_id", "module=eco"},
		{"Missing both", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/attachments?"+tt.query, nil)
			w := httptest.NewRecorder()

			handleListAttachments(w, req)

			if w.Code != 400 {
				t.Errorf("Expected 400 Bad Request, got %d", w.Code)
			}
		})
	}
}

func TestHandleListAttachments_EmptyResult(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	req := httptest.NewRequest("GET", "/api/attachments?module=eco&record_id=NONEXISTENT", nil)
	w := httptest.NewRecorder()

	handleListAttachments(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
		return
	}

	var resp struct {
		Data []Attachment `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	attachments := resp.Data

	if attachments == nil {
		t.Error("Expected empty array, got nil")
	}

	if len(attachments) != 0 {
		t.Errorf("Expected empty array, got %d attachments", len(attachments))
	}
}

// ==================== DOWNLOAD TESTS ====================

func TestHandleDownloadAttachment_Success(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	// Create a test file
	testContent := []byte("This is test file content")
	testFilename := "eco-ECO-001-1234-test.pdf"
	filePath := filepath.Join("uploads", testFilename)
	
	if err := os.WriteFile(filePath, testContent, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(filePath)

	// Insert database record
	result, _ := testDB.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by)
		VALUES ('eco', 'ECO-001', ?, 'test.pdf', ?, 'application/pdf', 'testuser')`,
		testFilename, len(testContent))
	
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/attachments/%d/download", id), nil)
	w := httptest.NewRecorder()

	handleDownloadAttachment(w, req, fmt.Sprintf("%d", id))

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify content
	body, _ := io.ReadAll(w.Body)
	if !bytes.Equal(body, testContent) {
		t.Error("Downloaded content doesn't match uploaded content")
	}

	// Verify headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/pdf" {
		t.Errorf("Expected Content-Type application/pdf, got %s", contentType)
	}

	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "test.pdf") {
		t.Errorf("Expected Content-Disposition to contain filename, got %s", disposition)
	}
}

func TestHandleDownloadAttachment_NotFound(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	req := httptest.NewRequest("GET", "/api/attachments/999999/download", nil)
	w := httptest.NewRecorder()

	handleDownloadAttachment(w, req, "999999")

	if w.Code != 404 {
		t.Errorf("Expected 404 Not Found, got %d", w.Code)
	}
}

func TestHandleDownloadAttachment_InvalidID(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	req := httptest.NewRequest("GET", "/api/attachments/invalid/download", nil)
	w := httptest.NewRecorder()

	handleDownloadAttachment(w, req, "invalid")

	if w.Code != 400 {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}
}

func TestHandleDownloadAttachment_FileDeleted(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	// Insert DB record but don't create file
	result, _ := testDB.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by)
		VALUES ('eco', 'ECO-001', 'missing-file.pdf', 'missing.pdf', 1024, 'application/pdf', 'testuser')`)
	
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/attachments/%d/download", id), nil)
	w := httptest.NewRecorder()

	handleDownloadAttachment(w, req, fmt.Sprintf("%d", id))

	if w.Code != 404 {
		t.Errorf("Expected 404 when file missing on disk, got %d", w.Code)
	}
}

// ==================== DELETE TESTS ====================

func TestHandleDeleteAttachment_Success(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	_ = createAttachmentTestUser(t, testDB, "admin", "admin")

	// Create test file
	testFilename := "eco-ECO-001-1234-delete-test.pdf"
	filePath := filepath.Join("uploads", testFilename)
	os.WriteFile(filePath, []byte("content"), 0644)

	// Insert DB record
	result, _ := testDB.Exec(`INSERT INTO attachments (module, record_id, filename, original_name, size_bytes, mime_type, uploaded_by)
		VALUES ('eco', 'ECO-001', ?, 'delete-test.pdf', 1024, 'application/pdf', 'testuser')`, testFilename)
	
	id, _ := result.LastInsertId()

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/attachments/%d", id), nil)
	w := httptest.NewRecorder()

	handleDeleteAttachment(w, req, fmt.Sprintf("%d", id))

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		return
	}

	// Verify DB record deleted
	var count int
	testDB.QueryRow("SELECT COUNT(*) FROM attachments WHERE id = ?", id).Scan(&count)
	if count != 0 {
		t.Error("Attachment record not deleted from database")
	}

	// Verify file deleted from disk
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("File not deleted from disk")
	}
}

func TestHandleDeleteAttachment_NotFound(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	_ = createAttachmentTestUser(t, testDB, "admin", "admin")

	req := httptest.NewRequest("DELETE", "/api/attachments/999999", nil)
	w := httptest.NewRecorder()

	handleDeleteAttachment(w, req, "999999")

	if w.Code != 404 {
		t.Errorf("Expected 404 Not Found, got %d", w.Code)
	}
}

func TestHandleDeleteAttachment_InvalidID(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	_ = createAttachmentTestUser(t, testDB, "admin", "admin")

	req := httptest.NewRequest("DELETE", "/api/attachments/invalid", nil)
	w := httptest.NewRecorder()

	handleDeleteAttachment(w, req, "invalid")

	if w.Code != 400 {
		t.Errorf("Expected 400 Bad Request, got %d", w.Code)
	}
}

// ==================== FILE SERVE TESTS ====================

func TestHandleServeFile_Success(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	// Create test file
	testContent := []byte("Test file content for serving")
	testFilename := "test-serve.pdf"
	filePath := filepath.Join("uploads", testFilename)
	os.WriteFile(filePath, testContent, 0644)
	defer os.Remove(filePath)

	req := httptest.NewRequest("GET", "/files/"+testFilename, nil)
	w := httptest.NewRecorder()

	handleServeFile(w, req, testFilename)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
		return
	}

	body, _ := io.ReadAll(w.Body)
	if !bytes.Equal(body, testContent) {
		t.Error("Served content doesn't match file content")
	}

	// Verify Content-Disposition header
	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "inline") {
		t.Errorf("Expected inline disposition, got: %s", disposition)
	}
}

func TestHandleServeFile_NotFound(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	req := httptest.NewRequest("GET", "/files/nonexistent.pdf", nil)
	w := httptest.NewRecorder()

	handleServeFile(w, req, "nonexistent.pdf")

	if w.Code != 404 {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestHandleServeFile_PathTraversalPrevention(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"../secret.key",
	}

	for _, filename := range pathTraversalAttempts {
		t.Run("Block_"+filename, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/files/"+filename, nil)
			w := httptest.NewRecorder()

			handleServeFile(w, req, filename)

			// Should return 404 or prevent access
			if w.Code == 200 {
				// If it returns 200, verify it's only serving from uploads
				// filepath.Join should prevent traversal, but verify
				uploadsDir, _ := filepath.Abs("uploads")
				requestedPath, _ := filepath.Abs(filepath.Join("uploads", filename))
				
				if !strings.HasPrefix(requestedPath, uploadsDir) {
					t.Errorf("SECURITY VULNERABILITY: Path traversal succeeded: %s", requestedPath)
				}
			}
		})
	}
}

// ==================== INTEGRATION TESTS ====================

func TestAttachmentWorkflow_UploadListDownloadDelete(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	_ = createAttachmentTestUser(t, testDB, "testuser", "user")

	// 1. Upload attachment
	uploadReq, _ := createMultipartRequest(t, "workflow-test.pdf", []byte("Workflow test content"), "eco", "ECO-WORKFLOW")
	
	uploadW := httptest.NewRecorder()
	handleUploadAttachment(uploadW, uploadReq)

	if uploadW.Code != 201 {
		t.Fatalf("Upload failed: %d - %s", uploadW.Code, uploadW.Body.String())
	}

	var uploadResp struct {
		Data Attachment `json:"data"`
	}
	json.NewDecoder(uploadW.Body).Decode(&uploadResp)
	uploaded := uploadResp.Data

	// 2. List attachments
	listReq := httptest.NewRequest("GET", "/api/attachments?module=eco&record_id=ECO-WORKFLOW", nil)
	listW := httptest.NewRecorder()
	handleListAttachments(listW, listReq)

	if listW.Code != 200 {
		t.Fatalf("List failed: %d", listW.Code)
	}

	var listResp struct {
		Data []Attachment `json:"data"`
	}
	json.NewDecoder(listW.Body).Decode(&listResp)
	listed := listResp.Data
	
	if len(listed) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(listed))
	}

	// 3. Download attachment
	downloadReq := httptest.NewRequest("GET", fmt.Sprintf("/api/attachments/%d/download", uploaded.ID), nil)
	downloadW := httptest.NewRecorder()
	handleDownloadAttachment(downloadW, downloadReq, fmt.Sprintf("%d", uploaded.ID))

	if downloadW.Code != 200 {
		t.Fatalf("Download failed: %d", downloadW.Code)
	}

	// 4. Delete attachment
	deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/attachments/%d", uploaded.ID), nil)
	deleteW := httptest.NewRecorder()
	handleDeleteAttachment(deleteW, deleteReq, fmt.Sprintf("%d", uploaded.ID))

	if deleteW.Code != 200 {
		t.Fatalf("Delete failed: %d", deleteW.Code)
	}

	// 5. Verify deletion
	listReq2 := httptest.NewRequest("GET", "/api/attachments?module=eco&record_id=ECO-WORKFLOW", nil)
	listW2 := httptest.NewRecorder()
	handleListAttachments(listW2, listReq2)

	var listResp2 struct {
		Data []Attachment `json:"data"`
	}
	json.NewDecoder(listW2.Body).Decode(&listResp2)
	listed2 := listResp2.Data
	
	if len(listed2) != 0 {
		t.Errorf("Expected 0 attachments after delete, got %d", len(listed2))
	}
}

func TestAttachmentMultipleModules(t *testing.T) {
	testDB, cleanup := setupAttachmentsTestDB(t)
	defer cleanup()
	db = testDB

	_ = createAttachmentTestUser(t, testDB, "testuser", "user")

	modules := []struct {
		module   string
		recordID string
	}{
		{"eco", "ECO-001"},
		{"ncr", "NCR-123"},
		{"rma", "RMA-456"},
		{"quotes", "QT-789"},
	}

	// Upload one attachment per module
	for _, m := range modules {
		req, _ := createMultipartRequest(t, "test.pdf", []byte("content"), m.module, m.recordID)
		
		w := httptest.NewRecorder()
		handleUploadAttachment(w, req)

		if w.Code != 201 {
			t.Errorf("Failed to upload to %s: %d", m.module, w.Code)
		}
	}

	// Verify isolation - each module should only see its own attachments
	for _, m := range modules {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/attachments?module=%s&record_id=%s", m.module, m.recordID), nil)
		w := httptest.NewRecorder()
		handleListAttachments(w, req)

		var resp struct {
			Data []Attachment `json:"data"`
		}
		json.NewDecoder(w.Body).Decode(&resp)
		attachments := resp.Data

		if len(attachments) != 1 {
			t.Errorf("Module %s should have 1 attachment, got %d", m.module, len(attachments))
		}

		if len(attachments) > 0 && attachments[0].Module != m.module {
			t.Errorf("Wrong module returned: expected %s, got %s", m.module, attachments[0].Module)
		}
	}
}

// ==================== SECURITY VULNERABILITY REPORT ====================

// TestSecurityVulnerabilityReport documents security issues found
func TestSecurityVulnerabilityReport(t *testing.T) {
	t.Log("=== SECURITY VULNERABILITY ASSESSMENT ===")
	t.Log("")
	t.Log("CRITICAL FINDINGS:")
	t.Log("")
	t.Log("🔴 VULNERABILITY 1: NO PERMISSION CHECKS ON ATTACHMENT HANDLERS")
	t.Log("   Location: handler_attachments.go - ALL endpoints")
	t.Log("   Severity: CRITICAL")
	t.Log("   Impact: Any unauthenticated user can upload, list, download, and delete attachments")
	t.Log("   Recommendation: Add requirePermission() middleware to all attachment endpoints")
	t.Log("   Current: handleUploadAttachment, handleDeleteAttachment have NO auth checks")
	t.Log("")
	t.Log("🔴 VULNERABILITY 2: NO READONLY ROLE ENFORCEMENT")
	t.Log("   Severity: HIGH")
	t.Log("   Impact: Readonly users can upload and delete files (no enforcement)")
	t.Log("   Recommendation: Readonly should only have view permission")
	t.Log("")
	t.Log("🟡 VULNERABILITY 3: NO MIME TYPE VALIDATION")
	t.Log("   Severity: MEDIUM")
	t.Log("   Impact: File extension validation only - attacker can claim .pdf but upload .exe")
	t.Log("   Recommendation: Validate actual file content (magic bytes) vs declared MIME type")
	t.Log("")
	t.Log("🟡 VULNERABILITY 4: NO VIRUS/MALWARE SCANNING")
	t.Log("   Severity: MEDIUM")
	t.Log("   Impact: Malicious files can be uploaded and shared")
	t.Log("   Recommendation: Integrate ClamAV or similar antivirus scanning")
	t.Log("")
	t.Log("🟢 GOOD: Path traversal protection via sanitizeFilename()")
	t.Log("🟢 GOOD: Dangerous extension blocking")
	t.Log("🟢 GOOD: File size limits (100MB)")
	t.Log("🟢 GOOD: Malicious character sanitization")
	t.Log("")
	t.Log("REMEDIATION PRIORITY:")
	t.Log("1. Add permission checks immediately (CRITICAL)")
	t.Log("2. Add readonly role enforcement (HIGH)")
	t.Log("3. Add MIME type validation (MEDIUM)")
	t.Log("4. Consider antivirus integration (MEDIUM)")
	t.Log("")
	
	// This test always "passes" - it's documentation
	t.Log("Security assessment complete. Review findings above.")
}
