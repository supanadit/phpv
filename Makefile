# PHPV Makefile

.PHONY: install uninstall test help

# Default installation directory
PREFIX ?= $(HOME)/.phpv
BINDIR = $(PREFIX)/bin

help:
	@echo "PHPV - PHP Version Manager"
	@echo ""
	@echo "Available targets:"
	@echo "  install    Install phpv to ~/.phpv (or PREFIX)"
	@echo "  uninstall  Remove phpv installation"
	@echo "  test       Run basic tests"
	@echo "  help       Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX     Installation prefix (default: ~/.phpv)"

install:
	@echo "Installing PHPV to $(PREFIX)..."
	@mkdir -p $(BINDIR)
	@cp phpv.sh $(BINDIR)/phpv
	@chmod +x $(BINDIR)/phpv
	@mkdir -p $(PREFIX)
	@cp -f setup.sh $(PREFIX)/
	@chmod +x $(PREFIX)/setup.sh
	@echo "Installation complete!"
	@echo ""
	@echo "To complete setup, run:"
	@echo "  $(PREFIX)/setup.sh"

uninstall:
	@echo "Uninstalling PHPV from $(PREFIX)..."
	@rm -rf $(PREFIX)
	@echo "PHPV uninstalled."
	@echo ""
	@echo "Note: You may need to remove the phpv source line from your shell config file."

test:
	@echo "Running basic tests..."
	@bash -n phpv.sh && echo "✓ Syntax check passed"
	@chmod +x phpv.sh
	@./phpv.sh help > /dev/null && echo "✓ Help command works"
	@./phpv.sh list > /dev/null && echo "✓ List command works"
	@./phpv.sh current > /dev/null && echo "✓ Current command works"
	@echo "✓ Basic tests passed"