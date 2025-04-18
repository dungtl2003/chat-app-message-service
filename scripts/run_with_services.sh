#!/bin/bash

# This script is used to run any command with the necessary services running.
# The script will start the necessary services, wait for them to be healthy, 
# run the command, and stop the services when the program exits. By default,
# the script will use the docker-compose.yaml file in the compose directory
# and the .env file in the environments directory. You can change the
# compose file and the env file by setting the COMPOSE_FILE and ENV_FILE environment.
# You can also set the DEBUG environment variable to "true" to enable debug mode.

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."
DEBUG=${DEBUG:-"false"}

COMPOSE_DIR=${COMPOSE_DIR:-"$ROOT_DIR/compose"}
COMPOSE_FILE=${COMPOSE_FILE:-"docker-compose.yaml"}
COMPOSE_PATH=${COMPOSE_PATH:-"$COMPOSE_DIR/$COMPOSE_FILE"}

services=("snowflake" "database")
default_services=("true" "true")

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

# from SO: https://stackoverflow.com/a/54261882/317605 (by https://stackoverflow.com/users/8207842/dols3m)
function prompt_for_multiselect {

    # little helpers for terminal print control and key input
    ESC=$( printf "\033")
    cursor_blink_on()   { printf "$ESC[?25h"; }
    cursor_blink_off()  { printf "$ESC[?25l"; }
    cursor_to()         { printf "$ESC[$1;${2:-1}H"; }
    print_inactive()    { printf "$2   $1 "; }
    print_active()      { printf "$2  $ESC[7m $1 $ESC[27m"; }
    get_cursor_row()    { IFS=';' read -sdR -p $'\E[6n' ROW COL; echo ${ROW#*[}; }
    key_input()         {
      local key
      IFS= read -rsn1 key 2>/dev/null >&2
      if [[ $key = ""      ]]; then echo enter; fi;
      if [[ $key = $'\x20' ]]; then echo space; fi;
      if [[ $key = $'\x1b' ]]; then
        read -rsn2 key
        if [[ $key = [A ]]; then echo up;    fi;
        if [[ $key = [B ]]; then echo down;  fi;
      fi 
    }
    toggle_option()    {
      local arr_name=$1
      eval "local arr=(\"\${${arr_name}[@]}\")"
      local option=$2
      if [[ ${arr[option]} == true ]]; then
        arr[option]=
      else
        arr[option]=true
      fi
      eval $arr_name='("${arr[@]}")'
    }

    local retval=$1
    local options
    local defaults

    IFS=';' read -r -a options <<< "$2"
    if [[ -z $3 ]]; then
      defaults=()
    else
      IFS=';' read -r -a defaults <<< "$3"
    fi
    local selected=()

    for ((i=0; i<${#options[@]}; i++)); do
      selected+=("${defaults[i]:-false}")
      printf "\n"
    done

    # determine current screen position for overwriting the options
    local lastrow=`get_cursor_row`
    local startrow=$(($lastrow - ${#options[@]}))

    # ensure cursor and input echoing back on upon a ctrl+c during read -s
    trap "cursor_blink_on; stty echo; printf '\n'; exit" 2
    cursor_blink_off

    local active=0
    while true; do
        # print options by overwriting the last lines
        local idx=0
        for option in "${options[@]}"; do
            local prefix="[ ]"
            if [[ ${selected[idx]} == true ]]; then
              prefix="[x]"
            fi

            cursor_to $(($startrow + $idx))
            if [ $idx -eq $active ]; then
                print_active "$option" "$prefix"
            else
                print_inactive "$option" "$prefix"
            fi
            ((idx++))
        done

        # user key control
        case `key_input` in
            space)  toggle_option selected $active;;
            enter)  break;;
            up)     ((active--));
                    if [ $active -lt 0 ]; then active=$((${#options[@]} - 1)); fi;;
            down)   ((active++));
                    if [ $active -ge ${#options[@]} ]; then active=0; fi;;
        esac
    done

    # cursor position back to normal
    cursor_to $lastrow
    printf "\n"
    cursor_blink_on

    eval $retval='("${selected[@]}")'
}

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

run_command() {
    shopt -s globstar # for ** pattern matching
    echo "Running command: ${command} ${extraArgs}"
    eval "${command} ${extraArgs}"
    return $?
}

main() {
    echo "Choose services you want to check: "
    services_str=$(array_to_string "${services[@]}")
    default_services_str=$(array_to_string "${default_services[@]}")
    prompt_for_multiselect result "${services_str}" "${default_services_str}"

    for i in ${!result[@]}; do
        if [ "${result[$i]}" == "true" ]; then
            selected_services+=("${services[$i]}")
        fi
    done

    echo "Selected services:"
    for service in "${selected_services[@]}"; do
        echo "  ${service}"
    done

    run_containers
    wait_for_containers "${selected_services[@]}"
    if [[ ${?} -ne 0 ]]; then
        quit
    fi

    run_command
    if [[ ${?} -ne 0 ]]; then
        quit
    fi
    stop_containers
}

main
