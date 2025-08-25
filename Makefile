# Makefile for go-innodb - InnoDB page parsing library

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Package name
PACKAGE=go-innodb

# Binary name
BINARY=go-innodb

# Build flags
BUILDFLAGS=-v

.PHONY: all build build-lib build-tool clean test fmt vet lint tidy help

# Default target
all: fmt vet build test

# Build both library and tool
build: build-lib build-tool

# Build the library only
build-lib:
	@echo "Building $(PACKAGE) library..."
	@$(GOBUILD) $(BUILDFLAGS) ./...

# Build the CLI tool
build-tool:
	@echo "Building $(BINARY)..."
	@$(GOBUILD) $(BUILDFLAGS) -o $(BINARY) ./cmd/$(BINARY)/

# Install the tool to $GOPATH/bin
install:
	@echo "Installing $(BINARY)..."
	@$(GOCMD) install ./cmd/$(BINARY)/

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f $(BINARY)
	@rm -f coverage.out coverage.html

# Run tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v -cover ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -w -s .
	@echo "Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	@$(GOVET) ./...

# Run all linting
lint: fmt vet
	@echo "Running staticcheck (if installed)..."
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"
	@echo "Running golint (if installed)..."
	@which golint > /dev/null 2>&1 && golint ./... || echo "golint not installed, skipping"

# Tidy go modules
tidy:
	@echo "Tidying go modules..."
	@$(GOMOD) tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download

# Initialize go module (if not exists)
init:
	@if [ ! -f go.mod ]; then \
		echo "Initializing go module..."; \
		$(GOMOD) init github.com/wilhasse/$(PACKAGE); \
	else \
		echo "go.mod already exists"; \
	fi

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

# Check for potential issues
check: fmt vet test
	@echo "All checks passed!"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@$(GOGET) -u golang.org/x/lint/golint
	@$(GOGET) -u honnef.co/go/tools/cmd/staticcheck
	@echo "Tools installed"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build      - Build both library and CLI tool"
	@echo "  make build-lib  - Build the library only"
	@echo "  make build-tool - Build the CLI tool only"
	@echo "  make install    - Install the CLI tool to GOPATH/bin"
	@echo "  make test       - Run tests"
	@echo "  make coverage   - Run tests with coverage report"
	@echo "  make fmt        - Format code"
	@echo "  make vet        - Run go vet"
	@echo "  make lint       - Run all linters"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make tidy       - Tidy go modules"
	@echo "  make deps       - Download dependencies"
	@echo "  make init       - Initialize go module (if needed)"
	@echo "  make bench      - Run benchmarks"
	@echo "  make check      - Run fmt, vet, and tests"
	@echo "  make install-tools - Install development tools"
	@echo "  make help       - Show this help message"