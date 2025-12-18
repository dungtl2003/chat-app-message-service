#!/bin/bash

# Exit immediately if a command exits with a non-zero status
# Treat unset variables as an error
# Return the exit status of the last command in the pipe that failed
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="${SCRIPT_DIR}/.."

# --- Environment Configuration ---

# Mark all subsequently defined variables for export automatically
set -a

# Infrastructure & Paths
DEBUG=${DEBUG:-"false"}
TEST_DIR=${TEST_DIR:-"$ROOT_DIR/tests"}
COMPOSE_FILE=${COMPOSE_FILE:-"docker-compose.test.yaml"}
ENVIRONMENT="${ENVIRONMENT:-test}"

# Service Configuration
DATABASE_URL=${DATABASE_URL:-"postgresql://message_service:msg1234@localhost:6000/chat-app?sslmode=disable"}
PORT=${PORT:-8100}
LOG_LEVEL=${LOG_LEVEL:-"DEBUG"}
LOG_KIND=${LOG_KIND:-"TEXT"}
ID_GENERATOR_ADDR=${ID_GENERATOR_ADDR:-"localhost:9000"} # tls
ID_GENERATOR_CERT_DIR=${ID_GENERATOR_CERT_DIR:-"$ROOT_DIR/environments/test/message/services/snowflake/ssl/certs"}
ENVIRONMENT=${ENVIRONMENT:-"test"}
KAFKA_BROKERS=${KAFKA_BROKERS:-"localhost:29092,localhost:39092,localhost:49092"}
MEDIA_SERVICE_URL=${MEDIA_SERVICE_URL:-"http://localhost:8300"}

# External Services
ID_GENERATOR_ADDR="${ID_GENERATOR_ADDR:-localhost:9000}"
ID_GENERATOR_CERT_DIR="${ID_GENERATOR_CERT_DIR:-$ROOT_DIR/environments/test/conversation/services/snowflake/ssl/certs}"
ID_GENERATOR_EPOCH="${ID_GENERATOR_EPOCH:-1654041600000}" # must match the epoch used by the ID generator service
MESSAGE_SERVICE_URL="${MESSAGE_SERVICE_URL:-http://localhost:8100}"
USER_SERVICE_URL="${USER_SERVICE_URL:-http://localhost:8400}"
MEDIA_SERVICE_URL="${MEDIA_SERVICE_URL:-http://localhost:8300}"
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:29092,localhost:39092,localhost:49092}"
OUTBOX_CHECK_INTERVAL_MS="${OUTBOX_CHECK_INTERVAL_MS:-500}"

# Test-Specific Variables
ADMIN_DATABASE_URL=${ADMIN_DATABASE_URL:-"postgresql://admin:testpass123@localhost:6000/chat-app?sslmode=disable"}
MESSAGE_SERVICE_URL=${MESSAGE_SERVICE_URL:-"http://localhost:$PORT"}
ID_GENERATOR_TLS_ADDR=${ID_GENERATOR_TLS_ADDR:-"localhost:9000"}
ID_GENERATOR_NON_TLS_ADDR=${ID_GENERATOR_NON_TLS_ADDR:-"localhost:9001"}
ID_GENERATOR_FAKE_CERT_DIR=${ID_GENERATOR_FAKE_CERT_DIR:-"$ROOT_DIR/environments/test/message/services/snowflake/fake_ssl"}
DATA_FILE_DIR=${DATA_FILE_DIR:-"$ROOT_DIR/tests/data"}
# One working broker address is enough for the tests to run
BROKER_ADDR=${BROKER_ADDR:-"localhost:29092"}

# Meta Configuration (Logs & Topics)
# Note: Keeping the newlines helps readability, just ensure the consuming script handles them.
LOG_META="
$ROOT_DIR/tests/logs/database_service.log=chat-app-db-service;
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

TOPICS="
message-resource-created:2:2;
asset-resource-delete:2:2;
asset-resource-delete-dlq:1:1;
participant-avatar-update:2:2;
participant-avatar-update-dlq:1:1
"

# Turn off auto-export
set +a

# --- Execution ---

if [ "$DEBUG" == "true" ]; then
    echo "Debug mode is on"
    set -x
    # Optional: Print env vars to verify they are set correctly
    env | grep -E "DATABASE_|PORT|KAFKA|SERVICE_URL"
fi

COMMAND="$1"
shift # Remove the first argument (command)
EXTRA_ARGS=("$@") # Capture the rest as an array to preserve spaces/structure

# Execute the service runner
"$SCRIPT_DIR/__test_with_services.sh" "$COMMAND" "${EXTRA_ARGS[@]}"
