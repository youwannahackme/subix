.PHONY: all build clean install test run

BINARY_NAME=subix
VERSION=2.0.0
BUILD_TIME=$(shell date -u +%Y%m%d%H%M%S)
LDFLAGS=-ldflags "-X github.com/youwannahackme/subix/cmd.version=$(VERSION) -X github.com/youwannahackme/subix/cmd.buildTime=$(BUILD_TIME)"

all: build

build:
    @echo "🔱 Building Subix v$(VERSION)..."
    go build $(LDFLAGS) -o $(BINARY_NAME) .
    @echo "✓ Build complete: ./$(BINARY_NAME)"

clean:
    @echo "Cleaning..."
    rm -f $(BINARY_NAME)
    go clean
    @echo "✓ Clean complete"

install: build
    @echo "Installing to $(GOPATH)/bin/..."
    cp $(BINARY_NAME) $(GOPATH)/bin/
    @echo "✓ Installed"

test:
    go test -v ./...

run: build
    ./$(BINARY_NAME) -d example.com

# Cross compilation
build-linux:
    GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
    @echo "✓ Built for linux/amd64"

build-windows:
    GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .
    @echo "✓ Built for windows/amd64"

build-mac:
    GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
    @echo "✓ Built for darwin/amd64 and darwin/arm64"

# Install config
install-config:
    @mkdir -p ~/.config/subix
    @cp configs/provider-config.yaml ~/.config/subix/provider-config.yaml
    @echo "✓ Config installed to ~/.config/subix/provider-config.yaml"

# Format code
fmt:
    go fmt ./...

# Vet code
vet:
    go vet ./...

# Lint (requires golangci-lint)
lint:
    golangci-lint run ./...