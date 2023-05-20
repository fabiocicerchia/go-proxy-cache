#!/bin/bash

mkdir dist > /dev/null 2>&1 || true

package_name=go-proxy-cache
platforms=("freebsd/amd64" "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64")

export CGO_CFLAGS="-march=native -O3"

for platform in "${platforms[@]}"; do
  declare -a "platform_split=(${platform//\// })"
  GOOS=${platform_split[0]}
  GOARCH=${platform_split[1]}
  output_name=$package_name'-'$GOOS'-'$GOARCH
  echo -n "Building $output_name..."
  if [ "$GOOS" = "windows" ]; then
    output_name+='.exe'
  fi

  env GOOS="$GOOS" GOARCH="$GOARCH" go build -o dist/$output_name main.go
  if [ $? -ne 0 ]; then
    echo "An error has occurred! Aborting..."
    exit 1
  fi
  echo "done"
done
