.PHONY: build clean install test

BINARY_NAME=datadog-monitor-manager
MAIN_PATH=./main.go

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

install:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Install complete"

test:
	@echo "Running tests..."
	@go test ./...

run: build
	@./$(BINARY_NAME)

help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  clean    - Remove built binaries"
	@echo "  install  - Download and tidy dependencies"
	@echo "  test     - Run tests"
	@echo "  run      - Build and run the binary"
	@echo "  help     - Show this help message"

