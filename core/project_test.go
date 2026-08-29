package core

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestNoSecretLeak adalah penegak ADR-006.
//
// Projection adalah jenis kode yang gagal secara DIAM-DIAM: kalau seseorang
// menambah field rahasia ke State dan lupa mengosongkannya di Project, tidak ada
// yang error -- data itu hanya diam-diam terkirim ke device pemain lain. Test ini
// membuat kegagalan itu berisik dengan cara memeriksa byte yang benar-benar akan
// dikirim ke jaringan.
//
// Pemeriksaannya di level serialisasi, bukan level field, supaya field baru
// otomatis ikut tercakup tanpa harus ingat memperbarui test ini.
func TestNoSecretLeak(t *testing.T) {
	c := DefaultContent()

	for _, seed := range []int64{1, 2, 3, 42, 999} {
		s, _, err := NewGame("m_leak", seed, testSetups(), c)
		if err != nil {
			t.Fatalf("NewGame: %v", err)
		}

		for _, viewer := range s.Players {
			// Kumpulkan objective yang TIDAK boleh dilihat viewer ini.
			var forbidden []ObjectiveID
			for _, other := range s.Players {
				if other.ID != viewer.ID && other.Objective != "" {
					forbidden = append(forbidden, other.Objective)
				}
			}

			view := Project(s, viewer.ID)
			blob, err := json.Marshal(view)
			if err != nil {
				t.Fatalf("marshal view: %v", err)
			}
			payload := string(blob)

			for _, secretID := range forbidden {
				if strings.Contains(payload, string(secretID)) {
					t.Errorf("seed %d: objective %q milik pemain lain bocor ke view %s",
						seed, secretID, viewer.ID)
				}
			}

			// Viewer harus tetap bisa melihat objective-nya sendiri, kalau tidak
			// projection-nya terlalu agresif dan game jadi tidak bisa dimainkan.
			if view.MyObjective != viewer.Objective {
				t.Errorf("seed %d: view %s kehilangan objective miliknya sendiri: got %q want %q",
					seed, viewer.ID, view.MyObjective, viewer.Objective)
			}

			// RNG state bocor sama berbahayanya: ia memungkinkan pemain
			// memprediksi lemparan dadu dan kocokan deck berikutnya.
			if view.State.RNGState != 0 {
				t.Errorf("seed %d: RNG state bocor ke view %s", seed, viewer.ID)
			}
		}
	}
}

// TestNoSecretLeakDuringPlay menjalankan partai penuh dan memeriksa setiap view
// di setiap langkah. Kebocoran bisa muncul di tengah permainan, bukan hanya saat
// setup.
func TestNoSecretLeakDuringPlay(t *testing.T) {
	c := DefaultContent()
	s, _, err := NewGame("m_leak2", 4242, testSetups(), c)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	rng := NewRNG(s.RNGState)
	pick := NewRNG(1)

	secrets := map[PlayerID]ObjectiveID{}
	for _, p := range s.Players {
		secrets[p.ID] = p.Objective
	}

	for step := 0; step < 200 && !s.Over(); step++ {
		for _, viewer := range s.Players {
			view := Project(s, viewer.ID)
			blob, _ := json.Marshal(view)
			for pid, obj := range secrets {
				if pid == viewer.ID || obj == "" {
					continue
				}
				if strings.Contains(string(blob), string(obj)) {
					t.Fatalf("langkah %d: objective %q milik %s bocor ke view %s",
						step, obj, pid, viewer.ID)
				}
			}
		}

		active := s.ActivePlayer()
		if active == nil {
			break
		}
		legal := LegalCommands(Project(s, active.ID), c)
		if len(legal) == 0 {
			break
		}
		evs, err := Decide(s, legal[pick.Intn(len(legal))], c, rng)
		if err != nil {
			t.Fatalf("langkah %d: %v", step, err)
		}
		ApplyAll(s, evs)
		s.RNGState = rng.Seed()
	}
}

