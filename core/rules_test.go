package core

import "testing"

// drive adalah pembantu test: jalankan satu command dan terapkan hasilnya.
func drive(t *testing.T, s *State, rng *RNG, cmd Command) []Event {
	t.Helper()
	evs, err := Decide(s, cmd, DefaultContent(), rng)
	if err != nil {
		t.Fatalf("Decide(%s oleh %s): %v", cmd.Kind, cmd.Player, err)
	}
	ApplyAll(s, evs)
	s.RNGState = rng.Seed()
	return evs
}

func newTestGame(t *testing.T, seed int64) (*State, *RNG) {
	t.Helper()
	s, _, err := NewGame("m_rules", seed, testSetups(), DefaultContent())
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return s, NewRNG(s.RNGState)
}

// grant memberi resource ke pemain lewat Apply, bukan mutasi langsung, supaya
// test tetap menghormati aturan bahwa Apply satu-satunya jalur mutasi.
func grant(s *State, p PlayerID, rs ResourceSet) {
	Apply(s, Event{Kind: EvResourceGained, Player: p, Resources: rs})
}

func TestContentIsValid(t *testing.T) {
	c := DefaultContent()
	if err := c.Validate(); err != nil {
		t.Fatalf("konten tertanam tidak valid: %v", err)
	}
	if c.Hash == "" {
		t.Error("content hash kosong; match tidak bisa dikunci ke versi konten")
	}
	// Kelima komponen mercusuar harus punya order berurutan tanpa lompatan,
	// karena GDD 7.1 mengandalkan urutan itu.
	for i, comp := range c.Lighthouse {
		if comp.Order != i+1 {
			t.Errorf("komponen %q punya order %d, diharapkan %d", comp.ID, comp.Order, i+1)
		}
	}
}

func TestNewGameRejectsBadSetup(t *testing.T) {
	c := DefaultContent()
	cases := []struct {
		name   string
		setups []PlayerSetup
	}{
		{"terlalu sedikit", []PlayerSetup{{ID: "p1", Character: "navigator"}}},
		{"terlalu banyak", []PlayerSetup{
			{ID: "p1", Character: "navigator"}, {ID: "p2", Character: "engineer"},
			{ID: "p3", Character: "hunter"}, {ID: "p4", Character: "scholar"},
			{ID: "p5", Character: "navigator"},
		}},
		{"id duplikat", []PlayerSetup{
			{ID: "p1", Character: "navigator"}, {ID: "p1", Character: "engineer"},
		}},
		{"karakter tidak dikenal", []PlayerSetup{
			{ID: "p1", Character: "wizard"}, {ID: "p2", Character: "engineer"},
		}},
	}
	for _, tc := range cases {
		if _, _, err := NewGame("m", 1, tc.setups, c); err == nil {
			t.Errorf("%s: diharapkan error, tapi diterima", tc.name)
		}
	}
}

func TestMoveRequiresAdjacency(t *testing.T) {
	s, rng := newTestGame(t, 1)
	p := s.ActivePlayer()

	// Lighthouse bersebelahan dengan harbor dan forest, bukan dengan cave.
	_, err := Decide(s, Command{Kind: CmdMove, Player: p.ID, To: "cave"}, DefaultContent(), rng)
	if err != ErrNotAdjacent {
		t.Errorf("pindah ke lokasi tak bersebelahan: got %v, want %v", err, ErrNotAdjacent)
	}

	_, err = Decide(s, Command{Kind: CmdMove, Player: p.ID, To: "atlantis"}, DefaultContent(), rng)
	if err != ErrUnknownLocation {
		t.Errorf("pindah ke lokasi tak dikenal: got %v, want %v", err, ErrUnknownLocation)
	}

	drive(t, s, rng, Command{Kind: CmdMove, Player: p.ID, To: "forest"})
	if got := s.Player(p.ID).At; got != "forest" {
		t.Errorf("setelah pindah, pemain di %q, want forest", got)
	}
	if got := s.Player(p.ID).AP; got != 2 {
		t.Errorf("AP setelah 1 aksi: got %d, want 2", got)
	}
}

