#!/bin/bash
# Run integration tests with PostgreSQL

set -e

echo "Starting PostgreSQL container..."

# Check if docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is required but not installed. Please install Docker."
    exit 1
fi

# Start PostgreSQL container
docker run -d \
  --name mnemosyne-test-db \
  -e POSTGRES_USER=mnemosyne_test \
  -e POSTGRES_PASSWORD=test_password \
  -e POSTGRES_DB=mnemosyne_test \
  -p 5432:5432 \
  postgres:15

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in {1..30}; do
  if docker exec mnemosyne-test-db pg_isready -U mnemosyne_test > /dev/null 2>&1; then
    echo "PostgreSQL is ready!"
    break
  fi
  echo -n "."
  sleep 1
done

# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=mnemosyne_test
export DB_PASSWORD=test_password
export DB_NAME=mnemosyne_test
export DB_SSLMODE=disable

# Run integration tests
echo "Running integration tests..."
cd backend
go test -v -tags=integration ./...

# Cleanup
echo "Cleaning up..."
docker stop mnemosyne-test-db
docker rm mnemosyne-test-db

echo "Integration tests completed!"