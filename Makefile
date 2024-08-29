GO ?= latest
GOBIN = $(CURDIR)/build/bin
GORUN = env GO111MODULE=on go run
GOPATH = $(shell go env GOPATH)

GIT_COMMIT ?= $(shell git rev-list -1 HEAD)

DOCKER := $(shell which docker)
HTTPS_GIT := https://github.com/0xPolygon/heimdall-v2.git

PACKAGE_NAME := github.com/0xPolygon/heimdall-v2
HTTPS_GIT := https://$(PACKAGE_NAME)
GOLANG_CROSS_VERSION  ?= v1.21.0

# Fetch git latest tag
LATEST_GIT_TAG:=$(shell git describe --tags $(git rev-list --tags --max-count=1))
VERSION := $(shell shell git describe --tags | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

ldflags = -X github.com/0xPolygon/heimdall-v2/version.Name=heimdall \
		  -X github.com/0xPolygon/heimdall-v2/version.ServerName=heimdalld \
		  -X github.com/0xPolygon/heimdall-v2/version.Version=$(VERSION) \
		  -X github.com/0xPolygon/heimdall-v2/version.Commit=$(COMMIT) \
		  -X github.com/cosmos/cosmos-sdk/version.Name=heimdall \
		  -X github.com/cosmos/cosmos-sdk/version.ServerName=heimdalld \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

.PHONY: clean
clean:
	rm -rf build

build: clean
	mkdir -p build
	go build $(BUILD_FLAGS) -o build/heimdalld ./cmd/heimdalld
	@echo "====================================================\n==================Build Successful==================\n===================================================="

test:
	go test  -v ./...

.PHONY: lint-deps
lint-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.57.1

.PHONY: lint
lint:
	@./build/bin/golangci-lint run --config ./.golangci.yml


###############################################################################
###                                Protobuf                                 ###
###############################################################################

protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

.PHONY: proto-all proto-gen proto-swagger-gen proto-format proto-lint proto-check-breaking proto-update-deps

mock:
	# TODO HV2: enrich the mockgen command with all other modules' mocks
	go install github.com/golang/mock/mockgen@latest
	mockgen -source=x/checkpoint/types/expected_keepers.go -destination=x/checkpoint/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/clerk/types/expected_keepers.go -destination=x/clerk/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/milestone/types/expected_keepers.go -destination=x/milestone/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/stake/types/expected_keepers.go -destination=x/stake/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/topup/types/expected_keepers.go -destination=x/topup/testutil/expected_keepers_mocks.go -package=testutil
	mockgen -source=x/bor/types/expected_keepers.go -destination=x/bor/testutil/expected_keepers_mocks.go  -package=testutil
	mockgen -destination=helper/mocks/mock_http_client.go.go -package=mocks --source=./helper/util.go HTTPClient
	go install github.com/vektra/mockery/v2/...@latest
	cd helper && mockery --name IContractCaller  --output ./mocks --filename=mock_contract_caller.go


###############################################################################
###                                docker                                   ###
###############################################################################

build-docker: # TODO-HV2: check this command once we have a proper docker build
	@echo Fetching latest tag: $(LATEST_GIT_TAG)
	git checkout $(LATEST_GIT_TAG)
	docker build -t "maticnetwork/heimdall:$(LATEST_GIT_TAG)" -f Dockerfile .

push-docker: # TODO-HV2: check this command once we have a proper docker push
	@echo Pushing docker tag image: $(LATEST_GIT_TAG)
	docker push "maticnetwork/heimdall:$(LATEST_GIT_TAG)"

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


.PHONY: help
help:
	@echo "Available targets:"
	@echo "  lint-deps           	- Install dependencies for GolangCI-Lint tool."
	@echo "  lint                	- Run the GolangCI-Lint tool on the codebase."
	@echo "  clean               	- Delete build folder."
	@echo "  test               	- Run the tests."
	@echo "  mock                	- Generate mocks."
	@echo "  proto-all           	- Format, lint and generate proto files."
	@echo "  proto-format        	- Format proto files."
	@echo "  proto-gen           	- Generate proto files."
	@echo "  proto-check-breaking   - Check if proto breaks against git head."
	@echo "  build-docker        	- Build a Docker image for the latest Git tag."
	@echo "  push-docker         	- Push the Docker image for the latest Git tag."
	@echo "  build-docker-develop	- Build a Docker image for the development branch."
	@echo "  release-dry-run     	- Perform a dry run of the release process."
	@echo "  release             	- Execute the actual release process."
