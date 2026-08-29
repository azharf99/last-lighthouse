package core

// Mesin fase, mengikuti struktur ronde GDD 12:
//
//	1. EVENT PHASE     -> tarik 1 Event Card          (M1)
//	2. PLAYER PHASE    -> tiap pemain dapat 3 AP      (M0)
//	3. MONSTER PHASE   -> monster bergerak & menyerang (M1)
//	4. DARKNESS PHASE  -> Darkness naik               (M0)
//	5. CHECK VICTORY / DEFEAT                          (M0)

// beginRound memulai ronde baru: geser token First Player, jalankan fase Event,
// lalu buka fase Pemain dengan giliran pertama.
func beginRound(em *emitter, c *Content, rng *RNG) {
	s := em.s
	if s.Over() {
		return
	}

	first := s.FirstIdx
	if s.Round > 0 {
		// GDD 27: token First Player bergeser searah jarum jam tiap ronde,
		// supaya keuntungan pemain pertama tidak menumpuk.
		first = (s.FirstIdx + 1) % len(s.TurnOrder)
	}

	em.emit(Event{Kind: EvRoundStarted, Round: s.Round + 1, Amount: first})

	// Regenerasi resource lokasi supaya pulau tidak habis kering permanen.
	// Tanpa ini, game jadi tidak bisa dimenangkan setelah beberapa ronde --
	// ditemukan saat menulis test loop penuh, bukan saat desain.
	regenLocations(em, c)

	// --- Fase 1: Event (GDD 13) ---
	em.emit(Event{Kind: EvPhaseChanged, Phase: PhaseEvent})
	runEventPhase(em, c, rng)
	if s.Over() {
		return
	}

	// --- Fase 2: Pemain (GDD 14) ---
	em.emit(Event{Kind: EvPhaseChanged, Phase: PhasePlayer})
	startTurn(em, c, first)
}

func regenLocations(em *emitter, c *Content) {
	s := em.s
	regen := c.Rules.LocationRegenPerRound
	if regen <= 0 {
		return
	}
	maxPer := c.Rules.LocationMaxAvailablePerResource

	for i := range s.Board.Locations {
		loc := &s.Board.Locations[i]
		lt := c.LocationType(loc.Type)
		if lt == nil || lt.Yields.IsEmpty() {
			continue
		}
		var add ResourceSet
		for r := range ResourceCount {
			if lt.Yields[r] == 0 {
				continue
			}
			if loc.Available[r] >= maxPer {
				continue
			}
			n := regen
			if loc.Available[r]+n > maxPer {
				n = maxPer - loc.Available[r]
			}
			add[r] = n
		}
		if add.IsEmpty() {
			continue
		}
		// Harus lewat emit, bukan mutasi langsung: kalau regenerasi tidak
		// tercatat sebagai event, replay dari event log akan menghasilkan stok
		// lokasi yang berbeda dan determinisme rusak.
		em.emit(Event{Kind: EvLocationRegen, From: loc.ID, Resources: add})
	}
}

// startTurn membuka giliran pemain pada indeks idx dan memberinya AP.
func startTurn(em *emitter, c *Content, idx int) {
	s := em.s
	pid := s.TurnOrder[idx]

	ap := c.Rules.ActionPointsPerTurn
	// GDD 23, ambang 7: pemain kehilangan 1 AP.
	if pen := c.Darkness.amountFor("ap_penalty", s.Darkness); pen > 0 {
		ap -= pen
		if ap < 0 {
			ap = 0
		}
	}

	em.emit(Event{Kind: EvTurnStarted, Player: pid, Amount: idx, Value: ap})

	// Kalau Darkness sudah menekan AP sampai nol, giliran langsung lewat --
	// kalau tidak, match macet menunggu aksi yang tidak mungkin dilakukan.
	if ap == 0 {
		endTurn(em, c, nil)
	}
}

// endTurn menutup giliran aktif dan berpindah ke pemain berikutnya, atau menutup
// ronde kalau semua pemain sudah bergerak.
//
// rng boleh nil selama fase yang belum memakai keacakan (M0). Saat fase Monster
// diimplementasikan di M1, ia jadi wajib.
func endTurn(em *emitter, c *Content, rng *RNG) {
	s := em.s
	if s.Over() {
		return
	}

	cur := s.TurnOrder[s.ActiveIdx]
	em.emit(Event{Kind: EvTurnEnded, Player: cur}) // Apply menaikkan TurnsTaken

	if s.TurnsTaken < len(s.TurnOrder) {
		next := (s.ActiveIdx + 1) % len(s.TurnOrder)
		startTurn(em, c, next)
		return
	}

	endRound(em, c, rng)
}

// endRound menjalankan fase Monster dan Darkness, memeriksa kondisi akhir, lalu
// memulai ronde berikutnya kalau match masih berjalan.
func endRound(em *emitter, c *Content, rng *RNG) {
	// TurnsTaken di-reset oleh EvRoundStarted di beginRound, bukan di sini,
	// supaya seluruh perubahan state tetap berasal dari event.

	// --- Fase 3: Monster (GDD 15) ---
	em.emit(Event{Kind: EvPhaseChanged, Phase: PhaseMonster})
	runMonsterPhase(em, c, rng)
	if em.s.Over() {
		return
	}

	// --- Fase 4: Darkness (GDD 22) ---
	em.emit(Event{Kind: EvPhaseChanged, Phase: PhaseDarkness})
	raiseDarkness(em, c, c.Darkness.RisePerRound, "akhir ronde")

	// GDD 23, ambang 6: monster mulai muncul lebih sering.
	if n := c.Darkness.amountFor("monster_spawn_per_round", em.s.Darkness); n > 0 {
		for range n {
			spawnMonster(em, c, rng, "kegelapan melahirkan monster baru")
		}
	}

	// --- Fase 5: Cek menang/kalah ---
	if checkEnd(em, c) {
		return
	}

	beginRound(em, c, rng)
}

