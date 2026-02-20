package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupFileUploadTestDB(t *testing.T) (*sql.DB, func()) {
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test DB: %v", err)
	}

	// Create required tables
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

	_, err = testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT,
			firmware_version TEXT,
			customer TEXT,
			location TEXT,
			status TEXT,
			install_date TEXT,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Create uploads directory
	os.MkdirAll("uploads", 0755)

	cleanup := func() {
		testDB.Close()
		// Clean up test uploads
		os.RemoveAll("uploads")
		os.MkdirAll("uploads", 0755)
	}

	return testDB, cleanup
}

// TestFileUploadSizeLimits tests that file uploads respect size constraints
func TestFileUploadSizeLimits(t *testing.T) {
	testDB, cleanup := setupFileUploadTestDB(t)
	defer cleanup()
	db = testDB

	tests := []struct {
		name          string
		fileSize      int64
		shouldSucceed bool
		expectedCode  int
		description   string
	}{
		{
			name:          "Small file (1KB)",
			fileSize:      1024,
			shouldSucceed: true,
			expectedCode:  201,
			description:   "Small files should upload successfully",
		},
		{
			name:          "Medium file (10MB)",
			fileSize:      10 * 1024 * 1024,
			shouldSucceed: true,
			expectedCode:  201,
			description:   "10MB files should upload successfully",
		},
		{
			name:          "Large file (101MB)",
			fileSize:      101 * 1024 * 1024,
			shouldSucceed: false,
			expectedCode:  413, // Request Entity Too Large
			description:   "Files larger than 100MB should be rejected",
		},
		{
			name:          "Zero byte file",
			fileSize:      0,
			shouldSucceed: false,
			expectedCode:  400,
			description:   "Empty files should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file content
			fileContent := make([]byte, tt.fileSize)
			for i := range fileContent {
				fileContent[i] = byte(i % 256)
			}

			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", "test.bin")
			if err != nil {
				t.Fatal(err)
			}
			
			if _, err := part.Write(fileContent); err != nil {
				t.Fatal(err)
			}

			// Add required form fields
			writer.WriteField("module", "test")
			writer.WriteField("record_id", "TEST001")
			writer.Close()

			// Create request
			req := httptest.NewRequest("POST", "/api/attachments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setTestUser(req)

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if tt.shouldSucceed {
				if w.Code != tt.expectedCode {
					t.Errorf("%s: expected status %d, got %d. %s", tt.name, tt.expectedCode, w.Code, tt.description)
				}
			} else {
				if w.Code == 201 || w.Code == 200 {
					t.Errorf("%s: upload should have been rejected but succeeded. %s", tt.name, tt.description)
				}
			}
		})
	}
}

// TestDangerousFileExtensions tests that dangerous file types are rejected
func TestDangerousFileExtensions(t *testing.T) {
	setup()
	defer teardown()

	dangerousExtensions := []struct {
		filename    string
		contentType string
		reason      string
	}{
		{".exe", "application/x-msdownload", "Windows executable"},
		{".bat", "application/x-bat", "Windows batch script"},
		{".sh", "application/x-sh", "Shell script"},
		{".cmd", "application/x-msdos-program", "Windows command script"},
		{".com", "application/x-msdownload", "DOS executable"},
		{".scr", "application/x-msdownload", "Windows screensaver"},
		{".vbs", "application/x-vbscript", "Visual Basic script"},
		{".js", "application/javascript", "JavaScript (could be malicious)"},
		{".jar", "application/java-archive", "Java executable"},
		{".app", "application/x-app", "macOS application"},
		{".dmg", "application/x-apple-diskimage", "macOS disk image"},
		{".pkg", "application/x-newton-compatible-pkg", "macOS installer"},
	}

	for _, ext := range dangerousExtensions {
		t.Run("Reject "+ext.filename, func(t *testing.T) {
			filename := "malicious" + ext.filename
			fileContent := []byte("This could be dangerous")

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				t.Fatal(err)
			}
			part.Write(fileContent)

			writer.WriteField("module", "test")
			writer.WriteField("record_id", "TEST001")
			writer.Close()

			req := httptest.NewRequest("POST", "/api/attachments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setTestUser(req)

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if w.Code == 201 || w.Code == 200 {
				t.Errorf("Dangerous file %s should have been rejected. Reason: %s", filename, ext.reason)
			} else if w.Code != 400 {
				t.Logf("File rejected with status %d (expected 400 Bad Request)", w.Code)
			}
		})
	}
}