// TestProjectEventRedaction memeriksa penyaringan event, bukan hanya state.
// Pembagian objective adalah kasus ujinya: pemilik melihat kartunya, yang lain
// hanya melihat bahwa sebuah kartu dibagikan.
func TestProjectEventRedaction(t *testing.T) {
	_, events, err := NewGame("m_ev", 5, testSetups(), DefaultContent())
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}

	var dealt *Event
	for i := range events {
		if events[i].Kind == EvObjectiveDealt {
			dealt = &events[i]
			break
		}
	}
	if dealt == nil {
		t.Fatal("tidak ada event objective_dealt di setup")
	}
	owner := dealt.Player

	mine := ProjectEvent(*dealt, owner)
	if mine == nil || mine.Objective == "" {
		t.Fatalf("pemilik harus melihat objective-nya, got %+v", mine)
	}
	if mine.Private != "" || mine.PublicVariant != nil {
		t.Error("metadata visibilitas harus dibersihkan sebelum dikirim ke client")
	}

	other := PlayerID("p_other")
	theirs := ProjectEvent(*dealt, other)
	if theirs == nil {
		t.Fatal("pemain lain harus tetap melihat bahwa objective dibagikan")
	}
	if theirs.Objective != "" {
		t.Errorf("objective bocor ke pemain lain lewat event: %q", theirs.Objective)
	}
	if theirs.Player != owner {
		t.Errorf("varian publik harus tetap menyebut siapa yang menerima, got %q", theirs.Player)
	}
}

// TestApplyOnProjectedViewMatchesServer memverifikasi klaim ADR-002 bahwa client
// dan server memakai Apply yang sama.
//
// Client memulai dari view awal, menerapkan event yang sudah diproyeksikan, dan
// bagian publiknya harus cocok dengan yang dilihat server. Kalau ini gagal,
// client akan menampilkan papan yang berbeda dari kebenaran server -- desync
// yang persis ingin dihindari arsitektur ini.
func TestApplyOnProjectedViewMatchesServer(t *testing.T) {
	c := DefaultContent()
	s, setupEvents, err := NewGame("m_sync", 77, testSetups(), c)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	viewer := s.TurnOrder[0]

	// Sisi client: bangun dari nol memakai HANYA event yang diproyeksikan.
	clientState := &State{}
	ApplyAll(clientState, ProjectEvents(setupEvents, viewer))

	rng := NewRNG(s.RNGState)
	pick := NewRNG(3)

	for step := 0; step < 120 && !s.Over(); step++ {
		active := s.ActivePlayer()
		if active == nil {
			break
		}
		legal := LegalCommands(Project(s, active.ID), c)
		if len(legal) == 0 {
			break
		}
		evs, err := Decide(s, legal[pick.Intn(len(legal))], c, rng)
		if err != nil {
			t.Fatalf("langkah %d: %v", step, err)
		}
		ApplyAll(s, evs)
		s.RNGState = rng.Seed()
		ApplyAll(clientState, ProjectEvents(evs, viewer))
	}

	// Bandingkan terhadap projection server pada state akhir.
	want := Project(s, viewer)
	clientState.RNGState = 0

	gotJSON, _ := json.Marshal(clientState)
	wantJSON, _ := json.Marshal(&want.State)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("state client menyimpang dari projection server\nclient=%s\nserver=%s",
			gotJSON, wantJSON)
	}
}

