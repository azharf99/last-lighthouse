package core

// Aksi M1: Explore, Fight, Investigate, Trade, dan Choose.

// --- Explore (GDD 11, 18) ---

func decideExplore(em *emitter, c *Content, rng *RNG, cmd Command) error {
	s := em.s
	p := s.Player(cmd.Player)

	// Target boleh dikosongkan: kalau begitu, ambil slot "?" pertama yang
	// bersebelahan. Ini menyederhanakan UI tanpa mengubah aturannya.
	target := cmd.To
	if target == "" {
		if adj := s.Board.Location(p.At); adj != nil {
			for _, id := range adj.Adjacent {
				if loc := s.Board.Location(id); loc != nil && !loc.Explored {
					target = id
					break
				}
			}
		}
	}
	if target == "" {
		return ErrAlreadyExplored
	}

	dest := s.Board.Location(target)
	if dest == nil {
		return ErrUnknownLocation
	}
	if dest.Explored {
		return ErrAlreadyExplored
	}
	if !s.Board.Adjacent(p.At, target) {
		return ErrNotAdjacent
	}

	// Pathfinder (GDD 10.1): "moving into an unexplored location costs 1 fewer
	// Action Point once per turn".
	//
	// Di implementasi ini masuk ke petak yang belum dibuka BUKAN gerakan biasa
	// melainkan aksi Explore tersendiri, jadi kemampuannya diterjemahkan menjadi
	// "Explore gratis sekali per giliran". Itu tafsir yang paling dekat dengan
	// maksudnya: Navigator adalah spesialis penyingkap pulau, dan biaya yang
	// dipotong GDD justru biaya menembus wilayah tak dikenal.
	free := isNavigator(c, p) && !p.AbilityUsedTurn
	if free {
		em.emit(Event{Kind: EvAbilityUsed, Player: cmd.Player, Reason: "turn"})
	} else {
		spendAP(em, cmd)
	}
	revealLocation(em, c, rng, target, cmd.Player, "eksplorasi")
	return nil
}

func isNavigator(c *Content, p *Player) bool {
	def := c.Character(p.Character)
	return def != nil && def.Ability.ID == "pathfinder"
}

// --- Fight (GDD 11, 16) ---

func decideFight(em *emitter, c *Content, rng *RNG, cmd Command) error {
	s := em.s
	p := s.Player(cmd.Player)

	// GDD 17: pemain Exhausted tidak bisa melakukan aksi Fight.
	if p.Exhausted {
		return ErrExhaustedNoFight
	}

	loc := s.Board.Location(p.At)
	if loc == nil {
		return ErrUnknownLocation
	}
	if loc.Monsters <= 0 {
		return ErrNoMonster
	}

	spendAP(em, cmd)
	resolveCombat(em, c, rng, cmd.Player, loc.ID)

	// Mengalahkan monster terakhir di sebuah desa berarti desa itu selamat.
	tryRescueVillage(em, c, loc.ID, cmd.Player)
	return nil
}

// --- Investigate (GDD 11, 20) ---

