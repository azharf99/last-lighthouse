# Build core.wasm dan salin runtime shim Go ke client.
#
# Jalankan dari root repo:
#   powershell -ExecutionPolicy Bypass -File scripts/build-wasm.ps1
#
# Ukuran gzip dilaporkan tiap build karena payload WASM adalah risiko yang
# ditandai di ADR-002: ia harus dipantau tiap rilis, bukan diperiksa sekali.

$ErrorActionPreference = 'Stop'

$outDir = 'client/public/wasm'
if (-not (Test-Path $outDir)) { New-Item -ItemType Directory -Force $outDir | Out-Null }

Write-Host 'Membangun core.wasm...'
$env:GOOS = 'js'
$env:GOARCH = 'wasm'
go build -ldflags='-s -w' -o "$outDir/core.wasm" ./cmd/wasm
Remove-Item Env:GOOS, Env:GOARCH

$goroot = (go env GOROOT)
$shim = Join-Path $goroot 'lib/wasm/wasm_exec.js'
if (-not (Test-Path $shim)) { $shim = Join-Path $goroot 'misc/wasm/wasm_exec.js' }
Copy-Item $shim "$outDir/wasm_exec.js" -Force

# Ukur raw dan gzip.
$raw = (Get-Item "$outDir/core.wasm").Length
$ms = New-Object System.IO.MemoryStream
$gz = New-Object System.IO.Compression.GZipStream($ms, [System.IO.Compression.CompressionLevel]::Optimal)
$bytes = [System.IO.File]::ReadAllBytes("$outDir/core.wasm")
$gz.Write($bytes, 0, $bytes.Length)
$gz.Close()
$gzipped = $ms.ToArray().Length
$ms.Close()

$rawMB = [math]::Round($raw / 1MB, 2)
$gzMB = [math]::Round($gzipped / 1MB, 2)
Write-Host "core.wasm: $rawMB MB mentah, $gzMB MB gzip"

# Anggaran dari ADR-002. Kalau terlampaui, opsinya: TinyGo, atau tidak memuat
# bot di mode online.
if ($gzMB -gt 3.0) {
    Write-Warning "Payload WASM $gzMB MB gzip melampaui anggaran 3 MB di ADR-002."
    exit 1
}
Write-Host 'Selesai.'
