#!/bin/bash
# ==============================================================================
# The Last Lighthouse — VPS Deployment Automation Script
# Usage: ./scripts/deploy-vps.sh
# ==============================================================================

set -e

echo "🚀 Memulai proses deployment The Last Lighthouse ke VPS..."

# 1. Check if .env exists
if [ ! -f .env ]; then
  echo "⚠️ File .env tidak ditemukan. Menyalin dari .env.example..."
  cp .env.example .env
  echo "🔑 Harap tinjau dan sesuaikan nilai rahasia di .env sebelum production!"
fi

# 2. Check docker and docker compose
if ! command -v docker &> /dev/null; then
  echo "❌ Docker tidak terdeteksi. Silakan pasang Docker terlebih dahulu."
  exit 1
fi

# 3. Build and launch containers
echo "📦 Membangun Docker images (Alpine Multi-Stage)..."
docker compose build --pull

echo "▶️ Menjalankan container di latar belakang..."
docker compose up -d

echo ""
echo "✅ Deployment Berhasil!"
echo "--------------------------------------------------------"
echo "🌐 Frontend & Nginx Proxy : http://localhost"
echo "⚙️ Go Backend Server      : http://localhost/api"
echo "📡 WebSocket Endpoint     : ws://localhost/ws"
echo "🐘 PostgreSQL Database    : postgres:5432 (Internal)"
echo "--------------------------------------------------------"
echo "Lihat log container: docker compose logs -f"
