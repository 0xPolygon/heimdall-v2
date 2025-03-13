GO ?= latest
GOBIN = $(CURDIR)/build/bin
GORUN = env GO111MODULE=on go run
GOPATH = $(shell go env GOPATH)

GIT_COMMIT ?= $(shell git rev-list -1 HEAD)

DOCKER := $(shell which docker)

PACKAGE_NAME := github.com/0xPolygon/heimdall-v2
HTTPS_GIT := https://$(PACKAGE_NAME).git
GOLANG_CROSS_VERSION  ?= v1.23.2

# Fetch git latest tag
LATEST_GIT_TAG:=$(shell git describe --tags $(git rev-list --tags --max-count=1))
VERSION := $(shell git describe --tags | sed 's/^v//')
CMT_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')
COSMOS_VERSION := $(shell go list -m github.com/cosmos/cosmos-sdk | sed 's:.* ::')
COMMIT := $(shell git log -1 --format='%H')

ldflags = -X github.com/0xPolygon/heimdall-v2/version.Name=heimdall \
		  -X github.com/0xPolygon/heimdall-v2/version.ServerName=heimdalld \
		  -X github.com/0xPolygon/heimdall-v2/version.Version=$(VERSION) \
		  -X github.com/0xPolygon/heimdall-v2/version.Commit=$(COMMIT) \
		  -X github.com/cosmos/cosmos-sdk/version.Name=heimdall \
		  -X github.com/cosmos/cosmos-sdk/version.ServerName=heimdalld \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(COSMOS_VERSION) \
		  -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(CMT_VERSION)

BUILD_FLAGS := -ldflags '$(ldflags)'

.PHONY: clean
clean:
	rm -rf build

heimdalld: clean
	mkdir -p build
	go build $(BUILD_FLAGS) -o build/heimdalld ./cmd/heimdalld
	@echo "====================================================\n==================Build Successful==================\n===================================================="

test:
	go test ./...

.PHONY: lint-deps
lint-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.61.0

.PHONY: lint
lint:
	@if [ -n "$(NEW_FROM_REV)" ]; then \
		echo "NEW_FROM_REV is set to: $(NEW_FROM_REV)"; \
	else \
		echo "NEW_FROM_REV is not set"; \
	fi
	@./build/bin/golangci-lint run --config ./.golangci.yml $(if $(NEW_FROM_REV),--new-from-rev $(NEW_FROM_REV))


###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen proto-swagger-gen

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against "$(HTTPS_GIT)#branch=develop"

proto-swagger-gen:
	@echo "Generating Protobuf Swagger"
	@$(protoImage) sh ./scripts/protoc-swagger-gen.sh

.PHONY: proto-all proto-gen proto-format proto-lint proto-check-breaking

mock:
	go install github.com/golang/mock/mockgen@latest
	mockgen -source=x/bor/types/expected_keepers.go -destination=x/bor/testutil/expected_keepers_mocks.go  -package=testutil
	mockgen -source=x/checkpoint/types/expected_keepers.go -destination=x/checkpoint/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/clerk/types/expected_keepers.go -destination=x/clerk/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/milestone/types/expected_keepers.go -destination=x/milestone/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/stake/types/expected_keepers.go -destination=x/stake/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/topup/types/expected_keepers.go -destination=x/topup/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -destination=helper/mocks/i_http_client.go -package=mocks --source=./helper/util.go HTTPClient
	go install github.com/vektra/mockery/v2/...@latest
	mockery --name IContractCaller --dir ./helper  --output ./helper/mocks --filename=i_contract_caller.go


###############################################################################
###                                docker                                   ###
###############################################################################

build-docker: # TODO-HV2: check this command once we have a proper docker build
	@echo Fetching latest tag: $(LATEST_GIT_TAG)
	git checkout $(LATEST_GIT_TAG)
	docker build -t "0xpolygon/heimdall-v2:$(LATEST_GIT_TAG)" -f Dockerfile .

push-docker: # TODO-HV2: check this command once we have a proper docker push
	@echo Pushing docker tag image: $(LATEST_GIT_TAG)
	docker push "0xpolygon/heimdall-v2:$(LATEST_GIT_TAG)"

###############################################################################
###                                release                                  ###
###############################################################################

