#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright 2026 Hygon Information Technology Co., Ltd.

set -euo pipefail

IMAGE=harbor.sourcefind.cn:5443/hcu/admin/base/hcu-device-plugin

# Derive the version from the git tag so it never drifts from a hand-maintained constant.
if ! VERSION=$(git describe --tags --dirty 2>/dev/null); then
    echo "ERROR: no git tag found, cannot determine the version." >&2
    echo "       Create a tag first, e.g.: git tag v2.4.4" >&2
    exit 1
fi
echo "Building version: ${VERSION}"

# Build the binary
export GOPROXY=https://goproxy.cn
export CGO_ENABLED=1
go mod tidy
go build -ldflags "-X 'main.version=${VERSION}'" -o k8s-device-plugin cmd/main.go

# Build and export the docker image
docker build --target dp -t "${IMAGE}:${VERSION}" .
docker save -o "hcu-device-plugin-${VERSION}.tar" "${IMAGE}:${VERSION}"
