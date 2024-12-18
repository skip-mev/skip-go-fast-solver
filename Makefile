SERVER_BIN=./build/skip_go_fast_solver
CLI_BIN=./build/solvercli
export PATH:=$(shell pwd)/tools/bin:$(PATH)
SHELL := env PATH='$(PATH)' /bin/sh

GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
GO_DEPS=go.mod go.sum

###############################################################################
###                                 Builds                                  ###
###############################################################################
${SERVER_BIN}: ${GO_FILES} ${GO_DEPS}
	go build -o ./build/skip_go_fast_solver github.com/skip-mev/go-fast-solver/cmd/solver

${CLI_BIN}: ${GO_FILES} ${GO_DEPS}
	go build -v -o ${CLI_BIN} github.com/skip-mev/go-fast-solver/cmd/solvercli

.PHONY: tidy build deps
tidy:
	go mod tidy

.PHONY: build
build: ${SERVER_BIN} 

.PHONY: build-cli
build-cli: ${CLI_BIN}

deps:
	go env
	go mod download

run-solver:
	${SERVER_BIN} --quickstart


###############################################################################
###                                 Testing                                 ###
###############################################################################
.PHONY: unit-test
unit-test:
	go test --tags=test -v -race $(shell go list ./... | grep -v /tests)

.PHONY: setup-foundry
setup-foundry:
	cd tests/e2e && forge install \
		foundry-rs/forge-std \
		OpenZeppelin/openzeppelin-contracts@v4.8.0 \
		OpenZeppelin/openzeppelin-contracts-upgradeable@v4.8.0 \
		hyperlane-xyz/hyperlane-monorepo \
		--no-commit

.PHONY: e2e-test
e2e-test:
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
