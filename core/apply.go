package core

// Apply adalah SATU-SATUNYA jalur mutasi State di seluruh codebase.
//
// Ia murni terhadap keacakan (tidak pernah memanggil RNG), tidak pernah gagal,
// dan tidak pernah memvalidasi -- semua itu sudah terjadi di Decide. Karena
// itulah event log bisa direplay bertahun-tahun kemudian dan menghasilkan state
// yang persis sama (ADR-004).
//
// Apply juga dipakai client: PlayerView membungkus State yang field rahasianya
// sudah dikosongkan, sehingga client menjalankan fungsi ini juga -- bukan
// implementasi kedua yang bisa menyimpang.
//
// Event yang tidak dikenal diabaikan diam-diam. Ini disengaja: client versi lama
// harus tetap bisa memproses stream dari server yang lebih baru tanpa crash.
func Apply(s *State, e Event) {
	switch e.Kind {
	case EvMatchStarted:
		s.Status = StatusActive
		if e.Setup != nil {
			// Deep copy: event masuk ke event log dan bisa direplay lagi nanti,
			// jadi state tidak boleh berbagi slice dengannya.
			su := e.Setup
			s.MatchID = su.MatchID
			s.ContentHash = su.ContentHash
			s.Darkness = su.Darkness
			s.TurnOrder = append([]PlayerID(nil), su.TurnOrder...)

			s.Players = make([]Player, len(su.Players))
			for i, p := range su.Players {
				p.Artifacts = append([]ArtifactID(nil), p.Artifacts...)
				s.Players[i] = p
			}

			s.Board.Locations = make([]Location, len(su.Board.Locations))
			for i, l := range su.Board.Locations {
				l.Adjacent = append([]LocationID(nil), l.Adjacent...)
				s.Board.Locations[i] = l
			}

			s.Lighthouse = make([]Component, len(su.Lighthouse))
			for i, comp := range su.Lighthouse {
				comp.Contributions = append([]Contribution(nil), comp.Contributions...)
				s.Lighthouse[i] = comp
			}
		}

	case EvObjectiveDealt:
		// Objective kosong pada varian publik yang diterima pemain lain.
		if p := s.Player(e.Player); p != nil && e.Objective != "" {
			p.Objective = e.Objective
		}

	case EvRoundStarted:
		s.Round = e.Round
		s.FirstIdx = e.Amount
		s.TurnsTaken = 0
		for i := range s.Players {
			s.Players[i].RepairDiscountUsed = false
		}
		// Efek kartu Event hanya berlaku satu ronde, dan lokasi yang bisa
		// diselidiki menyegar kembali.
		for i := range s.Board.Locations {
			s.Board.Locations[i].GatherBlocked = false
			s.Board.Locations[i].Investigated = false
		}

	case EvPhaseChanged:
		s.Phase = e.Phase

	case EvTurnStarted:
		s.ActiveIdx = e.Amount
		if p := s.Player(e.Player); p != nil {
			p.AP = e.Value
			p.ActedThisTurn = false
			p.FreeMoveUsed = false
			p.AbilityUsedTurn = false
		}

	case EvTurnEnded:
		if p := s.Player(e.Player); p != nil {
			p.AP = 0
		}
		// Penghitung giliran ikut diturunkan dari event, bukan dimutasi
		// langsung oleh mesin fase: kalau tidak, state hasil replay akan
		// berbeda dari state live dan ronde tidak pernah berakhir.
		s.TurnsTaken++

	case EvAPSpent:
		if p := s.Player(e.Player); p != nil {
			p.AP -= e.Amount
			p.ActedThisTurn = true
		}

	case EvMoved:
		if p := s.Player(e.Player); p != nil {
			p.At = e.To
		}

	case EvResourceGained:
		if p := s.Player(e.Player); p != nil {
			p.Inventory = p.Inventory.Add(e.Resources)
		}
		// From menandai lokasi asal resource; stoknya berkurang.
		if loc := s.Board.Location(e.From); loc != nil {
			loc.Available = loc.Available.Sub(e.Resources)
		}

	case EvResourceSpent:
		if p := s.Player(e.Player); p != nil {
			p.Inventory = p.Inventory.Sub(e.Resources)
		}

	case EvLocationRegen:
		if loc := s.Board.Location(e.From); loc != nil {
			loc.Available = loc.Available.Add(e.Resources)
		}

	case EvRepaired:
		if comp := s.Component(e.Component); comp != nil {
			comp.Progress = comp.Progress.Add(e.Resources)
			comp.contribute(e.Player, e.Resources.Total())
		}
		if p := s.Player(e.Player); p != nil {
			p.RepairsJoined++
		}

	case EvComponentDone:
		if comp := s.Component(e.Component); comp != nil {
			comp.Repaired = true
		}

	case EvVPAwarded:
		if p := s.Player(e.Player); p != nil {
			p.VP += e.Amount
		}

	case EvHealed:
		if p := s.Player(e.Player); p != nil {
			p.Health += e.Amount
			if p.Health > 0 {
				p.Exhausted = false
			}
		}

	case EvDamaged:
		if p := s.Player(e.Player); p != nil {
			p.Health -= e.Amount
			if p.Health < 0 {
				p.Health = 0
			}
		}

	case EvExhausted:
		// GDD 17: pemain tidak pernah tersingkir permanen, hanya Exhausted.
		if p := s.Player(e.Player); p != nil {
			p.Exhausted = e.Value != 0
			if p.Exhausted {
				// Dicatat permanen: objective "Sang Penyintas" menuntut TIDAK
				// PERNAH kelelahan, bukan sekadar sehat di akhir permainan.
				p.WasExhausted = true
			}
		}

	case EvDarknessRose:
		s.Darkness = e.Value

	case EvGameWon:
		s.Status = StatusWon

	case EvGameLost:
		s.Status = StatusLost

	// --- M1 ---

	case EvDeckShuffled:
		if e.Tile == "tiles" {
			s.TileStack = make([]LocationTypeID, len(e.Cards))
			for i, cid := range e.Cards {
				s.TileStack[i] = LocationTypeID(cid)
			}
			return
		}
		if d := s.deck(e.Deck); d != nil {
			d.Draw = toCardIDs(e.Cards)
			d.Discard = nil
		}

	case EvDeckReshuffled:
		if d := s.deck(e.Deck); d != nil {
			d.Draw = toCardIDs(e.Cards)
			d.Discard = nil
		}

	case EvCardDrawn:
		// Kartu teratas dilepas. Di sisi client tumpukan berisi kartu tertutup
		// (lihat Project), jadi yang berubah hanya jumlahnya -- dan itu memang
		// satu-satunya hal yang boleh diketahui client.
		if d := s.deck(e.Deck); d != nil && len(d.Draw) > 0 {
			d.Draw = d.Draw[1:]
		}

	case EvEventResolved:
		if d := s.deck(e.Deck); d != nil && e.Card != "" {
			d.Discard = append(d.Discard, CardID(e.Card))
		}

	case EvLocationRevealed:
		if loc := s.Board.Location(e.From); loc != nil {
			loc.Explored = true
			loc.Type = e.Tile
			loc.Name = e.Reason
			loc.Available = e.Resources
		}
		if len(s.TileStack) > 0 {
			s.TileStack = s.TileStack[1:]
		}
		if p := s.Player(e.Player); p != nil {
			p.Explored++
		}

	case EvGatherBlocked:
		if loc := s.Board.Location(e.From); loc != nil {
			loc.GatherBlocked = e.Value != 0
		}

	case EvMonsterSpawned:
		if loc := s.Board.Location(e.From); loc != nil {
			loc.Monsters += e.Amount
		}

	case EvMonsterMoved:
		if from := s.Board.Location(e.From); from != nil && from.Monsters > 0 {
			from.Monsters--
		}
		if to := s.Board.Location(e.To); to != nil {
			to.Monsters++
		}

	case EvMonsterDefeated:
		if loc := s.Board.Location(e.From); loc != nil && loc.Monsters > 0 {
			loc.Monsters--
		}
		if p := s.Player(e.Player); p != nil {
			p.MonstersSlain++
		}

	case EvArtifactGained:
		if p := s.Player(e.Player); p != nil {
			p.Artifacts = append(p.Artifacts, e.Artifact)
		}

	case EvMysteryOffered:
		if e.Choice != nil {
			pc := *e.Choice
			pc.Cards = append([]EventCardID(nil), e.Choice.Cards...)
			pc.Options = append([]string(nil), e.Choice.Options...)
			s.Pending = &pc
		}

	case EvMysteryResolved, EvChoiceCleared:
		s.Pending = nil

	case EvTraded:
		// Resource berpindah tangan. Kapasitas penerima sudah divalidasi di Decide.
		if from := s.Player(e.Player); from != nil {
			from.Inventory = from.Inventory.Sub(e.Resources)
			from.ResourcesGiven += e.Resources.Total()
		}
		if to := s.Player(e.Target); to != nil {
			to.Inventory = to.Inventory.Add(e.Resources)
		}

	case EvInvestigated:
		if loc := s.Board.Location(e.From); loc != nil {
			loc.Investigated = true
		}

	case EvVillageRescued:
		if loc := s.Board.Location(e.From); loc != nil {
			loc.Rescued = true
		}
		if p := s.Player(e.Player); p != nil {
			p.VillagesRescued++
		}

	case EvAbilityUsed:
		if p := s.Player(e.Player); p != nil {
			switch e.Reason {
			case "turn":
				p.AbilityUsedTurn = true
			case "round":
				p.RepairDiscountUsed = true
			case "free_move":
				p.FreeMoveUsed = true
			}
		}

	case EvDiceRolled, EvMonsterAttacked:
		// Murni informasi untuk log; efeknya dibawa event terpisah.
	}
}

// ApplyAll adalah pembantu replay: dipakai untuk merekonstruksi state dari
// snapshot + event log setelah restart server (ADR-004).
func ApplyAll(s *State, events []Event) {
	for _, e := range events {
		Apply(s, e)
	}
}

// emitter menjaga State dan event log tidak pernah menyimpang: setiap event yang
// dicatat langsung diterapkan lewat Apply, jalur mutasi yang sama dengan yang
// dipakai replay. Tidak ada tempat di core ini yang boleh mengubah State tanpa
// melewati emitter.
type emitter struct {
	s   *State
	out []Event
}

func newEmitter(s *State) *emitter { return &emitter{s: s} }

func (em *emitter) emit(e Event) {
	e.V = eventSchemaVersion
	em.out = append(em.out, e)
	Apply(em.s, e)
}

// emitSecret mencatat event yang hanya boleh dilihat satu pemain, dengan varian
// publik opsional untuk yang lain (ADR-006).
func (em *emitter) emitSecret(owner PlayerID, e Event, publicVariant *Event) {
	em.emit(secret(owner, e, publicVariant))
}
