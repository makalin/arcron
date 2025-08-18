.PHONY: help build clean test run install uninstall lint format deps

# Default target
help:
	@echo "Arcron - AI-Powered Autonomous Cron Agent"
	@echo ""
	@echo "Available targets:"
	@echo "  build     - Build the Arcron binary"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  run       - Build and run Arcron"
	@echo "  install   - Install Arcron to system"
	@echo "  uninstall - Remove Arcron from system"
	@echo "  lint      - Run linter"
	@echo "  format    - Format source code"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  docker    - Build Docker image"
	@echo "  release   - Build release binaries"

# Build variables
BINARY_NAME=arcron
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

# Build the binary
build:
	@echo "Building Arcron..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/arcron
	@echo "Binary built: ${BUILD_DIR}/${BINARY_NAME}"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf ${BUILD_DIR}
	@go clean -cache
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Build and run
run: build
	@echo "Running Arcron..."
	@${BUILD_DIR}/${BINARY_NAME} --help

# Install to system
install: build
	@echo "Installing Arcron..."
	@sudo cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	@sudo chmod +x /usr/local/bin/${BINARY_NAME}
	@echo "Arcron installed to /usr/local/bin/${BINARY_NAME}"

# Uninstall from system
uninstall:
	@echo "Uninstalling Arcron..."
	@sudo rm -f /usr/local/bin/${BINARY_NAME}
	@echo "Arcron uninstalled"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Format source code
format:
	@echo "Formatting source code..."
	go fmt ./...
	goimports -w .

# Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t arcron:latest .
	@echo "Docker image built: arcron:latest"

# Build release binaries for multiple platforms
release: clean
	@echo "Building release binaries..."
	@mkdir -p ${BUILD_DIR}
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ./cmd/arcron
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ./cmd/arcron
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ./cmd/arcron
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ./cmd/arcron
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ./cmd/arcron
	
	@echo "Release binaries built in ${BUILD_DIR}/"

# Development setup
dev-setup: deps
	@echo "Setting up development environment..."
	@mkdir -p config data logs models
	@echo "Development environment ready"

# Run with development config
dev: dev-setup
	@echo "Running in development mode..."
	@${BUILD_DIR}/${BINARY_NAME} --config config/arcron.yaml --log-level debug

# Generate documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p docs
	go doc -all ./... > docs/api.md
	@echo "Documentation generated in docs/"

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	gosec ./...
	@echo "Security check complete"

# Performance benchmark
bench:
	@echo "Running benchmarks..."
	go test -bench=. ./...

# Coverage report
coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
