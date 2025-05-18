#!/bin/bash

set -e

cd "$(dirname "$0")/.."

# Resolve base path
WHISPER_PATH="$(realpath ./third_party/whisper.cpp)"
WHISPER_BINDINGS_DIR="$WHISPER_PATH/bindings/go"
C_INCLUDE_PATH="$WHISPER_PATH/include"
GGML_INCLUDE_PATH="$WHISPER_PATH/ggml/include"
LIBRARY_PATH="$WHISPER_PATH/build_go/src"
GGML_LIBRARY_PATH="$WHISPER_PATH/build_go/ggml/src"

# Export paths for CGO
export CGO_ENABLED=1
export C_INCLUDE_PATH="$C_INCLUDE_PATH:$GGML_INCLUDE_PATH"
export CGO_LDFLAGS="-L$LIBRARY_PATH -L$GGML_LIBRARY_PATH -lwhisper"

# Build libwhisper.a if needed
if [ ! -f "$LIBRARY_PATH/libwhisper.a" ]; then
  echo "[INFO] libwhisper.a not found. Building with 'make whisper'..."
  (cd "$WHISPER_BINDINGS_DIR" && make whisper) || {
    echo "[ERROR] Failed to build libwhisper.a"
    exit 1
  }
else
  echo "[INFO] libwhisper.a found at $LIBRARY_PATH"
fi

# Debug info
echo ""
echo "========== Running Subvision Server =========="
echo "  C_INCLUDE_PATH=$C_INCLUDE_PATH"
echo "  CGO_LDFLAGS=$CGO_LDFLAGS"
echo "=============================================="
echo ""

# Run Go app
go run ./cmd/subvision
