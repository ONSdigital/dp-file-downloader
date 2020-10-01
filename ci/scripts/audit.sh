#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-file-downloader
  make audit
popd