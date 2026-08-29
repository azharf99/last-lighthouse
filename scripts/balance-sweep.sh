#!/usr/bin/env bash
# Jalankan beberapa konfigurasi simulator dan tabelkan hasilnya berdampingan.
#
# Tuning hanya bermakna kalau efek tiap perubahan bisa DIKAITKAN ke perubahan itu.
# Karena itu skrip ini selalu memakai seed awal yang sama untuk semua konfigurasi:
# perbedaan yang muncul berasal dari konfigurasinya, bukan dari partai yang
# kebetulan berbeda.
#
#   bash scripts/balance-sweep.sh                  # sweep bawaan
#   bash scripts/balance-sweep.sh 3000 "" "-ap 4"  # jumlah partai + konfigurasi sendiri

set -euo pipefail

GAMES="${1:-2000}"
shift || true

if [ "$#" -gt 0 ]; then
  CONFIGS=("$@")
else
  CONFIGS=("" "-ap 4" "-darkness-max 10" "-darkness-max 12" "-ap 4 -darkness-max 10" "-careless")
fi

printf "%-26s %8s %8s %8s %8s %9s\n" "KONFIGURASI" "MENANG" "RONDE" "DARK" "FIGHT" "MOVE%"
printf "%-26s %8s %8s %8s %8s %9s\n" "--------------------------" "------" "-----" "----" "-----" "-----"

for cfg in "${CONFIGS[@]}"; do
  out=$(go run ./cmd/simulator -games "$GAMES" -seed 1 $cfg 2>&1)

  win=$(echo "$out"    | grep 'Menang ('        | grep -oE '[0-9.]+%')
  rounds=$(echo "$out" | grep 'Ronde rata-rata' | grep -oE '[0-9.]+$')
  dark=$(echo "$out"   | grep 'Darkness akhir'  | grep -oE '[0-9.]+$')
  fight=$(echo "$out"  | grep '^  fight'        | awk '{print $2}')
  movep=$(echo "$out"  | grep '^  move'         | grep -oE '\( *[0-9.]+%' | grep -oE '[0-9.]+%')

  printf "%-26s %8s %8s %8s %8s %9s\n" \
    "${cfg:-dasar}" "${win:-?}" "${rounds:-?}" "${dark:-?}" "${fight:-?}" "${movep:-?}"
done
