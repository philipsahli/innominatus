#!/bin/bash
# test-embedded-ui.sh - Test embedded web UI in Go server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🧪 Testing Embedded Web UI"
echo ""

# Check if server is running
if ! curl -s http://localhost:8082/health > /dev/null 2>&1; then
    echo "❌ Server not running on port 8082"
    echo "   Start it with: go run ./cmd/server/main.go --disable-db --port 8082"
    exit 1
fi

echo "✅ Server is running"
echo ""

# Test cases
TESTS_PASSED=0
TESTS_FAILED=0

test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"
    local check_content="$4"

    echo -n "Testing $name... "

    response=$(curl -s -o /tmp/test-response.txt -w "%{http_code}" "$url")

    if [ "$response" != "$expected_status" ]; then
        echo "❌ FAILED (got $response, expected $expected_status)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi

    if [ -n "$check_content" ]; then
        if ! grep -q "$check_content" /tmp/test-response.txt; then
            echo "❌ FAILED (content check failed)"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    fi

    echo "✅ PASSED"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

# Run tests
echo "📋 Running tests..."
echo ""

test_endpoint "HTML index" "http://localhost:8082/" 200 "<!DOCTYPE html>"
test_endpoint "Webpack chunk" "http://localhost:8082/_next/static/chunks/webpack-5289856b5489d9f4.js" 200
test_endpoint "Main app chunk" "http://localhost:8082/_next/static/chunks/main-app-34e5f21fab611491.js" 200
test_endpoint "CSS file" "http://localhost:8082/_next/static/css/7e7d96b1e6991756.css" 200
test_endpoint "Favicon" "http://localhost:8082/favicon.ico" 200
test_endpoint "Swagger YAML" "http://localhost:8082/swagger-user.yaml" 200 "openapi:"
test_endpoint "Swagger UI" "http://localhost:8082/swagger" 200 "swagger-ui"
test_endpoint "Health endpoint" "http://localhost:8082/health" 200 "healthy"

echo ""
echo "📊 Test Results:"
echo "   ✅ Passed: $TESTS_PASSED"
echo "   ❌ Failed: $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "💥 Some tests failed"
    exit 1
fi
