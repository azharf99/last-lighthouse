package core

import "fmt"

// PlayerSetup adalah konfigurasi satu kursi saat match dibuat.
type PlayerSetup struct {
	ID        PlayerID
	Name      string
	Character CharacterID
}

// NewGame membangun State awal beserta event-event setup-nya.
//
// State awal tidak dirakit langsung ke dalam State, melainkan dibungkus ke dalam
// EvMatchStarted lalu dipasang lewat Apply. Ini menjaga aturan bahwa Apply
// adalah satu-satunya jalur mutasi, dan membuat event log swasembada: replay
// dari nol menghasilkan match yang identik tanpa perlu tahu konfigurasi awalnya.
//
// Pembagian objective bersifat rahasia (GDD 24) dan melewati mekanisme
// projection yang sama seperti langkah lain, sejak event pertama. Kalau setup
// memotong jalur itu, kebocoran pertama sudah terjadi sebelum ada yang bergerak.
func NewGame(matchID string, seed int64, setups []PlayerSetup, c *Content) (*State, []Event, error) {
	if c == nil {
		c = DefaultContent()
	}
	if len(setups) < 2 || len(setups) > 4 {
		return nil, nil, fmt.Errorf("core: butuh 2-4 pemain, ada %d", len(setups))
	}

	seen := map[PlayerID]bool{}
	for _, s := range setups {
		if s.ID == "" {
			return nil, nil, fmt.Errorf("core: player id kosong")
		}
		if seen[s.ID] {
			return nil, nil, fmt.Errorf("core: player id duplikat %q", s.ID)
		}
		seen[s.ID] = true
		if c.Character(s.Character) == nil {
			return nil, nil, fmt.Errorf("core: karakter tidak dikenal %q", s.Character)
		}
	}

	rng := NewRNG(seed)

	su := MatchSetup{
		MatchID:     matchID,
		ContentHash: c.Hash,
		Darkness:    c.Darkness.Start,
	}

	// Papan. Slot yang belum tereksplorasi belum punya tipe -- tipenya baru
	// ditentukan saat aksi Explore menarik tile (GDD 18), jadi di sini ia hanya
	// berupa tanda tanya tanpa resource.
	for _, ml := range c.Map.Locations {
		loc := Location{
			ID:       ml.ID,
			Type:     ml.Type,
			Name:     "?",
			Explored: ml.Explored,
			Adjacent: append([]LocationID(nil), ml.Adjacent...),
		}
		if lt := c.LocationType(ml.Type); lt != nil {
			loc.Name = lt.Name.Get("en")
			loc.Available = lt.Yields
		}
		su.Board.Locations = append(su.Board.Locations, loc)
	}

	// Mercusuar
	for _, cd := range c.Lighthouse {
		su.Lighthouse = append(su.Lighthouse, Component{
			ID:    cd.ID,
			Name:  cd.Name.Get("en"),
			Order: cd.Order,
			Cost:  cd.Cost,
			VP:    cd.VP,
		})
	}
	scaleComponentCosts(su.Lighthouse, len(setups), c)

	// Pemain. Objective sengaja dibiarkan kosong di sini -- ia dibagikan lewat
	// event rahasia di bawah, bukan lewat EvMatchStarted yang publik.
	for _, ps := range setups {
		su.Players = append(su.Players, Player{
			ID:        ps.ID,
			Name:      ps.Name,
			Character: ps.Character,
			At:        c.Map.StartLocation,
			Health:    c.Rules.MaxHealth,
			AP:        0,
		})
		su.TurnOrder = append(su.TurnOrder, ps.ID)
	}

	// Urutan giliran diacak: GDD 27 memakai "pemain yang terakhir melihat
	// mercusuar", yang tidak punya padanan digital. Token First Player tetap
	// bergeser tiap ronde untuk menekan keuntungan pemain pertama.
	rng.Shuffle(len(su.TurnOrder), func(i, j int) {
		su.TurnOrder[i], su.TurnOrder[j] = su.TurnOrder[j], su.TurnOrder[i]
	})

	s := &State{}
	em := newEmitter(s)
	em.emit(Event{Kind: EvMatchStarted, Reason: matchID, Setup: &su})

	// Bagikan personal objective (GDD 24). Tiap pemain dapat satu objective unik.
	deck := make([]ObjectiveID, 0, len(c.Objectives))
	for _, o := range c.Objectives {
		deck = append(deck, o.ID)
	}
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	for i := range s.TurnOrder {
		pid := s.TurnOrder[i]
		em.emitSecret(
			pid,
			Event{Kind: EvObjectiveDealt, Player: pid, Objective: deck[i]},
			// Yang dilihat pemain lain: bahwa pemain itu menerima objective,
			// tanpa objective-nya apa.
			&Event{Kind: EvObjectiveDealt, Player: pid},
		)
	}

	// Bangun deck. Hasil kocokan dikirim sebagai event supaya replay memakai
	// urutan yang sama tanpa perlu mereproduksi urutan pemanggilan RNG (ADR-002).
	eventIDs := make([]CardID, 0, len(c.Events))
	for _, e := range c.Events {
		eventIDs = append(eventIDs, CardID(e.ID))
	}
	buildDeck(em, DeckEvent, eventIDs, rng)

	mysteryIDs := make([]CardID, 0, len(c.Mysteries))
	for _, m := range c.Mysteries {
		mysteryIDs = append(mysteryIDs, CardID(m.ID))
	}
	buildDeck(em, DeckMystery, mysteryIDs, rng)

	artifactIDs := make([]CardID, 0, len(c.Artifacts))
	for _, a := range c.Artifacts {
		artifactIDs = append(artifactIDs, CardID(a.ID))
	}
	buildDeck(em, DeckArtifact, artifactIDs, rng)

	// Tumpukan tile eksplorasi (GDD 18), juga diacak.
	tiles := append([]LocationTypeID(nil), c.Map.TileStack...)
	rng.Shuffle(len(tiles), func(i, j int) { tiles[i], tiles[j] = tiles[j], tiles[i] })
	if c.Rules.GuaranteeCrystalTile {
		ensureCrystalReachable(tiles, countUnexplored(su.Board), c)
	}
	em.emitSecret(serverOnly,
		Event{Kind: EvDeckShuffled, Tile: "tiles", Cards: tileIDsToCards(tiles)},
		&Event{Kind: EvDeckShuffled, Tile: "tiles", Cards: hideCards(len(tiles))})

	// Mulai ronde pertama.
	beginRound(em, c, rng)

	s.RNGState = rng.Seed()
	return s, em.out, nil
}

