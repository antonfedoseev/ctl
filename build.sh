#!/bin/bash

set -e

DIR=$(cd "$(dirname "$0")"; pwd -P)

#check if MacOS
if [[ $OSTYPE == 'darwin'* ]]; then
  go build -o "$DIR/build/ctl" "$DIR/ctl.go" "$@"
  mkdir -p "$DIR/../producer/ctl"
  cp "$DIR/build/ctl" "$DIR/../producer/ctl/ctl"
#check if WSL
elif [[ ! -z "${PROGRAMFILES}" ]]; then
  go build -o "$DIR/build/ctl.exe" "$DIR/ctl.go" "$@"
  mkdir -p "$DIR/../producer/ctl"
  cp "$DIR/build/ctl.exe" "$DIR/../producer/ctl/ctl.exe"
#graceful fallback
else
  go build -o "$DIR/build/ctl.exe" "$DIR/ctl.go" "$@"
  mkdir -p "$DIR/../producer/ctl"
  cp "$DIR/build/ctl.exe" "$DIR/../producer/ctl/ctl.exe"
fi