func TestOnlyActivePlayerMayAct(t *testing.T) {
	s, rng := newTestGame(t, 2)
	active := s.ActivePlayer()

	var other PlayerID
	for _, p := range s.Players {
		if p.ID != active.ID {
			other = p.ID
			break
		}
	}

	_, err := Decide(s, Command{Kind: CmdRest, Player: other}, DefaultContent(), rng)
	if err != ErrNotYourTurn {
		t.Errorf("pemain non-aktif beraksi: got %v, want %v", err, ErrNotYourTurn)
	}

	_, err = Decide(s, Command{Kind: CmdRest, Player: "hantu"}, DefaultContent(), rng)
	if err != ErrUnknownPlayer {
		t.Errorf("pemain tak dikenal: got %v, want %v", err, ErrUnknownPlayer)
	}
}

func TestGatherRespectsInventoryCapacity(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 3)
	p := s.ActivePlayer()

	// Isi inventory sampai penuh (GDD 9.1: kapasitas default 6).
	grant(s, p.ID, NewResourceSet(map[Resource]int{Food: c.Rules.InventoryCapacity}))

	drive(t, s, rng, Command{Kind: CmdMove, Player: p.ID, To: "forest"})
	_, err := Decide(s, Command{Kind: CmdGather, Player: p.ID, Resource: Wood}, c, rng)
	if err != ErrInventoryFull {
		t.Errorf("gather dengan inventory penuh: got %v, want %v", err, ErrInventoryFull)
	}
}

func TestGatherClampsToRemainingRoom(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 4)
	p := s.ActivePlayer()

	// Sisakan ruang 1 slot; gather bernilai 2 harus dipotong jadi 1.
	grant(s, p.ID, NewResourceSet(map[Resource]int{Food: c.Rules.InventoryCapacity - 1}))
	drive(t, s, rng, Command{Kind: CmdMove, Player: p.ID, To: "forest"})
	drive(t, s, rng, Command{Kind: CmdGather, Player: p.ID, Resource: Wood})

	if got := s.Player(p.ID).Inventory.Total(); got != c.Rules.InventoryCapacity {
		t.Errorf("total inventory: got %d, want %d", got, c.Rules.InventoryCapacity)
	}
	if got := s.Player(p.ID).Inventory[Wood]; got != 1 {
		t.Errorf("wood terkumpul: got %d, want 1 (dipotong sisa ruang)", got)
	}
}

func TestGatherRejectsResourceNotHere(t *testing.T) {
	s, rng := newTestGame(t, 5)
	p := s.ActivePlayer()

	// Lighthouse tidak menghasilkan resource apa pun (GDD 19).
	_, err := Decide(s, Command{Kind: CmdGather, Player: p.ID, Resource: Wood}, DefaultContent(), rng)
	if err != ErrNothingToGather {
		t.Errorf("gather di lokasi tanpa resource: got %v, want %v", err, ErrNothingToGather)
	}
}

func TestRepairOrderEnforced(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 6)
	p := s.ActivePlayer()

	// Pemain mulai di mercusuar (GDD 32 prototype map).
	grant(s, p.ID, NewResourceSet(map[Resource]int{Crystal: 2}))

	// GDD 7.1: Lens tidak boleh diperbaiki sebelum Foundation dan Power Core.
	_, err := Decide(s, Command{
		Kind: CmdRepair, Player: p.ID, Component: "lens",
		Pay: NewResourceSet(map[Resource]int{Crystal: 2}),
	}, c, rng)
	if err != ErrWrongComponent {
		t.Errorf("repair di luar urutan: got %v, want %v", err, ErrWrongComponent)
	}
}

