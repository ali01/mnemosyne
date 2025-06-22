#!/bin/bash
# Run all tests, linting, and security checks for Mnemosyne

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track results
RESULTS=()

run_test() {
    local name="$1"
    local command="$2"

    echo ""
    echo -e "${BLUE}Running $name...${NC}"

    if (cd "$(dirname "$0")/.." && eval "$command") > /tmp/test_output 2>&1; then
        echo -e "${GREEN}✓ PASS${NC} - $name"
        RESULTS+=("PASS: $name")
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} - $name"
        echo "Error output:"
        cat /tmp/test_output
        RESULTS+=("FAIL: $name")
        return 1
    fi
}

echo "Running test suite for Mnemosyne..."
echo "=================================================="

# Backend Tests
run_test "Backend unit tests" "cd backend && go test -v -race -coverprofile=coverage.out ./..."

# Integration Tests
run_test "Integration tests" "cd backend && go test -v ./internal/vault/ -run TestIntegration"

# Backend Linting
run_test "Backend linting" "cd backend && golangci-lint run --config=.golangci.yml --timeout=5m"

# Security Scanning
if command -v gosec &> /dev/null; then
    run_test "Security scan" "cd backend && gosec -no-fail ./..."
else
    echo -e "${YELLOW}⚠ SKIP${NC} - Security scan (gosec not installed)"
    RESULTS+=("SKIP: Security scan")
fi

# Frontend Tests
run_test "Frontend dependencies" "cd frontend && npm ci"
run_test "Frontend type checking" "cd frontend && npm run check"
run_test "Frontend build" "cd frontend && npm run build"

# Summary
echo ""
echo "=================================================="
echo "TEST RESULTS SUMMARY:"
echo "=================================================="

for result in "${RESULTS[@]}"; do
    if [[ $result == PASS* ]]; then
        echo -e "${GREEN}$result${NC}"
    elif [[ $result == FAIL* ]]; then
        echo -e "${RED}$result${NC}"
    else
        echo -e "${YELLOW}$result${NC}"
    fi
done

# Coverage summary
PROJECT_ROOT="$(dirname "$0")/.."
if [ -f "$PROJECT_ROOT/backend/coverage.out" ]; then
    echo ""
    coverage_line=$(cd "$PROJECT_ROOT/backend" && go tool cover -func=coverage.out | grep total)
    coverage_percent=$(echo "$coverage_line" | awk '{print $3}')

    # Color based on coverage percentage
    coverage_num=$(echo "$coverage_percent" | sed 's/%//' | cut -d. -f1)
    if [ "$coverage_num" -ge 90 ]; then
        color="${GREEN}"
    elif [ "$coverage_num" -ge 70 ]; then
        color="${YELLOW}"
    else
        color="${RED}"
    fi

    echo -e "${BLUE}Coverage:${NC} ${color}${coverage_percent}${NC}"
fi

# Exit with failure if any tests failed
for result in "${RESULTS[@]}"; do
    if [[ $result == FAIL* ]]; then
        echo ""
        echo -e "${RED}Some tests failed. See output above for details.${NC}"
        exit 1
    fi
done

echo ""
echo -e "${GREEN}All tests passed successfully!${NC}"