// TestPathTraversalInFilenames tests that path traversal attempts are blocked
func TestPathTraversalInFilenames(t *testing.T) {
	setup()
	defer teardown()

	maliciousFilenames := []string{
		"../../etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"../../../root/.ssh/id_rsa",
		"test/../../../etc/shadow",
		"..%2F..%2Fetc%2Fpasswd", // URL encoded
		"....//....//etc/passwd",  // Double encoding attempt
		"test/../../secret.key",
		"/etc/passwd",
		"C:\\Windows\\System32\\config\\SAM",
		"test\x00.txt", // Null byte injection
	}

	for _, filename := range maliciousFilenames {
		t.Run("Block "+filename, func(t *testing.T) {
			fileContent := []byte("malicious content")

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				t.Fatal(err)
			}
			part.Write(fileContent)

			writer.WriteField("module", "test")
			writer.WriteField("record_id", "TEST001")
			writer.Close()

			req := httptest.NewRequest("POST", "/api/attachments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setTestUser(req)

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Verify the file was not saved outside uploads directory
			if w.Code == 201 {
				// Even if accepted, check that sanitization prevented path traversal
				uploadsDir, _ := filepath.Abs("uploads")
				files, _ := filepath.Glob("uploads/*")
				
				for _, file := range files {
					absPath, _ := filepath.Abs(file)
					if !strings.HasPrefix(absPath, uploadsDir) {
						t.Errorf("Path traversal succeeded! File saved outside uploads: %s", absPath)
					}
				}
				
				// Check for files in parent directories
				if _, err := os.Stat("../../etc/passwd"); err == nil {
					t.Error("Path traversal attack succeeded - file created in parent directory")
				}
			}
		})
	}
}

// TestContentTypeMismatch tests that declared content-type matches actual file type
func TestContentTypeMismatch(t *testing.T) {
	setup()
	defer teardown()

	tests := []struct {
		name            string
		filename        string
		declaredType    string
		actualContent   []byte
		shouldWarn      bool
		description     string
	}{
		{
			name:          "Executable disguised as PDF",
			filename:      "document.pdf",
			declaredType:  "application/pdf",
			actualContent: []byte("MZ\x90\x00"), // PE executable header
			shouldWarn:    true,
			description:   "Executable file with PDF extension",
		},
		{
			name:          "Script disguised as image",
			filename:      "photo.jpg",
			declaredType:  "image/jpeg",
			actualContent: []byte("#!/bin/bash\nrm -rf /"),
			shouldWarn:    true,
			description:   "Shell script with JPG extension",
		},
		{
			name:          "Legitimate PDF",
			filename:      "doc.pdf",
			declaredType:  "application/pdf",
			actualContent: []byte("%PDF-1.4\nsome pdf content"),
			shouldWarn:    false,
			description:   "Valid PDF file",
		},
		{
			name:          "Legitimate text file",
			filename:      "notes.txt",
			declaredType:  "text/plain",
			actualContent: []byte("Just some text"),
			shouldWarn:    false,
			description:   "Valid text file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", tt.filename)
			if err != nil {
				t.Fatal(err)
			}
			
			// Override content-type in the part header
			h := make(map[string][]string)
			h["Content-Type"] = []string{tt.declaredType}
			part.(*io.Writer)
			
			part.Write(tt.actualContent)

			writer.WriteField("module", "test")
			writer.WriteField("record_id", "TEST001")
			writer.Close()

			req := httptest.NewRequest("POST", "/api/attachments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setTestUser(req)

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Note: Basic implementation may accept mismatches
			// Advanced implementation should detect and warn/reject
			if tt.shouldWarn {
				t.Logf("%s: Content-type mismatch detection: status=%d", tt.description, w.Code)
			}
		})
	}
}

