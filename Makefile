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
PLUGIN_SRC = $(filter-out %_test.go,$(wildcard $(PLUGIN_DIR)/*/*.go))

.PHONY: all build plugins test test-install clean test-btrfs test-zfs

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
	rm -rf tests/bin

# Build the main Go application
build:
	$(GOBUILD) -o $(MAIN_OUT) .

# Copy xpire and plugins into a test directory, because the users of the
# integration tests cannot access a checkout in a private home directory
test-install: build plugins
	mkdir -p tests/bin
	cp $(MAIN_OUT) tests/bin/
	cp --parents $(PLUGIN_DIR)/*/*.so tests/bin/
	chmod -R a+rX tests/bin

# Run all plugin tests
test: test-setup test-all test-teardown

test-all: test-install
	@echo "running all tests"
	$(GOTEST) -tags integration . || { \
		$(MAKE) test-teardown; \
		exit 1; \
	}

# Run only BTRFS plugin tests
test-btrfs: test-setup-btrfs test-run-btrfs test-teardown-btrfs

test-run-btrfs: test-install
	@echo "running btrfs tests"
	$(GOTEST) -tags integration -run TestBTRFS . || { \
		$(MAKE) test-teardown-btrfs; \
		exit 1; \
	}

# Run only ZFS plugin tests
test-zfs: test-setup-zfs test-run-zfs test-teardown-zfs

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
