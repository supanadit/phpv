# PHPV — PHP Version Manager
#
# Usage:
#   make            Build the phpv binary into ./bin/phpv
#   make build      Same as `make` (alias)
#   make install    Build and install into $$PHPV_ROOT/bin/phpv
#                   (defaults PHPV_ROOT to $HOME/.phpv)
#   make clean      Remove build artifacts
#   make test       Run the test suite
#
# Overrides:
#   make install PHPV_ROOT=/custom/root        -> /custom/root/bin/phpv
#   make build BIN=/custom/phpv                 -> /custom/phpv
#   VERSION=1.2.3 make build                   -> bake version 1.2.3 into the binary

# --- Configuration -----------------------------------------------------------

# Package that holds the Version var, injected via -ldflags.
VERSION_PKG := github.com/supanadit/phpv/app

# Default to the nearest git tag; fall back to a short SHA, then "dev".
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

# Local build output (for `make` / `make build`).
BIN := bin/phpv

# Install destination. PHPV_ROOT defaults to $HOME/.phpv like install.sh.
PHPV_ROOT ?= $(HOME)/.phpv
INSTALL_BIN := $(PHPV_ROOT)/bin/phpv

# Go command. Respects PATH; falls back to common install locations if missing.
GO ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)

LDFLAGS := -ldflags "-X $(VERSION_PKG).Version=$(VERSION)"

# --- Targets -----------------------------------------------------------------

.PHONY: all build install clean test

all: build

build:
	@mkdir -p $(dir $(BIN))
	@echo "==> Building phpv $(VERSION)"
	$(GO) build $(LDFLAGS) -o $(BIN) ./app/phpv.go
	@echo "✓ Built $(BIN)"

install: build
	@mkdir -p $(dir $(INSTALL_BIN))
	@echo "==> Installing to $(INSTALL_BIN)"
	install -m 0755 $(BIN) $(INSTALL_BIN)
	@echo "✓ Installed phpv $(VERSION) to $(INSTALL_BIN)"
	@echo "    (PHPV_ROOT=$(PHPV_ROOT))"

clean:
	rm -rf bin
	@echo "✓ Cleaned build artifacts"

test:
	$(GO) test ./...
