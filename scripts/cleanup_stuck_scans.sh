#!/bin/bash
# Cleanup stuck discovery scans from rackd database

DB_PATH="${DATA_DIR:-./data}/rackd.db"

if [ ! -f "$DB_PATH" ]; then
    echo "Database not found at: $DB_PATH"
    echo "Set DATA_DIR environment variable if database is elsewhere"
    exit 1
fi

echo "Database: $DB_PATH"
echo ""
echo "Listing stuck scans (status = 'running' or 'pending'):"
echo ""

sqlite3 "$DB_PATH" <<'EOF'
.mode column
.headers on
SELECT id, network_id, status, scan_type, created_at, updated_at 
FROM discovery_scans 
WHERE status IN ('running', 'pending')
ORDER BY created_at DESC;
EOF

echo ""
echo "Choose an option:"
echo "  1) Update stuck scans to 'failed'"
echo "  2) Delete stuck scans"
echo "  3) Just show details (no action)"
read -p "Enter choice [1-3]: " choice

case $choice in
    1)
        echo ""
        echo "Updating stuck scans to 'failed'..."
        sqlite3 "$DB_PATH" <<'EOF'
UPDATE discovery_scans 
SET status = 'failed', 
    error_message = 'scan cancelled - cleanup',
    completed_at = datetime('now'),
    updated_at = datetime('now')
WHERE status IN ('running', 'pending');
EOF
        affected=$(sqlite3 "$DB_PATH" "SELECT changes();")
        echo "Updated $affected scan(s)"
        ;;
    2)
        echo ""
        read -p "Are you sure you want to DELETE stuck scans? [y/N]: " confirm
        if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
            sqlite3 "$DB_PATH" <<'EOF'
DELETE FROM discovery_scans 
WHERE status IN ('running', 'pending');
EOF
            affected=$(sqlite3 "$DB_PATH" "SELECT changes();")
            echo "Deleted $affected scan(s)"
        else
            echo "Cancelled"
        fi
        ;;
    3)
        echo ""
        echo "Showing full details of stuck scans:"
        sqlite3 "$DB_PATH" <<'EOF'
.mode column
.headers on
SELECT * 
FROM discovery_scans 
WHERE status IN ('running', 'pending')
ORDER BY created_at DESC;
EOF
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac
