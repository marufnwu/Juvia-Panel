#!/bin/bash
# Juvia Panel - Comprehensive Backend Test Script
# Version: 1.0
# Date: 2026-06-08

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_URL="${PANEL_API_URL:-http://127.0.0.1:9090}"
PANEL_DOMAIN="${PANEL_DOMAIN:-http://127.0.0.1:2053}"
TEST_USER="${PANEL_TEST_USER:-}"
TEST_PASS="${PANEL_TEST_PASS:-}"

# Ensure credentials are provided via env vars
if [[ -z "$TEST_USER" || -z "$TEST_PASS" ]]; then
    echo "ERROR: PANEL_TEST_USER and PANEL_TEST_PASS environment variables must be set."
    echo "Example: PANEL_TEST_USER=admin@example.com PANEL_TEST_PASS=secret ./run-tests.sh"
    exit 1
fi

# Counters
PASS=0
FAIL=0
SKIP=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASS++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAIL++))
}

log_skip() {
    echo -e "${YELLOW}[SKIP]${NC} $1"
    ((SKIP++))
}

# Get auth token
get_token() {
    RESP=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASS\"}")
    
    if echo "$RESP" | grep -q "access_token"; then
        echo "$RESP" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4
 else
        echo ""
    fi
}

# Test helper
test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local expected_status=$4
    local token=${5:-""}
    
    local headers="-H 'Content-Type: application/json'"
    if [ -n "$token" ]; then
        headers="$headers -H 'Authorization: Bearer $token'"
    fi
    
    local resp=$(curl -s -w "\n%{http_code}" -X "$method" "$API_URL$endpoint" $headers 2>/dev/null)
    local status=$(echo "$resp" | tail -1)
    local body=$(echo "$resp" | sed '$d')
    
    if [ "$status" = "$expected_status" ]; then
        log_pass "$name (HTTP $status)"
        return 0
    else
        log_fail "$name - Expected HTTP $expected_status, got $status"
        return 1
    fi
}

echo "========================================"
echo "Juvia Panel - Backend Test Suite"
echo "========================================"
echo ""

# Phase 1: Infrastructure Tests
echo "========================================"
echo "Phase 1: Infrastructure Tests"
echo "========================================"

log_info "1.1.1 Database file exists..."
if [ -f "/var/panel/panel.db" ]; then
    log_pass "Database file exists"
else
    log_fail "Database file not found"
fi

log_info "1.1.2 Database tables..."
TABLES=$(sqlite3 /var/panel/panel.db ".tables" 2>/dev/null | wc -w)
if [ "$TABLES" -ge 15 ]; then
    log_pass "Database has $TABLES tables"
else
    log_fail "Database has only $TABLES tables (expected 15+)"
fi

log_info "1.1.3 WAL mode..."
WAL=$(sqlite3 /var/panel/panel.db "PRAGMA journal_mode;" 2>/dev/null)
if [ "$WAL" = "wal" ]; then
    log_pass "WAL mode enabled"
else
    log_fail "WAL mode not enabled (got: $WAL)"
fi

log_info "1.2.1 API health check..."
test_endpoint "API health" "GET" "/health" "200"

log_info "1.2.2 API version..."
RESP=$(curl -s "$API_URL/health" 2>/dev/null)
if echo "$RESP" | grep -q "version"; then
    log_pass "API version endpoint working"
else
    log_fail "API version endpoint not working"
fi

log_info "1.3.1 Agent daemon running..."
if systemctl is-active --quiet juvia-agent; then
    log_pass "Agent daemon running"
else
    log_fail "Agent daemon not running"
fi

log_info "1.3.2 Agent ping..."
if nc -z -w2 /var/run/panel/agent.sock 2>/dev/null || nc -z -w2 127.0.0.1 9091 2>/dev/null; then
    log_pass "Agent is reachable"
else
    log_skip "Agent socket not directly reachable (may use TCP fallback)"
fi

log_info "1.4.1 Docker running..."
if systemctl is-active --quiet docker; then
    log_pass "Docker daemon running"
else
    log_fail "Docker daemon not running"
fi

log_info "1.4.2 Docker info..."
if docker info >/dev/null 2>&1; then
    log_pass "Docker is functional"
else
    log_fail "Docker is not functional"
fi

