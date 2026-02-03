#!/bin/bash
# Load test data into Rackd
# Usage: ./load-testdata.sh [cli|api]

set -e

MODE=${1:-cli}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RACKD_BIN="${RACKD_BIN:-./build/rackd}"
API_URL="${RACKD_API_URL:-http://localhost:8080}"

# Check if jq is available
if ! command -v jq &> /dev/null; then
    echo "❌ jq is required but not installed"
    exit 1
fi

echo "🚀 Loading Rackd test data (mode: $MODE)"
echo "================================================"

if [ "$MODE" = "cli" ]; then
    # CLI mode - use rackd import commands
    echo "📍 Importing datacenters..."
    $RACKD_BIN import datacenters --file "$SCRIPT_DIR/datacenters.json"
    
    echo ""
    echo "🌐 Importing networks..."
    $RACKD_BIN import networks --file "$SCRIPT_DIR/networks.json"
    
    echo ""
    echo "🏊 Importing network pools..."
    while IFS= read -r pool; do
        network_id=$(echo "$pool" | jq -r '.network_id')
        curl -s -X POST "$API_URL/api/networks/$network_id/pools" \
            -H "Content-Type: application/json" \
            -d "$pool" > /dev/null
    done < <(jq -c '.[]' "$SCRIPT_DIR/pools.json")
    echo "  ✓ Imported $(jq length "$SCRIPT_DIR/pools.json") pools"
    
    echo ""
    echo "💻 Importing devices..."
    $RACKD_BIN import devices --file "$SCRIPT_DIR/devices.json"
    
    echo ""
    echo "🔗 Importing relationships..."
    while IFS= read -r rel; do
        parent_id=$(echo "$rel" | jq -r '.parent_id')
        curl -s -X POST "$API_URL/api/devices/$parent_id/relationships" \
            -H "Content-Type: application/json" \
            -d "$rel" > /dev/null
    done < <(jq -c '.[]' "$SCRIPT_DIR/relationships.json")
    echo "  ✓ Imported $(jq length "$SCRIPT_DIR/relationships.json") relationships"
    
elif [ "$MODE" = "api" ]; then
    # API mode - use bulk endpoints
    echo "📍 Importing datacenters..."
    for dc in $(jq -c '.[]' "$SCRIPT_DIR/datacenters.json"); do
        curl -s -X POST "$API_URL/api/datacenters" \
            -H "Content-Type: application/json" \
            -d "$dc" | jq -r '"  ✓ \(.name)"'
    done
    
    echo ""
    echo "🌐 Importing networks (bulk)..."
    curl -s -X POST "$API_URL/api/networks/bulk" \
        -H "Content-Type: application/json" \
        -d @"$SCRIPT_DIR/networks.json" | jq -r '"  Total: \(.total), Success: \(.success), Failed: \(.failed)"'
    
    echo ""
    echo "🏊 Importing network pools..."
    while IFS= read -r pool; do
        network_id=$(echo "$pool" | jq -r '.network_id')
        curl -s -X POST "$API_URL/api/networks/$network_id/pools" \
            -H "Content-Type: application/json" \
            -d "$pool" > /dev/null
    done < <(jq -c '.[]' "$SCRIPT_DIR/pools.json")
    echo "  ✓ Imported $(jq length "$SCRIPT_DIR/pools.json") pools"
    
    echo ""
    echo "💻 Importing devices (bulk)..."
    curl -s -X POST "$API_URL/api/devices/bulk" \
        -H "Content-Type: application/json" \
        -d @"$SCRIPT_DIR/devices.json" | jq -r '"  Total: \(.total), Success: \(.success), Failed: \(.failed)"'
    
    echo ""
    echo "🔗 Importing relationships..."
    while IFS= read -r rel; do
        parent_id=$(echo "$rel" | jq -r '.parent_id')
        curl -s -X POST "$API_URL/api/devices/$parent_id/relationships" \
            -H "Content-Type: application/json" \
            -d "$rel" > /dev/null
    done < <(jq -c '.[]' "$SCRIPT_DIR/relationships.json")
    echo "  ✓ Imported $(jq length "$SCRIPT_DIR/relationships.json") relationships"
    
else
    echo "❌ Invalid mode: $MODE (use 'cli' or 'api')"
    exit 1
fi

echo ""
echo "================================================"
echo "✅ Test data loaded successfully!"
echo ""
echo "📊 Summary:"
echo "  - 3 datacenters (US-East, EU-West, APAC-SG)"
echo "  - 9 networks (3 per datacenter: Prod, DMZ, Mgmt)"
echo "  - 24 devices across all regions"
echo "  - 6 network pools (IP address pools)"
echo "  - 20 device relationships (dependencies, HA pairs, replication)"
echo ""
echo "🔍 Try these commands:"
echo "  $RACKD_BIN device list"
echo "  $RACKD_BIN device list --tags firewall"
echo "  $RACKD_BIN network list"
echo "  curl $API_URL/api/devices/us-web-01/relationships | jq"
echo "  curl $API_URL/api/networks/net-us-prod/pools | jq"
