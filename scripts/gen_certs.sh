#!/bin/bash

# --- Configuration ---
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="${SCRIPT_DIR}/.."
OPENSSL_CONFIG_FILE="${OPENSSL_CONFIG_FILE:-"$ROOT_DIR/etc/openssl.cnf"}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Target Directories
# We explicitly categorize them to allow filtering
DEV_CERT_DIRS=(
    "$ROOT_DIR/environments/dev/snowflake/ssl"
    "$ROOT_DIR/environments/dev/message/services/snowflake/ssl"
)

TEST_CERT_DIRS=(
    "$ROOT_DIR/environments/test/snowflake/ssl/certs"
    "$ROOT_DIR/environments/test/message/services/snowflake/ssl/certs"
    "$ROOT_DIR/environments/test/media/services/snowflake/ssl/certs"
)

FAKE_CERT_DIRS=(
    "$ROOT_DIR/environments/test/message/services/snowflake/fake_ssl/certs"
)

# --- Functions ---

# usage: print_usage
print_usage() {
    echo "Usage: $0 [env]"
    echo "  env: 'all', 'dev', or 'test' (default: test)"
    echo "  Example: $0 dev"
}

# cleanup: Automatically called on exit
cleanup() {
    if [ -d "${TEMP_CERT_DIR:-}" ]; then
        rm -rf "$TEMP_CERT_DIR"
    fi
}
trap cleanup EXIT

log() {
    echo -e "${BLUE}[GEN]${NC} $1"
}

check_prereqs() {
    if [ ! -f "$OPENSSL_CONFIG_FILE" ]; then
        echo -e "${RED}Error: OpenSSL config file not found at: $OPENSSL_CONFIG_FILE${NC}"
        exit 1
    fi
    if ! command -v openssl &> /dev/null; then
        echo -e "${RED}Error: openssl is not installed.${NC}"
        exit 1
    fi
}

generate_cert_set() {
    local target_dirs=("${@}")

    # Empty temp dir for new run
    rm -f "$TEMP_CERT_DIR"/*

    log "Creating CA Certificate..."
    openssl req -x509 -newkey rsa:4096 -nodes -days 3650 \
        -keyout "$TEMP_CERT_DIR/ca_key.pem" \
        -out "$TEMP_CERT_DIR/ca_cert.pem" \
        -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=chatapp/CN=chatapp_ca" \
        -config "$OPENSSL_CONFIG_FILE" \
        -extensions ca -sha256 > /dev/null 2>&1

    log "Creating Server Certificate..."
    openssl genrsa -out "$TEMP_CERT_DIR/server_key.pem" 4096 > /dev/null 2>&1
    openssl req -new -key "$TEMP_CERT_DIR/server_key.pem" \
        -out "$TEMP_CERT_DIR/server_csr.pem" \
        -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=mychatapp/CN=mychatapp.local" \
        -config "$OPENSSL_CONFIG_FILE" -reqexts server > /dev/null 2>&1

    openssl x509 -req -in "$TEMP_CERT_DIR/server_csr.pem" \
        -CAkey "$TEMP_CERT_DIR/ca_key.pem" -CA "$TEMP_CERT_DIR/ca_cert.pem" \
        -days 3650 -set_serial 1000 -out "$TEMP_CERT_DIR/server_cert.pem" \
        -extfile "$OPENSSL_CONFIG_FILE" -extensions server -sha256 > /dev/null 2>&1

    log "Creating Client Certificate..."
    openssl genrsa -out "$TEMP_CERT_DIR/client_key.pem" 4096 > /dev/null 2>&1
    openssl req -new -key "$TEMP_CERT_DIR/client_key.pem" \
        -out "$TEMP_CERT_DIR/client_csr.pem" \
        -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=mychatapp/CN=mychatapp.local" \
        -config "$OPENSSL_CONFIG_FILE" -reqexts client > /dev/null 2>&1

    openssl x509 -req -in "$TEMP_CERT_DIR/client_csr.pem" \
        -CAkey "$TEMP_CERT_DIR/ca_key.pem" -CA "$TEMP_CERT_DIR/ca_cert.pem" \
        -days 3650 -set_serial 1000 -out "$TEMP_CERT_DIR/client_cert.pem" \
        -extfile "$OPENSSL_CONFIG_FILE" -extensions client -sha256 > /dev/null 2>&1

    # Cleanup intermediate CSRs
    rm -f "$TEMP_CERT_DIR/"*_csr.pem

    # Distribution
    for dir in "${target_dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            mkdir -p "$dir"
        fi
        echo -e "  -> Installing to: ${YELLOW}$dir${NC}"
        cp "$TEMP_CERT_DIR"/*.pem "$dir/"
    done
}

# --- Main Execution ---

main() {
    TEMP_CERT_DIR=$(mktemp -d)
    check_prereqs

    TARGET_ENV="${1:-test}" # Default to test

    # Select directories based on input
    REAL_DIRS=()
    FAKE_DIRS=()

    case "$TARGET_ENV" in
        all)
            REAL_DIRS=("${DEV_CERT_DIRS[@]}" "${TEST_CERT_DIRS[@]}")
            FAKE_DIRS=("${FAKE_CERT_DIRS[@]}")
            ;;
        dev)
            REAL_DIRS=("${DEV_CERT_DIRS[@]}")
            ;;
        test)
            REAL_DIRS=("${TEST_CERT_DIRS[@]}")
            FAKE_DIRS=("${FAKE_CERT_DIRS[@]}")
            ;;
        *)
            print_usage
            exit 1
            ;;
    esac

    if [ ${#REAL_DIRS[@]} -gt 0 ]; then
        echo -e "${GREEN}=== Generating Valid Certificates for '$TARGET_ENV' ===${NC}"
        generate_cert_set "${REAL_DIRS[@]}"
    fi

    if [ ${#FAKE_DIRS[@]} -gt 0 ]; then
        echo -e "\n${GREEN}=== Generating Fake Certificates (Separate CA) ===${NC}"
        # We run this again to ensure the "Fake" certs have a different CA
        # than the "Valid" ones, ensuring validation tests fail as expected.
        generate_cert_set "${FAKE_DIRS[@]}"
    fi

    echo -e "\n${GREEN}Success! Certificates generated.${NC}"
}

main "$@"
