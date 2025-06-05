SHELL := /bin/bash
# Define the Go compiler
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST  = $(GOCMD) test

# Define the main Go application
MAIN_OUT = xpire

# Define the plugin directory and plugin output
PLUGIN_DIR = filesystems
PLUGIN_SRC = $(PLUGIN_DIR)/*.go

.PHONY: all build plugins test clean

# Default target
all: plugins build

# Build the Go plugin(s)
plugins:
	for src in $(PLUGIN_SRC); do \
		$(GOBUILD) -buildmode=plugin -o $(PLUGIN_DIR)/$$(basename $$src .go).so $$src; \
		done

# Clean up
clean:
	$(GOCLEAN)
	rm -f $(MAIN_OUT) $(PLUGIN_DIR)/*.so

## Build the main Go application
build:
	$(GOBUILD) -o $(MAIN_OUT) .

test: test-setup test-all test-teardown

test-setup:
	@echo "setup testing environment"
	@cd tests \
		&& ./setup.sh > /dev/null

test-all:
	@echo "running tests"
		@$(GOTEST) || { \
			$(MAKE) test-teardown; \
			exit 1; \
			}

test-teardown:
	@echo "teardown testing environment"
		@cd tests \
			&& ./teardown.sh
