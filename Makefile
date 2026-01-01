.PHONY: build install clean test list

# Display available tasks
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs

# Build the application
build:
	go build

# Install the application
install:
	go install

# Clean the build
clean:
	go clean -testcache

# Run tests
test:
	@echo "Running tests..."
	go test ./...
