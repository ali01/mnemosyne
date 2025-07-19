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
BACKEND_TEST_OUTPUT=""

run_test() {
    local name="$1"
    local command="$2"

    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}▸ Running $name...${NC}"

    if (cd "$(dirname "$0")/.." && eval "$command") > /tmp/test_output 2>&1; then
        # Save backend test output for later analysis
        if [[ "$name" == *"Backend"* ]] && [[ "$name" == *"tests"* ]]; then
            cp /tmp/test_output /tmp/backend_test_output
        fi

        # Check if tests were skipped (including sub-tests with indentation)
        skip_count=$(grep -c -- "--- SKIP:" /tmp/test_output 2>/dev/null || true)
        skip_count=${skip_count:-0}
        pass_count=$(grep -c "^--- PASS:" /tmp/test_output 2>/dev/null || true)
        pass_count=${pass_count:-0}

        if [ "$skip_count" -gt 0 ] && [ "$pass_count" -eq 0 ]; then
            # All tests were skipped
            echo -e "${YELLOW}⚠ SKIP${NC} - $name (all tests skipped)"
            RESULTS+=("SKIP: $name")

            # Always show skipped test names (including sub-tests)
            echo -e "${YELLOW}┌─ Skipped tests:${NC}"
            grep -- "--- SKIP:" /tmp/test_output | sed 's/.*--- SKIP: //' | awk '{print $1}' | sort | uniq | while read test_name; do
                echo -e "${YELLOW}│${NC}  ○ $test_name"
                # Show skip reason if verbose
                if [ "$VERBOSE" = true ]; then
                    skip_reason=$(grep -A1 -- "--- SKIP: $test_name" /tmp/test_output | tail -1 | sed 's/^[[:space:]]*//')
                    if [ -n "$skip_reason" ] && [ "$skip_reason" != "--" ]; then
                        echo -e "${YELLOW}│${NC}    └─ $skip_reason"
                    fi
                fi
            done
            echo -e "${YELLOW}└────────────────────────────────────────────────────────────${NC}"
        elif [ "$skip_count" -gt 0 ]; then
            # Some tests passed, some skipped
            echo -e "${GREEN}✓ PASS${NC} - $name ${YELLOW}($skip_count tests skipped)${NC}"
            RESULTS+=("PASS: $name (with $skip_count skips)")

            # Always show skipped test names (including sub-tests)
            echo -e "${YELLOW}┌─ Skipped tests:${NC}"
            grep -- "--- SKIP:" /tmp/test_output | sed 's/.*--- SKIP: //' | awk '{print $1}' | sort | uniq | while read test_name; do
                echo -e "${YELLOW}│${NC}  ○ $test_name"
                # Show skip reason if verbose
                if [ "$VERBOSE" = true ]; then
                    skip_reason=$(grep -A1 -- "--- SKIP: $test_name" /tmp/test_output | tail -1 | sed 's/^[[:space:]]*//')
                    if [ -n "$skip_reason" ] && [ "$skip_reason" != "--" ]; then
                        echo -e "${YELLOW}│${NC}    └─ $skip_reason"
                    fi
                fi
            done
            echo -e "${YELLOW}└────────────────────────────────────────────────────────────${NC}"
        else
            # All tests passed
            echo -e "${GREEN}✓ PASS${NC} - $name"
            RESULTS+=("PASS: $name")
        fi
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} - $name"

        # Extract failed test names from Go test output
        if [[ "$name" == *"tests"* ]] || [[ "$name" == *"integration"* ]]; then
            fail_count=$(grep -c "^--- FAIL:" /tmp/test_output 2>/dev/null || true)
            fail_count=${fail_count:-0}
            skip_count=$(grep -c -- "--- SKIP:" /tmp/test_output 2>/dev/null || true)
            skip_count=${skip_count:-0}

            if [ "$fail_count" -gt 0 ]; then
                echo -e "${RED}┌─ Failed tests ($fail_count):${NC}"
                grep "^--- FAIL:" /tmp/test_output | sed 's/--- FAIL: //' | awk '{print $1}' | sort | uniq | while read test_name; do
                    echo -e "${RED}│${NC}  ● $test_name"
                    # Show fail reason if verbose
                    if [ "$VERBOSE" = true ]; then
                        fail_reason=$(grep -A5 "^--- FAIL: $test_name" /tmp/test_output | grep -E "Error:|error:|expected|got" | head -1 | sed 's/^[[:space:]]*//')
                        if [ -n "$fail_reason" ]; then
                            echo -e "${RED}│${NC}    └─ $fail_reason"
                        fi
                    fi
                done
                echo -e "${RED}└────────────────────────────────────────────────────────────${NC}"
            fi

            if [ "$skip_count" -gt 0 ]; then
                echo -e "${YELLOW}┌─ Also skipped ($skip_count tests):${NC}"
                grep -- "--- SKIP:" /tmp/test_output | sed 's/.*--- SKIP: //' | awk '{print $1}' | sort | uniq | head -5 | while read test_name; do
                    echo -e "${YELLOW}│${NC}  ○ $test_name"
                done
                if [ "$skip_count" -gt 5 ]; then
                    echo -e "${YELLOW}│${NC}  ... and $((skip_count - 5)) more"
                fi
                echo -e "${YELLOW}└────────────────────────────────────────────────────────────${NC}"
            fi
        fi

        # Show lint errors summary for linting
        if [[ "$name" == *"linting"* ]]; then
            echo -e "${RED}┌─ Linting errors:${NC}"
            grep -E "^[^:]+:[0-9]+:[0-9]+:" /tmp/test_output | head -10 | while read line; do
                echo -e "${RED}│${NC}  ● $line"
            done
            error_count=$(grep -E "^[^:]+:[0-9]+:[0-9]+:" /tmp/test_output | wc -l)
            if [ $error_count -gt 10 ]; then
                echo -e "${RED}│${NC}  ... and $((error_count - 10)) more errors"
            fi
            echo -e "${RED}└────────────────────────────────────────────────────────────${NC}"
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

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}              MNEMOSYNE TEST SUITE RUNNER                     ${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}Test Categories:${NC}"
echo "  • Backend unit tests (with race detection)"
echo "  • Repository layer tests (mock & PostgreSQL)"
echo "  • Integration tests"
echo "  • Code quality (linting)"
echo "  • Frontend tests (type checking & build)"
echo ""
if [ "$VERBOSE" = false ]; then
    echo -e "${YELLOW}💡 Tip:${NC} Use -v or --verbose to see skip reasons"
