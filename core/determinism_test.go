package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// hashState menghasilkan sidik jari stabil dari sebuah State.
//
// Ini bergantung pada JSON marshalling yang deterministik, yang berlaku di Go
// karena field struct diserialisasi dalam urutan deklarasi dan seluruh koleksi
// di State adalah slice, bukan map (lihat kontrak kemurnian di doc.go). Kalau
// suatu saat ada map masuk ke State, test ini akan mulai gagal secara acak --
// dan itu justru sinyal yang kita inginkan.
func hashState(t *testing.T, s *State) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func testSetups() []PlayerSetup {
	return []PlayerSetup{
		{ID: "p1", Name: "Ana", Character: "navigator"},
		{ID: "p2", Name: "Budi", Character: "engineer"},
		{ID: "p3", Name: "Citra", Character: "hunter"},
	}
}

// playScripted menjalankan satu partai memakai bot acak sederhana. Ia
// mengembalikan state akhir plus seluruh event yang dihasilkan.
//
// Bot memilih dari LegalCommands memakai RNG-nya sendiri, terpisah dari RNG
// game, supaya urutan keputusan bot tidak menggeser keacakan game.
func playScripted(t *testing.T, seed int64, maxSteps int) (*State, []Event) {
	t.Helper()
	c := DefaultContent()

	s, events, err := NewGame("m_test", seed, testSetups(), c)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}

	rng := NewRNG(s.RNGState)
	pick := NewRNG(seed ^ 0x5DEECE66D)

	for step := 0; step < maxSteps && !s.Over(); step++ {
		active := s.ActivePlayer()
		if active == nil {
			t.Fatalf("langkah %d: tidak ada pemain aktif tapi match belum selesai", step)
		}

		view := Project(s, active.ID)
		legal := LegalCommands(view, c)
		if len(legal) == 0 {
			t.Fatalf("langkah %d: tidak ada aksi legal untuk %s (fase %s, AP %d)",
				step, active.ID, s.Phase, active.AP)
		}

		cmd := legal[pick.Intn(len(legal))]
		evs, err := Decide(s, cmd, c, rng)
		if err != nil {
			// Prediksi legalitas di client memang konservatif (lihat legal.go),
			// jadi penolakan itu sah. Yang tidak sah adalah command yang
			// dihasilkan LegalCommands lalu ditolak -- itu berarti keduanya
			// tidak sinkron.
			t.Fatalf("langkah %d: LegalCommands mengusulkan %+v tapi Decide menolak: %v",
				step, cmd, err)
		}
		ApplyAll(s, evs)
		s.RNGState = rng.Seed()
		events = append(events, evs...)
	}
	return s, events
}

// TestDeterminism adalah test yang menegakkan taruhan utama ADR-002: seed yang
// sama dan urutan command yang sama harus menghasilkan state akhir yang sama.
//
// Kalau test ini gagal, simulator balance jadi tidak bermakna, replay rusak, dan
// crash recovery server tidak bisa dipercaya. Ia dijalankan di CI.
func TestDeterminism(t *testing.T) {
	for _, seed := range []int64{1, 42, 1337, -99, 2026} {
		a, eventsA := playScripted(t, seed, 400)
		b, eventsB := playScripted(t, seed, 400)

		if got, want := hashState(t, b), hashState(t, a); got != want {
			t.Errorf("seed %d: state akhir berbeda antar dua run\n  run1=%s\n  run2=%s",
				seed, want, got)
		}
		if len(eventsA) != len(eventsB) {
			t.Errorf("seed %d: jumlah event berbeda: %d vs %d",
				seed, len(eventsA), len(eventsB))
		}
	}
}

// TestReplayReconstructsState memverifikasi klaim inti ADR-004: state bisa
// dibangun ulang hanya dari event log. Ini yang membuat crash recovery,
// hibernasi match async, dan resync client bisa dipercaya.
func TestReplayReconstructsState(t *testing.T) {
	final, events := playScripted(t, 7, 400)

	// Rekonstruksi dari nol memakai HANYA event log.
	replayed := &State{}
	ApplyAll(replayed, events)

	// RNGState adalah metadata snapshot, bukan sesuatu yang direkam event
	// (hasil acak dibakukan ke dalam event, lihat ADR-002), jadi ia disamakan
	// sebelum perbandingan.
	replayed.RNGState = final.RNGState
	replayed.MatchID = final.MatchID
	replayed.ContentHash = final.ContentHash

	if got, want := hashState(t, replayed), hashState(t, final); got != want {
		t.Errorf("replay tidak menghasilkan state yang sama\n  live  =%s\n  replay=%s", want, got)
	}
}

// TestRNGStable mengunci algoritma RNG. Kalau seseorang mengganti splitmix64
// dengan implementasi lain, setiap replay tersimpan dan setiap baseline
// simulator akan rusak diam-diam -- test ini membuat kerusakan itu berisik.
func TestRNGStable(t *testing.T) {
	r := NewRNG(12345)
	got := []int{}
	for range 10 {
		got = append(got, r.D6())
	}
	want := []int{3, 4, 4, 1, 4, 5, 3, 3, 2, 2}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("urutan RNG berubah: got %v, want %v\n"+
				"Kalau ini disengaja, perbarui nilai want DAN naikkan versi konten,\n"+
				"karena semua replay lama jadi tidak valid.", got, want)
		}
	}
}

// TestRNGUniform memeriksa bahwa Intn tidak bias modulo. Simulator balance
// menyandarkan validitas statistiknya pada ini.
func TestRNGUniform(t *testing.T) {
	r := NewRNG(99)
	var counts [6]int
	const n = 60000
	for range n {
		counts[r.D6()-1]++
	}
	expected := n / 6
	for face, c := range counts {
		if delta := c - expected; delta > expected/10 || delta < -expected/10 {
			t.Errorf("sisi %d muncul %d kali, diharapkan sekitar %d", face+1, c, expected)
		}
	}
}
