#!/bin/bash -e

# This script is used to generate SSL certificates for the services. By default,
# the script will generate certificates to all the directories specified in the CERT_DIRS
# array and the FAKE_CERT_DIRS array. Also, it will load openssl.cnf file from
# OPENSSL_CONFIG_FILE environment variable.
#
# You can choose to generate certificates for all the directories or just for the test
# environment by selecting the option when running the script. Note that if you choose
# to generate certificates for all the directories, the script will generate certificates
# for both the test and the production environment (it will delete the existing certificates
# in the directories and generate new certificates).
#
# This script is recommended to be run for development and testing purposes only. For
# production, you should use a proper CA to generate the certificates.

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."
OPENSSL_CONFIG_FILE=${OPENSSL_CONFIG_FILE:="$ROOT_DIR/etc/openssl.cnf"}
CERT_DIRS=(
    "$ROOT_DIR/environments/dev/snowflake/ssl" 
    "$ROOT_DIR/environments/dev/message/services/snowflake/ssl" 

    "$ROOT_DIR/environments/test/snowflake/ssl/certs" 
    "$ROOT_DIR/environments/test/message/services/snowflake/ssl/certs"
    "$ROOT_DIR/environments/test/media/services/snowflake/ssl/certs"
)
FAKE_CERT_DIRS=(
    "$ROOT_DIR/environments/test/message/services/snowflake/fake_ssl/certs"
)


TEMP_CERT_DIR=$(mktemp -d)

# user can choose: generate for all or just for test environment
options=("all" "test")

function gen() {
    rm -f $TEMP_CERT_DIR/* 

    # Create the CA certificate
    openssl req -x509 \
      -newkey rsa:4096 \
      -nodes \
      -days 3650 \
      -keyout $TEMP_CERT_DIR/ca_key.pem \
      -out $TEMP_CERT_DIR/ca_cert.pem \
      -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=chatapp/CN=chatapp_ca" \
      -config "$OPENSSL_CONFIG_FILE" \
      -extensions ca \
      -sha256

    # Generate a server private key
    openssl genrsa -out $TEMP_CERT_DIR/server_key.pem 4096

    # Create a Certificate Signing Request (CSR) for the server
    openssl req -new \
      -key $TEMP_CERT_DIR/server_key.pem \
      -out $TEMP_CERT_DIR/server_csr.pem \
      -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=mychatapp/CN=mychatapp.local" \
      -config "$OPENSSL_CONFIG_FILE" \
      -reqexts server

    # Sign the server CSR with the CA key to generate the server certificate
    openssl x509 -req \
      -in $TEMP_CERT_DIR/server_csr.pem \
      -CAkey $TEMP_CERT_DIR/ca_key.pem \
      -CA $TEMP_CERT_DIR/ca_cert.pem \
      -days 3650 \
      -set_serial 1000 \
      -out $TEMP_CERT_DIR/server_cert.pem \
      -extfile "$OPENSSL_CONFIG_FILE" \
      -extensions server \
      -sha256

    # Verify the server certificate
    openssl verify -verbose -CAfile $TEMP_CERT_DIR/ca_cert.pem $TEMP_CERT_DIR/server_cert.pem

    # Generate a client private key
    openssl genrsa -out $TEMP_CERT_DIR/client_key.pem 4096

    # Create a Certificate Signing Request (CSR) for the Client
    openssl req -new \
      -key $TEMP_CERT_DIR/client_key.pem \
      -out $TEMP_CERT_DIR/client_csr.pem \
      -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=mychatapp/CN=mychatapp.local" \
      -config "$OPENSSL_CONFIG_FILE" \
      -reqexts client

    # Sign the Client CSR with the Client CA Key to Generate the Client Certificate
    openssl x509 -req \
      -in $TEMP_CERT_DIR/client_csr.pem \
      -CAkey $TEMP_CERT_DIR/ca_key.pem \
      -CA $TEMP_CERT_DIR/ca_cert.pem \
      -days 3650 \
      -set_serial 1000 \
      -out $TEMP_CERT_DIR/client_cert.pem \
      -extfile "$OPENSSL_CONFIG_FILE" \
      -extensions client \
      -sha256

    # Verify the Client Certificate
    openssl verify -verbose -CAfile $TEMP_CERT_DIR/ca_cert.pem $TEMP_CERT_DIR/client_cert.pem

    rm -f $TEMP_CERT_DIR/server_csr.pem $TEMP_CERT_DIR/client_csr.pem
}

function copy_certs() {
    dirs=("$@")
    for dir in "${dirs[@]}"; do
        printf "Copying certificates to: $dir\n"
        cp $TEMP_CERT_DIR/*.pem $dir
    done
}

function init_dirs_if_not_exist() {
    dirs=("$@")
    for dir in "${dirs[@]}"; do
        printf "Checking directory: $dir\n"
        if [[ ! -d $dir ]]; then
            printf "Directory missing. Creating directory: $dir\n"
            mkdir -p $dir
        fi
    done
}

function main() {
    cert_dirs=("${CERT_DIRS[@]}")
    fake_cert_dirs=("${FAKE_CERT_DIRS[@]}")

    init_dirs_if_not_exist "${cert_dirs[@]}" "${fake_cert_dirs[@]}"

    printf "Generating certificates...\n"
    gen
    copy_certs "${cert_dirs[@]}"

    printf "Generating fake certificates...\n"
    gen
    copy_certs "${fake_cert_dirs[@]}"

    # printf "Choose the environment to generate certificates: \n"
    # cert_dirs=()
    # fake_cert_dirs=()
    # PS3="Enter your choice: "
    # select opt in "${options[@]}"; do
    #     case $opt in
    #         "all")
    #             cert_dirs=("${CERT_DIRS[@]}" "${TEST_CERT_DIRS[@]}")
    #             fake_cert_dirs=("${TEST_FAKE_CERT_DIRS[@]}")
    #             break
    #             ;;
    #         "test")
    #             cert_dirs=("${TEST_CERT_DIRS[@]}")
    #             fake_cert_dirs=("${TEST_FAKE_CERT_DIRS[@]}")
    #             break
    #             ;;
    #         *) echo "Invalid option $REPLY";;
    #     esac
    # done
    #
    # init_dirs_if_not_exist "${cert_dirs[@]}" "${fake_cert_dirs[@]}"
    #
    # printf "Generating certificates...\n"
    # gen
    # copy_certs "${cert_dirs[@]}"
    #
    # printf "Generating fake certificates...\n"
    # gen
    # copy_certs "${fake_cert_dirs[@]}"
}

main
