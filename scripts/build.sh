#!/bin/bash
set -e

COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")

LDFLAGS="-X github.com/supanadit/phpv/domain.AppGitCommit=${COMMIT} -X github.com/supanadit/phpv/domain.AppGitBranch=${BRANCH}"

echo "Building phpv..."
echo "  Commit: ${COMMIT}"
echo "  Branch: ${BRANCH}"

go build -ldflags "${LDFLAGS}" -o phpv ./app/phpv.go

echo "Build complete: ./phpv"
