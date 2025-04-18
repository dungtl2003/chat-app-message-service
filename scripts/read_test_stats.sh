#!/bin/bash

set -e

# This script is used to read the test stats from the output file and print the 
# total number of tests, the number of failed tests, and the list of failed tests. 
# The script uses jq to parse the JSON file and extract the test stats.
# You need to run tests and save the output to a file in JSON format before 
# running this script.
# You can also set the DEBUG environment variable to "true" to enable debug mode.
# The first argument is the path to the output file.

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."
TEMP_FAILED_TESTS_FILE="/tmp/failed_tests.json"
TEMP_TOTAL_TESTS_FILE="/tmp/total_tests.json"
TEMP_OUTPUT_FILE="/tmp/output.json"
DEBUG=${DEBUG:-"false"}

INPUT_FILE_PATH="$1"

if [ -z "$INPUT_FILE_PATH" ]; then
    echo "Usage: $0 <input_file_path>"
    exit 1
fi

if [ "$DEBUG" == "true" ]; then
    echo "Debug mode is on"
    set -x
fi

# This script need jq to be installed
# Check if jq is installed
if ! [ -x "$(command -v jq)" ]; then
  echo 'Error: jq is not installed.' >&2
  exit 1
fi

# Check if the file exists
if [ ! -f ${INPUT_FILE_PATH} ]; then
    echo "File not found: $INPUT_FILE_PATH"
    exit 1
fi

# Remove lines that don't start with { or [ (json objects), and lines with only whitespace
sed -i '/^$$/d; /^[^{[]/d' $INPUT_FILE_PATH

jq -s '.' < $INPUT_FILE_PATH > $TEMP_OUTPUT_FILE

jq '.[] | select(.Output | type == "string" and startswith("=== RUN"))' < $TEMP_OUTPUT_FILE > $TEMP_TOTAL_TESTS_FILE
jq '.[] | select(.Output | type == "string" and startswith("--- FAIL"))' < $TEMP_OUTPUT_FILE > $TEMP_FAILED_TESTS_FILE

# Reformat the output. We don't need TEMP_OUTPUT_FILE anymore, so we can use it to store the result temporarily
jq -s '.' < $TEMP_TOTAL_TESTS_FILE > $TEMP_OUTPUT_FILE
cat $TEMP_OUTPUT_FILE > $TEMP_TOTAL_TESTS_FILE

jq -s '.' < $TEMP_FAILED_TESTS_FILE > $TEMP_OUTPUT_FILE
cat $TEMP_OUTPUT_FILE > $TEMP_FAILED_TESTS_FILE

total_tests=$(jq length < $TEMP_TOTAL_TESTS_FILE)
failed_tests=$(jq length < $TEMP_FAILED_TESTS_FILE)
ok_tests=$((total_tests-failed_tests))

printf "Total tests: %s, OK: %s, Failed: %s\n" $total_tests $ok_tests $failed_tests
echo "Failed tests:"
jq < $TEMP_FAILED_TESTS_FILE
