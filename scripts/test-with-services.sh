#!/bin/bash -ex

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."
ENV_PATH=${ENV_PATH:-"$ROOT_DIR/.env.test"}

testCommand="$1"
extraArgs="$2"

export COMPOSE_FILE=${COMPOSE_FILE:="$ROOT_DIR/tests/docker-compose.yml"}

export_envs() {
    readarray -t lines < $ENV_PATH
    for line in "${lines[@]}"; do
        printf "export %s\n" $line;
        export $line;
    done
}

find_container_id() {
  echo $(docker ps \
    --filter "status=running" \
    --filter "label=custom.project=chat" \
    --filter "label=custom.service=id-generator" \
    --no-trunc \
    -q)
}

quit() {
  docker-compose -f "${COMPOSE_FILE}" down --remove-orphans -v
  exit 1
}

if [ -z ${DO_NOT_STOP} ]; then
  trap quit ERR
fi

if [ -f ${ENV_PATH} ]; then
    echo
    echo -e "Env file found. Load env file: $ENV_PATH"
    export_envs
    echo
fi

if [ -z "$(find_container_id)" ]; then
    echo -e "Start docker container(s)"
  NO_LOGS=1 "$SCRIPT_DIR/docker-compose-up.sh"
  if [ "1" = "$?" ]; then
    echo -e "Failed to start"
    exit 1
  fi
fi

tsx "$SCRIPT_DIR/wait-for-services.ts"
echo

set +x
echo
echo -e "Running tests with NODE_OPTIONS=${NODE_OPTIONS}"
echo -e "Heap size in MB:"
node -e "console.log((require('v8').getHeapStatistics().total_available_size / 1024 / 1024).toFixed(2))"
echo
set -x

shopt -s globstar # for ** pattern matching
eval "${testCommand} ${extraArgs}"
TEST_EXIT=$?
echo

if [ -z ${DO_NOT_STOP} ]; then
  docker-compose -f "${COMPOSE_FILE}" down --remove-orphans -v
fi
exit ${TEST_EXIT}

