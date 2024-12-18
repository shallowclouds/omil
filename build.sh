#!/usr/bin/env bash

set -e

APP_NAME="omil"

VERSION=$(git describe --tags --always)
FLAGS="-X 'main.compiledTimeString=$(date --rfc-3339='seconds')' -X main.version=${VERSION}"

mkdir -p build/bin build/conf
cp scripts/bootstrap.sh scripts/omil.service.template scripts/install.sh build/ 2>/dev/null
cp conf/example.config.yml build/conf/ 2>/dev/null

chmod +x build/bootstrap.sh

# Build for multiple platforms
PLATFORMS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
for platform in $PLATFORMS; do
  GOOS=${platform%/*}
  GOARCH=${platform#*/}
  output_name="build/bin/$APP_NAME-${VERSION}-${TIMESTAMP}-$GOOS-$GOARCH"
  if [ $GOOS = "windows" ]; then
    output_name+='.exe'
  fi
  GOOS=$GOOS GOARCH=$GOARCH GO111MODULE=on go build -ldflags "$FLAGS" -o "$output_name" main.go
done