// raiseDarkness menaikkan Darkness track dan menjepitnya pada nilai maksimum.
// Semua kenaikan Darkness melewati fungsi ini supaya GDD 38 ("pemain tidak
// paham kenapa Darkness naik") bisa dijawab: setiap kenaikan membawa Reason
// yang bisa ditampilkan di log.
func raiseDarkness(em *emitter, c *Content, amount int, reason string) {
	if amount <= 0 {
		return
	}
	next := em.s.Darkness + amount
	if next > c.Darkness.Max {
		next = c.Darkness.Max
	}
	if next == em.s.Darkness {
		return
	}
	em.emit(Event{
		Kind:   EvDarknessRose,
		Amount: next - em.s.Darkness,
		Value:  next,
		Reason: reason,
	})
}

// checkEnd mengevaluasi kondisi menang/kalah dan mengembalikan true kalau match
// berakhir (GDD 6, 26).
func checkEnd(em *emitter, c *Content) bool {
	s := em.s

	// GDD 6.1: Darkness mencapai batas -> semua kalah, tanpa pemenang individual.
	if s.Darkness >= c.Darkness.Max {
		em.emit(Event{Kind: EvGameLost, Reason: "darkness mencapai batas"})
		return true
	}

	// GDD 6.2 / 26.1: kelima komponen selesai -> mercusuar menyala.
	if s.NextComponent() == nil {
		em.emit(Event{Kind: EvGameWon, Reason: "mercusuar dinyalakan kembali"})
		awardObjectiveVP(em, c)
		return true
	}

	return false
}

// awardObjectiveVP memberi VP personal objective saat game dimenangkan (GDD 24).
//
// Semua penghitungnya kini tersedia di state, sehingga tidak perlu memutar ulang
// event log untuk menghitung skor akhir.
func awardObjectiveVP(em *emitter, c *Content) {
	s := em.s

	for i := range s.Players {
		p := &s.Players[i]
		def := c.Objective(p.Objective)
		if def == nil {
			continue
		}

		need := def.Requirement.Count
		met := false
		switch def.Requirement.Kind {
		case "hold_resources":
			met = p.Inventory.Total() >= need
		case "never_exhausted":
			met = !p.WasExhausted
		case "contribute_repairs":
			met = p.RepairsJoined >= need
		case "contribute_resources":
			met = contributedResources(s, p.ID) >= need
		case "explore_locations":
			met = p.Explored >= need
		case "defeat_monsters":
			met = p.MonstersSlain >= need
		case "rescue_villages":
			met = p.VillagesRescued >= need
		case "acquire_artifacts":
			met = len(p.Artifacts) >= need
		default:
			continue
		}

		if met {
			em.emitSecret(
				p.ID,
				Event{Kind: EvVPAwarded, Player: p.ID, Amount: def.VP,
					Objective: p.Objective, Reason: "personal objective"},
				// Skor akhir memang terbuka (GDD 26), jadi jumlah VP-nya publik;
				// yang disembunyikan sampai akhir hanyalah objective-nya apa.
				&Event{Kind: EvVPAwarded, Player: p.ID, Amount: def.VP,
					Reason: "personal objective"},
			)
		}
	}

	// VP dari artifact yang dipegang (GDD 21, 25).
	for i := range s.Players {
		p := &s.Players[i]
		total := 0
		for _, id := range p.Artifacts {
			if def := c.Artifact(id); def != nil {
				total += def.VP
			}
		}
		if total > 0 {
			em.emit(Event{Kind: EvVPAwarded, Player: p.ID, Amount: total,
				Reason: "artifact"})
		}
	}
}

func contributedResources(s *State, pid PlayerID) int {
	total := 0
	for _, comp := range s.Lighthouse {
		for _, con := range comp.Contributions {
			if con.Player == pid {
				total += con.Amount
			}
		}
	}
	return total
}

// runEventPhase menarik dan meresolusikan satu kartu Event (GDD 13).
//
// Kartunya terbuka untuk semua pemain: efeknya global, dan menyembunyikannya
// hanya akan membuat pemain bingung kenapa keadaan berubah -- persis risiko
// GDD 38 tentang Darkness yang naik tanpa sebab yang terlihat.
func runEventPhase(em *emitter, c *Content, rng *RNG) {
	card := drawCard(em, DeckEvent, rng, "", "fase event")
	if card == "" {
		return
	}
	def := c.EventCard(EventCardID(card))
	if def == nil {
		discardCard(em, DeckEvent, card)
		return
	}

	// Efek kartu Event bersifat global, jadi tidak ada actor -- efek yang
	// menyasar satu pemain (mis. gain_artifact) sengaja dilewati.
	applyEffects(em, c, rng, def.Effects, "", def.Name.Get("en"))
	discardCard(em, DeckEvent, card)
}
