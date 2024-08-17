#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
CONFIG_FILE="$SCRIPT_DIR/openssl.cnf"

gen() {
    rm -f *.pem
    # Create the CA certificate
    openssl req -x509 \
      -newkey rsa:4096 \
      -nodes \
      -days 3650 \
      -keyout $SCRIPT_DIR/fake_ca_key.pem \
      -out $SCRIPT_DIR/fake_ca_cert.pem \
      -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=chatapp/CN=chatapp_ca" \
      -config "$CONFIG_FILE" \
      -extensions ca \
      -sha256

    # Generate a client private key
    openssl genrsa -out $SCRIPT_DIR/fake_client_key.pem 4096

    # Create a Certificate Signing Request (CSR) for the Client
    openssl req -new \
      -key $SCRIPT_DIR/fake_client_key.pem \
      -out $SCRIPT_DIR/fake_client_csr.pem \
      -config "$CONFIG_FILE" \
      -subj "/C=VN/ST=Ha Noi/L=Ha Noi/O=mychatapp/CN=mychatapp.local" \
      -reqexts client

    # Sign the Client CSR with the Client CA Key to Generate the Client Certificate
    openssl x509 -req \
      -in $SCRIPT_DIR/fake_client_csr.pem \
      -CAkey $SCRIPT_DIR/fake_ca_key.pem \
      -CA $SCRIPT_DIR/fake_ca_cert.pem \
      -days 3650 \
      -set_serial 1000 \
      -out $SCRIPT_DIR/fake_client_cert.pem \
      -extfile "$CONFIG_FILE" \
      -extensions client \
      -sha256

    # Verify the Client Certificate
    openssl verify -verbose -CAfile $SCRIPT_DIR/fake_ca_cert.pem $SCRIPT_DIR/fake_client_cert.pem

    # Cleanup
    rm $SCRIPT_DIR/fake_client_csr.pem
}

main() {
    gen
}

main;
