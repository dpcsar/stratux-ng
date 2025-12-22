# Stratux-NG developer Makefile
#
# Common usage:
#   make test
#   make build
#   make run            # uses CONFIG (default: ./config.yaml)
#   CONFIG=/data/stratux-ng/config.yaml make run
#   make fmt vet
#
# Notes:
# - `run` uses STRATUX_NG_CONFIG under the hood so it matches the binary behavior.

APP_NAME := stratux-ng
CMD_DIR := ./cmd/stratux-ng
BIN_DIR := ./bin
BIN := $(BIN_DIR)/$(APP_NAME)

# Default config for dev. Override with:
#   CONFIG=/data/stratux-ng/config.yaml make run
CONFIG ?= ./config.yaml

.PHONY: help test build run fmt vet tidy clean

help:
	@printf "%s\n" "Targets:" \
	  "  make test        Run unit tests" \
	  "  make build       Build ./bin/stratux-ng" \
	  "  make run         Run via go run (CONFIG=$(CONFIG))" \
	  "  make fmt         gofmt all .go files" \
	  "  make vet         go vet ./..." \
	  "  make tidy        go mod tidy" \
	  "  make clean       Remove ./bin"

test:
	go test ./...

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) $(CMD_DIR)

run:
	STRATUX_NG_CONFIG=$(CONFIG) go run $(CMD_DIR)

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