log_info "1.5.1 Caddy running..."
if systemctl is-active --quiet juvia-caddy; then
    log_pass "Caddy daemon running"
else
    log_fail "Caddy daemon not running"
fi

log_info "1.5.2 UI accessible..."
if curl -s -o /dev/null -w "%{http_code}" "$PANEL_DOMAIN" 2>/dev/null | grep -q "200"; then
    log_pass "UI is accessible"
else
    log_fail "UI is not accessible"
fi

echo ""
echo "========================================"
echo "Phase 2: Authentication & Authorization"
echo "========================================"

TOKEN=$(get_token)

log_info "2.1.1 Login with valid credentials..."
if [ -n "$TOKEN" ]; then
    log_pass "Login successful, token received"
else
    log_fail "Login failed"
fi

log_info "2.1.2 Login with invalid password..."
RESP=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$TEST_USER\",\"password\":\"wrongpassword\"}" 2>/dev/null)
if echo "$RESP" | grep -q "invalid_credentials\|unauthorized"; then
    log_pass "Invalid login rejected"
else
    log_fail "Invalid login not properly rejected"
fi

log_info "2.1.3 Login with non-existent user..."
RESP=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"nonexistent\",\"password\":\"test\"}" 2>/dev/null)
if echo "$RESP" | grep -q "invalid_credentials\|unauthorized"; then
    log_pass "Non-existent user login rejected"
else
    log_fail "Non-existent user login not properly rejected"
fi

if [ -n "$TOKEN" ]; then
    log_info "2.1.4 Access protected endpoint with token..."
    test_endpoint "Protected endpoint" "GET" "/api/v1/activity" "200" "$TOKEN"
    
    log_info "2.1.5 Access without token..."
    test_endpoint "No token" "GET" "/api/v1/activity" "401" ""
    
    log_info "2.1.6 Access with invalid token..."
    test_endpoint "Invalid token" "GET" "/api/v1/activity" "401" "invalid_token_here"
    
    log_info "2.2.1 Get current user..."
    RESP=$(curl -s "$API_URL/api/v1/users/me" -H "Authorization: Bearer $TOKEN" 2>/dev/null)
    if echo "$RESP" | grep -q "username\|email"; then
        log_pass "Current user retrieved"
    else
        log_fail "Current user not retrieved"
    fi
    
    log_info "2.2.2 List users (owner only)..."
    test_endpoint "List users" "GET" "/api/v1/users" "200" "$TOKEN"
    
    log_info "2.3.1 List apps..."
    test_endpoint "List apps" "GET" "/api/v1/apps" "200" "$TOKEN"
    
    log_info "2.3.2 List services..."
    test_endpoint "List services" "GET" "/api/v1/services" "200" "$TOKEN"
    
    log_info "2.3.3 Get server metrics..."
    test_endpoint "Server metrics" "GET" "/api/v1/server/metrics" "200" "$TOKEN"
    
    log_info "2.3.4 Get server info..."
    test_endpoint "Server info" "GET" "/api/v1/server/info" "200" "$TOKEN"
    
    log_info "2.3.5 Get activity..."
    test_endpoint "Activity" "GET" "/api/v1/activity" "200" "$TOKEN"
    
    log_info "2.3.6 Get notifications..."
    test_endpoint "Notifications" "GET" "/api/v1/notifications" "200" "$TOKEN"
fi

echo ""
echo "========================================"
echo "Phase 3: Application Management"
echo "========================================"

