GO ?= latest
GOBIN = $(CURDIR)/build/bin
GORUN = env GO111MODULE=on go run
GOPATH = $(shell go env GOPATH)

GIT_COMMIT ?= $(shell git rev-list -1 HEAD)

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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.55.2

.PHONY: lint
lint:
	@./build/bin/golangci-lint run --config ./.golangci.yml

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  lint-deps           - Install dependencies for GolangCI-Lint tool."
	@echo "  lint                - Runs the GolangCI-Lint tool on the codebase."
