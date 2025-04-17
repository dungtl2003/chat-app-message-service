DOCKER_USERNAME ?= ilikeblue
DOCKER_FOLDER ?= ./docker
APPLICATION_NAME ?= chat-app-message-service
GIT_HASH ?= $(shell git log --format="%h" -n 1) 
SERVER_PORT ?= 80
CURRENT_DIR = $(shell pwd)
DOCKER_IMAGE_NAME = ${DOCKER_USERNAME}/${APPLICATION_NAME}

OUT_DIR = ./bin
OUT_FILE = $(OUT_DIR)/main
SRC_FILES = ./cmd/server/main.go $(shell find ./internal/ -name '*.go') $(shell find ./tests/ -name '*.go')

_BUILD_ARGS_TAG ?= ${GIT_HASH}
_BUILD_ARGS_RELEASE_TAG ?= latest
_BUILD_ARGS_DOCKERFILE ?= Dockerfile

.PHONY: proto
proto:
	$(info ==================== generating new proto files ====================)
	@mkdir -p internal/services/snowflake/proto
	protoc --proto_path=proto proto/*.proto  --go_out=:internal/services/snowflake/proto --go-grpc_out=:internal/services/snowflake/proto

.PHONY: run
run: build
	echo "Running application"
	./scripts/run.sh $(OUT_FILE)

.PHONY: build
build: $(OUT_FILE)

$(OUT_FILE): $(SRC_FILES)
	@echo "$(SRC_FILES)"
	@echo "Building application"
	@mkdir -p $(OUT_DIR)
	go build -o $(OUT_FILE) $<

.PHONY: clean
clean:
	@echo "Cleaning up"
	rm -rf $(OUT_DIR)

.PHONY: certs
certs:
	@echo "Generating certs"	
	./scripts/gen_certs.sh
