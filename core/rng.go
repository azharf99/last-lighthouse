package core

// RNG adalah generator acak deterministik milik core.
//
// Sengaja TIDAK memakai math/rand atau math/rand/v2: algoritma di stdlib boleh
// berubah antar versi Go, dan kalau itu terjadi setiap replay dan setiap
// baseline simulator yang tersimpan akan rusak diam-diam. splitmix64 di bawah
// ini terkunci pada implementasi kita sendiri, jadi seed yang sama menghasilkan
// urutan yang sama selamanya, lintas versi Go dan lintas target build
// (server dan WASM).
type RNG struct {
	state uint64
}

func NewRNG(seed int64) *RNG { return &RNG{state: uint64(seed)} }

// Seed mengembalikan state saat ini, sehingga bisa di-snapshot bersama State
// dan dilanjutkan persis dari titik yang sama setelah restart server.
func (r *RNG) Seed() int64 { return int64(r.state) }

func (r *RNG) next() uint64 {
	r.state += 0x9E3779B97F4A7C15
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Intn mengembalikan bilangan acak seragam di [0, n). Panic kalau n <= 0.
//
// Memakai penolakan (rejection) alih-alih modulo supaya distribusinya benar
// seragam; bias modulo akan merusak validitas statistik simulator balance.
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		panic("core: RNG.Intn butuh n > 0")
	}
	un := uint64(n)
	limit := ^uint64(0) - (^uint64(0) % un) - 1
	for {
		v := r.next()
		if v <= limit {
			return int(v % un)
		}
	}
}

// D6 melempar satu dadu enam sisi, mengembalikan 1..6 (GDD 16).
func (r *RNG) D6() int { return r.Intn(6) + 1 }

// Shuffle mengocok slice dengan Fisher-Yates.
func (r *RNG) Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		swap(i, j)
	}
}
