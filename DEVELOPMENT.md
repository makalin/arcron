# Arcron Development Guide

This document provides comprehensive information for developers who want to contribute to the Arcron project.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Development Environment Setup](#development-environment-setup)
3. [Project Structure](#project-structure)
4. [Building and Testing](#building-and-testing)
5. [Architecture](#architecture)
6. [Contributing Guidelines](#contributing-guidelines)
7. [Testing Strategy](#testing-strategy)
8. [Performance Considerations](#performance-considerations)
9. [Security Guidelines](#security-guidelines)
10. [Troubleshooting](#troubleshooting)

## Project Overview

Arcron is an AI-powered autonomous cron agent that replaces traditional static cron scheduling with intelligent, adaptive scheduling based on system metrics and machine learning predictions.

### Key Features

- **Dynamic Scheduling**: Optimizes job execution based on predicted system load
- **Machine Learning Core**: Learns patterns from historical system metrics
- **Resource Awareness**: Avoids collisions with peak usage periods
- **Self-Healing**: Adjusts schedules when predictions deviate from reality
- **System Agnostic**: Works across Linux-based environments

## Development Environment Setup

### Prerequisites

- Go 1.22 or later
- Git
- Make (optional, for using Makefile)
- Docker (optional, for containerized development)

### Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/makalin/arcron.git
   cd arcron
   ```

2. **Install dependencies**
   ```bash
   make deps
   # or manually:
   go mod download
   go mod tidy
   ```

3. **Build the project**
   ```bash
   make build
   # or manually:
   go build -o bin/arcron ./cmd/arcron
   ```

4. **Run tests**
   ```bash
   make test
   # or manually:
   go test ./...
   ```

### Development Setup

```bash
# Set up development environment
make dev-setup

# Run in development mode
make dev

# Format code
make format

# Run linter
make lint

# Run security scan
make security
```

## Project Structure

```
arcron/
├── cmd/arcron/           # Main application entry point
├── internal/             # Internal packages (not importable from outside)
│   ├── arcron/          # Main application logic
│   ├── config/          # Configuration management
│   ├── jobs/            # Job execution and management
│   ├── ml/              # Machine learning engine
│   ├── monitoring/      # System metrics collection
│   ├── scheduler/       # Intelligent job scheduling
│   ├── storage/         # Data persistence layer
│   └── types/           # Shared type definitions
├── config/              # Configuration files
├── data/                # Data storage (created at runtime)
├── logs/                # Log files (created at runtime)
├── models/              # ML models (created at runtime)
├── docs/                # Documentation
├── scripts/             # Utility scripts
├── .github/             # GitHub configuration
├── Dockerfile           # Container definition
├── docker-compose.yml   # Development environment
├── Makefile             # Build automation
├── go.mod               # Go module definition
├── go.sum               # Go module checksums
└── README.md            # Project overview
```

## Building and Testing

### Build Targets

```bash
# Build for current platform
make build

# Build for multiple platforms
make release

# Build Docker image
make docker

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run benchmarks
make bench

# Run specific package tests
go test ./internal/config
go test -v ./internal/jobs
```

### Code Quality

```bash
# Format code
make format

# Run linter
make lint

# Security scan
make security

# Generate documentation
make docs
```

## Architecture

### Core Components

1. **Configuration Manager** (`internal/config/`)
   - Loads and validates configuration files
   - Provides default values
   - Supports YAML configuration

2. **System Monitor** (`internal/monitoring/`)
   - Collects system metrics (CPU, memory, I/O, network)
   - Provides real-time system state
   - Configurable collection intervals

3. **Machine Learning Engine** (`internal/ml/`)
   - Predicts optimal job execution times
   - Learns from historical data
   - Supports multiple ML algorithms

4. **Intelligent Scheduler** (`internal/scheduler/`)
   - Manages job scheduling and execution
   - Adjusts schedules based on ML predictions
   - Handles job priorities and dependencies

5. **Job Manager** (`internal/jobs/`)
   - Executes jobs with proper isolation
   - Manages retries and error handling
   - Tracks execution history

6. **Storage Layer** (`internal/storage/`)
   - Persists job executions and metrics
   - Supports multiple database backends
   - Handles data cleanup and archival

### Data Flow

```
System Metrics → ML Engine → Predictions → Scheduler → Job Execution
     ↓              ↓           ↓           ↓           ↓
  Storage ←    Training ←   Feedback ←  Results ←   Monitoring
```

## Contributing Guidelines

### Code Style

- Follow Go standard formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and small
- Use interfaces for testability

### Commit Messages

Follow conventional commit format:
```
type(scope): description

[optional body]

[optional footer]
```

Examples:
```
feat(scheduler): add ML-based schedule adjustment
fix(monitoring): resolve memory leak in metrics collection
docs(readme): update installation instructions
```

### Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Update documentation if needed
7. Submit a pull request

### Testing Requirements

- All new code must have tests
- Maintain at least 80% code coverage
- Include integration tests for new features
- Add benchmarks for performance-critical code

## Testing Strategy

### Unit Tests

- Test individual functions and methods
- Mock external dependencies
- Use table-driven tests for multiple scenarios
- Test both success and failure cases

### Integration Tests

- Test component interactions
- Use test databases and mock services
- Test configuration loading and validation
- Verify error handling and recovery

### Performance Tests

- Benchmark critical code paths
- Test with realistic data volumes
- Monitor memory usage and GC pressure
- Profile CPU usage patterns

## Performance Considerations

### Optimization Areas

1. **Metrics Collection**
   - Use efficient system calls
   - Implement sampling for high-frequency metrics
   - Cache frequently accessed data

2. **ML Predictions**
   - Batch predictions when possible
   - Use efficient data structures
   - Implement model caching

3. **Job Execution**
   - Use goroutines for concurrent execution
   - Implement job queuing and prioritization
   - Monitor resource usage

### Monitoring

- Track execution times and success rates
- Monitor memory and CPU usage
- Log performance metrics
- Set up alerts for performance degradation

## Security Guidelines

### General Principles

- Never trust user input
- Validate all configuration values
- Use principle of least privilege
- Implement proper authentication and authorization
- Log security-relevant events

### Specific Considerations

1. **Job Execution**
   - Sanitize command inputs
   - Limit file system access
   - Implement resource limits
   - Use process isolation

2. **Configuration**
   - Validate file permissions
   - Encrypt sensitive data
   - Use secure defaults
   - Implement access controls

3. **Network Security**
   - Use HTTPS for web interfaces
   - Implement rate limiting
   - Validate API inputs
   - Monitor for suspicious activity

## Troubleshooting

### Common Issues

1. **Build Failures**
   ```bash
   # Clean and rebuild
   make clean
   make deps
   make build
   ```

2. **Test Failures**
   ```bash
   # Run tests with verbose output
   go test -v ./...
   
   # Run specific test
   go test -v ./internal/config -run TestLoadConfig
   ```

3. **Dependency Issues**
   ```bash
   # Update dependencies
   go get -u ./...
   go mod tidy
   ```

4. **Configuration Issues**
   ```bash
   # Validate configuration
   ./bin/arcron --config config/arcron.yaml --validate
   
   # Check configuration syntax
   yamllint config/arcron.yaml
   ```

### Debug Mode

```bash
# Run with debug logging
./bin/arcron --config config/arcron.yaml --log-level debug

# Enable profiling
./bin/arcron --config config/arcron.yaml --profile
```

### Log Analysis

```bash
# View recent logs
tail -f logs/arcron.log

# Search for errors
grep "ERROR" logs/arcron.log

# Analyze log patterns
grep "schedule adjustment" logs/arcron.log | jq .
```

## Getting Help

- **Issues**: Create GitHub issues for bugs and feature requests
- **Discussions**: Use GitHub Discussions for questions and ideas
- **Documentation**: Check the README and inline code comments
- **Code**: Review existing code for examples and patterns

## Next Steps

1. Set up your development environment
2. Explore the codebase and understand the architecture
3. Pick a small issue to start with
4. Join the community discussions
5. Contribute your first pull request

Welcome to the Arcron project! We're excited to have you contribute to building the future of intelligent job scheduling.