// tileIDsToCards mengangkut tumpukan tile lewat field Cards pada event.
//
// Tile bukan kartu, tapi keduanya sama-sama "daftar terurut yang dirahasiakan",
// jadi mereka berbagi jalur event yang sama alih-alih menambah jenis event baru
// yang isinya identik.
func tileIDsToCards(tiles []LocationTypeID) []EventCardID {
	out := make([]EventCardID, len(tiles))
	for i, t := range tiles {
		out[i] = EventCardID(t)
	}
	return out
}

func countUnexplored(b Board) int {
	n := 0
	for i := range b.Locations {
		if !b.Locations[i].Explored {
			n++
		}
	}
	return n
}

// ensureCrystalReachable menukar satu tile penghasil crystal ke bagian tumpukan
// yang benar-benar akan terpakai.
//
// Hanya "slots" tile teratas yang pernah diletakkan; sisanya tidak pernah
// terlihat. Jadi mengacak saja tidak cukup -- crystal bisa terkubur di bagian
// tumpukan yang tidak pernah dibuka.
//
// Penukaran dilakukan pada posisi TERAKHIR yang terpakai, bukan yang pertama:
// dengan begitu crystal tetap terasa sebagai temuan di ujung eksplorasi, bukan
// hadiah yang langsung terbuka di petak pertama.
func ensureCrystalReachable(tiles []LocationTypeID, slots int, c *Content) {
	if slots <= 0 || slots > len(tiles) {
		return
	}
	yieldsCrystal := func(id LocationTypeID) bool {
		lt := c.LocationType(id)
		return lt != nil && lt.Yields[Crystal] > 0
	}

	for _, id := range tiles[:slots] {
		if yieldsCrystal(id) {
			return // sudah ada, tidak perlu diutak-atik
		}
	}
	for i := slots; i < len(tiles); i++ {
		if yieldsCrystal(tiles[i]) {
			tiles[slots-1], tiles[i] = tiles[i], tiles[slots-1]
			return
		}
	}
	// Tidak ada tile crystal sama sekali di konten: itu kesalahan konten, tapi
	// bukan alasan untuk menggagalkan match. Validate yang seharusnya menangkapnya.
}

// scaleComponentCosts menaikkan biaya mercusuar untuk meja yang lebih ramai.
//
// Tambahan biaya disebar bergiliran ke seluruh komponen, dan pada tiap komponen
// ditambahkan ke resource yang SUDAH dibutuhkannya. Itu menjaga karakter tiap
// komponen tetap sama -- Lensa tetap terasa "kristal", Fondasi tetap "kayu dan
// logam" -- alih-alih mengaburkannya jadi campuran seragam.
func scaleComponentCosts(comps []Component, players int, c *Content) {
	per := c.Rules.ExtraComponentCostPerExtraPlayer
	extra := (players - 2) * per
	if extra <= 0 || len(comps) == 0 {
		return
	}

	for i := range extra {
		comp := &comps[i%len(comps)]

		// Naikkan resource yang paling banyak dibutuhkan komponen ini; seri
		// diputus oleh urutan enum, sehingga hasilnya deterministik.
		best := -1
		for r := range ResourceCount {
			if comp.Cost[r] > 0 && (best < 0 || comp.Cost[r] > comp.Cost[best]) {
				best = r
			}
		}
		if best >= 0 {
			comp.Cost[best]++
		}
	}
}
