#!/usr/bin/env bash

# PHPV - Main command functions

detect_readdir_r_variant() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/readdir_r.XXXXXX") || return 1

    cat > "$tmp_dir/test_posix.c" <<'EOF'
#define _REENTRANT
#include <sys/types.h>
#include <dirent.h>
int main(void) {
    DIR *dir = 0;
    struct dirent entry;
    struct dirent *result = 0;
    return readdir_r(dir, &entry, &result);
}
EOF

    local variant="unknown"
    if "$cc" -o "$tmp_dir/test_posix" "$tmp_dir/test_posix.c" >/dev/null 2>&1; then
        variant="posix"
    else
        cat > "$tmp_dir/test_old.c" <<'EOF'
#define _REENTRANT
#include <sys/types.h>
#include <dirent.h>
int main(void) {
    DIR *dir = 0;
    struct dirent entry;
    return readdir_r(dir, &entry);
}
EOF
        if "$cc" -o "$tmp_dir/test_old" "$tmp_dir/test_old.c" >/dev/null 2>&1; then
            variant="old"
        fi
    fi

    rm -rf "$tmp_dir"
    [[ "$variant" == "unknown" ]] && return 1
    printf '%s\n' "$variant"
}

detect_fork_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/fork_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_fork.c" <<'EOF'
#include <unistd.h>
int main(void) {
    return fork() == -1;
}
EOF

    if "$cc" -o "$tmp_dir/test_fork" "$tmp_dir/test_fork.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_waitpid_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/waitpid_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_waitpid.c" <<'EOF'
#include <sys/types.h>
#include <sys/wait.h>
int main(void) {
    int (*fn)(pid_t, int*, int) = waitpid;
    return fn == 0;
}
EOF

    if "$cc" -o "$tmp_dir/test_waitpid" "$tmp_dir/test_waitpid.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_sigaction_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/sigaction_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_sigaction.c" <<'EOF'
#include <signal.h>
static void noop(int sig) {(void)sig;}
int main(void) {
    struct sigaction sa;
    sa.sa_handler = noop;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    return sigaction(SIGINT, &sa, 0) == -1;
}
EOF

    if "$cc" -o "$tmp_dir/test_sigaction" "$tmp_dir/test_sigaction.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_wait_support() {
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/wait_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_wait.c" <<'EOF'
#include <sys/types.h>
#include <sys/wait.h>
int main(void) {
    int status = 0;
    return wait(&status) == -1;
}
EOF

    if "$cc" -o "$tmp_dir/test_wait" "$tmp_dir/test_wait.c" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}

detect_header_support() {
    local header="$1"
    local cc="${CC:-cc}"
    local base_dir="${PHPV_CACHE_DIR:-/tmp}"
    local tmp_dir
    tmp_dir=$(mktemp -d "$base_dir/header_probe.XXXXXX") || return 1

    cat > "$tmp_dir/test_header.c" <<EOF
#include <$header>
int main(void) { return 0; }
EOF

    if "$cc" -c "$tmp_dir/test_header.c" -o "$tmp_dir/test_header.o" >/dev/null 2>&1; then
        rm -rf "$tmp_dir"
        return 0
    fi

    rm -rf "$tmp_dir"
    return 1
}