// decideInvestigate menarik kartu Mystery dan MENAWARKAN pilihannya.
//
// Aksinya berhenti di sini: pemain harus melihat kartunya dulu, lalu menjawab
// lewat CmdChoose. Menggabungkan keduanya jadi satu command akan menghapus
// keputusan yang justru merupakan inti mekanik ini (GDD 20).
func decideInvestigate(em *emitter, c *Content, rng *RNG, cmd Command) error {
	s := em.s
	p := s.Player(cmd.Player)

	loc := s.Board.Location(p.At)
	if loc == nil {
		return ErrUnknownLocation
	}
	// Arah 2 (BALANCE-M1.md): InvestigateAnywhere membuka semua lokasi tereksplorasi.
	canInvestigate := false
	if lt := c.LocationType(loc.Type); lt != nil && lt.CanInvestigate {
		canInvestigate = true
	}
	if c.Rules.InvestigateAnywhere && loc.Explored {
		canInvestigate = true
	}
	if !canInvestigate {
		return ErrCannotInvestigate
	}
	if loc.Investigated {
		return ErrAlreadyInvestigated
	}
	if s.MysteryDeck.Len() == 0 && len(s.MysteryDeck.Discard) == 0 {
		return ErrNoMysteryLeft
	}

	spendAP(em, cmd)

	// Kemampuan Scholar (GDD 10.4): tarik 2, pilih 1.
	scholar := isScholar(c, p) && !p.AbilityUsedTurn && s.MysteryDeck.Len()+len(s.MysteryDeck.Discard) >= 2

	if scholar {
		first := drawCard(em, DeckMystery, rng, cmd.Player, "investigate")
		second := drawCard(em, DeckMystery, rng, cmd.Player, "investigate")
		em.emit(Event{Kind: EvAbilityUsed, Player: cmd.Player, Reason: "turn"})

		cards := []EventCardID{}
		for _, cid := range []CardID{first, second} {
			if cid != "" {
				cards = append(cards, EventCardID(cid))
			}
		}
		if len(cards) == 0 {
			return ErrNoMysteryLeft
		}
		if len(cards) == 1 {
			offerMysteryOptions(em, c, cmd.Player, cards[0])
			return nil
		}
		em.emitSecret(cmd.Player,
			Event{Kind: EvMysteryOffered, Player: cmd.Player, From: loc.ID,
				Choice: &PendingChoice{Kind: "mystery_card", Player: cmd.Player, Cards: cards}},
			// Pemain lain tahu ada yang sedang memilih, tapi bukan kartu apa
			// saja yang terlihat -- itulah nilai kemampuan Scholar.
			&Event{Kind: EvMysteryOffered, Player: cmd.Player, From: loc.ID,
				Choice: &PendingChoice{Kind: "mystery_card", Player: cmd.Player}})
		return nil
	}

	card := drawCard(em, DeckMystery, rng, "", "investigate")
	if card == "" {
		return ErrNoMysteryLeft
	}
	offerMysteryOptions(em, c, cmd.Player, EventCardID(card))
	return nil
}

func isScholar(c *Content, p *Player) bool {
	def := c.Character(p.Character)
	return def != nil && def.Ability.ID == "ancient_knowledge"
}

// offerMysteryOptions menyaring pilihan yang benar-benar bisa diambil lalu
// menahan permainan sampai pemain menjawab.
func offerMysteryOptions(em *emitter, c *Content, pid PlayerID, card EventCardID) {
	def := c.MysteryCard(card)
	if def == nil {
		return
	}
	p := em.s.Player(pid)

	var options []string
	for _, opt := range def.Options {
		if affordable(c, p, opt) {
			options = append(options, opt.ID)
		}
	}
	if len(options) == 0 {
		// Semua pilihan di luar jangkauan; kartunya dibuang tanpa efek alih-alih
		// mengunci permainan pada pilihan yang tidak bisa diambil.
		discardCard(em, DeckMystery, CardID(card))
		return
	}

	// Kartu Mystery terbuka begitu ditarik: seluruh meja melihat dilema yang
	// sedang dihadapi, hanya keputusannya yang belum diketahui.
	em.emit(Event{
		Kind:   EvMysteryOffered,
		Player: pid,
		Card:   card,
		Choice: &PendingChoice{Kind: "mystery_option", Player: pid, Card: card, Options: options},
	})
}

// affordable memeriksa apakah pemain sanggup membayar biaya sebuah pilihan.
func affordable(c *Content, p *Player, opt MysteryOptionDef) bool {
	if p == nil {
		return false
	}
	for _, e := range opt.Effects {
		if e.Op == "pay_resource" && e.Resource != nil {
			if p.Inventory[*e.Resource] < e.Amount {
				return false
			}
		}
	}
	return true
}

