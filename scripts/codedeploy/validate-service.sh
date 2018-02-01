#!/bin/bash

if [[ $(docker inspect --format="{{ .State.Running }}" dp-file-downloader) == "false" ]]; then
  exit 1;
fi
