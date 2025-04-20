#!/bin/bash -e

# This script is used to run any command with environment variables specific to
# the test environment. By default, it will also run all necessary services in
# docker-compose.test.yaml file. You can change the compose file by setting the
# COMPOSE_FILE environment variable. You can also set the DEBUG environment variable
# to "true" to enable debug mode.

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."
TEST_DIR=${TEST_DIR:-"$ROOT_DIR/tests"}
DEBUG=${DEBUG:-"false"}
COMPOSE_FILE=${COMPOSE_FILE:-"docker-compose.test.yaml"}

PORT=${PORT:-8100}
ENV=${ENV:-"test"}
LOG_LEVEL=${LOG_LEVEL:-"DEBUG"}
LOG_KIND=${LOG_KIND:-"TEXT"}
ID_GENERATOR_SERVICE_ADDR=${ID_GENERATOR_SERVICE_ADDR:-"localhost:9000"}
DATABASE_URL=${DATABASE_URL:-"postgresql://message_service:msg1234@localhost:6000/chat-app?sslmode=disable"}
ID_GENERATOR_SERVICE_CERT_DIR=${ID_GENERATOR_SERVICE_CERT_DIR:-"$ROOT_DIR/environments/test/message/services/snowflake/ssl/certs"}

# Test's specific environment variables
ADMIN_DATABASE_URL=${ADMIN_DATABASE_URL:-"postgresql://admin:testpass123@localhost:6000/chat-app?sslmode=disable"}
MESSAGE_SERVICE_URL=${MESSAGE_SERVICE_URL:-"http://localhost:$PORT"}

command="$1"
extraArgs="${@:2}"

if [ "$DEBUG" == "true" ]; then
    echo "Debug mode is on"
    set -x
fi

function export_envs() {
    printf "export COMPOSE_FILE=%s\n" $COMPOSE_FILE
    export COMPOSE_FILE

    printf "export PORT=%s\n" $PORT
    export PORT
    printf "export LOG_LEVEL=%s\n" $LOG_LEVEL
    export LOG_LEVEL
    printf "export LOG_KIND=%s\n" $LOG_KIND
    export LOG_KIND
    printf "export ENV=%s\n" $ENV
    export ENV
    printf "export ID_GENERATOR_SERVICE_ADDR=%s\n" $ID_GENERATOR_SERVICE_ADDR
    export ID_GENERATOR_SERVICE_ADDR
    printf "export ID_GENERATOR_SERVICE_CERT_DIR=%s\n" $ID_GENERATOR_SERVICE_CERT_DIR
    export ID_GENERATOR_SERVICE_CERT_DIR 
    printf "export DATABASE_URL=%s\n" $DATABASE_URL
    export DATABASE_URL

    printf "export ADMIN_DATABASE_URL=%s\n" $ADMIN_DATABASE_URL
    export ADMIN_DATABASE_URL
    printf "export MESSAGE_SERVICE_URL=%s\n" $MESSAGE_SERVICE_URL
    export MESSAGE_SERVICE_URL 
}

function main() {
    export_envs
    $SCRIPT_DIR/run_with_services.sh "$command" $extraArgs
}

main
