#!/bin/bash

# --- Configuration ---
# Use current directory if variables aren't set
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="${SCRIPT_DIR}/.."
COMPOSE_DIR="${COMPOSE_DIR:-"$ROOT_DIR/compose"}"
COMPOSE_FILE="${COMPOSE_FILE:-"docker-compose.yaml"}"
COMPOSE_PATH="${COMPOSE_PATH:-"$COMPOSE_DIR/$COMPOSE_FILE"}"

# Health check settings
MAX_RETRIES=20   # 20 * 3s = 60s timeout
SLEEP_DELAY=3

# Debug mode
DEBUG="${DEBUG:-false}"

# Define the services to wait for.
# Format: "service_label option_label" (option is optional)
DEPENDENCY_SERVICES=(
    "snowflake TLS"
    "snowflake non-TLS"
    "database"
    "controller 1"
    "controller 2"
    "controller 3"
    "broker 1"
    "broker 2"
    "broker 3"
    "topics-init"
)

# --- Colors for Output ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# --- Setup ---
set -o pipefail # Fail if any part of a pipe fails

if [ "$DEBUG" == "true" ]; then
    echo -e "${YELLOW}Debug mode is on${NC}"
    set -x
fi

# Arguments
COMMAND="$1"
shift
EXTRA_ARGS=("$@")

# --- Functions ---

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"
}

cleanup() {
    EXIT_CODE=$?
    echo ""
    log "${YELLOW}Stopping containers...${NC}"
    # Stop containers, remove orphans and volumes
    docker compose -f "${COMPOSE_PATH}" down --remove-orphans -v > /dev/null 2>&1

    if [ $EXIT_CODE -ne 0 ]; then
        log "${RED}Script failed with exit code ${EXIT_CODE}${NC}"
    else
        log "${GREEN}Script finished successfully.${NC}"
    fi
    exit $EXIT_CODE
}

# Trap signals.
# We don't trap ERR if DO_NOT_STOP is set, allowing manual debugging.
trap cleanup SIGINT SIGTERM EXIT
if [ -z "$DO_NOT_STOP" ]; then
    trap cleanup ERR
fi

run_containers() {
    log "Running compose file: ${COMPOSE_PATH}"
    docker compose -f "${COMPOSE_PATH}" up --force-recreate -d
}

find_container_id() {
    local node="$1"
    local option="$2"

    # improved docker ps command avoiding eval
    local args=(docker ps --filter "status=running" --filter "label=custom.project=chat" --filter "label=custom.service=${node}")

    if [ -n "${option}" ]; then
        args+=(--filter "label=custom.option=${option}")
    fi

    args+=(--no-trunc -q)

    "${args[@]}"
}

wait_for_single_container() {
    local container_id="$1"
    local description="$2"

    for ((i=1; i<=MAX_RETRIES; i++)); do
        # Check health
        local health_status
        health_status=$(docker inspect --format='{{.State.Health.Status}}' "${container_id}" 2>/dev/null)

        if [ "$health_status" == "healthy" ]; then
            log "${GREEN}✔ ${description} (${container_id}) is healthy${NC}"
            return 0
        fi

        # If container died or doesn't exist anymore
        if [ -z "$health_status" ]; then
            log "${RED}✘ ${description} (${container_id}) appears to have crashed or stopped.${NC}"
            return 1
        fi

        log "Waiting for ${description}... (${i}/${MAX_RETRIES}) [Status: ${health_status}]"
        sleep "${SLEEP_DELAY}"
    done

    log "${RED}Timeout waiting for ${description} (${container_id})${NC}"
    return 1
}

wait_for_all_containers() {
    local services=("${@}")
    local container_ids=()
    local pids=()
    local map_descriptions=()
    local errors=0

    log "Resolving container IDs..."

    # 1. Resolve all IDs first
    for service_str in "${services[@]}"; do
        # Split string "node option"
        read -r node option <<< "$service_str"

        id=$(find_container_id "$node" "$option")

        if [ -z "$id" ]; then
            log "${RED}Could not find running container for: $service_str${NC}"
            errors=1
        else
            container_ids+=("$id")
            map_descriptions+=("$service_str")
        fi
    done

    if [ $errors -ne 0 ]; then
        log "${RED}Aborting: Could not find all required containers.${NC}"
        return 1
    fi

    log "Waiting for ${#container_ids[@]} containers to be healthy..."

    # 2. Wait in parallel
    for i in "${!container_ids[@]}"; do
        wait_for_single_container "${container_ids[$i]}" "${map_descriptions[$i]}" &
        pids+=($!)
    done

    # 3. Collect results
    for pid in "${pids[@]}"; do
        wait "$pid"
        if [ $? -ne 0 ]; then
            errors=1
        fi
    done

    if [ $errors -ne 0 ]; then
        log "${RED}One or more containers failed to become healthy.${NC}"
        # Dump logs for debugging before exiting
        docker compose -f "${COMPOSE_PATH}" ps
        return 1
    fi

    log "${GREEN}All dependency containers are healthy.${NC}"
    return 0
}

run_target_command() {
    if [ -z "$COMMAND" ]; then
        log "${YELLOW}No command provided. Exiting.${NC}"
        return 0
    fi

    log "------------------------------------------------"
    log "Executing: ${COMMAND} ${EXTRA_ARGS[*]}"
    log "------------------------------------------------"

    # Enable globstar for patterns like **/*.js if needed by the command
    shopt -s globstar

    # Execute command preserving spaces in arguments
    "$COMMAND" "${EXTRA_ARGS[@]}"
}

# --- Main Execution ---

main() {
    # Check if docker is available
    if ! command -v docker &> /dev/null; then
        echo "Error: docker could not be found."
        exit 1
    fi

    run_containers

    wait_for_all_containers "${DEPENDENCY_SERVICES[@]}"

    # If wait failed, the trap will catch it due to 'set -e' or explicit return check
    # However, since we are inside a function, we handle return 1 explicitly
    if [ $? -ne 0 ]; then
        exit 1
    fi

    run_target_command
}

main
