.PHONY: build install clean list

# Define the binary name
BINARY_NAME=pomodoro

# Display available tasks
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | xargs

# Build the application
build:
	go build -o $(BINARY_NAME)

# Install the application
install: build
	go install

# Clean the build
clean:
	rm -f $(BINARY_NAME)

