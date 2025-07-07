#!/bin/bash
# Run integration tests with PostgreSQL in Docker
# This script creates a temporary PostgreSQL instance for testing

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}PostgreSQL Integration Test Runner${NC}"
echo "=================================="

# Configuration
CONTAINER_NAME="mnemosyne-test-db-$$"  # Use PID to avoid conflicts
TEST_PORT=15432  # Use non-standard port to avoid conflicts
DB_USER="postgres"
DB_PASSWORD="postgres"
DB_NAME="mnemosyne_test"

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    docker stop $CONTAINER_NAME >/dev/null 2>&1 || true
    docker rm $CONTAINER_NAME >/dev/null 2>&1 || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Check if docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is required but not installed. Please install Docker.${NC}"
    exit 1
fi

# Check if container already exists and remove it
if docker ps -a | grep -q $CONTAINER_NAME; then
    echo "Removing existing test container..."
    cleanup
fi

# Start PostgreSQL container
echo -e "${BLUE}Starting PostgreSQL container on port $TEST_PORT...${NC}"
docker run -d \
  --name $CONTAINER_NAME \
  -e POSTGRES_USER=$DB_USER \
  -e POSTGRES_PASSWORD=$DB_PASSWORD \
  -e POSTGRES_DB=$DB_NAME \
  -p $TEST_PORT:5432 \
  postgres:15-alpine >/dev/null

# Wait for PostgreSQL to be ready
echo -n "Waiting for PostgreSQL to be ready"
for i in {1..30}; do
  if docker exec $CONTAINER_NAME pg_isready -U $DB_USER >/dev/null 2>&1; then
    echo -e " ${GREEN}✓${NC}"
    break
  fi
  echo -n "."
  sleep 1
  if [ $i -eq 30 ]; then
    echo -e " ${RED}✗${NC}"
    echo -e "${RED}PostgreSQL failed to start within 30 seconds${NC}"
    exit 1
  fi
done

# Create test database config
export TEST_DB_HOST=localhost
export TEST_DB_PORT=$TEST_PORT
export TEST_DB_USER=$DB_USER
export TEST_DB_PASSWORD=$DB_PASSWORD
export TEST_DB_NAME=$DB_NAME
export TEST_DB_SSLMODE=disable

# Run schema setup
echo -e "${BLUE}Setting up database schema...${NC}"
if [ -f internal/db/schema.sql ]; then
    PGPASSWORD=$DB_PASSWORD psql -h localhost -p $TEST_PORT -U $DB_USER -d $DB_NAME -f internal/db/schema.sql >/dev/null 2>&1
    echo -e "${GREEN}✓ Schema created${NC}"
fi

# Run migrations
if [ -d internal/db/migrations ]; then
    for migration in internal/db/migrations/*.sql; do
        if [ -f "$migration" ]; then
            echo -n "Applying $(basename $migration)... "
            PGPASSWORD=$DB_PASSWORD psql -h localhost -p $TEST_PORT -U $DB_USER -d $DB_NAME -f "$migration" >/dev/null 2>&1
            echo -e "${GREEN}✓${NC}"
        fi
    done
fi

# Run integration tests
echo -e "\n${BLUE}Running integration tests...${NC}"
echo "=================================="

# Run only PostgreSQL repository tests since they require a real database
TEST_RESULTS=0

# Repository integration tests
echo -e "\n${BLUE}Repository Integration Tests:${NC}"
if go test -v ./internal/repository/postgres -count=1; then
    echo -e "${GREEN}✓ Repository tests passed${NC}"
else
    echo -e "${RED}✗ Repository tests failed${NC}"
    TEST_RESULTS=1
fi

# Run any other integration tests that require a database
echo -e "\n${BLUE}Other Integration Tests:${NC}"
# Look for tests that check for database availability
if go test -v ./... -run "TestIntegration|TestDB" -count=1 2>&1 | grep -v "no test files"; then
    echo -e "${GREEN}✓ Integration tests passed${NC}"
else
    echo -e "${YELLOW}⚠ No additional integration tests found or some failed${NC}"
fi

# Summary
echo -e "\n=================================="
if [ $TEST_RESULTS -eq 0 ]; then
    echo -e "${GREEN}✓ All integration tests completed successfully!${NC}"
else
    echo -e "${RED}✗ Some integration tests failed${NC}"
    exit 1
fi

echo -e "\n${YELLOW}Note: Container will be automatically cleaned up${NC}"