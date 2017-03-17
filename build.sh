#!/usr/bin/env bash
set -e

VERSION="$(git describe)"

echo "Building version $VERSION"

docker build -t gowncloudbuilder .
docker run --rm -v "$PWD":/go/src/github.com/gowncloud/gowncloud --entrypoint sh gowncloudbuilder -c "go generate && go build -ldflags '-s -X main.version=$VERSION' -v -o dist/gowncloud"
docker build -t gowncloud/gowncloud:"$VERSION" -f DockerfileMinimal .
