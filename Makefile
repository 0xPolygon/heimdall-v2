GO ?= latest
GOBIN = $(CURDIR)/build/bin
GORUN = env GO111MODULE=on go run
GOPATH = $(shell go env GOPATH)

GIT_COMMIT ?= $(shell git rev-list -1 HEAD)

PACKAGE = github.com/0xPolygon/heimdall-v2

# LDFlags
# BUILD_FLAGS


clean:
	rm -rf build


LINT_COMMAND := $(shell command -v golangci-lint 2> /dev/null)
lint:
ifndef LINT_COMMAND
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3
endif
	golangci-lint run --config ./.golangci.yml