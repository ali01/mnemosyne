#!/bin/bash
# Run all tests with race detection and coverage

set -e

echo "Running backend tests..."
cd backend

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate coverage report
go tool cover -html=coverage.out -o coverage.html

# Show coverage summary
go tool cover -func=coverage.out | grep total

echo "Tests completed successfully!"