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
PLUGIN_SRC = $(PLUGIN_DIR)/*/*.go

.PHONY: all build plugins test clean test-btrfs test-zfs

# Default target
all: plugins build

# Build the Go plugin(s)
plugins:
	for src in $(PLUGIN_SRC); do \
		$(GOBUILD) -buildmode=plugin -o $(PLUGIN_DIR)/$$(basename $$src .go)/$$(basename $$src .go).so $$src; \
	done

# Clean up
clean:
	$(GOCLEAN)
	rm -f $(MAIN_OUT) $(PLUGIN_DIR)/*/*.so

# Build the main Go application
build:
	$(GOBUILD) -o $(MAIN_OUT) .

# Run all plugin tests
test: test-setup test-all test-teardown

test-all:
	@echo "running all tests"
	$(GOTEST) || { \
		$(MAKE) test-teardown; \
		exit 1; \
	}

# Run only BTRFS plugin tests
test-btrfs: test-setup test-btrfs-run test-teardown

test-btrfs-run:
	@echo "running btrfs tests"
	$(GOTEST) -run TestBTRFS

# Run only ZFS plugin tests
test-zfs: test-setup test-zfs-run test-teardown

test-zfs-run:
	@echo "running zfs tests"
	$(GOTEST) -run TestZFS

test-setup:
	@echo "setting up testing environment"
	@cd tests \
		&& ./setup.sh > /dev/null

test-teardown:
	@echo "tearing down testing environment"
	@cd tests \
		&& ./teardown.sh
