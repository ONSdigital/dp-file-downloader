#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-file-downloader
  make build && mv build/$(go env GOOS)-$(go env GOARCH)/* $cwd/build
  cp Dockerfile.concourse $cwd/build
popd