// --- Choose: menjawab pilihan yang tertunda (GDD 20) ---

func decideChoose(em *emitter, c *Content, rng *RNG, cmd Command) error {
	s := em.s
	pending := s.Pending
	if pending == nil {
		return ErrNoChoicePending
	}
	if pending.Player != cmd.Player {
		return ErrNotYourChoice
	}

	switch pending.Kind {
	case "mystery_card":
		// Scholar memilih salah satu dari dua kartu; sisanya dibuang.
		chosen := cmd.Card
		found := false
		for _, id := range pending.Cards {
			if id == chosen {
				found = true
			}
		}
		if !found {
			return ErrBadOption
		}
		for _, id := range pending.Cards {
			if id != chosen {
				discardCard(em, DeckMystery, CardID(id))
			}
		}
		em.emit(Event{Kind: EvChoiceCleared, Player: cmd.Player})
		offerMysteryOptions(em, c, cmd.Player, chosen)
		return nil

	case "mystery_option":
		def := c.MysteryCard(pending.Card)
		if def == nil {
			em.emit(Event{Kind: EvChoiceCleared, Player: cmd.Player})
			return nil
		}
		var opt *MysteryOptionDef
		for i := range def.Options {
			if def.Options[i].ID == cmd.Option && slicesContains(pending.Options, cmd.Option) {
				opt = &def.Options[i]
				break
			}
		}
		if opt == nil {
			return ErrBadOption
		}

		card := pending.Card
		em.emit(Event{Kind: EvMysteryResolved, Player: cmd.Player, Card: card,
			Option: opt.ID, Reason: opt.Text.Get("en")})

		// Lokasi ditandai supaya tidak diselidiki dua kali dalam ronde yang sama;
		// tandanya hilang saat ronde berikutnya dimulai.
		if p := s.Player(cmd.Player); p != nil {
			markInvestigated(em, p.At)
		}

		applyEffects(em, c, rng, opt.Effects, cmd.Player, "mystery: "+def.Name.Get("en"))
		discardCard(em, DeckMystery, CardID(card))
		return nil
	}

	return ErrBadOption
}

func markInvestigated(em *emitter, at LocationID) {
	loc := em.s.Board.Location(at)
	if loc == nil || loc.Investigated {
		return
	}
	em.emit(Event{Kind: EvInvestigated, From: at})
}

func slicesContains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// --- Trade (GDD 11, 28) ---

// decideTrade memindahkan resource ke pemain lain di lokasi yang sama.
//
// Sengaja SATU ARAH: pemain memberi, tidak menukar secara atomik. Pertukaran dua
// arah butuh persetujuan pihak kedua, dan itu memerlukan mekanisme tawar-menawar
// yang jauh lebih besar daripada nilainya di prototype ini. Memberi sudah cukup
// untuk menciptakan kerja sama yang diinginkan GDD 28, dan tetap bisa dibalas
// pada giliran pemain lain.
func decideTrade(em *emitter, c *Content, cmd Command) error {
	s := em.s
	p := s.Player(cmd.Player)

	target := s.Player(cmd.Target)
	if target == nil {
		return ErrUnknownPlayer
	}
	if target.ID == p.ID || target.At != p.At {
		return ErrTargetNotHere
	}
	give := cmd.Give
	if give.HasNegative() {
		return ErrBadCommand
	}
	if give.IsEmpty() {
		return ErrTradeEmpty
	}
	if give.Total() > c.Rules.TradeMaxPerAction {
		return ErrBadCommand
	}
	if !p.Inventory.Covers(give) {
		return ErrNotEnoughRes
	}
	room := playerCapacity(c, target) - target.Inventory.Total()
	if room < give.Total() {
		return ErrTargetFull
	}

	spendAP(em, cmd)
	em.emit(Event{Kind: EvTraded, Player: p.ID, Target: target.ID, Resources: give})
	return nil
}
