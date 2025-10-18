.PHONY: build run test clean dev

# Build the application
build:
	@echo "Building server..."
	@go build -o bin/server cmd/server/main.go
	@echo "Build complete: bin/server"

# Run the application
run:
	@echo "Starting server..."
	@go run cmd/server/main.go

# Run development mode with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@air

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf storage/audio/*.mp3
	@echo "Clean complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Create .env from example
setup:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created. Please update with your credentials."; \
	else \
		echo ".env file already exists."; \
	fi

help:
	@echo "Available commands:"
	@echo "  make build  - Build the application"
	@echo "  make run    - Run the application"
	@echo "  make dev    - Run in development mode (requires air)"
	@echo "  make test   - Run tests"
	@echo "  make clean  - Clean build artifacts"
	@echo "  make deps   - Install dependencies"
	@echo "  make fmt    - Format code"
	@echo "  make lint   - Run linter"
	@echo "  make setup  - Create .env file from example"
