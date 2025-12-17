#!/bin/bash

set -e

# Usage: ./summarize_tests.sh <path_to_output.json>
#
# This script parses the JSON output from 'go test -json' to provide a summary
# of passed, failed, and skipped tests.
#
# It relies on the 'Action' field in the JSON objects rather than parsing
# raw text output, making it robust for current Go versions.

INPUT_FILE_PATH="$1"

# Colors for better visibility
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# 1. Validation
if [ -z "$INPUT_FILE_PATH" ]; then
    echo "Usage: $0 <input_file_path>"
    exit 1
fi

if ! [ -x "$(command -v jq)" ]; then
    echo "Error: jq is not installed." >&2
    exit 1
fi

if [ ! -f "${INPUT_FILE_PATH}" ]; then
    echo "File not found: ${INPUT_FILE_PATH}"
    exit 1
fi

# 2. Extract Data using jq
# We filter the input using grep to ensure only lines starting with '{' are passed to jq.
# This avoids errors if the log file contains non-JSON noise (like compiler warnings).

# A. Calculate Counts
# We look for lines where 'Test' is defined (ignoring package summaries for the count)
# and the Action is one of the final states: pass, fail, or skip.
STATS=$(grep '^{.*}' "$INPUT_FILE_PATH" | jq -n '
  reduce inputs as $i (
    {pass: 0, fail: 0, skip: 0};
    if $i.Test != null and ($i.Action == "pass" or $i.Action == "fail" or $i.Action == "skip")
    then .[$i.Action] += 1
    else .
    end
  )
')

PASS_COUNT=$(echo "$STATS" | jq '.pass')
FAIL_COUNT=$(echo "$STATS" | jq '.fail')
SKIP_COUNT=$(echo "$STATS" | jq '.skip')
TOTAL_TESTS=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))

# B. Extract Failed Tests (Action="fail" AND Test name exists)
FAILED_TESTS=$(grep '^{.*}' "$INPUT_FILE_PATH" | jq -r 'select(.Action == "fail" and .Test != null) | "\(.Package): \(.Test)"')

# C. Extract Package Failures (Action="fail" AND Test name is null - e.g., build failed)
FAILED_PACKAGES=$(grep '^{.*}' "$INPUT_FILE_PATH" | jq -r 'select(.Action == "fail" and .Test == null) | .Package')

# 3. Print Results

echo "---------------------------------------------------"
printf "${BOLD}Test Summary for: ${NC} %s\n" "$INPUT_FILE_PATH"
echo "---------------------------------------------------"

# Print Totals
printf "Total Tests: %s\n" "$TOTAL_TESTS"
printf "Passed:      ${GREEN}%s${NC}\n" "$PASS_COUNT"
printf "Skipped:     ${YELLOW}%s${NC}\n" "$SKIP_COUNT"

if [ "$FAIL_COUNT" -gt 0 ]; then
    printf "Failed:      ${RED}%s${NC}\n" "$FAIL_COUNT"
else
    printf "Failed:      %s\n" "$FAIL_COUNT"
fi
echo "---------------------------------------------------"

# Print List of Failed Tests
if [ -n "$FAILED_TESTS" ]; then
    printf "${RED}${BOLD}Failed Tests List:${NC}\n"
    echo "$FAILED_TESTS" | sed 's/^/  - /'
    echo ""
fi

# Print List of Failed Packages (Build errors, timeouts, etc.)
if [ -n "$FAILED_PACKAGES" ]; then
    printf "${RED}${BOLD}Package Level Failures (Build/Timeout):${NC}\n"
    echo "$FAILED_PACKAGES" | sed 's/^/  - /'
    echo ""
fi

# Exit with status 1 if there were failures, useful for CI/CD
if [ "$FAIL_COUNT" -gt 0 ] || [ -n "$FAILED_PACKAGES" ]; then
    exit 1
fi

exit 0