.PHONY: release-dry-run # TODO-HV2: check this command once we have a proper release process
release-dry-run:
	@docker run \
		--platform linux/amd64 \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-e CGO_CFLAGS=-Wno-unused-function \
		-e GITHUB_TOKEN \
		-e DOCKER_USERNAME \
		-e DOCKER_PASSWORD \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.docker/config.json:/root/.docker/config.json \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		--rm-dist --skip-validate --skip-publish

.PHONY: release # TODO-HV2: check this command once we have a proper release process
release:
	@docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-e GITHUB_TOKEN \
		-e DOCKER_USERNAME \
		-e DOCKER_PASSWORD \
		-e SLACK_WEBHOOK \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.docker/config.json:/root/.docker/config.json \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		goreleaser/goreleaser-cross:${GOLANG_CROSS_VERSION} \
		--rm-dist --skip-validate

###############################################################################
###                                Engine API POC                           ###
###############################################################################

engine-api-poc-build: 
	ARCH=$(ARCH) docker compose -f ./engine-api-poc/docker-compose.yaml build heimdalld-init erigon-init
.PHONY: engine-api-poc-build

engine-api-poc-init: 
	ARCH=$(ARCH) docker compose -f ./engine-api-poc/docker-compose.yaml up heimdalld-init erigon-init
.PHONY: engine-api-poc-init

engine-api-poc-start:
	rm -rf ./engine-api-poc/logs
	mkdir ./engine-api-poc/logs
	ARCH=$(ARCH) docker compose -f ./engine-api-poc/docker-compose.yaml up -d  node0 node1 node2 node3 node4
	@echo "Tailing logs for all nodes. Press CTRL+C to stop..."
	@bash -c '\
		trap "kill 0" SIGINT; \
		tail -F ./engine-api-poc/build/node0/heimdalld/app.log | grep --line-buffered "🕒" >> ./engine-api-poc/logs/node0.log & \
		tail -F ./engine-api-poc/build/node1/heimdalld/app.log | grep --line-buffered "🕒" >> ./engine-api-poc/logs/node1.log & \
		tail -F ./engine-api-poc/build/node2/heimdalld/app.log | grep --line-buffered "🕒" >> ./engine-api-poc/logs/node2.log & \
		tail -F ./engine-api-poc/build/node3/heimdalld/app.log | grep --line-buffered "🕒" >> ./engine-api-poc/logs/node3.log & \
		tail -F ./engine-api-poc/build/node4/heimdalld/app.log | grep --line-buffered "🕒" >> ./engine-api-poc/logs/node4.log & \
		wait'	
.PHONY: engine-api-poc-start

engine-api-poc-stop:
	ARCH=$(ARCH) docker compose -f ./engine-api-poc/docker-compose.yaml  stop
.PHONY: engine-api-poc-stop

engine-api-poc-destroy:
	ARCH=$(ARCH) docker compose -f ./engine-api-poc/docker-compose.yaml  down --remove-orphans
	ARCH=$(ARCH) rm -rf ./engine-api-poc/build
.PHONY: engine-api-poc-stop

engine-api-poc-test:
	ARCH=$(ARCH) bash ./engine-api-poc/run_test.sh
.PHONY: engine-api-poc-test

engine-api-poc-monitor-test:
	ARCH=$(ARCH) polycli monitor
.PHONY: engine-api-poc-test

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  lint-deps           	- Install dependencies for GolangCI-Lint tool."
	@echo "  lint                	- Run the GolangCI-Lint tool on the codebase."
	@echo "  clean               	- Delete build folder."
	@echo "  heimdalld              - Compiles the Heimdall binaries."
	@echo "  test               	- Run the tests."
	@echo "  mock                	- Generate mocks."
	@echo "  proto-all           	- Format, lint and generate proto files."
	@echo "  proto-lint        		- Lint proto files."
	@echo "  proto-format        	- Format proto files."
	@echo "  proto-gen           	- Generate proto files."
	@echo "  proto-check-breaking   - Check if proto breaks against git head."
	@echo "  build-docker        	- Build a Docker image for the latest Git tag."
	@echo "  push-docker         	- Push the Docker image for the latest Git tag."
	@echo "  release-dry-run     	- Perform a dry run of the release process."
	@echo "  release             	- Execute the actual release process."
