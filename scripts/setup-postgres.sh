#!/bin/bash
# Setup PostgreSQL database for innominatus
set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Database configuration (from environment or defaults)
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-idp_orchestrator2}

echo -e "${GREEN}innominatus PostgreSQL Setup${NC}"
echo "================================"
echo ""
echo "Database configuration:"
echo "  Host:     $DB_HOST"
echo "  Port:     $DB_PORT"
echo "  User:     $DB_USER"
echo "  Database: $DB_NAME"
echo ""

# Check if PostgreSQL is running
echo -e "${YELLOW}Checking PostgreSQL connection...${NC}"
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c '\q' 2>/dev/null; then
    echo -e "${RED}✗ Cannot connect to PostgreSQL at $DB_HOST:$DB_PORT${NC}"
    echo ""
    echo "Please ensure PostgreSQL is running:"
    echo "  - macOS (Homebrew):    brew services start postgresql"
    echo "  - Linux (systemd):     sudo systemctl start postgresql"
    echo "  - Docker:              docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:15"
    echo ""
    exit 1
fi
echo -e "${GREEN}✓ PostgreSQL is running${NC}"
echo ""

# Check if database exists
echo -e "${YELLOW}Checking if database '$DB_NAME' exists...${NC}"
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -lqt | cut -d \| -f 1 | grep -qw $DB_NAME; then
    echo -e "${GREEN}✓ Database '$DB_NAME' already exists${NC}"
else
    echo -e "${YELLOW}Creating database '$DB_NAME'...${NC}"
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;"
    echo -e "${GREEN}✓ Database '$DB_NAME' created${NC}"
fi
echo ""

# Test connection to the database
echo -e "${YELLOW}Testing database connection...${NC}"
if PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT version();" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Successfully connected to database '$DB_NAME'${NC}"
    VERSION=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c "SELECT version();" | head -1 | xargs)
    echo -e "${GREEN}  PostgreSQL version: $VERSION${NC}"
else
    echo -e "${RED}✗ Failed to connect to database '$DB_NAME'${NC}"
    exit 1
fi
echo ""

# Export environment variables for convenience
echo -e "${GREEN}✓ PostgreSQL setup complete!${NC}"
echo ""
echo "You can now start innominatus with:"
echo -e "  ${YELLOW}./innominatus${NC}  (or ${YELLOW}make dev${NC})"
echo ""
echo "To use custom database settings, set environment variables:"
echo "  export DB_HOST=localhost"
echo "  export DB_PORT=5432"
echo "  export DB_USER=postgres"
echo "  export DB_PASSWORD=postgres"
echo "  export DB_NAME=idp_orchestrator2"
echo ""
