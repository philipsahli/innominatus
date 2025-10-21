#!/bin/bash
# Comprehensive debug script to trace through entire workflow
# This script helps identify where data flow breaks

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}================================================${NC}"
echo -e "${GREEN}  Innominatus Integration Testing & Debugging${NC}"
echo -e "${GREEN}================================================${NC}\n"

# Get API token
echo -e "${YELLOW}=== 0. Getting API Token ===${NC}"
if [ -f ~/.innominatus/credentials ]; then
    TOKEN=$(grep api_key ~/.innominatus/credentials | cut -d: -f2 | tr -d ' ')
    echo "✓ Token found in credentials file"
else
    echo "⚠️  No credentials file found, trying environment variable"
    TOKEN=${IDP_API_KEY:-""}
fi

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ No API token available${NC}"
    echo "Please run: ./innominatus-ctl login"
    exit 1
fi

# Test 1: Server Health
echo -e "\n${YELLOW}=== 1. Checking Server Health ===${NC}"
if curl -s http://localhost:8081/health | jq '.' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Server is healthy${NC}"
    curl -s http://localhost:8081/health | jq '.'
else
    echo -e "${RED}✗ Server health check failed${NC}"
    exit 1
fi

# Test 2: Deploy Application
echo -e "\n${YELLOW}=== 2. Deploying Application (world-app3) ===${NC}"
if ./innominatus-ctl run deploy-app score-spec-k8s.yaml; then
    echo -e "${GREEN}✓ Deployment command succeeded${NC}"
else
    echo -e "${RED}✗ Deployment command failed${NC}"
    exit 1
fi

# Wait for async processing
echo "Waiting 3 seconds for async processing..."
sleep 3

# Test 3: Database - Applications Table
echo -e "\n${YELLOW}=== 3. Checking Database - Applications Table ===${NC}"
APP_CHECK=$(psql -h localhost -U postgres -d idp_orchestrator2 -t -c "SELECT COUNT(*) FROM applications WHERE name='world-app3';" 2>&1)
if echo "$APP_CHECK" | grep -q "1"; then
    echo -e "${GREEN}✓ Application found in database${NC}"
    psql -h localhost -U postgres -d idp_orchestrator2 -c "SELECT id, name, created_at FROM applications WHERE name='world-app3';"
else
    echo -e "${YELLOW}⚠️  Application not in applications table (count: $APP_CHECK)${NC}"
fi

# Test 4: Database - Queue Tasks
echo -e "\n${YELLOW}=== 4. Checking Database - Queue Tasks ===${NC}"
QUEUE_COUNT=$(psql -h localhost -U postgres -d idp_orchestrator2 -t -c "SELECT COUNT(*) FROM queue_tasks;" 2>&1)
echo "Total queue tasks: $QUEUE_COUNT"
psql -h localhost -U postgres -d idp_orchestrator2 -c "SELECT task_id, app_name, workflow_name, status, enqueued_at FROM queue_tasks ORDER BY enqueued_at DESC LIMIT 5;"

# Test 5: Database - Workflow Executions
echo -e "\n${YELLOW}=== 5. Checking Database - Workflow Executions ===${NC}"
WORKFLOW_COUNT=$(psql -h localhost -U postgres -d idp_orchestrator2 -t -c "SELECT COUNT(*) FROM workflow_executions;" 2>&1)
echo "Total workflow executions: $WORKFLOW_COUNT"
psql -h localhost -U postgres -d idp_orchestrator2 -c "SELECT id, application_name, workflow_name, status, started_at FROM workflow_executions ORDER BY started_at DESC LIMIT 5;"

# Test 6: Database - Graph Nodes
echo -e "\n${YELLOW}=== 6. Checking Database - Graph Nodes ===${NC}"
NODE_COUNT=$(psql -h localhost -U postgres -d idp_orchestrator2 -t -c "SELECT COUNT(*) FROM graph_nodes;" 2>&1)
echo "Total graph nodes: $NODE_COUNT"

