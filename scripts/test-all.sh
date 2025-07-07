#!/bin/bash
# Run all tests, linting, and security checks for Mnemosyne

# Parse command line arguments
VERBOSE=false
SECURITY=false
for arg in "$@"; do
    case $arg in
        -v|--verbose)
            VERBOSE=true
            ;;
        -s|--security)
            SECURITY=true
            ;;
    esac
done

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
        # Check if tests were skipped
        skip_count=$(grep -c "^--- SKIP:" /tmp/test_output 2>/dev/null || true)
        skip_count=${skip_count:-0}
        pass_count=$(grep -c "^--- PASS:" /tmp/test_output 2>/dev/null || true)
        pass_count=${pass_count:-0}
        
        if [ "$skip_count" -gt 0 ] && [ "$pass_count" -eq 0 ]; then
            # All tests were skipped
            echo -e "${YELLOW}⚠ SKIP${NC} - $name (all tests skipped)"
            RESULTS+=("SKIP: $name")
            
            # Show skip reasons when verbose
            if [ "$VERBOSE" = true ]; then
                echo -e "${YELLOW}Skipped tests:${NC}"
                grep -A1 "^--- SKIP:" /tmp/test_output | grep -v "^--$" | sed 's/^/  /' | while read line; do
                    if [[ "$line" == *"SKIP:"* ]]; then
                        echo -e "  ${YELLOW}○${NC} ${line#*SKIP: }"
                    else
                        echo -e "    ${YELLOW}↳${NC} $line"
                    fi
                done
            fi
        elif [ "$skip_count" -gt 0 ]; then
            # Some tests passed, some skipped
            echo -e "${GREEN}✓ PASS${NC} - $name ${YELLOW}($skip_count tests skipped)${NC}"
            RESULTS+=("PASS: $name (with $skip_count skips)")
            
            # Show skip reasons when verbose
            if [ "$VERBOSE" = true ]; then
                echo -e "${YELLOW}Skipped tests:${NC}"
                grep -A1 "^--- SKIP:" /tmp/test_output | grep -v "^--$" | sed 's/^/  /' | while read line; do
                    if [[ "$line" == *"SKIP:"* ]]; then
                        echo -e "  ${YELLOW}○${NC} ${line#*SKIP: }"
                    else
                        echo -e "    ${YELLOW}↳${NC} $line"
                    fi
                done
            fi
        else
            # All tests passed
            echo -e "${GREEN}✓ PASS${NC} - $name"
            RESULTS+=("PASS: $name")
        fi
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} - $name"
        
        # Extract failed test names from Go test output
        if [[ "$name" == *"unit tests"* ]] || [[ "$name" == *"integration tests"* ]]; then
            fail_count=$(grep -c "^--- FAIL:" /tmp/test_output 2>/dev/null || true)
            fail_count=${fail_count:-0}
            skip_count=$(grep -c "^--- SKIP:" /tmp/test_output 2>/dev/null || true)
            skip_count=${skip_count:-0}
            
            if [ "$fail_count" -gt 0 ]; then
                echo -e "${RED}Failed tests ($fail_count):${NC}"
                grep -E "^--- FAIL:|FAIL.*Test" /tmp/test_output | head -5 | while read line; do
                    echo -e "  ${RED}●${NC} ${line#*FAIL: }"
                done
                if [ "$fail_count" -gt 5 ]; then
                    echo -e "  ${RED}... and $((fail_count - 5)) more failures${NC}"
                fi
            fi
            
            if [ "$skip_count" -gt 0 ]; then
                echo -e "${YELLOW}Skipped tests ($skip_count)${NC}"
            fi
        fi
        
        # Show lint errors summary for linting
        if [[ "$name" == *"linting"* ]]; then
            echo -e "${RED}Linting errors:${NC}"
            grep -E "^[^:]+:[0-9]+:[0-9]+:" /tmp/test_output | head -10 | while read line; do
                echo -e "  ${RED}●${NC} $line"
            done
            error_count=$(grep -E "^[^:]+:[0-9]+:[0-9]+:" /tmp/test_output | wc -l)
            if [ $error_count -gt 10 ]; then
                echo -e "  ${YELLOW}... and $((error_count - 10)) more errors${NC}"
            fi
        fi
        
        if [ "$VERBOSE" = true ]; then
            echo ""
            echo "Full error output:"
            cat /tmp/test_output
        else
            echo ""
            echo -e "${YELLOW}Run with -v flag for full error output${NC}"
        fi
        RESULTS+=("FAIL: $name")
        return 1
    fi
}

echo "Running test suite for Mnemosyne..."
echo "=================================================="
echo "Test Categories:"
echo "  • Backend unit tests (with race detection)"
echo "  • Repository layer tests (mock & PostgreSQL)"
echo "  • Integration tests"
echo "  • Code quality (linting)"
echo "  • Frontend tests (type checking & build)"
if [ "$VERBOSE" = false ]; then
    echo ""
    echo "Tip: Use -v or --verbose to see details of skipped tests"
fi
echo "=================================================="

# Backend Tests - ALL tests (including those requiring external services)
run_test "Backend ALL tests (unit + integration)" "cd backend && go test -v -race -coverprofile=coverage.out ./..."

# PostgreSQL Integration Tests (only if PostgreSQL is available)
if command -v psql &> /dev/null && psql -U postgres -c '\q' 2>/dev/null; then
    run_test "PostgreSQL integration tests" "cd backend && go test -v ./internal/repository/postgres"
else
    echo -e "${YELLOW}⚠ SKIP${NC} - PostgreSQL integration tests (PostgreSQL not available)"
    RESULTS+=("SKIP: PostgreSQL integration tests")
fi

# Backend Linting
run_test "Backend linting" "cd backend && golangci-lint run --config=.golangci.yml --timeout=5m"

# Security Scanning
if [ "$SECURITY" = true ]; then
    if command -v gosec &> /dev/null; then
        run_test "Security scan" "cd backend && gosec -no-fail ./..."
    else
        echo -e "${YELLOW}⚠ SKIP${NC} - Security scan (gosec not installed)"
        RESULTS+=("SKIP: Security scan")
    fi
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

pass_count=0
fail_count=0
skip_count=0

for result in "${RESULTS[@]}"; do
    if [[ $result == PASS* ]]; then
        echo -e "${GREEN}✓ $result${NC}"
        ((pass_count++))
    elif [[ $result == FAIL* ]]; then
        echo -e "${RED}✗ $result${NC}"
        ((fail_count++))
    elif [[ $result == SKIP* ]]; then
        echo -e "${YELLOW}⚠ $result${NC}"
        ((skip_count++))
    else
        echo -e "${YELLOW}$result${NC}"
    fi
done

# Show totals
echo ""
echo -e "Totals: ${GREEN}$pass_count passed${NC}, ${RED}$fail_count failed${NC}, ${YELLOW}$skip_count skipped${NC}"

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
