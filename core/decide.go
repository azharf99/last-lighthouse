package core

// Decide memvalidasi satu command dan menghasilkan event yang mendeskripsikan
// akibatnya. Ini satu-satunya tempat validasi dan keacakan terjadi.
//
// Decide bekerja di atas SALINAN state, bukan state aslinya. Pemanggil (Match
// Actor di server, atau facade session di client offline) yang menerapkan event
// yang dikembalikan ke state sungguhan lewat Apply. Ini menjaga Apply tetap satu-
// satunya jalur mutasi: kalau Decide memutasi state secara langsung, akan ada dua
// implementasi perubahan state yang harus sepakat, dan cepat atau lambat keduanya
// menyimpang.
//
// Biaya clone per command tidak signifikan: state satu match hanya beberapa KB
// dan command datang sekitar 10 kali per menit.
//
// Error yang dikembalikan adalah penolakan yang wajar, bukan bug -- lihat errors.go.
func Decide(s *State, cmd Command, c *Content, rng *RNG) ([]Event, error) {
	if c == nil {
		c = DefaultContent()
	}
	if s.Over() {
		return nil, ErrMatchOver
	}
	if cmd.Pay.HasNegative() {
		// Command datang dari mesin pemain; jangan pernah percaya isinya.
		return nil, ErrBadCommand
	}

	p := s.Player(cmd.Player)
	if p == nil {
		return nil, ErrUnknownPlayer
	}
	if s.Phase != PhasePlayer {
		return nil, ErrNotPlayerPhase
	}

	// Pilihan yang tertunda menahan seluruh permainan (GDD 20). Selama ada,
	// hanya jawabannya yang diterima -- kalau tidak, pemain bisa mengabaikan
	// dilema yang seharusnya wajib dijawab.
	if s.Pending != nil {
		if cmd.Kind != CmdChoose {
			return nil, ErrChoicePending
		}
		scratch := s.Clone()
		em := newEmitter(scratch)
		if err := decideChoose(em, c, rng, cmd); err != nil {
			return nil, err
		}
		autoEndTurn(em, c, rng, cmd.Player)
		return em.out, nil
	}
	if cmd.Kind == CmdChoose {
		return nil, ErrNoChoicePending
	}
	active := s.ActivePlayer()
	if active == nil || active.ID != cmd.Player {
		return nil, ErrNotYourTurn
	}
	if cost := cmd.APCost(); p.AP < cost {
		return nil, ErrNoAP
	}

	// Mulai dari sini kita bekerja di scratch.
	scratch := s.Clone()
	em := newEmitter(scratch)

	var err error
	switch cmd.Kind {
	case CmdMove:
		err = decideMove(em, c, cmd)
	case CmdGather:
		err = decideGather(em, c, cmd)
	case CmdRepair:
		err = decideRepair(em, c, cmd)
	case CmdRest:
		err = decideRest(em, c, cmd)
	case CmdEndTurn:
		endTurn(em, c, rng)
		return em.out, nil
	case CmdExplore:
		err = decideExplore(em, c, rng, cmd)
	case CmdFight:
		err = decideFight(em, c, rng, cmd)
	case CmdInvestigate:
		err = decideInvestigate(em, c, rng, cmd)
	case CmdTrade:
		err = decideTrade(em, c, cmd)
	default:
		return nil, ErrBadCommand
	}
	if err != nil {
		return nil, err
	}

	autoEndTurn(em, c, rng, cmd.Player)
	return em.out, nil
}

// autoEndTurn menutup giliran begitu AP habis.
//
// Tanpa ini setiap pemain harus menekan "akhiri giliran" pada giliran yang sudah
// jelas selesai. Giliran TIDAK ditutup selama masih ada pilihan tertunda --
// pemain harus menjawabnya lebih dulu (GDD 20).
func autoEndTurn(em *emitter, c *Content, rng *RNG, pid PlayerID) {
	s := em.s
	if s.Over() || s.Pending != nil {
		return
	}
	if cur := s.ActivePlayer(); cur != nil && cur.ID == pid && cur.AP <= 0 {
		endTurn(em, c, rng)
	}
}

func spendAP(em *emitter, cmd Command) {
	em.emit(Event{Kind: EvAPSpent, Player: cmd.Player, Amount: cmd.APCost(),
		Reason: string(cmd.Kind)})
}

// --- Move (GDD 11) ---

