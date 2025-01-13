#!/usr/bin/env bash

set -e

APP_NAME="omil"

# Set version from environment variable or default to 1.0.0
VERSION=${VERSION:-"1.0.0"}
GIT_HASH=$(git rev-parse --short HEAD)

# Include both version and git hash in binary for backward compatibility
FLAGS="-X 'main.compiledTimeString=$(date --rfc-3339='seconds')' -X main.version=${VERSION} -X main.gitHash=${GIT_HASH}"

mkdir -p build/bin build/conf
cp scripts/bootstrap.sh scripts/omil.service.template scripts/install.sh build/ 2>/dev/null
cp conf/example.config.yml build/conf/ 2>/dev/null

chmod +x build/bootstrap.sh

# Build binary with version in name for clarity
GO111MODULE=on go build -ldflags "$FLAGS" -o "build/bin/${APP_NAME}-v${VERSION}-${GIT_HASH}" main.go

# Create symlink to versioned binary for backward compatibility
ln -sf "${APP_NAME}-v${VERSION}-${GIT_HASH}" "build/bin/${APP_NAME}"
