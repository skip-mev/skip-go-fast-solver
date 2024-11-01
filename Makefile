SERVER_BIN=./build/fast_transfer_solver
export PATH:=$(shell pwd)/tools/bin:$(PATH)
SHELL := env PATH='$(PATH)' /bin/sh

GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
GO_DEPS=go.mod go.sum

###############################################################################
###                                 Builds                                  ###
###############################################################################
${SERVER_BIN}: ${GO_FILES} ${GO_DEPS}
	go build -o ./build/fast_transfer_solver github.com/skip-mev/go-fast-solver/cmd/solver

.PHONY: tidy build deps
tidy:
	go mod tidy

.PHONY: build
build: ${SERVER_BIN} 

deps:
	go env
	go mod download

run-solver:
	quickstart=true ${SERVER_BIN}

###############################################################################
###                                 Testing                                 ###
###############################################################################
.PHONY: unit-test
unit-test:
	go test --tags=test -v -race $(shell go list ./... | grep -v /tests)

.PHONY: setup-foundry
setup-foundry:
	cd tests/e2e && forge install foundry-rs/forge-std OpenZeppelin/openzeppelin-contracts OpenZeppelin/openzeppelin-contracts-upgradeable --no-commit
	
.PHONY: e2e-test
e2e-test: setup-foundry
	cd tests/e2e && go test -v ./

###############################################################################
###                                 Developer Tools                         ###
###############################################################################
.PHONY: db-exec db-clean tidy test
db-exec:
	sqlite3 solver.db

db-clean:
	rm solver.db

.PHONY: tools
tools:
	make -C tools local