func decideMove(em *emitter, c *Content, cmd Command) error { //nolint:unparam
	s := em.s
	p := s.Player(cmd.Player)

	dest := s.Board.Location(cmd.To)
	if dest == nil {
		return ErrUnknownLocation
	}
	if !s.Board.Adjacent(p.At, cmd.To) {
		return ErrNotAdjacent
	}
	// M1: lokasi belum tereksplorasi butuh aksi Explore lebih dulu (GDD 18),
	// dan kemampuan Pathfinder Navigator memotong biayanya (GDD 10.1).

	// Perpindahan gratis: artifact Kompas Kuno (GDD 21), sekali per giliran ke
	// lokasi yang sudah tereksplorasi.
	free := dest.Explored && !p.FreeMoveUsed && hasArtifactEffect(c, p, "free_move_per_turn")

	from := p.At
	if free {
		em.emit(Event{Kind: EvAbilityUsed, Player: cmd.Player, Reason: "free_move"})
	} else {
		spendAP(em, cmd)
	}
	em.emit(Event{Kind: EvMoved, Player: cmd.Player, From: from, To: cmd.To})
	return nil
}

// --- Gather (GDD 11, 19) ---

func decideGather(em *emitter, c *Content, cmd Command) error {
	s := em.s
	p := s.Player(cmd.Player)

	if int(cmd.Resource) >= ResourceCount {
		return ErrBadCommand
	}

	loc := s.Board.Location(p.At)
	if loc == nil {
		return ErrUnknownLocation
	}
	if loc.GatherBlocked {
		return ErrGatherBlocked
	}
	if loc.Available[cmd.Resource] <= 0 {
		return ErrNothingToGather
	}

	// playerCapacity, bukan c.Rules.InventoryCapacity: kapasitas efektif juga
	// dipengaruhi status Exhausted (GDD 17) dan artifact Ransel Peziarah (GDD 21).
	// Memakai angka dasar di sini membuat Decide lebih ketat daripada
	// LegalCommands, sehingga UI menawarkan aksi yang lalu ditolak server.
	room := playerCapacity(c, p) - p.Inventory.Total()
	if room <= 0 {
		return ErrInventoryFull
	}

	amount := c.Rules.GatherBaseAmount
	// GDD 23, ambang 4: gathering jadi kurang efisien. Lantainya 1, bukan 0 --
	// gathering yang menghasilkan nol akan membuat game tidak bisa dimenangkan
	// alih-alih sekadar lebih sulit.
	if pen := c.Darkness.amountFor("gather_penalty", s.Darkness); pen > 0 {
		amount -= pen
		if amount < 1 {
			amount = 1
		}
	}
	if hasArtifactEffect(c, p, "gather_bonus") {
		amount++
	}
	amount = min(amount, loc.Available[cmd.Resource], room)

	var got ResourceSet
	got[cmd.Resource] = amount

	spendAP(em, cmd)
	em.emit(Event{Kind: EvResourceGained, Player: cmd.Player, From: loc.ID, Resources: got})

	// GDD 19: menambang Crystal menaikkan Darkness. Ini contoh GDD 38 -- kenaikan
	// Darkness diikat ke aksi yang terlihat, supaya pemain paham penyebabnya.
	if lt := c.LocationType(loc.Type); lt != nil && lt.DarknessOnGather > 0 && cmd.Resource == Crystal {
		raiseDarkness(em, c, lt.DarknessOnGather, "menambang kristal di "+loc.Name)
		if checkEnd(em, c) {
			return nil
		}
	}
	return nil
}

// --- Repair (GDD 7, 11) ---

func decideRepair(em *emitter, c *Content, cmd Command) error {
	s := em.s
	p := s.Player(cmd.Player)

	loc := s.Board.Location(p.At)
	if loc == nil {
		return ErrUnknownLocation
	}
	if lt := c.LocationType(loc.Type); lt == nil || !lt.CanRepair {
		return ErrNotAtLighthouse
	}

	next := s.NextComponent()
	if next == nil {
		return ErrComponentDone
	}
	// GDD 7.1: komponen harus diperbaiki berurutan. Kalau client menyebut
	// komponen lain, tolak alih-alih diam-diam mengalihkan ke yang benar.
	if cmd.Component != "" && cmd.Component != next.ID {
		if target := s.Component(cmd.Component); target == nil {
			return ErrBadCommand
		} else if target.Repaired {
			return ErrComponentDone
		}
		return ErrWrongComponent
	}

	if !p.Inventory.Covers(cmd.Pay) {
		return ErrNotEnoughRes
	}

	// Diskon perbaikan: kemampuan Insinyur (GDD 10.2) dan artifact Perkakas
	// Terlupakan (GDD 21). Keduanya sekali per ronde dan berbagi satu bendera,
	// supaya menumpuk keduanya tidak memberi dua diskon di ronde yang sama.
	discountAvailable := false
	if !p.RepairDiscountUsed {
		if def := c.Character(p.Character); def != nil && def.Ability.ID == "efficient_repair" {
			discountAvailable = true
		} else if hasArtifactEffect(c, p, "repair_discount_per_round") {
			discountAvailable = true
		}
	}

	// Buang kelebihan setoran: menyetor 3 Wood ke komponen yang hanya butuh 1
	// akan membuang 2 Wood tanpa manfaat. Potong ke yang benar-benar dibutuhkan.
	need := next.Progress.Missing(next.Cost)
	pay := cmd.Pay
	for r := range ResourceCount {
		pay[r] = min(pay[r], need[r])
	}
	if pay.IsEmpty() && !discountAvailable {
		return ErrUselessPayment
	}

	spendAP(em, cmd)

	// Diskon dipakai LEBIH DULU, lalu kebutuhan dihitung ulang dan setoran
	// dipotong ulang. Urutan ini penting: kalau setoran dipotong dulu, pemain
	// yang membayar penuh akan tetap menghabiskan jatah diskonnya tanpa manfaat
	// -- dan itu membuat aksi yang ditawarkan UI ditolak server.
	if discountAvailable && need.Total() > 0 {
		best := -1
		for r := range ResourceCount {
			if need[r] > 0 && (best < 0 || need[r] > need[best]) {
				best = r
			}
		}
		if best >= 0 {
			var free ResourceSet
			free[best] = 1
			em.emit(Event{Kind: EvAbilityUsed, Player: cmd.Player, Reason: "round"})
			em.emit(Event{Kind: EvRepaired, Player: cmd.Player, Component: next.ID,
				Resources: free, Reason: "diskon perbaikan"})

			need = s.Component(next.ID).Progress.Missing(s.Component(next.ID).Cost)
			for r := range ResourceCount {
				pay[r] = min(pay[r], need[r])
			}
		}
	}

	if !pay.IsEmpty() {
		em.emit(Event{Kind: EvResourceSpent, Player: cmd.Player, Resources: pay,
			Reason: "repair " + string(next.ID)})
		em.emit(Event{Kind: EvRepaired, Player: cmd.Player, Component: next.ID, Resources: pay})
	}

	if comp := s.Component(next.ID); comp != nil && comp.Complete() {
		em.emit(Event{Kind: EvComponentDone, Component: comp.ID})
		distributeComponentVP(em, comp.ID)
		checkEnd(em, c)
	}
	return nil
}

