#!/bin/bash
#
# fix-gitea-oauth.sh - Fix Gitea OAuth2 auto-registration with Keycloak
#
# This script ensures that Gitea is properly configured to automatically
# create user accounts when logging in via Keycloak OIDC.
#

set -e

NAMESPACE="gitea"
KEYCLOAK_REALM="demo-realm"
KEYCLOAK_URL="http://keycloak.localtest.me"
OAUTH_NAME="Keycloak"

echo "🔧 Fixing Gitea OAuth2 configuration for auto-registration..."
echo ""

# Get Gitea pod name
echo "📦 Finding Gitea pod..."
POD_NAME=$(kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=gitea -o jsonpath='{.items[0].metadata.name}')

if [ -z "$POD_NAME" ]; then
    echo "❌ Error: No Gitea pod found in namespace $NAMESPACE"
    echo "   Make sure the demo environment is running: ./innominatus-ctl demo-time"
    exit 1
fi

echo "   Found pod: $POD_NAME"
echo ""

# List existing OAuth2 sources
echo "🔍 Checking existing OAuth2 sources..."
kubectl exec -n $NAMESPACE $POD_NAME -- gitea admin auth list || echo "   (No auth sources found or command failed)"
echo ""

# Try to delete existing Keycloak OAuth2 source (ignore errors if it doesn't exist)
echo "🗑️  Removing existing OAuth2 source (if any)..."
kubectl exec -n $NAMESPACE $POD_NAME -- \
    gitea admin auth delete --id 1 2>/dev/null || echo "   (No existing source to remove or different ID)"
echo ""

# Add OAuth2 source with auto-registration enabled
echo "➕ Adding OAuth2 source with auto-registration..."
kubectl exec -n $NAMESPACE $POD_NAME -- \
    gitea admin auth add-oauth \
    --name "$OAUTH_NAME" \
    --provider "openidConnect" \
    --key "gitea" \
    --secret "gitea-client-secret" \
    --auto-discover-url "$KEYCLOAK_URL/realms/$KEYCLOAK_REALM/.well-known/openid-configuration" \
    --skip-local-2fa \
    --scopes "openid email profile" \
    --auto-register

if [ $? -eq 0 ]; then
    echo "   ✅ OAuth2 source added successfully with auto-registration!"
else
    echo "   ⚠️  OAuth2 source might already exist, trying to update instead..."

    # Alternative: Update via Gitea's app.ini (requires pod restart)
    echo "   Checking Gitea configuration..."
    kubectl exec -n $NAMESPACE $POD_NAME -- cat /data/gitea/conf/app.ini | grep -A 10 "oauth2" || echo "   OAuth2 section not found in app.ini"
fi

echo ""

# Verify the OAuth2 source was added
echo "✅ Verifying OAuth2 sources..."
kubectl exec -n $NAMESPACE $POD_NAME -- gitea admin auth list
echo ""

# Check Gitea configuration
echo "📋 Checking Gitea OAuth2 settings in app.ini..."
echo "   ENABLE_AUTO_REGISTRATION should be true:"
kubectl exec -n $NAMESPACE $POD_NAME -- cat /data/gitea/conf/app.ini | grep -E "(ENABLE_AUTO_REGISTRATION|ALLOW_ONLY_EXTERNAL_REGISTRATION|DISABLE_REGISTRATION)" || echo "   Settings not found, using defaults"
echo ""

echo "🎉 OAuth2 configuration fix completed!"
echo ""
echo "📝 Next steps:"
echo "   1. Go to http://gitea.localtest.me"
echo "   2. Click 'Sign In'"
echo "   3. Click 'Sign in with OAuth' and select 'Keycloak'"
echo "   4. Login with Keycloak credentials:"
echo "      - Username: demo-user"
echo "      - Password: password123"
echo "   5. Your account should be automatically created in Gitea!"
echo ""
echo "💡 If it still doesn't work:"
echo "   - Check Keycloak is running: http://keycloak.localtest.me"
echo "   - Verify the OAuth2 client exists in Keycloak (admin/adminpassword)"
echo "   - Check Gitea logs: kubectl logs -n gitea $POD_NAME -f"
echo ""
