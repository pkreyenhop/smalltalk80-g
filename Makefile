# Smalltalk-80 Go Port Makefile

BINARY_NAME := Smalltalk
FILES_DIR := files
HEADLESS_CYCLES := 10000

.PHONY: all build test test-unit test-headless run clean help

all: build

## build: Compiles the Smalltalk-80 virtual machine executable
build:
	go build -o $(BINARY_NAME) .

## test: Runs unit tests and headless VM integration check
test: test-unit test-headless

## test-unit: Runs standard Go package unit tests
test-unit:
	go test ./...

## test-headless: Runs headless VM test for $(HEADLESS_CYCLES) cycles
test-headless: build
	./$(BINARY_NAME) -directory $(FILES_DIR) -headless -cycles $(HEADLESS_CYCLES)

## run: Launches the Smalltalk-80 graphical environment
run: build
	./$(BINARY_NAME) -directory $(FILES_DIR)

## clean: Removes build binary and temporary files
clean:
	rm -f $(BINARY_NAME)

## help: Display available targets
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build          Build the Smalltalk-80 binary"
	@echo "  test           Run unit tests and headless integration check"
	@echo "  test-unit      Run Go unit tests"
	@echo "  test-headless  Run headless VM integration test"
	@echo "  run            Launch Smalltalk-80 GUI"
	@echo "  clean          Remove build artifacts"
