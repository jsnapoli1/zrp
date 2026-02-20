package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
	_ "modernc.org/sqlite"
)

func TestDevicesDebug(t *testing.T) {
	testDB, _ := sql.Open("sqlite", ":memory:")
	defer testDB.Close()

	// Create devices table
	_, err := testDB.Exec(`
		CREATE TABLE devices (
			serial_number TEXT PRIMARY KEY,
			ipn TEXT NOT NULL,
			firmware_version TEXT,
			customer TEXT,
			location TEXT,
			status TEXT DEFAULT 'active',
			install_date TEXT,
			last_seen DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create devices table: %v", err)
	}

	// Insert a device
	_, err = testDB.Exec(`INSERT INTO devices (serial_number, ipn, customer, status, created_at) 
		VALUES (?, ?, ?, ?, ?)`,
		"SN-001", "DEV-100", "Acme Corp", "active", time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test device: %v", err)
	}

	// Query devices
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM devices").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count devices: %v", err)
	}
	fmt.Printf("Device count: %d\n", count)

	rows, err := testDB.Query("SELECT serial_number, ipn, customer FROM devices")
	if err != nil {
		t.Fatalf("Failed to query devices: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sn, ipn, customer string
		rows.Scan(&sn, &ipn, &customer)
		fmt.Printf("Device: %s, %s, %s\n", sn, ipn, customer)
	}
}