func TestRepairRejectsUselessPayment(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 7)
	p := s.ActivePlayer()

	// Foundation butuh wood + metal. Menyetor crystal tidak mengurangi apa pun.
	grant(s, p.ID, NewResourceSet(map[Resource]int{Crystal: 3}))

	// Diskon perbaikan sekali per ronde membuat aksi Repair tetap berguna walau
	// setorannya nihil, jadi jatah itu dihabiskan dulu supaya yang diuji benar-
	// benar "setoran yang tidak mengurangi apa pun".
	Apply(s, Event{Kind: EvAbilityUsed, Player: p.ID, Reason: "round"})

	_, err := Decide(s, Command{
		Kind: CmdRepair, Player: p.ID,
		Pay: NewResourceSet(map[Resource]int{Crystal: 3}),
	}, c, rng)
	if err != ErrUselessPayment {
		t.Errorf("setoran tidak berguna: got %v, want %v", err, ErrUselessPayment)
	}
}

// TestRepairDiscountMakesEmptyPaymentUseful mengunci sisi lain dari aturan yang
// sama: bagi Insinyur, datang ke mercusuar tanpa membawa apa pun tetap berguna
// karena kemampuannya sendiri mengurangi kebutuhan (GDD 10.2).
func TestRepairDiscountMakesEmptyPaymentUseful(t *testing.T) {
	s, rng := newTestGame(t, 7)

	// Cari Insinyur dan jadikan dia pemain aktif.
	var eng PlayerID
	for _, pl := range s.Players {
		if pl.Character == "engineer" {
			eng = pl.ID
		}
	}
	if eng == "" {
		t.Skip("tidak ada insinyur di meja ini")
	}
	for s.ActivePlayer().ID != eng && !s.Over() {
		drive(t, s, rng, Command{Kind: CmdEndTurn, Player: s.ActivePlayer().ID})
	}
	if s.Over() {
		t.Skip("match berakhir sebelum giliran insinyur")
	}

	before := s.Component("foundation").Progress.Total()
	drive(t, s, rng, Command{Kind: CmdRepair, Player: eng})
	if after := s.Component("foundation").Progress.Total(); after <= before {
		t.Errorf("diskon insinyur tidak menambah progress: %d -> %d", before, after)
	}
}

func TestRepairTrimsOverpayment(t *testing.T) {
	s, rng := newTestGame(t, 8)
	p := s.ActivePlayer()

	// Setoran berlebih harus dipotong ke yang benar-benar dibutuhkan, bukan
	// ditelan diam-diam.
	//
	// Biaya dibaca dari state, bukan ditulis sebagai angka tetap: biaya komponen
	// berskala dengan jumlah pemain (lihat scaleComponentCosts), jadi ekspektasi
	// hardcode akan usang setiap kali penskalaannya disetel.
	need := s.Component("foundation").Cost
	const carry = 3
	grant(s, p.ID, NewResourceSet(map[Resource]int{Wood: carry, Metal: carry}))

	// Insinyur punya diskon sekali per ronde yang akan mengubah perhitungan;
	// habiskan dulu supaya yang diuji murni pemotongan setoran.
	Apply(s, Event{Kind: EvAbilityUsed, Player: p.ID, Reason: "round"})

	drive(t, s, rng, Command{
		Kind: CmdRepair, Player: p.ID,
		Pay: NewResourceSet(map[Resource]int{Wood: carry, Metal: carry}),
	})

	inv := s.Player(p.ID).Inventory
	wantWood := carry - need[Wood]
	wantMetal := carry - need[Metal]
	if inv[Wood] != wantWood || inv[Metal] != wantMetal {
		t.Errorf("setoran berlebih tidak dipotong: sisa wood=%d metal=%d, want %d dan %d "+
			"(biaya fondasi %v)", inv[Wood], inv[Metal], wantWood, wantMetal, need)
	}
	if !s.Component("foundation").Repaired {
		t.Error("foundation seharusnya selesai")
	}
}

