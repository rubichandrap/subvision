#!/bin/bash

set -e

cd "$(dirname "$0")/.."

echo "========== Running Subvision Server =========="
echo "  PARAKEET_URL=${PARAKEET_URL:-<unset>}"
echo "=============================================="
echo ""

go run ./cmd/subvision
