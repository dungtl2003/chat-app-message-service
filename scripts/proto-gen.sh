#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT_DIR="$SCRIPT_DIR/.."

npx proto-loader-gen-types --grpcLib=@grpc/grpc-js --outDir=$ROOT_DIR/src/loaders/services/id-generator/proto $ROOT_DIR/src/loaders/services/id-generator/proto/*.proto