// TestLighthouseCompletionWins menelusuri jalur kemenangan penuh (GDD 6.2, 26.1).
func TestLighthouseCompletionWins(t *testing.T) {
	s, rng := newTestGame(t, 9)

	for range 40 {
		if s.Over() {
			break
		}
		next := s.NextComponent()
		if next == nil {
			break
		}
		p := s.ActivePlayer()
		if p == nil {
			t.Fatal("tidak ada pemain aktif")
		}
		if p.AP < 1 {
			drive(t, s, rng, Command{Kind: CmdEndTurn, Player: p.ID})
			continue
		}
		need := next.Progress.Missing(next.Cost)
		grant(s, p.ID, need)
		drive(t, s, rng, Command{Kind: CmdRepair, Player: p.ID, Component: next.ID, Pay: need})
	}

	if s.Status != StatusWon {
		t.Fatalf("status: got %q, want %q (darkness=%d)", s.Status, StatusWon, s.Darkness)
	}
	for _, comp := range s.Lighthouse {
		if !comp.Repaired {
			t.Errorf("komponen %q belum selesai padahal game menang", comp.ID)
		}
	}
	// VP perbaikan harus terbagi ke pemain, bukan menguap.
	total := 0
	for _, p := range s.Players {
		total += p.VP
	}
	if total == 0 {
		t.Error("tidak ada VP yang diberikan padahal mercusuar selesai")
	}
}

// TestDarknessReachesMaxAndLoses menelusuri jalur kekalahan (GDD 6.1, 22).
func TestDarknessReachesMaxAndLoses(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 10)

	// Semua pemain hanya mengakhiri giliran; Darkness naik 1 tiap ronde.
	for range 200 {
		if s.Over() {
			break
		}
		p := s.ActivePlayer()
		if p == nil {
			t.Fatal("tidak ada pemain aktif")
		}
		drive(t, s, rng, Command{Kind: CmdEndTurn, Player: p.ID})
	}

	if s.Status != StatusLost {
		t.Fatalf("status: got %q, want %q (darkness=%d, round=%d)",
			s.Status, StatusLost, s.Darkness, s.Round)
	}
	if s.Darkness != c.Darkness.Max {
		t.Errorf("darkness: got %d, want %d", s.Darkness, c.Darkness.Max)
	}

	// Setelah match selesai, tidak ada aksi yang boleh diterima.
	_, err := Decide(s, Command{Kind: CmdRest, Player: s.TurnOrder[0]}, c, rng)
	if err != ErrMatchOver {
		t.Errorf("aksi setelah match selesai: got %v, want %v", err, ErrMatchOver)
	}
}

// TestDarknessThresholdsApply memverifikasi dua ambang yang sudah aktif di M0:
// penalti gather di 4 dan penalti AP di 7 (GDD 23).
func TestDarknessThresholdsApply(t *testing.T) {
	c := DefaultContent()

	if got := c.Darkness.amountFor("gather_penalty", 3); got != 0 {
		t.Errorf("darkness 3 seharusnya belum kena penalti gather, got %d", got)
	}
	if got := c.Darkness.amountFor("gather_penalty", 4); got != 1 {
		t.Errorf("darkness 4 seharusnya kena penalti gather 1, got %d", got)
	}
	if got := c.Darkness.amountFor("ap_penalty", 6); got != 0 {
		t.Errorf("darkness 6 seharusnya belum kena penalti AP, got %d", got)
	}
	if got := c.Darkness.amountFor("ap_penalty", 7); got != 1 {
		t.Errorf("darkness 7 seharusnya kena penalti AP 1, got %d", got)
	}

	// Efek nyata dari penalti gather: hasil turun tapi tidak pernah nol,
	// karena gather bernilai nol membuat game tidak bisa dimenangkan.
	s, rng := newTestGame(t, 11)
	Apply(s, Event{Kind: EvDarknessRose, Value: 4})

	p := s.ActivePlayer()
	drive(t, s, rng, Command{Kind: CmdMove, Player: p.ID, To: "forest"})
	drive(t, s, rng, Command{Kind: CmdGather, Player: p.ID, Resource: Wood})

	got := s.Player(p.ID).Inventory[Wood]
	want := c.Rules.GatherBaseAmount - 1
	if got != want {
		t.Errorf("gather saat darkness 4: got %d, want %d", got, want)
	}
	if got < 1 {
		t.Error("gather tidak boleh menghasilkan nol")
	}
}

