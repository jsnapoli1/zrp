package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"mime/multipart"
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
			
			part, err := writer.CreateFormFile("file", "test.pdf")
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
	testDB, cleanup := setupFileUploadTestDB(t)
	defer cleanup()
	db = testDB

	dangerousExtensions := []struct {
		filename    string
		contentType string
		reason      string
	}{
		{"malicious.exe", "application/x-msdownload", "Windows executable"},
		{"script.bat", "application/x-bat", "Windows batch script"},
		{"shell.sh", "application/x-sh", "Shell script"},
		{"command.cmd", "application/x-msdos-program", "Windows command script"},
		{"program.com", "application/x-msdownload", "DOS executable"},
		{"saver.scr", "application/x-msdownload", "Windows screensaver"},
		{"script.vbs", "application/x-vbscript", "Visual Basic script"},
		{"code.js", "application/javascript", "JavaScript (could be malicious)"},
		{"archive.jar", "application/java-archive", "Java executable"},
		{"program.app", "application/x-app", "macOS application"},
	}

	for _, ext := range dangerousExtensions {
		t.Run("Reject "+ext.filename, func(t *testing.T) {
			fileContent := []byte("This could be dangerous")

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			part, err := writer.CreateFormFile("file", ext.filename)
			if err != nil {
				t.Fatal(err)
			}
			part.Write(fileContent)

			writer.WriteField("module", "test")
			writer.WriteField("record_id", "TEST001")
			writer.Close()

			req := httptest.NewRequest("POST", "/api/attachments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if w.Code == 201 || w.Code == 200 {
				t.Errorf("Dangerous file %s should have been rejected. Reason: %s", ext.filename, ext.reason)
			} else if w.Code != 400 {
				t.Logf("File rejected with status %d (expected 400 Bad Request)", w.Code)
			}
		})
	}
}

// TestPathTraversalInFilenames tests that path traversal attempts are blocked
func TestPathTraversalInFilenames(t *testing.T) {
	testDB, cleanup := setupFileUploadTestDB(t)
	defer cleanup()
	db = testDB

	maliciousFilenames := []string{
		"../../etc/passwd",
		"..\\..\\windows\\system32\\config\\sam",
		"../../../root/.ssh/id_rsa",
		"test/../../../etc/shadow",
		"test/../../secret.key",
		"/etc/passwd",
		"C:\\Windows\\System32\\config\\SAM",
	}

	for _, filename := range maliciousFilenames {
		t.Run("Block "+filename, func(t *testing.T) {
			fileContent := []byte("malicious content")

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			
			// Use .txt extension to avoid extension rejection
			testFilename := filename
			if !strings.Contains(filename, ".") {
				testFilename = filename + ".txt"
			}
			
			part, err := writer.CreateFormFile("file", testFilename)
			if err != nil {
				t.Fatal(err)
			}
			part.Write(fileContent)

			writer.WriteField("module", "test")
			writer.WriteField("record_id", "TEST001")
			writer.Close()

			req := httptest.NewRequest("POST", "/api/attachments/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Should be rejected due to path traversal
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

// TestCSVImportSizeLimits tests file upload limits for CSV import endpoints
func TestCSVImportSizeLimits(t *testing.T) {
	testDB, cleanup := setupFileUploadTestDB(t)
	defer cleanup()
	db = testDB

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
			name:         "Large CSV (10000 rows)",
			rowCount:     10000,
			shouldSucceed: true,
			description:  "Large CSV should import successfully (under 50MB)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create CSV content
			csvBuffer := &bytes.Buffer{}
			csvWriter := csv.NewWriter(csvBuffer)
			
			// Write header
			csvWriter.Write([]string{"serial_number", "ipn", "firmware_version", 
				"customer", "location", "status", "install_date", "notes"})
			
			// Write data rows
			for i := 0; i < tt.rowCount; i++ {
				sn := "SN" + strings.Repeat("X", i%10)
				csvWriter.Write([]string{
					sn + string(rune('0'+(i%10))),
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

			w := httptest.NewRecorder()
			handleImportDevices(w, req)

			if tt.shouldSucceed {
				if w.Code >= 400 {
					t.Errorf("%s: import failed with status %d - %s", tt.name, w.Code, w.Body.String())
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

// TestMaliciousFilenameChars tests special characters in filenames
func TestMaliciousFilenameChars(t *testing.T) {
	testDB, cleanup := setupFileUploadTestDB(t)
	defer cleanup()
	db = testDB

	maliciousNames := []string{
		"file;rm -rf.txt",
		"file|ls.txt",
		"file&whoami&.txt",
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

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			// Check if special chars were sanitized
			if w.Code == 201 {
				// Verify no command injection occurred
				files, _ := filepath.Glob("uploads/*")
				for _, file := range files {
					if strings.ContainsAny(filepath.Base(file), ";|&") {
						t.Errorf("Special characters not sanitized in filename: %s", file)
					}
				}
			}
		})
	}
}

// TestSafeFileExtensions tests that allowed file types work correctly
func TestSafeFileExtensions(t *testing.T) {
	testDB, cleanup := setupFileUploadTestDB(t)
	defer cleanup()
	db = testDB

	safeFiles := []string{
		"document.pdf",
		"spreadsheet.xlsx",
		"image.png",
		"archive.zip",
		"notes.txt",
		"data.csv",
		"drawing.dxf",
	}

	for _, filename := range safeFiles {
		t.Run("Allow "+filename, func(t *testing.T) {
			fileContent := []byte("safe content")

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

			w := httptest.NewRecorder()
			handleUploadAttachment(w, req)

			if w.Code != 201 {
				t.Errorf("Safe file %s should have been accepted, got status %d - %s", 
					filename, w.Code, w.Body.String())
			}
		})
	}
}
