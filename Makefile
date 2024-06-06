GO ?= latest
GOBIN = $(CURDIR)/build/bin
GORUN = env GO111MODULE=on go run
GOPATH = $(shell go env GOPATH)

GIT_COMMIT ?= $(shell git rev-list -1 HEAD)

DOCKER := $(shell which docker)
HTTPS_GIT := https://github.com/0xPolygon/heimdall-v2.git

PACKAGE_NAME := github.com/0xPolygon/heimdall-v2
GOLANG_CROSS_VERSION  ?= v1.21.0

# LDFlags
# BUILD_FLAGS

.PHONY: clean
clean:
	rm -rf build

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

mock:
	go install github.com/vektra/mockery/v2/...@latest
	cd helper && mockery --name IContractCaller  --output ./mocks --filename=mock_contract_caller.go
	go install github.com/golang/mock/mockgen@latest
	mockgen -destination=helper/mocks/mock_http_client.go.go -package=mocks --source=./helper/util.go HTTPClient

.PHONY: proto-all proto-gen proto-swagger-gen proto-format proto-lint proto-check-breaking proto-update-deps

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  lint-deps           - Install dependencies for GolangCI-Lint tool."
	@echo "  lint                - Runs the GolangCI-Lint tool on the codebase."
