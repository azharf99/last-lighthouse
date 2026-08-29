package core

import "testing"

// --- Eksplorasi (GDD 18) ---

func TestExploreRevealsTileAndCountsProgress(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 101)
	p := s.ActivePlayer()

	// Bawa pemain ke lokasi yang bersebelahan dengan slot "?".
	Apply(s, Event{Kind: EvMoved, Player: p.ID, To: "ruins"})
	before := len(s.TileStack)

	if loc := s.Board.Location("site_a"); loc == nil || loc.Explored {
		t.Fatal("site_a seharusnya ada dan belum tereksplorasi")
	}

	drive(t, s, rng, Command{Kind: CmdExplore, Player: p.ID, To: "site_a"})

	loc := s.Board.Location("site_a")
	if !loc.Explored {
		t.Error("site_a seharusnya tereksplorasi")
	}
	if loc.Type == "" || c.LocationType(loc.Type) == nil {
		t.Errorf("tile yang dibuka tidak punya tipe valid: %q", loc.Type)
	}
	if len(s.TileStack) != before-1 {
		t.Errorf("tumpukan tile: got %d, want %d", len(s.TileStack), before-1)
	}
	if got := s.Player(p.ID).Explored; got != 1 {
		t.Errorf("penghitung eksplorasi: got %d, want 1", got)
	}

	// Mengeksplorasi lokasi yang sama dua kali harus ditolak.
	if _, err := Decide(s, Command{Kind: CmdExplore, Player: p.ID, To: "site_a"}, c, rng); err != ErrAlreadyExplored {
		t.Errorf("eksplor ulang: got %v, want %v", err, ErrAlreadyExplored)
	}
}

func TestExploreRequiresAdjacency(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 102)
	p := s.ActivePlayer()

	// Pemain mulai di mercusuar; site_a tidak bersebelahan dengannya.
	_, err := Decide(s, Command{Kind: CmdExplore, Player: p.ID, To: "site_a"}, c, rng)
	if err != ErrNotAdjacent {
		t.Errorf("eksplor lokasi jauh: got %v, want %v", err, ErrNotAdjacent)
	}
}

// --- Combat (GDD 16) ---

func TestFightResolvesByDiceRange(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 103)
	p := s.ActivePlayer()

	loc := s.Board.Location(p.At)
	Apply(s, Event{Kind: EvMonsterSpawned, From: loc.ID, Amount: 1})

	evs := drive(t, s, rng, Command{Kind: CmdFight, Player: p.ID})

	var roll *Event
	for i := range evs {
		if evs[i].Kind == EvDiceRolled {
			roll = &evs[i]
		}
	}
	if roll == nil {
		t.Fatal("fight harus menghasilkan lemparan dadu")
	}
	if roll.Amount < 1 || roll.Amount > 6 {
		t.Errorf("lemparan 1D6 di luar rentang: %d", roll.Amount)
	}

	// Hasilnya harus konsisten dengan rentang di konten (GDD 16).
	total := roll.Value
	defeated, damaged := false, false
	for _, e := range evs {
		switch e.Kind {
		case EvMonsterDefeated:
			defeated = true
		case EvDamaged:
			damaged = true
		}
	}
	switch {
	case total >= c.Monsters.Combat.MonsterDefeatedMin:
		if !defeated {
			t.Errorf("total %d seharusnya mengalahkan monster", total)
		}
	case total <= c.Monsters.Combat.PlayerDamagedMax:
		if !damaged {
			t.Errorf("total %d seharusnya melukai pemain", total)
		}
	default:
		if defeated || damaged {
			t.Errorf("total %d seharusnya seri, tapi ada kekalahan/luka", total)
		}
	}
}

func TestExhaustedPlayerCannotFight(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 104)
	p := s.ActivePlayer()

	loc := s.Board.Location(p.At)
	Apply(s, Event{Kind: EvMonsterSpawned, From: loc.ID, Amount: 1})
	Apply(s, Event{Kind: EvDamaged, Player: p.ID, Amount: c.Rules.MaxHealth})
	Apply(s, Event{Kind: EvExhausted, Player: p.ID, Value: 1})

	// GDD 17: pemain Exhausted tidak bisa bertarung, tapi juga tidak tersingkir.
	if _, err := Decide(s, Command{Kind: CmdFight, Player: p.ID}, c, rng); err != ErrExhaustedNoFight {
		t.Errorf("fight saat kelelahan: got %v, want %v", err, ErrExhaustedNoFight)
	}
	if s.Player(p.ID).Health != 0 {
		t.Error("pemain kelelahan seharusnya di 0 HP, bukan terhapus")
	}
}

