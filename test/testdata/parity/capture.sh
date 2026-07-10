#!/usr/bin/env bash
# Regenerate golden files. Run from repo root:
#   go build -o /tmp/structlint-golden ./cmd/structlint
#   bash test/testdata/parity/capture.sh /tmp/structlint-golden
set -euo pipefail
BIN="${1:-./bin/structlint}"
ROOT="$(cd "$(dirname "$0")" && pwd)"
GOLDENS="$ROOT/goldens"
mkdir -p "$GOLDENS"
for fixture in "$ROOT"/*/; do
  fixture="${fixture%/}"
  name="$(basename "$fixture")"
  [[ "$name" == "goldens" || "$name" == "capture.sh" || "$name" == "README.md" ]] && continue
  [[ ! -f "$fixture/.structlint.yaml" ]] && continue

  set +e
  "$BIN" validate --path "$fixture" --config "$fixture/.structlint.yaml" > "$GOLDENS/${name}.text" 2>&1
  code=$?
  set -e
  echo "$code" > "$GOLDENS/${name}.exit"

  set +e
  "$BIN" validate --path "$fixture" --config "$fixture/.structlint.yaml" --format json > "$GOLDENS/${name}.json" 2>&1
  set -e

  echo "captured $name (exit=$code)"
done