if [ "$NODE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Graph nodes exist${NC}"
    psql -h localhost -U postgres -d idp_orchestrator2 -c "SELECT COUNT(*), type, state FROM graph_nodes GROUP BY type, state;"
else
    echo -e "${RED}✗ No graph nodes found!${NC}"
    echo "This indicates graph adapter is not creating nodes"
fi

# Test 7: Database - Graph Edges
echo -e "\n${YELLOW}=== 7. Checking Database - Graph Edges ===${NC}"
EDGE_COUNT=$(psql -h localhost -U postgres -d idp_orchestrator2 -t -c "SELECT COUNT(*) FROM graph_edges;" 2>&1)
echo "Total graph edges: $EDGE_COUNT"

if [ "$EDGE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ Graph edges exist${NC}"
    psql -h localhost -U postgres -d idp_orchestrator2 -c "SELECT COUNT(*), type FROM graph_edges GROUP BY type;"
else
    echo -e "${YELLOW}⚠️  No graph edges found${NC}"
fi

# Test 8: API - List Specs
echo -e "\n${YELLOW}=== 8. Checking API - GET /api/specs ===${NC}"
SPECS_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/specs)
if echo "$SPECS_RESPONSE" | jq '.' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ API returned valid JSON${NC}"
    echo "Applications found:"
    echo "$SPECS_RESPONSE" | jq 'keys'

    if echo "$SPECS_RESPONSE" | jq -e '.["world-app3"]' > /dev/null 2>&1; then
        echo -e "${GREEN}✓ world-app3 found in API response${NC}"
    else
        echo -e "${RED}✗ world-app3 NOT found in API response${NC}"
    fi
else
    echo -e "${RED}✗ API returned invalid response: $SPECS_RESPONSE${NC}"
fi

# Test 9: API - Graph Endpoint
echo -e "\n${YELLOW}=== 9. Checking API - GET /api/graph/world-app3 ===${NC}"
GRAPH_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/graph/world-app3)
if echo "$GRAPH_RESPONSE" | jq '.' > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Graph API returned valid JSON${NC}"
    NODE_COUNT_API=$(echo "$GRAPH_RESPONSE" | jq '.nodes | length')
    EDGE_COUNT_API=$(echo "$GRAPH_RESPONSE" | jq '.edges | length')
    echo "Nodes in response: $NODE_COUNT_API"
    echo "Edges in response: $EDGE_COUNT_API"

    if [ "$NODE_COUNT_API" -gt 0 ]; then
        echo -e "${GREEN}✓ Graph has nodes${NC}"
        echo "Node types:"
        echo "$GRAPH_RESPONSE" | jq '[.nodes[].type] | unique'
    else
        echo -e "${RED}✗ No nodes in graph response${NC}"
    fi
else
    echo -e "${RED}✗ Graph API returned invalid response: $GRAPH_RESPONSE${NC}"
fi

# Test 10: CLI List
echo -e "\n${YELLOW}=== 10. Checking CLI - innominatus-ctl list ===${NC}"
if ./innominatus-ctl list | grep -q "world-app3"; then
    echo -e "${GREEN}✓ world-app3 visible in CLI${NC}"
    ./innominatus-ctl list | grep -A 15 "world-app3"
else
    echo -e "${RED}✗ world-app3 NOT visible in CLI${NC}"
fi

# Test 11: CLI Graph Status
echo -e "\n${YELLOW}=== 11. Checking CLI - graph-status ===${NC}"
if ./innominatus-ctl graph-status world-app3 2>&1 | grep -q "Total Nodes"; then
    echo -e "${GREEN}✓ Graph status command succeeded${NC}"
    ./innominatus-ctl graph-status world-app3
else
    echo -e "${RED}✗ Graph status command failed${NC}"
    ./innominatus-ctl graph-status world-app3 2>&1
fi

# Summary
echo -e "\n${GREEN}================================================${NC}"
echo -e "${GREEN}  Test Summary${NC}"
echo -e "${GREEN}================================================${NC}"

echo -e "\nDatabase Status:"
echo "  - Applications table: $(psql -h localhost -U postgres -d idp_orchestrator2 -t -c "SELECT COUNT(*) FROM applications WHERE name='world-app3';" | tr -d ' ') entries"
echo "  - Queue tasks: $QUEUE_COUNT total"
echo "  - Workflow executions: $WORKFLOW_COUNT total"
echo "  - Graph nodes: $NODE_COUNT total"
echo "  - Graph edges: $EDGE_COUNT total"

echo -e "\nAPI Status:"
echo "  - /api/specs: $(echo "$SPECS_RESPONSE" | jq 'keys | length') apps"
echo "  - /api/graph/world-app3: $NODE_COUNT_API nodes, $EDGE_COUNT_API edges"

echo -e "\n${YELLOW}Next Steps:${NC}"
if [ "$NODE_COUNT" -eq 0 ]; then
    echo "  ${RED}⚠️  CRITICAL: No graph nodes created${NC}"
    echo "     - Check server logs for graph adapter errors"
    echo "     - Verify graph adapter is initialized"
    echo "     - Check innominatus-graph SDK compatibility"
elif [ "$NODE_COUNT_API" -eq 0 ]; then
    echo "  ${RED}⚠️  CRITICAL: Graph API not returning nodes${NC}"
    echo "     - Nodes exist in DB but API doesn't return them"
    echo "     - Check API handler: internal/server/handlers.go"
    echo "     - Verify app_id matching logic"
else
    echo "  ${GREEN}✓ All systems operational${NC}"
    echo "     - Ready for UI testing"
    echo "     - Run Puppeteer tests next"
fi

echo -e "\n${GREEN}Done!${NC}\n"
