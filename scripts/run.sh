#!/bin/bash -e

# This script is used to run any command with the environment variables loaded 
# from the .env file. Default .env file is located at environments/.env. 
# You can change the path by setting the ENV_FILE environment variable.
# You can also set the DEBUG environment variable to "true" to enable debug mode.

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."
ENV_DIR=${ENV_DIR:="$ROOT_DIR/environments"}
ENV_FILE=${ENV_FILE:-".env"}
ENV_PATH=${ENV_PATH:-"$ENV_DIR/$ENV_FILE"}
DEBUG=${DEBUG:-"false"}

command="$1"
extraArgs="${@:2}"

export_envs() {
    readarray -t lines < $ENV_PATH
    for line in "${lines[@]}"; do
        printf "export %s\n" $line;
        export $line;
    done
}

if [ "$DEBUG" == "true" ]; then
    echo "Debug mode is on"
    set -x
fi

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
