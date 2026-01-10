.PHONY: build install clean test list release-dry-run install-deps

VERSION := $(shell git describe --tags 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Display available tasks
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs

# Build the application
build:
	go build $(LDFLAGS)

# Install the application
install:
	go install $(LDFLAGS)

# Clean the build
clean:
	@rm -f tiny-timer
	go clean -testcache

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Install development dependencies
install-deps:
	@echo "Installing development dependencies..."
	@go install github.com/goreleaser/goreleaser@latest
	@echo "Dependencies installed successfully!"

# Test release process with GoReleaser (dry-run)
release-dry-run:
	@echo "Running GoReleaser dry-run..."
	goreleaser release --snapshot --skip-publish --clean
