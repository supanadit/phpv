#!/usr/bin/env bash
#
# PHPV Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/uninstall.sh | bash
#        curl -fsSL https://raw.githubusercontent.com/supanadit/phpv/main/uninstall.sh | bash -s -- --yes
#

set -e

PHPV_UNINSTALLER_VERSION="1.0.0"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
	echo -e "${BLUE}==>${NC} $*" >&2
}

log_success() {
	echo -e "${GREEN}✓${NC} $*" >&2
}

log_warn() {
	echo -e "${YELLOW}⚠${NC} $*" >&2
}

log_error() {
	echo -e "${RED}✗${NC} $*" >&2
}

usage() {
	cat <<EOF
Usage: uninstall.sh [OPTIONS]

Options:
  -y, --yes    Skip confirmation and uninstall immediately
  -h, --help   Show this help message

Environment variables:
  PHPV_ROOT    Path to the phpv installation directory (default: \$HOME/.phpv)
EOF
}

detect_shell_configs() {
	local configs=()

	# Known shell config files
	local known_files=(
		"${HOME}/.bashrc"
		"${HOME}/.bash_profile"
		"${HOME}/.profile"
		"${HOME}/.zshrc"
		"${HOME}/.zprofile"
		"${HOME}/.config/fish/config.fish"
		"${HOME}/.config/powershell/Microsoft.PowerShell_profile.ps1"
		"${HOME}/.config/powershell/profile.ps1"
		"${HOME}/.kshrc"
		"${HOME}/.mkshrc"
	)

	for file in "${known_files[@]}"; do
		if [ -f "$file" ]; then
			configs+=("$file")
		fi
	done

	# Also discover any file under HOME that references phpv init
	local discovered=""
	if command -v rg >/dev/null 2>&1; then
		discovered=$(rg -l --hidden --max-depth 3 "phpv init" "$HOME" 2>/dev/null || true)
	elif command -v grep >/dev/null 2>&1 && command -v find >/dev/null 2>&1; then
		discovered=$(find "$HOME" -maxdepth 3 -type f -not -path '*/\.*' -exec grep -l "phpv init" {} + 2>/dev/null || true)
	fi

	for file in $discovered; do
		# Skip if already in the list
		local found=0
		for existing in "${configs[@]}"; do
			if [ "$existing" = "$file" ]; then
				found=1
				break
			fi
		done
		if [ "$found" -eq 0 ] && [ -f "$file" ]; then
			configs+=("$file")
		fi
	done

	printf '%s\n' "${configs[@]}"
}

remove_shell_integration() {
	local file="$1"

	if [ ! -f "$file" ]; then
		return 0
	fi

	if ! grep -q "phpv init\|PHPV - PHP Version Manager" "$file" 2>/dev/null; then
		return 0
	fi

	log_info "Removing phpv shell integration from ${file}..."

	local tmp_file
	tmp_file=$(mktemp)

	# Remove the PHPV block and the blank line that follows it (if any).
	awk '
		/phpv init/ { skip_blank = 1; next }
		/^# PHPV - PHP Version Manager/ { skip_blank = 1; next }
		skip_blank && /^[[:space:]]*$/ { skip_blank = 0; next }
		{ skip_blank = 0; print }
	' "$file" >"$tmp_file"

	if ! diff -q "$file" "$tmp_file" >/dev/null 2>&1; then
		mv "$tmp_file" "$file"
		log_success "Removed phpv shell integration from ${file}"
	else
		rm -f "$tmp_file"
	fi
}

remove_phpv_root() {
	local phpv_root="$1"

	if [ ! -d "$phpv_root" ]; then
		log_warn "PHPV_ROOT directory not found: ${phpv_root}"
		return 0
	fi

	log_info "Removing phpv installation directory: ${phpv_root}..."
	rm -rf "$phpv_root"
	log_success "Removed ${phpv_root}"
}

confirm_uninstall() {
	local phpv_root="$1"
	shift
	local configs=("$@")

	echo
	echo "This will uninstall phpv by:"
	echo "  - Removing the PHPV_ROOT directory: ${phpv_root}"
	if [ "${#configs[@]}" -gt 0 ] && [ -n "${configs[0]}" ]; then
		echo "  - Removing phpv initialization from the following shell config files:"
		for cfg in "${configs[@]}"; do
			echo "      ${cfg}"
		done
	else
		echo "  - No shell config files with phpv integration were found."
	fi
	echo

	local answer
	read -r -p "Are you sure you want to continue? [y/N] " answer
	case "$answer" in
	[yY] | [yY][eE][sS]) return 0 ;;
	*) return 1 ;;
	esac
}

main() {
	local yes=0

	while [ "$#" -gt 0 ]; do
		case "$1" in
		-y | --yes) yes=1 ;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			log_error "Unknown option: $1"
			usage
			exit 1
			;;
		esac
		shift
	done

	log_info "PHPV Uninstaller v${PHPV_UNINSTALLER_VERSION}"
	echo

	local phpv_root="${PHPV_ROOT:-$HOME/.phpv}"
	log_info "PHPV_ROOT: ${phpv_root}"

	local configs=""
	configs=$(detect_shell_configs)

	# Parse newline-separated config list into an array (portable for bash 3+)
	local config_array=()
	while IFS= read -r line; do
		[ -n "$line" ] && config_array+=("$line")
	done <<<"$configs"

	if [ "$yes" -eq 0 ]; then
		if ! confirm_uninstall "$phpv_root" "${config_array[@]}"; then
			echo
			log_info "Uninstall cancelled."
			exit 0
		fi
	fi

	echo

	for cfg in "${config_array[@]}"; do
		remove_shell_integration "$cfg"
	done

	remove_phpv_root "$phpv_root"

	echo
	log_success "phpv has been uninstalled successfully."
	log_info "Please restart your shell or run: source <your-shell-config>"
}

main "$@"
