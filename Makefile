DOCKER_USERNAME ?= ilikeblue
DOCKER_FOLDER ?= ./docker
APPLICATION_NAME ?= chat-app-message-service
GIT_HASH ?= $(shell git log --format="%h" -n 1) 
SERVER_PORT ?= 80
CURRENT_DIR = $(shell pwd)
DOCKER_IMAGE_NAME = ${DOCKER_USERNAME}/${APPLICATION_NAME}

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
OUT_DIR = ./bin
OUT_FILE = $(OUT_DIR)/main
SRC_FILES = ./cmd/server/main.go $(shell find ./internal/ -name '*.go') $(shell find ./tests/ -name '*.go')

_BUILD_ARGS_TAG ?= ${GIT_HASH}
_BUILD_ARGS_RELEASE_TAG ?= latest
_BUILD_ARGS_DOCKERFILE ?= Dockerfile

.PHONY: test
test: export TEST_OUT = $(ROOT_DIR)/reports/results
test: export DB_LOG = $(ROOT_DIR)/reports/db.log
test: export USER_SERVICE_LOG = $(ROOT_DIR)/reports/user_service.log
test: export SNOWFLAKE_SERVICE_LOG = $(ROOT_DIR)/reports/snowflake_service.log
test: build certs
	@rm -rf reports
	@mkdir reports
	@echo "Running tests"
ifdef JSON
	TEST_OUT=$(TEST_OUT) ./scripts/test_local.sh go run ./cmd/test/run_tests.go -json
	./scripts/read_test_stats.sh $(TEST_OUT)
else
	./scripts/test_local.sh go run ./cmd/test/run_tests.go
endif

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

.PHONY: up_% certs
up_%:
	$(info ==================== up docker compose ====================)
	docker-compose -f compose/docker-compose.$*.yaml up -d

.PHONY: down_%
down_%:
	$(info ==================== down docker compose ====================)
	docker-compose -f compose/docker-compose.$*.yaml down

_dbuilder:
	$(info ==================== building dockerfile ====================)
	docker buildx build --platform linux/amd64 --tag ${DOCKER_IMAGE_NAME}:${_BUILD_ARGS_TAG} -f ${DOCKER_FOLDER}/${_BUILD_ARGS_DOCKERFILE} .

_dbuilder_debug:
	$(info ==================== building dockerfile with debug on ====================)
	docker buildx build --debug --progress=plain --no-cache --platform linux/amd64 --tag ${DOCKER_IMAGE_NAME}:${_BUILD_ARGS_TAG} -f ${DOCKER_FOLDER}/${_BUILD_ARGS_DOCKERFILE} . 

_dpusher:
	$(info ==================== pushing dockerfile ====================)
	docker push ${DOCKER_IMAGE_NAME}:${_BUILD_ARGS_TAG}

_dreleaser:
	$(info ==================== releasing dockerfile ====================)
	docker pull ${DOCKER_IMAGE_NAME}:${_BUILD_ARGS_TAG}
	docker tag  ${DOCKER_IMAGE_NAME}:${_BUILD_ARGS_TAG} ${DOCKER_IMAGE_NAME}:${_BUILD_ARGS_RELEASE_TAG}
	docker push ${DOCKER_IMAGE_NAME}:${_BUILD_ARGS_RELEASE_TAG}

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
