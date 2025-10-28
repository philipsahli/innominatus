#!/bin/bash
# dev-server.sh - Development server startup script
# Builds web-ui, prepares embedded files, and starts the server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "🔨 Building development environment..."

# Check if web-ui needs building
if [ ! -d "$PROJECT_ROOT/web-ui/out" ] || [ "$PROJECT_ROOT/web-ui" -nt "$PROJECT_ROOT/web-ui/out" ]; then
    echo "📦 Building web-ui..."
    cd "$PROJECT_ROOT/web-ui"
    npm run build
    cd "$PROJECT_ROOT"
else
    echo "✅ Web-UI already built (skipping)"
fi

# Prepare embedded files
echo "📋 Preparing embedded files..."
"$SCRIPT_DIR/prepare-embed.sh"

# Start server
echo "🚀 Starting development server..."
echo ""
cd "$PROJECT_ROOT"
exec go run ./cmd/server/main.go "$@"