// TestCrystalGatheringRaisesDarkness memeriksa GDD 19 dan 38: kenaikan Darkness
// harus terikat ke aksi yang terlihat, dengan alasan yang bisa ditampilkan.
func TestCrystalGatheringRaisesDarkness(t *testing.T) {
	c := DefaultContent()
	lt := c.LocationType("crystal_cavern")
	if lt == nil {
		t.Fatal("tipe lokasi crystal_cavern tidak ada")
	}
	if lt.DarknessOnGather <= 0 {
		t.Error("menambang kristal seharusnya menaikkan Darkness (GDD 19)")
	}
	// Peta prototype (GDD 32) belum memuat crystal_cavern; efeknya sudah ada di
	// konten dan diuji lewat definisinya sampai peta M1 memasukkannya.
}

func TestRestRules(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 12)
	p := s.ActivePlayer()

	// Health penuh -> tidak ada gunanya istirahat.
	_, err := Decide(s, Command{Kind: CmdRest, Player: p.ID}, c, rng)
	if err != ErrHealthFull {
		t.Errorf("istirahat saat health penuh: got %v, want %v", err, ErrHealthFull)
	}

	Apply(s, Event{Kind: EvDamaged, Player: p.ID, Amount: 2})
	drive(t, s, rng, Command{Kind: CmdRest, Player: p.ID})
	if got := s.Player(p.ID).Health; got != 2 {
		t.Errorf("health setelah istirahat: got %d, want 2", got)
	}

	// GDD 11: tidak bisa istirahat kalau ada monster.
	s.Board.Location(s.Player(p.ID).At).Monsters = 1
	_, err = Decide(s, Command{Kind: CmdRest, Player: p.ID}, c, rng)
	if err != ErrMonsterPresent {
		t.Errorf("istirahat dengan monster: got %v, want %v", err, ErrMonsterPresent)
	}
}

func TestTurnAutoEndsWhenAPExhausted(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 13)
	first := s.ActivePlayer().ID

	for range c.Rules.ActionPointsPerTurn {
		p := s.ActivePlayer()
		if p.ID != first {
			break
		}
		drive(t, s, rng, Command{Kind: CmdMove, Player: p.ID,
			To: pickAdjacent(s, p)})
	}

	if s.ActivePlayer().ID == first {
		t.Error("giliran seharusnya berpindah otomatis setelah AP habis")
	}
}

func pickAdjacent(s *State, p *Player) LocationID {
	loc := s.Board.Location(p.At)
	return loc.Adjacent[0]
}

func TestUnknownCommandRejected(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 14)
	p := s.ActivePlayer()

	_, err := Decide(s, Command{Kind: "teleport", Player: p.ID}, c, rng)
	if err != ErrBadCommand {
		t.Errorf("command tak dikenal: got %v, want %v", err, ErrBadCommand)
	}
}

// TestLegalCommandsAreAllAccepted adalah kontrak antara client dan server:
// apa pun yang ditawarkan LegalCommands harus diterima Decide. Kalau tidak,
// UI akan menyalakan tombol yang lalu ditolak, dan pemain mengira game rusak.
func TestLegalCommandsAreAllAccepted(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 15)
	pick := NewRNG(21)

	for step := 0; step < 300 && !s.Over(); step++ {
		active := s.ActivePlayer()
		if active == nil {
			break
		}
		view := Project(s, active.ID)
		legal := LegalCommands(view, c)
		if len(legal) == 0 {
			t.Fatalf("langkah %d: tidak ada aksi legal untuk pemain aktif", step)
		}

		// Setiap usulan harus lolos Decide di state yang sama.
		for _, cmd := range legal {
			if _, err := Decide(s, cmd, c, rng); err != nil {
				t.Fatalf("langkah %d: LegalCommands mengusulkan %+v tapi Decide menolak: %v",
					step, cmd, err)
			}
		}

		evs, err := Decide(s, legal[pick.Intn(len(legal))], c, rng)
		if err != nil {
			t.Fatalf("langkah %d: %v", step, err)
		}
		ApplyAll(s, evs)
		s.RNGState = rng.Seed()
	}
}