func TestHunterGetsCombatBonus(t *testing.T) {
	c := DefaultContent()
	var hunter, other Player
	for _, ch := range []CharacterID{"hunter", "navigator"} {
		p := Player{Character: ch}
		if ch == "hunter" {
			hunter = p
		} else {
			other = p
		}
	}
	if combatModifier(c, &hunter) <= combatModifier(c, &other) {
		t.Error("Hunter seharusnya punya bonus combat lebih tinggi (GDD 10.3)")
	}
}

// --- Mystery & pilihan tertunda (GDD 20) ---

func TestInvestigateOffersChoiceAndBlocksOtherActions(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 105)
	p := s.ActivePlayer()

	Apply(s, Event{Kind: EvMoved, Player: p.ID, To: "ruins"})
	drive(t, s, rng, Command{Kind: CmdInvestigate, Player: p.ID})

	if s.Pending == nil {
		t.Fatal("investigate harus menghasilkan pilihan tertunda")
	}
	if s.Pending.Player != p.ID {
		t.Errorf("pilihan milik %q, want %q", s.Pending.Player, p.ID)
	}
	if len(s.Pending.Options) == 0 && len(s.Pending.Cards) == 0 {
		t.Error("pilihan tertunda tanpa opsi apa pun")
	}

	// Selama tertunda, aksi lain harus ditolak: dilema GDD 20 wajib dijawab.
	if _, err := Decide(s, Command{Kind: CmdRest, Player: p.ID}, c, rng); err != ErrChoicePending {
		t.Errorf("aksi lain saat ada pilihan tertunda: got %v, want %v", err, ErrChoicePending)
	}

	// Pemain lain tidak boleh menjawab pilihan orang.
	var other PlayerID
	for _, pl := range s.Players {
		if pl.ID != p.ID {
			other = pl.ID
			break
		}
	}
	if _, err := Decide(s, Command{Kind: CmdChoose, Player: other, Option: "a"}, c, rng); err != ErrNotYourChoice {
		t.Errorf("pemain lain menjawab: got %v, want %v", err, ErrNotYourChoice)
	}
}

func TestChooseResolvesMystery(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 106)
	p := s.ActivePlayer()

	Apply(s, Event{Kind: EvMoved, Player: p.ID, To: "ruins"})
	drive(t, s, rng, Command{Kind: CmdInvestigate, Player: p.ID})
	if s.Pending == nil {
		t.Fatal("tidak ada pilihan tertunda")
	}

	// Kalau kemampuan Scholar aktif, jawab tahap pemilihan kartu lebih dulu.
	if s.Pending.Kind == "mystery_card" {
		drive(t, s, rng, Command{Kind: CmdChoose, Player: p.ID, Card: s.Pending.Cards[0]})
		if s.Pending == nil || s.Pending.Kind != "mystery_option" {
			t.Fatalf("setelah memilih kartu seharusnya lanjut ke pilihan opsi, got %+v", s.Pending)
		}
	}

	opt := s.Pending.Options[0]
	drive(t, s, rng, Command{Kind: CmdChoose, Player: p.ID, Option: opt})

	if s.Pending != nil {
		t.Errorf("pilihan seharusnya selesai, masih ada %+v", s.Pending)
	}
	if !s.Board.Location("ruins").Investigated {
		t.Error("lokasi seharusnya ditandai sudah diselidiki")
	}

	// Dalam ronde yang sama, lokasi itu tidak bisa diselidiki lagi.
	if s.ActivePlayer() != nil && s.ActivePlayer().ID == p.ID && s.Player(p.ID).AP > 0 {
		if _, err := Decide(s, Command{Kind: CmdInvestigate, Player: p.ID}, c, rng); err != ErrAlreadyInvestigated {
			t.Errorf("selidiki ulang: got %v, want %v", err, ErrAlreadyInvestigated)
		}
	}
}

