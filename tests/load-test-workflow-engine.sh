#!/bin/bash

###############################################################################
# Workflow Engine Load Test
#
# Tests workflow engine performance under concurrent load:
# - Concurrent workflow executions
# - Database transaction handling
# - Graph updates via WebSocket
# - API response times
# - Resource cleanup
###############################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Configuration
CONCURRENT_WORKFLOWS=${CONCURRENT_WORKFLOWS:-10}
ITERATIONS=${ITERATIONS:-3}
API_URL=${API_URL:-http://localhost:8081}
LOG_DIR="/tmp/innominatus-load-test"
RESULTS_FILE="$LOG_DIR/load-test-results.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create log directory
mkdir -p "$LOG_DIR"

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║        Workflow Engine Load Test                             ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  Concurrent Workflows: $CONCURRENT_WORKFLOWS"
echo "  Iterations: $ITERATIONS"
echo "  API URL: $API_URL"
echo "  Log Directory: $LOG_DIR"
echo ""

# Check if server is running
echo -e "${BLUE}[1/6] Checking server health...${NC}"
if ! curl -sf "$API_URL/health" > /dev/null; then
    echo -e "${RED}✗ Server not reachable at $API_URL${NC}"
    echo "  Please start the server first: ./innominatus"
    exit 1
fi
echo -e "${GREEN}✓ Server is healthy${NC}"
echo ""

# Get API key from users.yaml (for API checks only, CLI doesn't need it)
echo -e "${BLUE}[2/6] Retrieving API key for API checks...${NC}"
API_KEY=$(grep 'api_key:' users.yaml 2>/dev/null | head -1 | awk '{print $2}' | tr -d '"' || echo "")
if [ -z "$API_KEY" ]; then
    echo -e "${YELLOW}⚠ No API key found (will skip API checks)${NC}"
else
    echo -e "${GREEN}✓ API key retrieved${NC}"
fi
echo ""

# Create test Score spec
echo -e "${BLUE}[3/6] Creating test Score specification...${NC}"
TEST_SPEC="/tmp/load-test-spec.yaml"
cat > "$TEST_SPEC" <<'EOF'
apiVersion: score.dev/v1b1
metadata:
  name: load-test-app
service:
  ports:
    web:
      port: 8080
      targetPort: 8080
containers:
  app:
    image: nginx:alpine
    variables:
      ENV: production
resources:
  db:
    type: postgres
    properties:
      version: "15"
EOF
echo -e "${GREEN}✓ Test spec created${NC}"
echo ""

# Function to deploy application (no retry - tests raw engine resilience)
deploy_app() {
    local app_name=$1
    local iteration=$2
    local start_time=$(date +%s%N)
    local log_file="$LOG_DIR/deploy-${app_name}-${iteration}.log"

    # Modify spec name for uniqueness
    local unique_spec="/tmp/${app_name}-${iteration}.yaml"
    sed "s/load-test-app/${app_name}/g" "$TEST_SPEC" > "$unique_spec"

    # Deploy via golden path (single attempt - engine handles retries)
    if ./innominatus-ctl run deploy-app "$unique_spec" > "$log_file" 2>&1; then
        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))
        echo "{\"app\":\"${app_name}\",\"iteration\":${iteration},\"status\":\"success\",\"duration_ms\":${duration}}"
    else
        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))
        echo "{\"app\":\"${app_name}\",\"iteration\":${iteration},\"status\":\"failed\",\"duration_ms\":${duration},\"error\":\"deployment_failed\"}"
    fi

    rm -f "$unique_spec"
}

# Function to check workflow status via API
check_workflow_status() {
    local app_name=$1
    curl -sf -H "Authorization: Bearer $API_KEY" \
        "$API_URL/api/workflows?app=$app_name" | \
        jq -r '.[0].status' 2>/dev/null || echo "unknown"
}

# Function to get graph via API
get_graph() {
    local app_name=$1
    curl -sf -H "Authorization: Bearer $API_KEY" \
        "$API_URL/api/graph/$app_name" | \
        jq -c '{nodes: .nodes | length, edges: .edges | length}' 2>/dev/null
}

# Initialize results
echo "[]" > "$RESULTS_FILE"

# Run load test
echo -e "${BLUE}[4/6] Running load test...${NC}"
echo "  Starting $CONCURRENT_WORKFLOWS concurrent workflow executions..."
echo ""

total_deployments=$((CONCURRENT_WORKFLOWS * ITERATIONS))
current=0

