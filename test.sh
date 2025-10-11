#!/usr/bin/env bash

# PHPV Test Script
# Basic functionality tests

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHPV_SCRIPT="$SCRIPT_DIR/phpv.sh"

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

run_test() {
    local test_name="$1"
    local command="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "Testing $test_name... "
    
    if eval "$command" &>/dev/null; then
        echo -e "${GREEN}PASS${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}FAIL${NC}"
    fi
}

echo "PHPV Test Suite"
echo "==============="

# Check if script exists and is executable
if [[ ! -f "$PHPV_SCRIPT" ]]; then
    echo -e "${RED}Error: phpv.sh not found${NC}"
    exit 1
fi

if [[ ! -x "$PHPV_SCRIPT" ]]; then
    echo -e "${YELLOW}Making phpv.sh executable...${NC}"
    chmod +x "$PHPV_SCRIPT"
fi

# Basic functionality tests
run_test "syntax check" "bash -n '$PHPV_SCRIPT'"
run_test "help command" "'$PHPV_SCRIPT' help"
run_test "current command" "'$PHPV_SCRIPT' current"
run_test "list command" "'$PHPV_SCRIPT' list"
run_test "list-available command" "'$PHPV_SCRIPT' list-available"
run_test "which command" "'$PHPV_SCRIPT' which"

# Test error handling
run_test "invalid command handling" "! '$PHPV_SCRIPT' invalid-command"
run_test "install without version" "! '$PHPV_SCRIPT' install"
run_test "use without version" "! '$PHPV_SCRIPT' use"

echo
echo "Test Results:"
echo "============="
echo "Tests run: $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$((TESTS_RUN - TESTS_PASSED))${NC}"

if [[ $TESTS_PASSED -eq $TESTS_RUN ]]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed.${NC}"
    exit 1
fi