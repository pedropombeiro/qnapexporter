# Variables
PKG := "github.com/pedropombeiro/qnapexporter"
VERSION_PKG := PKG + "/lib/utils"
PACKAGE_VERSION := env_var_or_default("PACKAGE_VERSION", "dev")
REVISION := `git rev-parse --short=8 HEAD || echo unknown`
BRANCH := `git show-ref | grep "$(git rev-parse --short=8 HEAD || echo unknown)" | grep -v HEAD | awk '{print $2}' | sed 's|refs/remotes/origin/||' | sed 's|refs/heads/||' | sort | head -n 1`
BUILT := `date -u +%Y-%m-%dT%H:%M:%S%z`

GO_LDFLAGS := "-X " + VERSION_PKG + ".REVISION=" + REVISION + " -X " + VERSION_PKG + ".BUILT=" + BUILT + " -X " + VERSION_PKG + ".BRANCH=" + BRANCH + " -X " + VERSION_PKG + ".VERSION=" + PACKAGE_VERSION + " -s -w"

BIN_PATH := "bin/qnapexporter"

# Default recipe (show available commands)
[private]
default:
    @just --list

# Install mise and dependencies
install:
    @echo "Installing dependencies..."
    mise install
    pre-commit install
    @echo "Dependencies ready"

# Update Go module dependencies
update:
    @echo "Updating Go modules..."
    go mod tidy
    go mod vendor
    @echo "Go modules updated"

# Run tests with optional parameters
test *args:
    @go test {{args}} ./...

# Run comprehensive linters
lint:
    @echo "Running linters..."
    golangci-lint run

# Run formatters and auto-fixable linters
fix:
    @echo "Running pre-commit hooks..."
    pre-commit run --all-files

# Build binary with version embedding
build:
    @echo "Building qnapexporter..."
    @mkdir -p ./bin
    go build -mod=readonly -ldflags "{{GO_LDFLAGS}}" -o {{BIN_PATH}} .
    @echo "Build complete: {{BIN_PATH}}"

# Generate test mocks
mocks:
    @echo "Generating mocks..."
    @find . -name mock_*.go -delete
    mockery --dir=. --recursive --all --inpackage
    @echo "Mocks generated"

# Vendor management (alias to update)
vendor:
    @just update

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    @rm -rf ./bin
    @echo "Clean complete"

# Show build metadata
info:
    @echo "Build Information:"
    @echo "  Package: {{PKG}}"
    @echo "  Version: {{PACKAGE_VERSION}}"
    @echo "  Revision: {{REVISION}}"
    @echo "  Branch: {{BRANCH}}"
    @echo "  Built: {{BUILT}}"

# Aliases for common commands
alias b := build
alias t := test
alias v := vendor