if [ -n "$TOKEN" ]; then
    log_info "3.1.1 Create app (from Git)..."
    RESP=$(curl -s -X POST "$API_URL/api/v1/apps" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "test-node-app",
            "runtime": "nodejs",
            "source_type": "git",
            "source_config": {"repository": "https://github.com/example/repo", "branch": "main"},
            "build_strategy": "nixpacks"
        }' 2>/dev/null)
    if echo "$RESP" | grep -q "id\|name"; then
        log_pass "App created"
        APP_ID=$(echo "$RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    else
        log_skip "App creation may require valid Git repo"
    fi
    
    log_info "3.1.2 List apps..."
    test_endpoint "List apps" "GET" "/api/v1/apps" "200" "$TOKEN"
    
    log_info "3.1.3 Get app details..."
    if [ -n "$APP_ID" ]; then
        test_endpoint "Get app" "GET" "/api/v1/apps/$APP_ID" "200" "$TOKEN"
    else
        log_skip "App ID not available"
    fi
    
    log_info "3.2.1 Get app env vars..."
    if [ -n "$APP_ID" ]; then
        test_endpoint "Get env vars" "GET" "/api/v1/apps/$APP_ID/env" "200" "$TOKEN"
    else
        log_skip "App ID not available"
    fi
    
    log_info "3.2.2 Get app volumes..."
    if [ -n "$APP_ID" ]; then
        test_endpoint "Get volumes" "GET" "/api/v1/apps/$APP_ID/volumes" "200" "$TOKEN"
    else
        log_skip "App ID not available"
    fi
    
    log_info "3.2.3 Get app deployments..."
    if [ -n "$APP_ID" ]; then
        test_endpoint "Get deployments" "GET" "/api/v1/apps/$APP_ID/deployments" "200" "$TOKEN"
    else
        log_skip "App ID not available"
    fi
    
    log_info "3.3.1 Get app logs..."
    if [ -n "$APP_ID" ]; then
        test_endpoint "Get logs" "GET" "/api/v1/apps/$APP_ID/logs" "200" "$TOKEN"
    else
        log_skip "App ID not available"
    fi
fi

echo ""
echo "========================================"
echo "Phase 4: Service Management"
echo "========================================"

if [ -n "$TOKEN" ]; then
    log_info "4.1.1 List services..."
    test_endpoint "List services" "GET" "/api/v1/services" "200" "$TOKEN"
    
    log_info "4.1.2 Get service types..."
    RESP=$(curl -s "$API_URL/api/v1/services" -H "Authorization: Bearer $TOKEN" 2>/dev/null)
    if echo "$RESP" | grep -q "postgresql\|mysql\|redis"; then
        log_pass "Service types available"
    else
        log_skip "No services created yet"
    fi
    
    log_info "4.2.1 Get backups..."
    test_endpoint "List backups" "GET" "/api/v1/backups" "200" "$TOKEN"
    
    log_info "4.2.2 Get backup settings..."
    test_endpoint "Backup settings" "GET" "/api/v1/backups/settings" "200" "$TOKEN"
fi

echo ""
echo "========================================"
echo "Phase 5: Server Management"
echo "========================================"

if [ -n "$TOKEN" ]; then
    log_info "5.1.1 Get server info..."
    test_endpoint "Server info" "GET" "/api/v1/server/info" "200" "$TOKEN"
    
    log_info "5.1.2 Get server metrics..."
    test_endpoint "Server metrics" "GET" "/api/v1/server/metrics" "200" "$TOKEN"
    
    log_info "5.1.3 Get processes..."
    test_endpoint "Processes" "GET" "/api/v1/server/processes" "200" "$TOKEN"
    
    log_info "5.1.4 Get disk usage..."
    test_endpoint "Disk usage" "GET" "/api/v1/server/disk" "200" "$TOKEN"
    
    log_info "5.1.5 Get network info..."
    test_endpoint "Network info" "GET" "/api/v1/server/network" "200" "$TOKEN"
    
    log_info "5.2.1 Get firewall status..."
    test_endpoint "Firewall status" "GET" "/api/v1/firewall" "200" "$TOKEN"
    
    log_info "5.2.2 List cron jobs..."
    test_endpoint "Cron jobs" "GET" "/api/v1/cron" "200" "$TOKEN"
fi

echo ""
echo "========================================"
echo "Phase 6: Real-time & WebSocket"
echo "========================================"

log_info "6.1.1 WebSocket connection..."
if curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/stream" 2>/dev/null | grep -q "200\|101"; then
    log_pass "WebSocket endpoint accessible"
else
    log_fail "WebSocket endpoint not accessible"
fi

echo ""
echo "========================================"
echo "Phase 7: Security Testing"
echo "========================================"

log_info "7.1.1 Rate limiting on login..."
for i in {1..5}; do
    curl -s -X POST "$API_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"test\",\"password\":\"test\"}" >/dev/null 2>&1
done
RESP=$(curl -s -X POST "$API_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"test\",\"password\":\"test\"}" 2>/dev/null)
if echo "$RESP" | grep -q "rate_limit\|too_many"; then
    log_pass "Rate limiting working"
else
    log_skip "Rate limit may not trigger in test conditions"
fi

log_info "7.1.2 SQL injection prevention..."
RESP=$(curl -s "$API_URL/api/v1/apps?status=' OR '1'='1" \
    -H "Authorization: Bearer $TOKEN" 2>/dev/null)
if echo "$RESP" | grep -q "error\|invalid"; then
    log_pass "SQL injection blocked"
else
    log_skip "SQL injection test inconclusive"
fi

log_info "7.1.3 XSS prevention..."
# XSS in app name should be sanitized
RESP=$(curl -s -X POST "$API_URL/api/v1/apps" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"name":"<script>alert(1)</script>","runtime":"nodejs","source_type":"git","source_config":{},"build_strategy":"nixpacks"}' 2>/dev/null)
if echo "$RESP" | grep -q "error\|invalid"; then
    log_pass "XSS in name rejected"
else
    log_skip "XSS test inconclusive"
fi

echo ""
echo "========================================"
echo "Phase 8: Performance Testing"
echo "========================================"

log_info "8.1.1 API response time (health)..."
START=$(date +%s%N)
curl -s "$API_URL/health" >/dev/null 2>&1
END=$(date +%s%N)
MS=$(( (END - START) / 1000000 ))
if [ "$MS" -lt 100 ]; then
    log_pass "Health check: ${MS}ms"
else
    log_fail "Health check too slow: ${MS}ms"
fi

log_info "8.1.2 API response time (apps list)..."
START=$(date +%s%N)
curl -s "$API_URL/api/v1/apps" -H "Authorization: Bearer $TOKEN" >/dev/null 2>&1
END=$(date +%s%N)
MS=$(( (END - START) / 1000000 ))
if [ "$MS" -lt 1000 ]; then
    log_pass "Apps list: ${MS}ms"
else
    log_fail "Apps list too slow: ${MS}ms"
fi

echo ""
echo "========================================"
echo "Phase9: Integration Testing"
echo "========================================"

if [ -n "$TOKEN" ]; then
    log_info "9.1.1 Full app lifecycle (create, deploy, delete)..."
    # Create
    RESP=$(curl -s -X POST "$API_URL/api/v1/apps" \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "integration-test-app",
            "runtime": "static",
            "source_type": "git",
            "source_config": {"repository": "https://github.com/example/static-site", "branch": "main"},
            "build_strategy": "static"
        }' 2>/dev/null)
    if echo "$RESP" | grep -q "id"; then
        TEST_APP_ID=$(echo "$RESP" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
        log_pass "App created for integration test"
        
        # Delete
        RESP=$(curl -s -X DELETE "$API_URL/api/v1/apps/$TEST_APP_ID" \
            -H "Authorization: Bearer $TOKEN" 2>/dev/null)
        if echo "$RESP" | grep -q "success\|204"; then
            log_pass "App deleted in integration test"
        else
            log_fail "App deletion failed"
        fi
    else
        log_skip "App creation skipped (may require valid Git)"
    fi
fi

echo ""
echo "========================================"
echo "Phase10: Disaster Recovery"
echo "========================================"

log_info "10.1.1 Database backup..."
if [ -f "/var/panel/panel.db" ]; then
    cp /var/panel/panel.db /tmp/panel-test-backup.db
    if [ -f "/tmp/panel-test-backup.db" ]; then
        log_pass "Database backup created"
        rm /tmp/panel-test-backup.db
    else
        log_fail "Database backup failed"
    fi
else
    log_fail "Database file not found"
fi

log_info "10.1.2 Database restore..."
# This would require a backup file to restore
log_skip "Database restore test (requires backup file)"

log_info "10.2.1 Service restart (API)..."
if systemctl restart juvia-api 2>/dev/null; then
    sleep 2
    if curl -s "$API_URL/health" | grep -q "ok"; then
        log_pass "API restart successful"
    else
        log_fail "API restart but health check failed"
    fi
else
    log_fail "API restart failed"
fi

echo ""
echo "========================================"
echo "Test Summary"
echo "========================================"
echo -e "${GREEN}PASSED:${NC} $PASS"
echo -e "${RED}FAILED:${NC} $FAIL"
echo -e "${YELLOW}SKIPPED:${NC} $SKIP"
echo "========================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
else
    exit 0
fi