// TestNoDeckOrderLeak menegakkan kerahasiaan urutan deck dan tumpukan tile.
//
// Ini lebih ketat daripada kerahasiaan objective: objective hanya disembunyikan
// dari lawan, sedangkan urutan deck harus tidak diketahui SIAPA PUN. Kalau
// bocor, pertanyaan "apakah imbalannya sepadan dengan risikonya" (GDD 20)
// berubah jadi perhitungan tanpa risiko, dan seluruh mekanik Mystery kehilangan
// maknanya.
func TestNoDeckOrderLeak(t *testing.T) {
	c := DefaultContent()
	s, setupEvents, err := NewGame("m_deck", 31, testSetups(), c)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}

	// Kartu yang belum ditarik: satu pun tidak boleh muncul di view atau di
	// event yang dikirim ke pemain.
	var hidden []string
	for _, id := range s.EventDeck.Draw {
		hidden = append(hidden, string(id))
	}
	for _, id := range s.MysteryDeck.Draw {
		hidden = append(hidden, string(id))
	}
	for _, id := range s.ArtifactDeck.Draw {
		hidden = append(hidden, string(id))
	}
	if len(hidden) < 10 {
		t.Fatalf("deck terlalu kecil untuk menguji apa pun: %d kartu", len(hidden))
	}

	for _, viewer := range s.Players {
		blob, err := json.Marshal(Project(s, viewer.ID))
		if err != nil {
			t.Fatalf("marshal view: %v", err)
		}
		for _, id := range hidden {
			if strings.Contains(string(blob), id) {
				t.Errorf("kartu %q yang belum ditarik bocor ke view %s", id, viewer.ID)
			}
		}

		// Event setup memuat hasil kocokan; yang dikirim harus versi tertutup.
		for _, e := range ProjectEvents(setupEvents, viewer.ID) {
			eb, _ := json.Marshal(e)
			for _, id := range hidden {
				if strings.Contains(string(eb), id) {
					t.Errorf("kartu %q bocor lewat event %s ke %s", id, e.Kind, viewer.ID)
				}
			}
		}
	}

	// Jumlah kartu tetap terlihat: itu informasi publik di board game fisik,
	// dan client memakainya untuk menampilkan tebal-tipisnya tumpukan.
	view := Project(s, s.Players[0].ID)
	if view.State.MysteryDeck.Len() != s.MysteryDeck.Len() {
		t.Errorf("jumlah kartu mystery berubah setelah projection: %d vs %d",
			view.State.MysteryDeck.Len(), s.MysteryDeck.Len())
	}
	if len(view.State.TileStack) != len(s.TileStack) {
		t.Errorf("jumlah tile berubah setelah projection: %d vs %d",
			len(view.State.TileStack), len(s.TileStack))
	}
}

// TestScholarDrawStaysPrivate memeriksa bahwa dua kartu yang dilihat Scholar
// tidak bocor ke pemain lain (GDD 10.4). Nilai kemampuan itu justru terletak
// pada informasi yang hanya ia miliki.
func TestScholarDrawStaysPrivate(t *testing.T) {
	c := DefaultContent()
	setups := []PlayerSetup{
		{ID: "s1", Name: "Scholar", Character: "scholar"},
		{ID: "s2", Name: "Lain", Character: "hunter"},
	}
	s, _, err := NewGame("m_scholar", 77, setups, c)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	rng := NewRNG(s.RNGState)

	// Bawa Scholar ke Reruntuhan lalu selidiki.
	for s.ActivePlayer().ID != "s1" && !s.Over() {
		evs, _ := Decide(s, Command{Kind: CmdEndTurn, Player: s.ActivePlayer().ID}, c, rng)
		ApplyAll(s, evs)
	}
	Apply(s, Event{Kind: EvMoved, Player: "s1", To: "ruins"})

	evs, err := Decide(s, Command{Kind: CmdInvestigate, Player: "s1"}, c, rng)
	if err != nil {
		t.Fatalf("investigate: %v", err)
	}
	ApplyAll(s, evs)

	if s.Pending == nil {
		t.Fatal("investigate seharusnya menghasilkan pilihan tertunda")
	}
	if s.Pending.Kind != "mystery_card" {
		t.Skipf("scholar tidak menarik 2 kartu di seed ini (kind=%s)", s.Pending.Kind)
	}

	secretCards := append([]EventCardID(nil), s.Pending.Cards...)
	if len(secretCards) < 2 {
		t.Fatalf("scholar seharusnya melihat 2 kartu, dapat %d", len(secretCards))
	}

	blob, _ := json.Marshal(Project(s, "s2"))
	for _, id := range secretCards {
		if strings.Contains(string(blob), string(id)) {
			t.Errorf("kartu %q yang dilihat Scholar bocor ke pemain lain", id)
		}
	}
	for _, e := range ProjectEvents(evs, "s2") {
		eb, _ := json.Marshal(e)
		for _, id := range secretCards {
			if strings.Contains(string(eb), string(id)) {
				t.Errorf("kartu %q bocor lewat event %s", id, e.Kind)
			}
		}
	}
}
