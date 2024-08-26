#!/bin/bash

set -e

DIR=$(cd "$(dirname "$0")"; pwd -P)

#check if MacOS
if [[ $OSTYPE == 'darwin'* ]]; then
  go build -o "$DIR/build/ctl" "$DIR/ctl.go" "$@"
#check if WSL
elif [[ ! -z "${PROGRAMFILES}" ]]; then
  go build -o "$DIR/build/ctl.exe" "$DIR/ctl.go" "$@"
#graceful fallback
else
  go build -o "$DIR/build/ctl.exe" "$DIR/ctl.go" "$@"
fi