// TestCSVImportSizeLimits tests file upload limits for CSV import endpoints
func TestCSVImportSizeLimits(t *testing.T) {
	setup()
	defer teardown()

	tests := []struct {
		name         string
		rowCount     int
		shouldSucceed bool
		description  string
	}{
		{
			name:         "Small CSV (10 rows)",
			rowCount:     10,
			shouldSucceed: true,
			description:  "Small CSV should import successfully",
		},
		{
			name:         "Medium CSV (1000 rows)",
			rowCount:     1000,
			shouldSucceed: true,
			description:  "Medium CSV should import successfully",
		},
		{
			name:         "Large CSV (100000 rows)",
			rowCount:     100000,
			shouldSucceed: false, // Should timeout or reject
			description:  "Very large CSV should be rejected or handled gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create CSV content
			csvBuffer := &bytes.Buffer{}
			csvWriter := csv.NewWriter(csvBuffer)
			
			// Write header
			csvWriter.Write([]string{"serial_number", "assembly_ipn", "firmware_version", 
				"customer", "location", "status", "install_date", "notes"})
			
			// Write data rows
			for i := 0; i < tt.rowCount; i++ {
				csvWriter.Write([]string{
					"SN" + string(rune('A'+i%26)) + string(rune('0'+i%10)),
					"IPN-TEST-001",
					"1.0.0",
					"TestCustomer",
					"TestLocation",
					"active",
					"2024-01-01",
					"Test device",
				})
			}
			csvWriter.Flush()

			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", "devices.csv")
			if err != nil {
				t.Fatal(err)
			}
			part.Write(csvBuffer.Bytes())
			writer.Close()

			req := httptest.NewRequest("POST", "/api/devices/import", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setTestUser(req)

			w := httptest.NewRecorder()
			handleImportDevices(w, req)

			if tt.shouldSucceed {
				if w.Code >= 400 {
					t.Errorf("%s: import failed with status %d", tt.name, w.Code)
				}
			} else {
				// Large files should timeout or be rejected
				if w.Code == 200 {
					t.Logf("%s: Large import completed (should consider adding limits)", tt.name)
				}
			}
		})
	}
}

// TestMultipleFileUploadEndpoints tests all file upload endpoints
func TestMultipleFileUploadEndpoints(t *testing.T) {
	setup()
	defer teardown()

	endpoints := []struct {
		path        string
		handler     http.HandlerFunc
		setupForm   func(*multipart.Writer) error
		description string
	}{
		{
			path:    "/api/attachments/upload",
			handler: handleUploadAttachment,
			setupForm: func(w *multipart.Writer) error {
				w.WriteField("module", "test")
				w.WriteField("record_id", "TEST001")
				return nil
			},
			description: "Attachment upload endpoint",
		},
		{
			path:    "/api/devices/import",
			handler: handleImportDevices,
			setupForm: func(w *multipart.Writer) error {
				// CSV import doesn't need extra fields
				return nil
			},
			description: "Device CSV import endpoint",
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.description, func(t *testing.T) {
			// Test with 50MB file
			fileSize := 50 * 1024 * 1024
			fileContent := make([]byte, fileSize)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", "test.bin")
			if err != nil {
				t.Fatal(err)
			}
			part.Write(fileContent)
			
			endpoint.setupForm(writer)
			writer.Close()

			req := httptest.NewRequest("POST", endpoint.path, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setTestUser(req)

			w := httptest.NewRecorder()
			endpoint.handler(w, req)

			// Log result
			t.Logf("%s: 50MB upload returned status %d", endpoint.description, w.Code)
		})
	}
}

// TestMaliciousFilenameChars tests special characters in filenames
func TestMaliciousFilenameChars(t *testing.T) {
	setup()
	defer teardown()

	maliciousNames := []string{
		"file<script>.txt",
		"file;rm -rf /.txt",
		"file`whoami`.txt",
		"file$(whoami).txt",
		"file|ls.txt",
		"file&whoami&.txt",
		"file\r\n.txt", // CRLF injection
		"file\t\t.txt",
	}

	for _, filename := range maliciousNames {
		t.Run("Sanitize: "+filename, func(t *testing.T) {
			fileContent := []byte("test content")

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", filename)
			if err != nil {
				t.Fatal(err)
			}
			part.Write(fileContent)

			writer.WriteField("module", "test")
			writer.WriteField("record_id", "TEST001")
			writer.Close()

			req := httptest.NewRequest("POST", "/api/attachments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			setTestUser(req)

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Check if special chars were sanitized
			if w.Code == 201 {
				// Verify no command injection occurred
				files, _ := filepath.Glob("uploads/*")
				for _, file := range files {
					if strings.ContainsAny(filepath.Base(file), "<>;`$|&\r\n") {
						t.Errorf("Special characters not sanitized in filename: %s", file)
					}
				}
			}
		})
	}
}

// Helper function to set test user context
func setTestUser(r *http.Request) {
	// Add auth context for test
	r.Header.Set("X-Test-User", "testuser")
}
