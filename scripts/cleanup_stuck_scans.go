//go:build ignore

// Cleanup utility for stuck discovery scans
// Usage: go run cleanup_stuck_scans.go [update|delete|show]
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run cleanup_stuck_scans.go [update|delete|show]")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  update  - Update stuck scans to 'failed' status")
		fmt.Println("  delete  - Delete stuck scans from database")
		fmt.Println("  show    - Show details of stuck scans")
		os.Exit(1)
	}

	action := os.Args[1]
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}
	dbPath := fmt.Sprintf("%s/rackd.db", dataDir)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("Database not found at: %s\nSet DATA_DIR environment variable if database is elsewhere", dbPath)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// List stuck scans first
	scans, err := listStuckScans(db)
	if err != nil {
		log.Fatal(err)
	}

	if len(scans) == 0 {
		fmt.Println("No stuck scans found (status = 'running' or 'pending')")
		os.Exit(0)
	}

	fmt.Printf("Found %d stuck scan(s):\n\n", len(scans))
	printScans(scans)

	if action == "show" {
		os.Exit(0)
	}

	fmt.Printf("\nAre you sure you want to %s these scans? [y/N]: ", action)
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("Cancelled")
		os.Exit(0)
	}

	switch action {
	case "update":
		err = updateStuckScans(db)
	case "delete":
		err = deleteStuckScans(db)
	default:
		fmt.Printf("Unknown action: %s\n", action)
		os.Exit(1)
	}

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nDone!")
}

type Scan struct {
	ID           string
	NetworkID    string
	Status       string
	ScanType     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	TotalHosts   int
	ScannedHosts int
	FoundHosts   int
}

func listStuckScans(db *sql.DB) ([]Scan, error) {
	rows, err := db.Query(`
		SELECT id, network_id, status, scan_type, created_at, updated_at, total_hosts, scanned_hosts, found_hosts
		FROM discovery_scans
		WHERE status IN ('running', 'pending')
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []Scan
	for rows.Next() {
		var s Scan
		err := rows.Scan(&s.ID, &s.NetworkID, &s.Status, &s.ScanType, &s.CreatedAt, &s.UpdatedAt, &s.TotalHosts, &s.ScannedHosts, &s.FoundHosts)
		if err != nil {
			return nil, err
		}
		scans = append(scans, s)
	}
	return scans, nil
}

func updateStuckScans(db *sql.DB) error {
	result, err := db.Exec(`
		UPDATE discovery_scans
		SET status = 'failed',
		    error_message = 'scan cancelled - cleanup',
		    completed_at = datetime('now'),
		    updated_at = datetime('now')
		WHERE status IN ('running', 'pending')
	`)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	fmt.Printf("\nUpdated %d scan(s) to 'failed' status\n", affected)
	return nil
}

func deleteStuckScans(db *sql.DB) error {
	result, err := db.Exec(`
		DELETE FROM discovery_scans
		WHERE status IN ('running', 'pending')
	`)
	if err != nil {
		return err
	}

	affected, _ := result.RowsAffected()
	fmt.Printf("\nDeleted %d scan(s)\n", affected)
	return nil
}

func printScans(scans []Scan) {
	fmt.Printf("%-36s %-12s %-12s %-12s %-12s %-12s\n", "ID", "Status", "Type", "Total", "Scanned", "Found")
	fmt.Println("----------------------------------------------------------------------------------------------------")
	for _, s := range scans {
		fmt.Printf("%-36s %-12s %-12s %-12d %-12d %-12d\n", s.ID, s.Status, s.ScanType, s.TotalHosts, s.ScannedHosts, s.FoundHosts)
	}
}
