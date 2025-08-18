#!/bin/bash

# Arcron Development Script
# This script provides common development tasks for the Arcron project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    local missing_commands=()
    
    if ! command_exists go; then
        missing_commands+=("go")
    fi
    
    if ! command_exists git; then
        missing_commands+=("git")
    fi
    
    if ! command_exists make; then
        print_warning "Make not found. Some commands may not work."
    fi
    
    if [ ${#missing_commands[@]} -ne 0 ]; then
        print_error "Missing required commands: ${missing_commands[*]}"
        print_error "Please install the missing commands and try again."
        exit 1
    fi
    
    # Check Go version
    local go_version=$(go version | awk '{print $3}' | sed 's/go//')
    local required_version="1.22"
    
    if [ "$(printf '%s\n' "$required_version" "$go_version" | sort -V | head -n1)" != "$required_version" ]; then
        print_error "Go version $required_version or later is required. Current version: $go_version"
        exit 1
    fi
    
    print_success "All prerequisites met"
}

# Function to setup development environment
setup_dev() {
    print_status "Setting up development environment..."
    
    cd "$PROJECT_ROOT"
    
    # Create necessary directories
    mkdir -p config data logs models docs scripts
    
    # Download dependencies
    print_status "Downloading Go dependencies..."
    go mod download
    go mod tidy
    
    # Create default config if it doesn't exist
    if [ ! -f "config/arcron.yaml" ]; then
        print_status "Creating default configuration..."
        # The config package will create this automatically
    fi
    
    print_success "Development environment setup complete"
}

# Function to build the project
build_project() {
    print_status "Building Arcron..."
    
    cd "$PROJECT_ROOT"
    
    if command_exists make; then
        make build
    else
        mkdir -p bin
        go build -o bin/arcron ./cmd/arcron
    fi
    
    print_success "Build complete. Binary available at bin/arcron"
}

# Function to run tests
run_tests() {
    print_status "Running tests..."
    
    cd "$PROJECT_ROOT"
    
    if command_exists make; then
        make test
    else
        go test -v ./...
    fi
    
    print_success "Tests completed"
}

# Function to run the application
run_app() {
    print_status "Starting Arcron..."
    
    cd "$PROJECT_ROOT"
    
    if [ ! -f "bin/arcron" ]; then
        print_warning "Binary not found. Building first..."
        build_project
    fi
    
    local config_file="config/arcron.yaml"
    local log_level="info"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --config)
                config_file="$2"
                shift 2
                ;;
            --log-level)
                log_level="$2"
                shift 2
                ;;
            --daemon)
                print_status "Starting in daemon mode..."
                ./bin/arcron --config "$config_file" --log-level "$log_level" --daemon
                return
                ;;
            *)
                print_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    ./bin/arcron --config "$config_file" --log-level "$log_level"
}

# Function to clean the project
clean_project() {
    print_status "Cleaning project..."
    
    cd "$PROJECT_ROOT"
    
    if command_exists make; then
        make clean
    else
        rm -rf bin/
        go clean -cache
    fi
    
    print_success "Clean complete"
}

# Function to format code
format_code() {
    print_status "Formatting code..."
    
    cd "$PROJECT_ROOT"
    
    if command_exists make; then
        make format
    else
        go fmt ./...
        # Note: goimports is not included in standard Go installation
        print_warning "goimports not available. Install it for better formatting."
    fi
    
    print_success "Code formatting complete"
}

# Function to run linter
run_linter() {
    print_status "Running linter..."
    
    cd "$PROJECT_ROOT"
    
    if command_exists make; then
        make lint
    else
        if command_exists golangci-lint; then
            golangci-lint run
        else
            print_error "golangci-lint not found. Install it or use 'make lint'"
            exit 1
        fi
    fi
    
    print_success "Linting complete"
}

# Function to run security scan
run_security() {
    print_status "Running security scan..."
    
    cd "$PROJECT_ROOT"
    
    if command_exists make; then
        make security
    else
        if command_exists gosec; then
            gosec ./...
        else
            print_error "gosec not found. Install it or use 'make security'"
            exit 1
        fi
    fi
    
    print_success "Security scan complete"
}

# Function to show project status
show_status() {
    print_status "Project Status:"
    echo
    
    cd "$PROJECT_ROOT"
    
    # Show Go module info
    echo "Go Module:"
    go list -m
    echo
    
    # Show dependencies
    echo "Dependencies:"
    go list -m all | head -10
    echo "... (showing first 10)"
    echo
    
    # Show build status
    if [ -f "bin/arcron" ]; then
        print_success "Binary exists: bin/arcron"
        ls -lh bin/arcron
    else
        print_warning "Binary not found. Run 'build' to create it."
    fi
    echo
    
    # Show configuration
    if [ -f "config/arcron.yaml" ]; then
        print_success "Configuration exists: config/arcron.yaml"
    else
        print_warning "Configuration not found. Run 'setup' to create it."
    fi
}

# Function to show help
print_usage() {
    echo "Usage: $0 <command> [options]"
    echo
    echo "Commands:"
    echo "  setup     - Setup development environment"
    echo "  build     - Build the project"
    echo "  test      - Run tests"
    echo "  run       - Run the application"
    echo "  clean     - Clean build artifacts"
    echo "  format    - Format source code"
    echo "  lint      - Run linter"
    echo "  security  - Run security scan"
    echo "  status    - Show project status"
    echo "  help      - Show this help message"
    echo
    echo "Run options:"
    echo "  --config <file>     - Specify config file (default: config/arcron.yaml)"
    echo "  --log-level <level> - Set log level (default: info)"
    echo "  --daemon            - Run in daemon mode"
    echo
    echo "Examples:"
    echo "  $0 setup"
    echo "  $0 build"
    echo "  $0 run --log-level debug"
    echo "  $0 run --daemon"
}

# Main script logic
main() {
    local command="${1:-help}"
    
    case "$command" in
        setup)
            check_prerequisites
            setup_dev
            ;;
        build)
            check_prerequisites
            build_project
            ;;
        test)
            check_prerequisites
            run_tests
            ;;
        run)
            check_prerequisites
            shift
            run_app "$@"
            ;;
        clean)
            clean_project
            ;;
        format)
            check_prerequisites
            format_code
            ;;
        lint)
            check_prerequisites
            run_linter
            ;;
        security)
            check_prerequisites
            run_security
            ;;
        status)
            check_prerequisites
            show_status
            ;;
        help|--help|-h)
            print_usage
            ;;
        *)
            print_error "Unknown command: $command"
            print_usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
