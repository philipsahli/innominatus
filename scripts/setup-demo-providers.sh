#!/bin/bash
# Setup script to push product team providers to Gitea repositories
# This creates organizations and repositories for each product team

set -e

# Configuration
GITEA_URL="${GITEA_URL:-http://gitea.localtest.me}"
GITEA_USER="${GITEA_USER:-giteaadmin}"
GITEA_PASS="${GITEA_PASS:-admin123}"
GITEA_API="${GITEA_URL}/api/v1"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 Setting up Product Team Providers in Gitea"
echo "=============================================="
echo ""

# Product teams to setup
TEAMS=("container-team" "database-team" "storage-team" "vault-team")

# Function to create Gitea organization
create_organization() {
    local org_name=$1
    echo -e "${BLUE}📁 Creating organization: ${org_name}${NC}"

    curl -s -X POST "${GITEA_API}/orgs" \
        -u "${GITEA_USER}:${GITEA_PASS}" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"${org_name}\",
            \"full_name\": \"${org_name}\",
            \"description\": \"Product team: ${org_name}\",
            \"visibility\": \"public\"
        }" > /dev/null 2>&1 || echo "   (Organization may already exist)"

    echo -e "${GREEN}   ✅ Organization ready${NC}"
}

# Function to create Gitea repository
create_repository() {
    local org_name=$1
    local repo_name="${org_name}-provider"

    echo -e "${BLUE}📦 Creating repository: ${org_name}/${repo_name}${NC}"

    curl -s -X POST "${GITEA_API}/org/${org_name}/repos" \
        -u "${GITEA_USER}:${GITEA_PASS}" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"${repo_name}\",
            \"description\": \"Provider configuration for ${org_name}\",
            \"private\": false,
            \"auto_init\": false
        }" > /dev/null 2>&1 || echo "   (Repository may already exist)"

    echo -e "${GREEN}   ✅ Repository ready${NC}"
}

# Function to push provider content to repository
push_provider_content() {
    local team_name=$1
    local repo_name="${team_name}-provider"
    local source_dir="providers/${team_name}"
    local temp_dir="/tmp/gitea-provider-${team_name}"

    # Save current working directory
    local original_dir=$(pwd)

    if [ ! -d "${source_dir}" ]; then
        echo -e "${YELLOW}   ⚠️  Source directory not found: ${source_dir}${NC}"
        return 1
    fi

    echo -e "${BLUE}📤 Pushing ${team_name} provider content to Git${NC}"

    # Clean up temp directory
    rm -rf "${temp_dir}"
    mkdir -p "${temp_dir}"

    # Initialize git repo
    cd "${temp_dir}"
    git init -q
    git config user.name "Platform Admin"
    git config user.email "admin@innominatus.local"

    # Copy provider content from original directory
    cp -r "${original_dir}/${source_dir}"/* .

    # Add and commit
    git add .
    git commit -q -m "feat: initial provider configuration for ${team_name}"

    # Add remote and push
    git remote add origin "${GITEA_URL}/${team_name}/${repo_name}.git"

    # Push with credentials
    echo "   Pushing to ${GITEA_URL}/${team_name}/${repo_name}"
    git push -f "http://${GITEA_USER}:${GITEA_PASS}@${GITEA_URL#http://}/${team_name}/${repo_name}.git" main 2>&1 | grep -v "Username\|Password" || true

    cd - > /dev/null
    rm -rf "${temp_dir}"

    echo -e "${GREEN}   ✅ Content pushed to Git${NC}"
}

# Main execution
echo -e "${BLUE}Step 1: Creating Gitea organizations${NC}"
echo "-----------------------------------"
for team in "${TEAMS[@]}"; do
    create_organization "${team}"
done

echo ""
echo -e "${BLUE}Step 2: Creating Git repositories${NC}"
echo "-------------------------------"
for team in "${TEAMS[@]}"; do
    create_repository "${team}"
done

echo ""
echo -e "${BLUE}Step 3: Pushing provider content${NC}"
echo "------------------------------"
for team in "${TEAMS[@]}"; do
    push_provider_content "${team}"
done

echo ""
echo -e "${GREEN}🎉 Setup Complete!${NC}"
echo ""
echo "Product team providers are now available in Gitea:"
for team in "${TEAMS[@]}"; do
    echo "  • ${GITEA_URL}/${team}/${team}-provider"
done
echo ""
echo "Next steps:"
echo "1. Update admin-config.yaml to register these Git-based providers"
echo "2. Restart innominatus server: pkill innominatus && ./innominatus &"
echo "3. Verify providers: ./innominatus-ctl list-providers"
echo ""
