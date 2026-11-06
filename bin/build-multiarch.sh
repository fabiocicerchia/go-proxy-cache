#!/bin/bash

mkdir dist >/dev/null 2>&1 || true

package_name=go-proxy-cache
platforms=("freebsd/amd64" "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64")

export CGO_CFLAGS="-O3"

for platform in "${platforms[@]}"; do
  GOOS=${platform%/*}
  GOARCH=${platform#*/}
  output_name=$package_name'-'$GOOS'-'$GOARCH
  echo -n "Building $output_name..."
  if [ "$GOOS" = "windows" ]; then
    output_name+='.exe'
  fi

  if ! env GOOS="$GOOS" GOARCH="$GOARCH" go build -o "dist/$output_name" main.go; then
    echo "An error has occurred! Aborting..."
    exit 1
  fi
  echo "done"
done
