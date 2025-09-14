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

DATABASE_URL=${DATABASE_URL:-"postgresql://message_service:msg1234@localhost:6000/chat-app?sslmode=disable"}
PORT=${PORT:-8100}
LOG_LEVEL=${LOG_LEVEL:-"DEBUG"}
LOG_KIND=${LOG_KIND:-"TEXT"}
ID_GENERATOR_ADDR=${ID_GENERATOR_ADDR:-"localhost:9000"} # tls
ID_GENERATOR_CERT_DIR=${ID_GENERATOR_CERT_DIR:-"$ROOT_DIR/environments/test/message/services/snowflake/ssl/certs"}
ENVIRONMENT=${ENVIRONMENT:-"test"}
KAFKA_BROKERS=${KAFKA_BROKERS:-"localhost:29092,localhost:39092,localhost:49092"}
MEDIA_SERVICE_URL=${MEDIA_SERVICE_URL:-"http://localhost:8300"}

# Test's specific environment variables
ADMIN_DATABASE_URL=${ADMIN_DATABASE_URL:-"postgresql://admin:testpass123@localhost:6000/chat-app?sslmode=disable"}
MESSAGE_SERVICE_URL=${MESSAGE_SERVICE_URL:-"http://localhost:$PORT"}
ID_GENERATOR_TLS_ADDR=${ID_GENERATOR_TLS_ADDR:-"localhost:9000"}
ID_GENERATOR_NON_TLS_ADDR=${ID_GENERATOR_NON_TLS_ADDR:-"localhost:9001"}
ID_GENERATOR_FAKE_CERT_DIR=${ID_GENERATOR_FAKE_CERT_DIR:-"$ROOT_DIR/environments/test/message/services/snowflake/fake_ssl"}
DATA_FILE_DIR=${DATA_FILE_DIR:-"$ROOT_DIR/tests/data"}
# One working broker address is enough for the tests to run
BROKER_ADDR=${BROKER_ADDR:-"localhost:29092"}

LOG_META="
$ROOT_DIR/tests/logs/database_service.log=chat-app-db-service;
$ROOT_DIR/tests/logs/media_service.log=chat-app-media-service;
$ROOT_DIR/tests/logs/snowflake_tls_service.log=chat-app-snowflake-tls-service;
$ROOT_DIR/tests/logs/snowflake_non_tls_service.log=chat-app-snowflake-non-tls-service;
$ROOT_DIR/tests/logs/topic_init_service.log=chat-app-kafka-topics-init;
$ROOT_DIR/tests/logs/controller_1.log=chat-app-kafka-controller-1;
$ROOT_DIR/tests/logs/controller_2.log=chat-app-kafka-controller-2;
$ROOT_DIR/tests/logs/controller_3.log=chat-app-kafka-controller-3;
$ROOT_DIR/tests/logs/broker_1.log=chat-app-kafka-broker-1;
$ROOT_DIR/tests/logs/broker_2.log=chat-app-kafka-broker-2;
$ROOT_DIR/tests/logs/broker_3.log=chat-app-kafka-broker-3
"

command="$1"
extraArgs="${@:2}"

if [ "$DEBUG" == "true" ]; then
    echo "Debug mode is on"
    set -x
fi

function export_envs() {
    printf "export COMPOSE_FILE=%s\n" $COMPOSE_FILE
    export COMPOSE_FILE

    printf "export DATABASE_URL=%s\n" $DATABASE_URL
    export DATABASE_URL
    printf "export PORT=%s\n" $PORT
    export PORT
    printf "export LOG_LEVEL=%s\n" $LOG_LEVEL
    export LOG_LEVEL
    printf "export LOG_KIND=%s\n" $LOG_KIND
    export LOG_KIND
    printf "export ENVIRONMENT=%s\n" $ENV
    export ENVIRONMENT
    printf "export ID_GENERATOR_ADDR=%s\n" $ID_GENERATOR_ADDR
    export ID_GENERATOR_ADDR
    printf "export ID_GENERATOR_CERT_DIR=%s\n" $ID_GENERATOR_CERT_DIR
    export ID_GENERATOR_CERT_DIR
    printf "export KAFKA_BROKERS=%s\n" $KAFKA_BROKERS
    export KAFKA_BROKERS
    printf "export MEDIA_SERVICE_URL=%s\n" $MEDIA_SERVICE_URL
    export MEDIA_SERVICE_URL

    printf "export ADMIN_DATABASE_URL=%s\n" $ADMIN_DATABASE_URL
    export ADMIN_DATABASE_URL
    printf "export MESSAGE_SERVICE_URL=%s\n" $MESSAGE_SERVICE_URL
    export MESSAGE_SERVICE_URL 
    printf "export ID_GENERATOR_TLS_ADDR=%s\n" $ID_GENERATOR_TLS_ADDR
    export ID_GENERATOR_TLS_ADDR
    printf "export ID_GENERATOR_NON_TLS_ADDR=%s\n" $ID_GENERATOR_NON_TLS_ADDR
    export ID_GENERATOR_NON_TLS_ADDR
    printf "export ID_GENERATOR_FAKE_CERT_DIR=%s\n" $ID_GENERATOR_FAKE_CERT_DIR
    export ID_GENERATOR_FAKE_CERT_DIR
    printf "export DATA_FILE_DIR=%s\n" $DATA_FILE_DIR
    export DATA_FILE_DIR    
    printf "export BROKER_ADDR=%s\n" $BROKER_ADDR
    export BROKER_ADDR
    printf "export LOG_META=%s\n" "$LOG_META"
    export LOG_META
}

function main() {
    export_envs
    $SCRIPT_DIR/__test_with_services.sh "$command" $extraArgs
}

main
