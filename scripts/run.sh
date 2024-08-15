#!/bin/bash -ex

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."

ENV_PATH=${ENV_PATH:-"$ROOT_DIR/.env"}

command="$1"
extraArgs="$2"

export_envs() {
    readarray -t lines < $ENV_PATH
    for line in "${lines[@]}"; do
        printf "export %s\n" $line;
        export $line;
    done
}

if [ -f ${ENV_PATH} ]; then
    echo
    echo -e "Env file found. Load env file: $ENV_PATH"
    export_envs
    echo
fi

shopt -s globstar # for ** pattern matching
eval "${command} ${extraArgs}"
TEST_EXIT=$?
echo