func TestChooseRejectsUnavailableOption(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 107)
	p := s.ActivePlayer()

	Apply(s, Event{Kind: EvMoved, Player: p.ID, To: "ruins"})
	drive(t, s, rng, Command{Kind: CmdInvestigate, Player: p.ID})
	if s.Pending == nil {
		t.Fatal("tidak ada pilihan tertunda")
	}

	if _, err := Decide(s, Command{Kind: CmdChoose, Player: p.ID, Option: "zz"}, c, rng); err != ErrBadOption {
		t.Errorf("opsi tidak ada: got %v, want %v", err, ErrBadOption)
	}
}

// --- Trade (GDD 11, 28) ---

func TestTradeMovesResourcesBetweenPlayersHere(t *testing.T) {
	s, rng := newTestGame(t, 108)
	giver := s.ActivePlayer()

	var receiver PlayerID
	for _, pl := range s.Players {
		if pl.ID != giver.ID {
			receiver = pl.ID
			break
		}
	}

	grant(s, giver.ID, NewResourceSet(map[Resource]int{Wood: 2}))
	give := NewResourceSet(map[Resource]int{Wood: 1})

	// Keduanya mulai di mercusuar, jadi sudah berada di lokasi yang sama.
	drive(t, s, rng, Command{Kind: CmdTrade, Player: giver.ID, Target: receiver, Give: give})

	if got := s.Player(giver.ID).Inventory[Wood]; got != 1 {
		t.Errorf("pemberi menyisakan %d wood, want 1", got)
	}
	if got := s.Player(receiver).Inventory[Wood]; got != 1 {
		t.Errorf("penerima punya %d wood, want 1", got)
	}
	if got := s.Player(giver.ID).ResourcesGiven; got != 1 {
		t.Errorf("penghitung resource diberikan: got %d, want 1", got)
	}
}

func TestTradeRequiresSameLocation(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 109)
	giver := s.ActivePlayer()

	var receiver PlayerID
	for _, pl := range s.Players {
		if pl.ID != giver.ID {
			receiver = pl.ID
			break
		}
	}
	grant(s, giver.ID, NewResourceSet(map[Resource]int{Wood: 2}))
	Apply(s, Event{Kind: EvMoved, Player: receiver, To: "forest"})

	_, err := Decide(s, Command{Kind: CmdTrade, Player: giver.ID, Target: receiver,
		Give: NewResourceSet(map[Resource]int{Wood: 1})}, c, rng)
	if err != ErrTargetNotHere {
		t.Errorf("trade lintas lokasi: got %v, want %v", err, ErrTargetNotHere)
	}
}

func TestTradeRejectsWhatYouDoNotHave(t *testing.T) {
	c := DefaultContent()
	s, rng := newTestGame(t, 110)
	giver := s.ActivePlayer()

	var receiver PlayerID
	for _, pl := range s.Players {
		if pl.ID != giver.ID {
			receiver = pl.ID
			break
		}
	}

	_, err := Decide(s, Command{Kind: CmdTrade, Player: giver.ID, Target: receiver,
		Give: NewResourceSet(map[Resource]int{Crystal: 2})}, c, rng)
	if err != ErrNotEnoughRes {
		t.Errorf("trade tanpa stok: got %v, want %v", err, ErrNotEnoughRes)
	}

	// Nilai negatif adalah upaya menyedot resource dari pemain lain lewat trade.
	// Command datang dari mesin pemain, jadi ini harus ditolak mentah-mentah.
	_, err = Decide(s, Command{Kind: CmdTrade, Player: giver.ID, Target: receiver,
		Give: ResourceSet{-5, 0, 0, 0}}, c, rng)
	if err != ErrBadCommand {
		t.Errorf("trade dengan jumlah negatif: got %v, want %v", err, ErrBadCommand)
	}
}

// --- Fase Event & Monster (GDD 13, 15) ---

func TestEventPhaseDrawsACardEachRound(t *testing.T) {
	s, _, err := NewGame("m_event", 111, testSetups(), DefaultContent())
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}

	total := s.EventDeck.Len() + len(s.EventDeck.Discard)
	if total == 0 {
		t.Fatal("deck event kosong setelah setup")
	}
	// Ronde pertama sudah dimulai di NewGame, jadi satu kartu sudah terpakai.
	if len(s.EventDeck.Discard) != 1 {
		t.Errorf("setelah ronde 1, buangan berisi %d kartu, want 1", len(s.EventDeck.Discard))
	}
}

