#!/bin/bash
# Load test data into Rackd
# Usage: ./load-testdata.sh [options]
#
# Options:
#   --url URL          API base URL (default: http://localhost:8080)
#   --user USERNAME    Username for login (default: admin)
#   --pass PASSWORD    Password for login (default: admin)
#   --token TOKEN      Use a Bearer API token instead of username/password
#   --mode cli|api     Import mode for core data (default: api)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_URL="http://localhost:8080"
USERNAME="admin"
PASSWORD="admin"
TOKEN=""
MODE="api"
RACKD_BIN="${RACKD_BIN:-./build/rackd}"

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --url)   API_URL="$2";   shift 2 ;;
        --user)  USERNAME="$2";  shift 2 ;;
        --pass)  PASSWORD="$2";  shift 2 ;;
        --token) TOKEN="$2";     shift 2 ;;
        --mode)  MODE="$2";      shift 2 ;;
        cli|api) MODE="$1";      shift   ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if ! command -v jq &> /dev/null; then
    echo "❌ jq is required but not installed"
    exit 1
fi

COOKIE_JAR=$(mktemp)
trap "rm -f $COOKIE_JAR" EXIT

# ── Authentication ────────────────────────────────────────────────────────────

if [ -n "$TOKEN" ]; then
    echo "� Using provided Bearer token"
    # Verify it works
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "$API_URL/api/auth/me")
    if [ "$STATUS" != "200" ]; then
        echo "❌ Token authentication failed (HTTP $STATUS)"
        exit 1
    fi
    AUTH_ARGS=(-H "Authorization: Bearer $TOKEN")
    WRITE_ARGS=(-H "Authorization: Bearer $TOKEN")
