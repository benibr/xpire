SHELL := /bin/bash
# Define the Go compiler
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST  = $(GOCMD) test -v

# Define the main Go application
MAIN_OUT = xpire

# Define the plugin directory and plugin output
PLUGIN_DIR = filesystems
PLUGIN_SRC = $(filter-out %_test.go,$(wildcard $(PLUGIN_DIR)/*/*.go))

# Define where test filesystems and the test binaries are created, must be
# accessible for unprivileged users to run tests as other users
export XPIRE_TEST_DIR ?= /var/tmp/xpire-test

.PHONY: all build plugins test test-unit test-install clean test-btrfs test-zfs

# Default target
all: plugins build

# Build the Go plugin(s)
plugins:
	for src in $(PLUGIN_SRC); do \
		$(GOBUILD) -buildmode=plugin -o $(PLUGIN_DIR)/$$(basename $$src .go)/$$(basename $$src .go).so $$src || exit 1; \
	done

# Clean up
clean:
	$(GOCLEAN)
	rm -f $(MAIN_OUT) $(PLUGIN_DIR)/*/*.so
	rm -rf $(XPIRE_TEST_DIR)/
	rm -rf tests/mnt tests/*.img

# Build the main Go application
build:
	$(GOBUILD) -o $(MAIN_OUT) .

# Copy xpire and plugins into the test directory, because the users of the
# integration tests cannot access a checkout in a private home directory
test-install: build plugins
	mkdir -p $(XPIRE_TEST_DIR)/bin
	cp $(MAIN_OUT) $(XPIRE_TEST_DIR)/bin/
	cp --parents $(PLUGIN_DIR)/*/*.so $(XPIRE_TEST_DIR)/bin/
	chmod -R a+rX $(XPIRE_TEST_DIR)

# Run all tests
test: test-unit test-install test-setup test-integration test-teardown

# Run unit tests, no root permissions or test filesystems needed
test-unit:
	@echo "running unit tests"
	$(GOTEST) ./...

test-integration: test-install
	@echo "running all tests"
	$(GOTEST) -tags integration . || { \
		$(MAKE) test-teardown; \
		exit 1; \
	}

# Run only BTRFS plugin tests
test-btrfs: test-install test-setup-btrfs test-run-btrfs test-teardown-btrfs

test-run-btrfs: test-install
	@echo "running btrfs tests"
	$(GOTEST) -tags integration -run TestBTRFS . || { \
		$(MAKE) test-teardown-btrfs; \
		exit 1; \
	}

# Run only ZFS plugin tests
test-zfs: test-install test-setup-zfs test-run-zfs test-teardown-zfs

test-run-zfs: test-install
	@echo "running zfs tests"
	$(GOTEST) -tags integration -run TestZFS . || { \
		$(MAKE) test-teardown-zfs; \
		exit 1; \
	}

test-setup:
	@echo "setting up testing environment"
	@cd tests \
		&& ./setup.sh > /dev/null

test-teardown:
	@echo "tearing down testing environment"
	@cd tests \
		&& ./teardown.sh

test-setup-btrfs:
	@echo "setting up btrfs test environment"
	@cd tests \
		&& ./setup-btrfs.sh > /dev/null

test-teardown-btrfs:
	@echo "tearing down btrfs test environment"
	@cd tests \
		&& ./teardown-btrfs.sh

test-setup-zfs:
	@echo "setting up zfs test environment"
	@cd tests \
		&& ./setup-zfs.sh > /dev/null

test-teardown-zfs:
	@echo "tearing down zfs test environment"
	@cd tests \
		&& ./teardown-zfs.sh
