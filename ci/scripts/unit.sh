#!/bin/bash -eux

export GOPATH=$(pwd)/go

pushd $GOPATH/dp-file-downloader
  make test
popd