#!/usr/bin/env bash
# Build core.wasm dan salin runtime shim Go ke client.
#
# Jalankan dari root repo:  bash scripts/build-wasm.sh
#
# Ukuran gzip dilaporkan tiap build karena payload WASM adalah risiko yang
# ditandai di ADR-002: ia harus dipantau tiap rilis, bukan diperiksa sekali.

set -euo pipefail

OUT_DIR="client/public/wasm"
mkdir -p "$OUT_DIR"

echo "Membangun core.wasm..."
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o "$OUT_DIR/core.wasm" ./cmd/wasm

GOROOT="$(go env GOROOT)"
if [ -f "$GOROOT/lib/wasm/wasm_exec.js" ]; then
  cp "$GOROOT/lib/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
else
  cp "$GOROOT/misc/wasm/wasm_exec.js" "$OUT_DIR/wasm_exec.js"
fi

python - "$OUT_DIR/core.wasm" <<'PY'
import gzip, sys, os
path = sys.argv[1]
raw = open(path, 'rb').read()
gz = gzip.compress(raw, 9)
raw_mb, gz_mb = len(raw)/1048576, len(gz)/1048576
print(f"core.wasm: {raw_mb:.2f} MB mentah, {gz_mb:.2f} MB gzip")
# Anggaran dari ADR-002. Kalau terlampaui: TinyGo, atau jangan muat bot online.
if gz_mb > 3.0:
    print(f"PERINGATAN: payload {gz_mb:.2f} MB gzip melampaui anggaran 3 MB di ADR-002.")
    sys.exit(1)
PY

echo "Selesai."
