#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLES_DIR="$ROOT_DIR/examples"

examples=(
  "counter"
  "form-remote"
  "prefork"
  "counter-test-prefork"
)

for example in "${examples[@]}"; do
  echo "==> validating example: ${example}"
  (
    cd "$EXAMPLES_DIR/$example"
    go build ./...
  )
done

echo "All examples built successfully."