func TestMonsterMovesTowardPlayers(t *testing.T) {
	s, rng := newTestGame(t, 112)

	// Taruh semua pemain di gua, dan satu monster di reruntuhan (bersebelahan).
	for _, p := range s.Players {
		Apply(s, Event{Kind: EvMoved, Player: p.ID, To: "cave"})
	}
	Apply(s, Event{Kind: EvMonsterSpawned, From: "ruins", Amount: 1})

	// Habiskan ronde supaya fase Monster berjalan.
	for range len(s.TurnOrder) {
		if s.Over() {
			break
		}
		drive(t, s, rng, Command{Kind: CmdEndTurn, Player: s.ActivePlayer().ID})
	}

	if s.Board.Location("ruins").Monsters != 0 {
		t.Error("monster seharusnya meninggalkan reruntuhan menuju pemain")
	}
	if s.Board.Location("cave").Monsters == 0 {
		t.Error("monster seharusnya tiba di gua tempat para pemain berada")
	}
}

func TestMonsterAttacksPlayersOnItsLocation(t *testing.T) {
	s, rng := newTestGame(t, 113)

	for _, p := range s.Players {
		Apply(s, Event{Kind: EvMoved, Player: p.ID, To: "cave"})
	}
	Apply(s, Event{Kind: EvMonsterSpawned, From: "cave", Amount: 1})

	totalHealthBefore := 0
	for _, p := range s.Players {
		totalHealthBefore += p.Health
	}

	for range len(s.TurnOrder) {
		if s.Over() {
			break
		}
		drive(t, s, rng, Command{Kind: CmdEndTurn, Player: s.ActivePlayer().ID})
	}

	totalHealthAfter := 0
	for _, p := range s.Players {
		totalHealthAfter += p.Health
	}
	if totalHealthAfter >= totalHealthBefore {
		t.Errorf("monster di lokasi pemain seharusnya menyerang: %d -> %d",
			totalHealthBefore, totalHealthAfter)
	}
}

// --- Artifact (GDD 21) ---

func TestArtifactGrantsCapacityAndScoresVP(t *testing.T) {
	c := DefaultContent()
	s, _ := newTestGame(t, 114)
	p := s.ActivePlayer()

	base := playerCapacity(c, s.Player(p.ID))
	Apply(s, Event{Kind: EvArtifactGained, Player: p.ID, Artifact: "ar_pilgrims_pack"})
	withPack := playerCapacity(c, s.Player(p.ID))

	if withPack <= base {
		t.Errorf("Ransel Peziarah seharusnya menambah kapasitas: %d -> %d", base, withPack)
	}

	def := c.Artifact("ar_black_pearl")
	if def == nil || def.VP <= 0 {
		t.Error("Mutiara Hitam seharusnya bernilai VP di akhir permainan (GDD 21)")
	}
}

// --- Objective M1 (GDD 24) ---

func TestObjectiveCountersTrackProgress(t *testing.T) {
	s, _ := newTestGame(t, 115)
	p := s.ActivePlayer()

	Apply(s, Event{Kind: EvLocationRevealed, Player: p.ID, From: "site_a", Tile: "forest"})
	Apply(s, Event{Kind: EvMonsterSpawned, From: "cave", Amount: 1})
	Apply(s, Event{Kind: EvMonsterDefeated, Player: p.ID, From: "cave"})
	Apply(s, Event{Kind: EvVillageRescued, Player: p.ID, From: "village"})

	got := s.Player(p.ID)
	if got.Explored != 1 {
		t.Errorf("Explored: got %d, want 1", got.Explored)
	}
	if got.MonstersSlain != 1 {
		t.Errorf("MonstersSlain: got %d, want 1", got.MonstersSlain)
	}
	if got.VillagesRescued != 1 {
		t.Errorf("VillagesRescued: got %d, want 1", got.VillagesRescued)
	}

	// WasExhausted harus permanen: objective "Sang Penyintas" menuntut TIDAK
	// PERNAH kelelahan, bukan sekadar sehat di akhir permainan.
	Apply(s, Event{Kind: EvExhausted, Player: p.ID, Value: 1})
	Apply(s, Event{Kind: EvHealed, Player: p.ID, Amount: 3})
	if !s.Player(p.ID).WasExhausted {
		t.Error("riwayat kelelahan seharusnya tidak terhapus oleh penyembuhan")
	}
	if s.Player(p.ID).Exhausted {
		t.Error("status kelelahan seharusnya hilang setelah sembuh")
	}
}