// distributeComponentVP membagi VP satu komponen ke para penyetornya secara
// proporsional (GDD 7, 25).
//
// CATATAN DESAIN: GDD memberi nilai VP pada komponen (3-7) dan contoh skor di
// GDD 26 menunjukkan tiap pemain punya total "Repair" sendiri, tetapi tidak
// menyatakan bagaimana VP satu komponen dibagi antar penyumbang. Pembagian
// proporsional dengan metode sisa terbesar dipilih di sini karena ia memberi
// imbalan sebanding dengan kontribusi dan tidak menghukum kerja sama. Ini
// keputusan desain terbuka -- lihat ringkasan M0.
func distributeComponentVP(em *emitter, id ComponentID) {
	comp := em.s.Component(id)
	if comp == nil || len(comp.Contributions) == 0 {
		return
	}

	total := 0
	for _, con := range comp.Contributions {
		total += con.Amount
	}
	if total <= 0 {
		return
	}

	// Bagian bulat dulu, lalu sisanya ke kontribusi terbesar. Iterasi memakai
	// slice Contributions yang urutannya deterministik (urutan penyetoran
	// pertama), sehingga seri selalu diputus dengan cara yang sama.
	awarded := make([]int, len(comp.Contributions))
	given := 0
	for i, con := range comp.Contributions {
		share := comp.VP * con.Amount / total
		awarded[i] = share
		given += share
	}

	for given < comp.VP {
		best, bestRem := -1, -1
		for i, con := range comp.Contributions {
			rem := (comp.VP * con.Amount) % total
			if rem > bestRem || (rem == bestRem && best >= 0 && con.Amount > comp.Contributions[best].Amount) {
				best, bestRem = i, rem
			}
		}
		if best < 0 {
			break
		}
		awarded[best]++
		given++
	}

	for i, con := range comp.Contributions {
		if awarded[i] <= 0 {
			continue
		}
		em.emit(Event{Kind: EvVPAwarded, Player: con.Player, Amount: awarded[i],
			Component: comp.ID, Reason: "kontribusi perbaikan"})
	}
}

// --- Rest (GDD 11, 17) ---

func decideRest(em *emitter, c *Content, cmd Command) error {
	s := em.s
	p := s.Player(cmd.Player)

	loc := s.Board.Location(p.At)
	if loc == nil {
		return ErrUnknownLocation
	}
	if loc.Monsters > 0 {
		return ErrMonsterPresent
	}
	if p.Health >= c.Rules.MaxHealth {
		return ErrHealthFull
	}

	heal := min(c.Rules.RestHealAmount, c.Rules.MaxHealth-p.Health)

	spendAP(em, cmd)
	em.emit(Event{Kind: EvHealed, Player: cmd.Player, Amount: heal})

	// GDD 17: Exhausted berakhir begitu health di atas nol. Apply sudah
	// menghapus flag-nya; event ini membuat perubahan itu eksplisit di log.
	if p.Exhausted && p.Health+heal > 0 {
		em.emit(Event{Kind: EvExhausted, Player: cmd.Player, Value: 0})
	}
	return nil
}
