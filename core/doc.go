// Package core adalah rules engine The Last Lighthouse.
//
// Package ini adalah SATU-SATUNYA sumber kebenaran aturan game. Ia ditautkan
// langsung ke binary server dan juga dikompilasi ke WebAssembly untuk
// dijalankan di client (mode hotseat offline & solo vs bot). Lihat
// docs/adr/ADR-002-shared-go-core-wasm.md.
//
// # Kontrak kemurnian
//
// Package ini WAJIB tetap murni dan deterministik. Yang dilarang:
//
//   - import net/*, os, database/*, syscall, atau apa pun dari internal/
//   - time.Now() atau sumber waktu apa pun; waktu diinjeksikan lewat parameter
//   - math/rand global; seluruh keacakan lewat *RNG ber-seed yang dioper eksplisit
//   - goroutine dan channel
//   - iterasi map yang memengaruhi hasil; gunakan slice atau key terurut
//
// Alasannya: replay, simulator balance, dan crash recovery server semuanya
// bergantung pada properti "seed sama + urutan command sama => state akhir sama".
// Lihat determinism_test.go yang menegakkan ini di CI.
//
// # Model
//
// Alur mengikuti pola decide/apply (event sourcing):
//
//	Decide(state, cmd, rng) -> []Event   // validasi + RNG terjadi DI SINI
//	Apply(state, event)                  // murni, tanpa RNG, tanpa error, bisa direplay
//
// Hasil acak dibakukan ke dalam Event, sehingga Apply tidak pernah butuh RNG dan
// replay tidak perlu mereproduksi urutan pemanggilan RNG.
//
// # Status implementasi (M0)
//
// Irisan vertikal M0: Move, Gather, Repair, Rest, EndTurn, fase Darkness, dan
// kondisi menang/kalah. Aksi Explore, Fight, Investigate, dan Trade sudah punya
// tempat di Command tetapi mengembalikan ErrNotImplemented sampai M1.
package core
