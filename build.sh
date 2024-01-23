#!/bin/bash

# install gofumpt if not installed
if ! [ -x "$(command -v gofumpt)" ]; then
  go install mvdan.cc/gofumpt@latest
fi

export PATH=$PATH:$(go env GOPATH)/bin

make
