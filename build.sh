#!/usr/bin/env bash

set -e

APP_NAME="omil"

FLAGS="-X main.compiledTimeString=$(date --rfc-3339='seconds') -X main.version=$(git rev-parse --short HEAD)"

mkdir -p build/bin build/conf
cp script/bootstrap.sh build/ 2>/dev/null

chmod +x build/bootstrap.sh

GO111MODULE=on go build -ldflags "$FLAGS" -o "build/bin/$APP_NAME" main.go