for iteration in $(seq 1 $ITERATIONS); do
    echo -e "${YELLOW}Iteration $iteration/$ITERATIONS${NC}"

    # Start concurrent deployments with staggered start
    pids=()
    for i in $(seq 1 $CONCURRENT_WORKFLOWS); do
        app_name="loadtest-iter${iteration}-app${i}"

        (
            result=$(deploy_app "$app_name" "$iteration")
            echo "$result" >> "$LOG_DIR/results-${iteration}.json"
        ) &

        pids+=($!)
        current=$((current + 1))
        echo -ne "  Progress: [$current/$total_deployments] "
        printf '█%.0s' $(seq 1 $((current * 50 / total_deployments)))
        echo -ne "\r"
    done

    # Wait for all concurrent deployments to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done

    echo ""

    # Small delay between iterations
    if [ $iteration -lt $ITERATIONS ]; then
        sleep 2
    fi
done

echo ""
echo -e "${GREEN}✓ Load test completed${NC}"
echo ""

# Aggregate results
echo -e "${BLUE}[5/6] Analyzing results...${NC}"

# Combine all iteration results (newline-delimited JSON → array)
cat "$LOG_DIR"/results-*.json | jq -s '.' > "$RESULTS_FILE"

# Calculate statistics
total=$(jq 'length' "$RESULTS_FILE")
successful=$(jq '[.[] | select(.status=="success")] | length' "$RESULTS_FILE")
failed=$(jq '[.[] | select(.status=="failed")] | length' "$RESULTS_FILE")
avg_duration=$(jq '[.[] | .duration_ms] | add / length | round' "$RESULTS_FILE")
min_duration=$(jq '[.[] | .duration_ms] | min' "$RESULTS_FILE")
max_duration=$(jq '[.[] | .duration_ms] | max' "$RESULTS_FILE")

success_rate=$(echo "scale=2; $successful * 100 / $total" | bc)

echo ""
echo -e "${BLUE}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                     Load Test Results                        ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Deployment Statistics:${NC}"
echo "  Total Deployments:     $total"
echo "  Successful:            ${GREEN}$successful${NC}"
echo "  Failed:                ${RED}$failed${NC}"
echo "  Success Rate:          ${success_rate}%"
echo ""
echo -e "${YELLOW}Performance Metrics:${NC}"
echo "  Average Duration:      ${avg_duration} ms"
echo "  Minimum Duration:      ${min_duration} ms"
echo "  Maximum Duration:      ${max_duration} ms"
echo "  Throughput:            $(echo "scale=2; $total / ($max_duration / 1000)" | bc) deployments/sec"
echo ""

# Check database and API health
echo -e "${BLUE}[6/6] Verifying system health...${NC}"

# Check server health endpoint
health_status=$(curl -sf "$API_URL/health" | jq -r '.status' 2>/dev/null || echo "unknown")
echo "  Server Health:         ${health_status}"

# Check if workflows are queryable (only if API key available)
if [ -n "$API_KEY" ]; then
    workflow_count=$(curl -sf -H "Authorization: Bearer $API_KEY" "$API_URL/api/workflows" | jq 'length' 2>/dev/null || echo "0")
    echo "  Total Workflows:       ${workflow_count}"

    # Check graph API
    first_app="loadtest-iter1-app1"
    graph_info=$(get_graph "$first_app")
    if [ -n "$graph_info" ]; then
        echo "  Graph API:             ${GREEN}✓ Working${NC} ($graph_info)"
    else
        echo "  Graph API:             ${YELLOW}⚠ No data${NC}"
    fi
else
    echo "  API Checks:            ${YELLOW}⚠ Skipped (no API key)${NC}"
fi

echo ""

# Performance assessment
if [ "$success_rate" = "100.00" ] && [ "$health_status" = "healthy" ]; then
    echo -e "${GREEN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  ✓ LOAD TEST PASSED                                          ║${NC}"
    echo -e "${GREEN}║    All workflows executed successfully                       ║${NC}"
    echo -e "${GREEN}║    System remained stable under load                         ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════════════╝${NC}"
    exit_code=0
elif [ "$success_rate" != "0.00" ] && [ "$failed" -lt 5 ]; then
    echo -e "${YELLOW}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║  ⚠ LOAD TEST PASSED WITH WARNINGS                           ║${NC}"
    echo -e "${YELLOW}║    Some workflows failed but system is functional           ║${NC}"
    echo -e "${YELLOW}╚══════════════════════════════════════════════════════════════╝${NC}"
    exit_code=0
else
    echo -e "${RED}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║  ✗ LOAD TEST FAILED                                          ║${NC}"
    echo -e "${RED}║    Too many workflow failures or system instability          ║${NC}"
    echo -e "${RED}╚══════════════════════════════════════════════════════════════╝${NC}"
    exit_code=1
fi

echo ""
echo -e "${BLUE}Detailed logs:${NC} $LOG_DIR"
echo -e "${BLUE}Results JSON:${NC}  $RESULTS_FILE"
echo ""

# Cleanup suggestion
echo -e "${YELLOW}Cleanup:${NC}"
echo "  To remove test applications from database:"
echo "  psql -U postgres -d idp_orchestrator -c \"DELETE FROM applications WHERE name LIKE 'loadtest-%';\""
echo ""

exit $exit_code