else
    echo "🔐 Logging in as '$USERNAME'..."
    LOGIN_RESP=$(curl -s -c "$COOKIE_JAR" -X POST "$API_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

    if echo "$LOGIN_RESP" | jq -e '.error' > /dev/null 2>&1; then
        echo "❌ Login failed: $(echo "$LOGIN_RESP" | jq -r '.error')"
        echo "   Try: --user <username> --pass <password>"
        echo "   Or:  --token <api-token>"
        exit 1
    fi

    echo "  ✓ Logged in as $(echo "$LOGIN_RESP" | jq -r '.user.username')"
    # Session cookie auth - GET requests just need the cookie
    # POST/PUT/DELETE also need X-Requested-With for CSRF
    AUTH_ARGS=(-b "$COOKIE_JAR")
    WRITE_ARGS=(-b "$COOKIE_JAR" -H "X-Requested-With: XMLHttpRequest")
fi

# ── Helper functions ──────────────────────────────────────────────────────────

# POST a single JSON object, suppress output
api_post() {
    local endpoint="$1"
    local data="$2"
    curl -s -X POST "$API_URL$endpoint" \
        "${WRITE_ARGS[@]}" \
        -H "Content-Type: application/json" \
        -d "$data"
}

# POST a file
api_post_file() {
    local endpoint="$1"
    local file="$2"
    curl -s -X POST "$API_URL$endpoint" \
        "${WRITE_ARGS[@]}" \
        -H "Content-Type: application/json" \
        -d @"$file"
}

# Import each item from a JSON array, print count
import_array() {
    local label="$1"
    local endpoint="$2"
    local file="$3"
    local count
    count=$(jq length "$file")
    local ok=0
    local fail=0
    while IFS= read -r item; do
        resp=$(api_post "$endpoint" "$item")
        if echo "$resp" | jq -e '.error' > /dev/null 2>&1; then
            fail=$((fail + 1))
            echo "    ⚠ $(echo "$resp" | jq -r '.error // .message // "unknown error"') — $(echo "$item" | jq -r '.name // .id // ""')"
        else
            ok=$((ok + 1))
        fi
    done < <(jq -c '.[]' "$file")
    echo "  ✓ $label: $ok/$count imported$([ $fail -gt 0 ] && echo " ($fail failed)" || true)"
}

echo ""
echo "� Loading Rackd test data"
echo "   URL:  $API_URL"
echo "   Mode: $MODE"
echo "================================================"

# ── Core infrastructure ───────────────────────────────────────────────────────

echo ""
echo "� Datacenters..."
while IFS= read -r dc; do
    resp=$(api_post "/api/datacenters" "$dc")
    echo "  ✓ $(echo "$resp" | jq -r '.name // .error')"
done < <(jq -c '.[]' "$SCRIPT_DIR/datacenters.json")

echo ""
echo "🌐 Networks (bulk)..."
api_post_file "/api/networks/bulk" "$SCRIPT_DIR/networks.json" \
    | jq -r '"  ✓ Total: \(.total), Success: \(.success), Failed: \(.failed)"'

echo ""
echo "🏊 Network pools..."
while IFS= read -r pool; do
    network_id=$(echo "$pool" | jq -r '.network_id')
    resp=$(api_post "/api/networks/$network_id/pools" "$pool")
    if echo "$resp" | jq -e '.error' > /dev/null 2>&1; then
        echo "    ⚠ pool $(echo "$pool" | jq -r '.name'): $(echo "$resp" | jq -r '.error')"
    fi
done < <(jq -c '.[]' "$SCRIPT_DIR/pools.json")
echo "  ✓ $(jq length "$SCRIPT_DIR/pools.json") pools imported"

echo ""
echo "� Devices (bulk)..."
api_post_file "/api/devices/bulk" "$SCRIPT_DIR/devices.json" \
    | jq -r '"  ✓ Total: \(.total), Success: \(.success), Failed: \(.failed)"'

echo ""
echo "💻 Extra devices (varied statuses)..."
api_post_file "/api/devices/bulk" "$SCRIPT_DIR/devices_extra.json" \
    | jq -r '"  ✓ Total: \(.total), Success: \(.success), Failed: \(.failed)"'

echo ""
echo "� Relationships..."
ok=0; fail=0
while IFS= read -r rel; do
    parent_id=$(echo "$rel" | jq -r '.parent_id')
    resp=$(api_post "/api/devices/$parent_id/relationships" "$rel")
    if echo "$resp" | jq -e '.error' > /dev/null 2>&1; then
        fail=$((fail + 1))
    else
        ok=$((ok + 1))
    fi
done < <(jq -c '.[]' "$SCRIPT_DIR/relationships.json")
echo "  ✓ $(jq length "$SCRIPT_DIR/relationships.json") relationships: $ok ok, $fail failed"

# ── Network services ──────────────────────────────────────────────────────────

echo ""
echo "� Circuits..."
import_array "circuits" "/api/circuits" "$SCRIPT_DIR/circuits.json"

echo ""
echo "🔀 NAT mappings..."
import_array "NAT mappings" "/api/nat" "$SCRIPT_DIR/nat.json"

echo ""
echo "� IP reservations..."
import_array "reservations" "/api/reservations" "$SCRIPT_DIR/reservations.json"

# ── Configuration ─────────────────────────────────────────────────────────────

echo ""
echo "🔧 Custom field definitions..."
import_array "custom fields" "/api/custom-fields" "$SCRIPT_DIR/custom_fields.json"

echo ""
echo "� Credentials..."
import_array "credentials" "/api/credentials" "$SCRIPT_DIR/credentials.json"

echo ""
echo "🔔 Webhooks..."
import_array "webhooks" "/api/webhooks" "$SCRIPT_DIR/webhooks.json"

# ── Discovery ─────────────────────────────────────────────────────────────────

echo ""
echo "� Scan profiles..."
import_array "scan profiles" "/api/scan-profiles" "$SCRIPT_DIR/scan_profiles.json"

echo ""
echo "📅 Scheduled scans..."
import_array "scheduled scans" "/api/scheduled-scans" "$SCRIPT_DIR/scheduled_scans.json"

echo ""
echo "🔎 Discovery rules..."
import_array "discovery rules" "/api/discovery/rules" "$SCRIPT_DIR/discovery_rules.json"

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "================================================"
echo "✅ Done!"
echo ""
echo "🔍 Quick checks:"
echo "  curl -b $COOKIE_JAR $API_URL/api/circuits | jq length"
echo "  curl -b $COOKIE_JAR $API_URL/api/nat | jq length"
echo "  curl -b $COOKIE_JAR $API_URL/api/custom-fields | jq length"
echo "  curl -b $COOKIE_JAR $API_URL/api/credentials | jq length"
echo "  curl -b $COOKIE_JAR $API_URL/api/webhooks | jq length"
