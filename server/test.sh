#!/bin/sh
# Run the server test suite with the whisper CGO link paths set.
#
# The whisper binding links libwhisper/libggml from
# third_party/whisper.cpp/build_go (built once via `make whisper` in
# third_party/whisper.cpp/bindings/go). Plain `go test` fails at link
# time with "cannot find -lwhisper" because the linker is never told
# where those archives live — the Dockerfile sets these same variables,
# the local shell does not. `go vet` passes without them (no linking),
# which is why the failure only shows up at test time.
set -eu
ROOT="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
export CGO_ENABLED=1
export C_INCLUDE_PATH="$ROOT/third_party/whisper.cpp/include:$ROOT/third_party/whisper.cpp/ggml/include"
export LIBRARY_PATH="$ROOT/third_party/whisper.cpp/build_go/src:$ROOT/third_party/whisper.cpp/build_go/ggml/src"
if [ "$#" -eq 0 ]; then
  set -- ./...
fi
exec go test "$@"
