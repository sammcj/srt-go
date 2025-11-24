.PHONY: build universal install test clean lint fmt run

# Build for current architecture (default)
build:
	@echo "Building for current architecture..."
	mkdir -p bin
	@go build -ldflags="-s -w" -o bin/srt ./cmd/srt
	@echo "✓ Built srt"

# Build universal binary (ARM64 + AMD64)
universal:
	mkdir -p bin
	@echo "Building for ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/srt-arm64 ./cmd/srt
	@echo "Building for AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/srt-amd64 ./cmd/srt
	@echo "Creating universal binary..."
	@lipo -create srt-arm64 srt-amd64 -output bin/srt
	@rm srt-arm64 srt-amd64 bin/*
	@echo "✓ Built bin/srt (universal binary)"

# Install to /usr/local/bin
install: universal
	@echo "Installing to /usr/local/bin..."
	@install -m 700 srt /usr/local/bin/srt
	@echo "✓ Installed srt to /usr/local/bin"

# Run tests
test:
	@echo "Checking system requirements..."
	@sw_vers -productVersion | awk -F. '{if ($$1 < 26) exit 1}' || \
		(echo "⚠ Tests require macOS 26+" && exit 1)
	@echo "Running tests..."
	@go test -v -race ./...
	@echo "✓ Tests passed"

# Lint and format code
lint:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Linting complete"

# Format code
fmt:
	@go fmt ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies ready"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f srt srt-arm64 srt-amd64 bin/*
	@echo "✓ Cleaned"

# Run the tool (for testing)
run:
	@go run ./cmd/srt $(ARGS)

# Show help
help:
	@echo "Available targets:"
	@echo "  build     - Build for current architecture (default)"
	@echo "  universal - Build universal binary (ARM64 + AMD64)"
	@echo "  install   - Build universal and install to /usr/local/bin"
	@echo "  test      - Run tests (requires macOS 26+)"
	@echo "  lint      - Format and lint code"
	@echo "  fmt       - Format code only"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  clean     - Remove build artifacts"
	@echo "  run       - Run tool with ARGS='your command'"
	@echo "  help      - Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make universal"
	@echo "  make run ARGS='\"ls -la\"'"
	@echo "  make universal install"
