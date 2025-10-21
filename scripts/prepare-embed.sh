#!/bin/bash
# prepare-embed.sh - Copies static files to cmd/server/ for Go embed directives

set -e  # Exit on error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EMBED_DIR="$PROJECT_ROOT/cmd/server"

echo "Preparing static files for embedding..."

# Create cmd/server directory if it doesn't exist
mkdir -p "$EMBED_DIR"

# Copy migrations
echo "Copying migrations..."
rm -rf "$EMBED_DIR/migrations"
cp -r "$PROJECT_ROOT/migrations" "$EMBED_DIR/migrations"

# Copy swagger files
echo "Copying swagger files..."
rm -f "$EMBED_DIR/swagger-admin.yaml" "$EMBED_DIR/swagger-user.yaml"
cp "$PROJECT_ROOT/swagger-admin.yaml" "$EMBED_DIR/swagger-admin.yaml"
cp "$PROJECT_ROOT/swagger-user.yaml" "$EMBED_DIR/swagger-user.yaml"

# Copy web-ui build output
echo "Copying web-ui output..."
rm -rf "$EMBED_DIR/web-ui-out"
if [ -d "$PROJECT_ROOT/web-ui/out" ]; then
    cp -r "$PROJECT_ROOT/web-ui/out" "$EMBED_DIR/web-ui-out"
else
    # If web-ui/out doesn't exist, create an empty directory with index.html
    echo "Warning: web-ui/out not found, creating minimal placeholder"
    mkdir -p "$EMBED_DIR/web-ui-out"
    echo '<!DOCTYPE html><html><body><p>Web UI not built. Run: cd web-ui && npm run build</p></body></html>' > "$EMBED_DIR/web-ui-out/index.html"
fi

echo "Static files prepared successfully!"
echo "  - $EMBED_DIR/migrations/"
echo "  - $EMBED_DIR/swagger-*.yaml"
echo "  - $EMBED_DIR/web-ui-out/"
