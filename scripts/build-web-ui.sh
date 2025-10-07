#!/bin/bash
# Build Web UI for innominatus
# This script builds the Next.js web UI and makes it ready for serving

set -e

echo "🔨 Building Web UI..."

# Navigate to web-ui directory
cd "$(dirname "$0")/../web-ui"

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
  echo "📦 Installing dependencies..."
  npm install
fi

# Build the web UI
echo "⚙️  Building Next.js application..."
npm run build

echo "✅ Web UI build complete!"
echo "📍 Output: web-ui/out/"
echo "🌐 Server will serve from: http://localhost:8081/"
