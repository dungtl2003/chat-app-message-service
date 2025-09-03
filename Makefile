DOCKER_USERNAME ?= ilikeblue
DOCKER_FOLDER ?= ./docker
APPLICATION_NAME ?= chat-app-message-service
GIT_HASH ?= $(shell git log --format="%h" -n 1) 
SERVER_PORT ?= 80
CURRENT_DIR = $(shell pwd)
DOCKER_IMAGE_NAME = ${DOCKER_USERNAME}/${APPLICATION_NAME}

_BUILD_ARGS_TAG ?= ${GIT_HASH}
_BUILD_ARGS_RELEASE_TAG ?= latest
_BUILD_ARGS_DOCKERFILE ?= Dockerfile

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
OUT_DIR = ./bin
OUT_FILE = $(OUT_DIR)/main
SRC_FILES = ./cmd/server/main.go $(shell find ./internal/ -name '*.go')
TEST_LOGS_DIR = $(ROOT_DIR)/tests/logs

PROTO_IN_DIR = ./proto
PROTO_OUT_DIRS = ./internal/services/idgen/proto

.PHONY: test 
test: export TEST_OUT = $(TEST_LOGS_DIR)/results
test: build certs
	@echo "Running tests"
# ifdef FORCE
# 	go clean -testcache
# endif
	@rm -rf $(TEST_LOGS_DIR)
	@go clean -testcache
ifdef JSON
	TEST_OUT=$(TEST_OUT) ./scripts/test_local.sh go run ./cmd/test/run_tests.go -json
	./scripts/read_test_stats.sh $(TEST_OUT)
else
	./scripts/test_local.sh go run ./cmd/test/run_tests.go
endif

# use this command to run service without building to container yet
.PHONY: run_with_services
run_with_services: $(OUT_FILE)
ifdef ENVIRONMENT
	@echo "Running application with services in $(ENVIRONMENT) environment"
	COMPOSE_FILE=docker-compose.$(ENVIRONMENT).yaml ./scripts/run_with_services.sh $(OUT_FILE)
else
	@echo "Running application with services in default environment"
	./scripts/run_with_services.sh $(OUT_FILE)
endif

.PHONY: certs
certs:
	@echo "Generating certs"	
	./scripts/gen_certs.sh

.PHONY: build
build: $(OUT_FILE)

$(OUT_FILE): $(SRC_FILES) $(OUT_DIR) $(PROTO_OUT_DIRS)/*.pb.go
	echo "Building application"
	go build -o $(OUT_FILE) $<

$(PROTO_OUT_DIRS)/%.pb.go: $(PROTO_IN_DIR) $(PROTO_OUT_DIRS) $(PROTO_IN_DIR)/%.proto
	echo "Generating proto files" 
	protoc --proto_path=$(PROTO_IN_DIR) $(PROTO_IN_DIR)/*.proto  --go_out=:$(PROTO_OUT_DIRS) --go-grpc_out=:$(PROTO_OUT_DIRS)

$(PROTO_OUT_DIRS):
	echo "Creating proto output directories"
	mkdir -p $@

$(PROTO_GRPC_OUT_DIRS):
	echo "Creating proto grpc output directories"
	mkdir -p $(PROTO_GRPC_OUT_DIRS)

$(PROTO_IN_DIR):
	echo "Creating proto directory"
	mkdir -p $(PROTO_IN_DIR)

$(OUT_DIR):
	echo "Creating output directory"
	mkdir -p $(OUT_DIR)


_dbuilder:
	$(info ==================== building dockerfile ====================)
	docker buildx build --platform linux/amd64 --tag ${DOCKER_USERNAME}/${APPLICATION_NAME}:${_BUILD_ARGS_TAG} -f ${DOCKER_FOLDER}/${_BUILD_ARGS_DOCKERFILE} .

_dbuilder_debug:
	$(info ==================== building dockerfile with debug on ====================)
	docker buildx build --debug --progress=plain --no-cache --platform linux/amd64 --tag ${DOCKER_USERNAME}/${APPLICATION_NAME}:${_BUILD_ARGS_TAG} -f ${DOCKER_FOLDER}/${_BUILD_ARGS_DOCKERFILE} . 

_dpusher:
	$(info ==================== pushing dockerfile ====================)
	docker push ${DOCKER_USERNAME}/${APPLICATION_NAME}:${_BUILD_ARGS_TAG}

_dreleaser:
	$(info ==================== releasing dockerfile ====================)
	docker pull ${DOCKER_USERNAME}/${APPLICATION_NAME}:${_BUILD_ARGS_TAG}
	docker tag  ${DOCKER_USERNAME}/${APPLICATION_NAME}:${_BUILD_ARGS_TAG} ${DOCKER_USERNAME}/${APPLICATION_NAME}:${_BUILD_ARGS_RELEASE_TAG}
	docker push ${DOCKER_USERNAME}/${APPLICATION_NAME}:${_BUILD_ARGS_RELEASE_TAG}

.PHONY: dbuild
dbuild:
	$(MAKE) _dbuilder
 
.PHONY: dpush
dpush:
	$(MAKE) _dpusher
 
.PHONY: drelease
drelease:
	$(MAKE) _dreleaser

.PHONY: dbuild_debug
dbuild_debug:
	$(MAKE) _dbuilder_debug

.PHONY: dbuild_%
dbuild_%: 
	$(MAKE) _dbuilder \
		-e _BUILD_ARGS_TAG="$*-${GIT_HASH}" \
		-e _BUILD_ARGS_DOCKERFILE="Dockerfile.$*"

.PHONY: dbuild_debug_%
dbuild_debug_%:
	$(MAKE) _dbuilder_debug \
		-e _BUILD_ARGS_TAG="$*-${GIT_HASH}" \
		-e _BUILD_ARGS_DOCKERFILE="Dockerfile.$*"
 
.PHONY: dpush_%
dpush_%:
	$(MAKE) _dpusher \
		-e _BUILD_ARGS_TAG="$*-${GIT_HASH}"
 
.PHONY: drelease_%
drelease_%:
	$(MAKE) _dreleaser \
		-e _BUILD_ARGS_TAG="$*-${GIT_HASH}" \
		-e _BUILD_ARGS_RELEASE_TAG="$*-latest"

.PHONY: clean_image
clean_image:
	$(info ==================== cleaning dangling images ====================)
	docker images --filter "dangling=true" --filter "reference=${DOCKER_IMAGE_NAME}" -q | xargs -r docker rmi

.PHONY: ci_%
ci_%:
	$(MAKE) dbuild_$*
	$(MAKE) dpush_$*
	$(MAKE) drelease_$*
	$(MAKE) clean_image

.PHONY: up_% certs
up_%:
	$(info ==================== up docker compose ====================)
	docker-compose -f compose/docker-compose.$*.yaml up -d

.PHONY: down_%
down_%:
	$(info ==================== down docker compose ====================)
	docker-compose -f compose/docker-compose.$*.yaml down
