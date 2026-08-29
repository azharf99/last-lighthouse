# The Last Lighthouse

Implementasi digital dari board game semi-koperatif *The Last Lighthouse*
(2–4 pemain, eksplorasi & strategi). Target: Web, PC, dan Mobile dengan backend Go.

- Desain game: `The_Last_Lighthouse_GDD_v0.2.docx`
- Arsitektur teknis: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Keputusan arsitektur: [docs/adr/](docs/adr/)

## Status: M2 — Server Online, Match Actor, WebSocket Hub, Postgres Event Log

Server online multi-pemain dengan model Match Actor (1 goroutine/match), WebSocket hub dengan resync & heartbeat, persistensi event log append-only di Postgres, guest auth JWT, turn timer 90 detik dengan takeover bot otomatis, serta client UI yang mendukung mode Hotseat lokal dan Online multiplayer.

| Sudah jalan | Belum |
|-------------|-------|
| Move, Gather, Repair, Rest, End Turn | UI Layout Penuh & Animasi PixiJS (M3) |
| Explore + tile modular (§18) | Tauri Desktop & Capacitor Mobile (M4) |
| Fight + combat 1D6 (§16) | Async Play Lintas Hari & Push (M5) |
| Trade antar pemain (§11, §28) | Telemetri & Replay Spectator (M6) |
| Investigate + Mystery dua tahap (§20) | |
| Fase Event & Monster (§13, §15) | |
| Deck Event / Mystery / Artifact (§8.3) | |
| Keempat kemampuan karakter (§10) | |
| Penskalaan biaya per jumlah pemain | |
| Penghitung & scoring objective (§24, §25) | |
| Bot heuristik + `cmd/simulator` | |
| VP eksplorasi (Arah 1) + Investigate di semua lokasi (Arah 2) | |
| **Server Online WebSocket + REST Lobby (`cmd/server`)** | |
| **Match Actor + Event Sourcing (`internal/match`)** | |
| **PostgreSQL & In-Memory Store (`internal/store`)** | |
| **Guest Auth JWT (`internal/auth`)** | |
| **Reconnect, Delta Resync & Snapshot Projection (ADR-003/006)** | |
| **Turn Timer 90s + Auto Takeover Bot AFK (ADR-007)** | |
| **Client Online Lobby & Realtime Session (`client/src`)** | |

## Struktur

```
core/            rules engine murni & deterministik (server + WASM)
bot/             AI heuristik (online, solo, simulator, AFK takeover)
internal/
  store/         abstraksi penyimpanan + Postgres event log & Memory store
  auth/          JWT guest authenticator
  match/         Match Actor engine (1 goroutine/match) & Registry
  transport/
    ws/          WebSocket Hub, Connection wrapper, Heartbeat, Envelopes
    http/        REST HTTP server (auth, lobby, match status)
cmd/server/      binary server online HTTP & WebSocket
cmd/wasm/        binding JS untuk client
cmd/simulator/   harness balance headless
client/          React + TypeScript (Hotseat lokal + Online multiplayer)
scripts/         build WASM + smoke test
docs/            arsitektur, ADR, laporan balance
```

## Menjalankan

Butuh Go 1.24+ dan Node 20+.

Test lengkap, termasuk determinisme, kebocoran rahasia, kemurnian, dan server online:

```bash
go test ./...
```

Jalankan server online (secara default memakai In-Memory Store):

```bash
go run ./cmd/server
```

Atau dengan PostgreSQL (ADR-004):

```bash
LLH_DB_URL="postgres://user:pass@localhost:5432/lastlighthouse?sslmode=disable" go run ./cmd/server
```

Jalankan client web:

```bash
npm run dev --prefix client
```

Buka `http://localhost:5173/`, klik tombol **Mode Online**, buat atau gabung ke match bersama pemain lain di tab/perangkat berbeda!

## Berikutnya

**M3: UI Sesungguhnya** — Layout §34 penuh, peta canvas PixiJS interaktif, animasi kartu & token, art placeholder, audio, serta dukungan i18n (ID/EN) untuk playtest eksternal.