fi
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Backend Tests - ALL tests (including those requiring external services)
run_test "Backend ALL tests (unit + integration)" "cd backend && go test -v -race -coverprofile=coverage.out ./..."

# PostgreSQL Integration Tests
# Check if Docker is available for testcontainers
if command -v docker &> /dev/null && docker info &> /dev/null 2>&1; then
    # Docker is available, run full integration tests
    run_test "PostgreSQL integration tests" "cd backend && go test -v ./internal/repository/postgres"
else
    # Docker not available, run with -short to see which tests would be skipped
    run_test "PostgreSQL integration tests (Docker not available)" "cd backend && go test -v -short ./internal/repository/postgres"
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
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}                    TEST RESULTS SUMMARY                      ${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

pass_count=0
fail_count=0
skip_count=0

# Track all skipped test names and count
SKIPPED_TESTS=()
total_skipped_tests=0

# Gather skipped tests from backend test output
if [ -f /tmp/backend_test_output ]; then
    # Count total skipped tests (including sub-tests)
    total_skipped_tests=$(grep -c -- "--- SKIP:" /tmp/backend_test_output 2>/dev/null || echo 0)
    # Ensure it's a clean integer
    total_skipped_tests=${total_skipped_tests//[^0-9]/}
    total_skipped_tests=${total_skipped_tests:-0}

    # Get skipped test names
    while IFS= read -r test_name; do
        SKIPPED_TESTS+=("$test_name")
    done < <(grep -- "--- SKIP:" /tmp/backend_test_output | sed 's/.*--- SKIP: //' | awk '{print $1}' | sort | uniq)
fi

for result in "${RESULTS[@]}"; do
    if [[ $result == PASS* ]]; then
        echo -e "${GREEN}  ✓ $result${NC}"
        ((pass_count++))
    elif [[ $result == FAIL* ]]; then
        echo -e "${RED}  ✗ $result${NC}"
        ((fail_count++))
    elif [[ $result == SKIP* ]]; then
        echo -e "${YELLOW}  ⚠ $result${NC}"
        ((skip_count++))
    else
        echo -e "${YELLOW}  $result${NC}"
    fi
done

# Show totals with visual bar
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
total_count=$((pass_count + fail_count + skip_count))
if [ $total_count -gt 0 ]; then
    pass_pct=$((pass_count * 100 / total_count))
    fail_pct=$((fail_count * 100 / total_count))
    skip_pct=$((skip_count * 100 / total_count))

    echo -e "  Total: ${total_count} test suites"
    echo -e "  ${GREEN}Passed: $pass_count ($pass_pct%)${NC} | ${RED}Failed: $fail_count ($fail_pct%)${NC} | ${YELLOW}Skipped: $skip_count ($skip_pct%)${NC}"
else
    echo -e "  Total: ${total_count} test suites"
fi

# Show individual skipped tests count if any
if [ "$total_skipped_tests" -gt 0 ]; then
    echo ""
    echo -e "  ${YELLOW}Individual tests skipped: $total_skipped_tests${NC}"
fi

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
