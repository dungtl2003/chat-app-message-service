#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."
DEBUG=${DEBUG:-"false"}

COMPOSE_DIR=${COMPOSE_DIR:-"$ROOT_DIR/compose"}
COMPOSE_FILE=${COMPOSE_FILE:-"docker-compose.yaml"}
COMPOSE_PATH=${COMPOSE_PATH:-"$COMPOSE_DIR/$COMPOSE_FILE"}

services=("snowflake" "database")

command="$1"
extraArgs="${@:2}"

quit() {
  echo "Stopping containers..."
  stop_containers
  exit 0
}

if [ "$DEBUG" == "true" ]; then
    echo "Debug mode is on"
    set -x
fi

# Trap the signals to stop the containers (e.g., CTRL+C)
trap quit SIGINT SIGTERM EXIT

# If DO_NOT_STOP is not set, trap the ERR signal to stop the containers (every command that fails will trigger the trap)
if [ -z "$DO_NOT_STOP" ]; then
  trap quit ERR
fi

function array_to_string() {
    str=""
    arr=("$@")
    for s in "${arr[@]}"; do
        str+="$s;"
    done

    echo $str
}

run_containers() {
  echo "Running compose file: ${COMPOSE_PATH}:"
  docker compose -f "${COMPOSE_PATH}" up --force-recreate -d
}

stop_containers() {
  docker-compose -f "${COMPOSE_PATH}" down --remove-orphans -v
}

find_container_id() {
    local node=$1
    local option=$2
    local cmd="docker ps \
        --filter \"status=running\" \
        --filter \"label=custom.project=chat\" \
        --filter \"label=custom.service=${node}\"" 


    if [ -n "${option}" ]; then
        cmd="${cmd} --filter \"label=custom.option=${option}\""
    fi

    cmd="${cmd} --no-trunc -q"

    container_id=$(eval ${cmd})
    echo ${container_id}
}

wait_for_container() {
    local container_id=$1
    local retries=10
    local delay=5

    for ((i=1; i<=${retries}; i++)); do
        health_status=$(docker inspect --format='{{.State.Health.Status}}' ${container_id} 2>/dev/null)
        echo "Health status for container ${container_id}: ${health_status}"

        if [ "${health_status}" == "healthy" ]; then
            echo "Container ${container_id} is healthy"
            return 0
        fi

        echo "Container ${container_id} is not healthy yet (attempt ${i}/${retries})"
        if [ ${i} -lt ${retries} ]; then
            echo "Retrying to check health status of container ${container_id} in ${delay} seconds..."
            sleep ${delay}
        fi
    done

    echo "Container ${container_id} is not healthy after ${retries} retries"
    return 1
}

wait_for_containers() {
    services=("$@")
    container_ids=()
    pids=()

    for service in "${services[@]}"; do
        # service = "node option" (option can be optional)
        node=$(echo ${service} | cut -d ' ' -f 1)
        if [[ ${service} == *" "* ]]; then
            option=$(echo ${service} | cut -d ' ' -f 2)
        else
            option=""
        fi

        echo "Finding container ID for ${service} service..." 
        container_id=$(find_container_id ${node} ${option})
        container_ids+=(${container_id})
    done

    for i in ${!container_ids[@]}; do
        echo "Container ID for ${services[$i]}: ${container_ids[$i]}"
    done

    echo -e "\nWaiting for containers to be healthy..."
    for container_id in ${container_ids[@]}; do
        wait_for_container ${container_id} &
        pids+=($!)
    done

    for pid in ${pids[@]}; do
        wait $pid
        status=$?
        if [ $status -ne 0 ]; then
            echo "One or more containers failed to start"
            return 1
        fi
    done

    echo "All containers are healthy"
    docker compose -f ${COMPOSE_PATH} ps
    return 0
}

create_topic() {
    local topic_name=$1
    local container_name=$2
    local broker_port=${3:-9092}
    local partitions=${4:-3}
    local replication_factor=${5:-2}

    echo "Creating topic ${topic_name} on broker ${container_name} with ${partitions} partitions and replication factor ${replication_factor}"
    docker exec -i ${container_name} /opt/kafka/bin/kafka-topics.sh --create --topic ${topic_name} --bootstrap-server localhost:${broker_port} --partitions ${partitions} --replication-factor ${replication_factor} --if-not-exists
}

run_command() {
    shopt -s globstar # for ** pattern matching
    echo "Running command: ${command} ${extraArgs}"
    eval "${command} ${extraArgs}"
    return $?
}

main() {
    echo "Services:"
    for service in "${services[@]}"; do
        echo "  ${service}"
    done

    run_containers
    wait_for_containers "${services[@]}"
    if [[ ${?} -ne 0 ]]; then
        quit
    fi

    # create_topic "participant-avatar-updates" "${access_broker}" "19092" "3" "2"

    # press enter to continue
    # read -p "Press Enter to continue..."

    run_command
    if [[ ${?} -ne 0 ]]; then
        quit
    fi
    stop_containers
}

main
