#!/bin/sh
# Run the server test suite. The Parakeet worker is an HTTP service, so the Go
# tests need no local model weights or CGO link paths to run.
set -eu
if [ "$#" -eq 0 ]; then
  set -- ./...
fi
exec go test "$@"
