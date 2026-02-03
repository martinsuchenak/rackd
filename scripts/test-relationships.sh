#!/bin/bash
# Test script for relationship management UI

set -e

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api"

echo "=== Testing Relationship Management API ==="
echo

# Create test devices
echo "1. Creating test devices..."
DEVICE1=$(curl -s -X POST "$API_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server-01",
    "hostname": "web01.example.com",
    "make_model": "Dell PowerEdge R640",
    "os": "Ubuntu 22.04",
    "description": "Primary web server"
  }' | jq -r '.id')

DEVICE2=$(curl -s -X POST "$API_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "database-server-01",
    "hostname": "db01.example.com",
    "make_model": "Dell PowerEdge R740",
    "os": "Ubuntu 22.04",
    "description": "Primary database server"
  }' | jq -r '.id')

DEVICE3=$(curl -s -X POST "$API_URL/devices" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "switch-core-01",
    "hostname": "sw01.example.com",
    "make_model": "Cisco Catalyst 9300",
    "os": "IOS-XE",
    "description": "Core network switch"
  }' | jq -r '.id')

echo "   Created devices:"
echo "   - web-server-01: $DEVICE1"
echo "   - database-server-01: $DEVICE2"
echo "   - switch-core-01: $DEVICE3"
echo

# Add relationships
echo "2. Adding relationships..."
curl -s -X POST "$API_URL/devices/$DEVICE1/relationships" \
  -H "Content-Type: application/json" \
  -d "{\"child_id\": \"$DEVICE2\", \"type\": \"depends_on\"}" > /dev/null
echo "   ✓ web-server-01 depends_on database-server-01"

curl -s -X POST "$API_URL/devices/$DEVICE3/relationships" \
  -H "Content-Type: application/json" \
  -d "{\"child_id\": \"$DEVICE1\", \"type\": \"connected_to\"}" > /dev/null
echo "   ✓ switch-core-01 connected_to web-server-01"

curl -s -X POST "$API_URL/devices/$DEVICE3/relationships" \
  -H "Content-Type: application/json" \
  -d "{\"child_id\": \"$DEVICE2\", \"type\": \"connected_to\"}" > /dev/null
echo "   ✓ switch-core-01 connected_to database-server-01"
echo

# Get relationships
echo "3. Retrieving relationships..."
echo "   Relationships for web-server-01:"
curl -s "$API_URL/devices/$DEVICE1/relationships" | jq -r '.[] | "   - \(.type): \(.parent_id) -> \(.child_id)"'
echo

echo "   Relationships for database-server-01:"
curl -s "$API_URL/devices/$DEVICE2/relationships" | jq -r '.[] | "   - \(.type): \(.parent_id) -> \(.child_id)"'
echo

echo "   Relationships for switch-core-01:"
curl -s "$API_URL/devices/$DEVICE3/relationships" | jq -r '.[] | "   - \(.type): \(.parent_id) -> \(.child_id)"'
echo

# Test UI
echo "4. Testing UI..."
echo "   Open your browser to:"
echo "   - Web Server: $BASE_URL/devices/detail?id=$DEVICE1"
echo "   - Database Server: $BASE_URL/devices/detail?id=$DEVICE2"
echo "   - Switch: $BASE_URL/devices/detail?id=$DEVICE3"
echo
echo "   You should see:"
echo "   ✓ Relationships section with 'Add Relationship' button"
echo "   ✓ Existing relationships displayed with color-coded badges"
echo "   ✓ Remove button for each relationship"
echo "   ✓ Clickable device names to navigate"
echo

# Cleanup option
echo "5. Cleanup (optional)..."
echo "   To remove test devices, run:"
echo "   curl -X DELETE $API_URL/devices/$DEVICE1"
echo "   curl -X DELETE $API_URL/devices/$DEVICE2"
echo "   curl -X DELETE $API_URL/devices/$DEVICE3"
echo

echo "=== Test Complete ==="
echo "The relationship management UI is ready to use!"
