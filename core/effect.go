package core

// Himpunan efek yang bisa dipakai kartu Event dan pilihan Mystery.
//
// Ini batas yang dijaga ADR-005: JSON mendeskripsikan APA yang terjadi, Go
// mengimplementasikan BAGAIMANA. Menambah kartu baru cukup mengedit JSON;
// menambah JENIS efek baru harus lewat sini, dengan sadar.
//
// Himpunan ini sengaja dijaga tetap kecil. ADR-005 menetapkan ambang: kalau
// jumlah op melewati sekitar 40 dan masih terus bertambah, artinya kita sedang
// membangun bahasa skrip yang lebih buruk daripada Lua, dan opsi runtime skrip
// harus ditinjau ulang secara jujur.

type effectSpec struct {
	needsResource bool
	needsAmount   bool
	needsTag      bool
}

var knownEffectOps = map[string]effectSpec{
	// Global
	"darkness":       {needsAmount: true},
	"spawn_monster":  {needsAmount: true},
	"heal_all":       {needsAmount: true},
	"damage_all":     {needsAmount: true},
	"grant_all":      {needsResource: true, needsAmount: true},
	"block_gather":   {needsTag: true},
	"damage_at_tag":  {needsTag: true, needsAmount: true},
	"restock":        {needsTag: true, needsAmount: true},

	// Menyasar pemain yang memicu efeknya (pilihan Mystery)
	"gain_resource": {needsResource: true, needsAmount: true},
	"pay_resource":  {needsResource: true, needsAmount: true},
	"gain_artifact": {},
	"gain_vp":       {needsAmount: true},
	"damage":        {needsAmount: true},
	"reveal_tile":   {},
}

// knownArtifactEffects adalah kemampuan pasif artifact (GDD 21).
// "none" berarti kartunya murni bernilai VP.
var knownArtifactEffects = map[string]bool{
	"none":                       true,
	"free_move_per_turn":         true,
	"safe_at_lighthouse":         true,
	"repair_discount_per_round":  true,
	"gather_bonus":               true,
	"combat_bonus":               true,
	"capacity_bonus":             true,
}

// applyEffects menjalankan sederet efek.
//
// actor boleh kosong untuk efek global (kartu Event); efek yang menyasar pemain
// dilewati kalau tidak ada actor, alih-alih memilih korban secara sembarangan.
func applyEffects(em *emitter, c *Content, rng *RNG, effects []EffectDef, actor PlayerID, reason string) {
	for _, e := range effects {
		applyEffect(em, c, rng, e, actor, reason)
		if em.s.Over() {
			return
		}
	}
}

func applyEffect(em *emitter, c *Content, rng *RNG, e EffectDef, actor PlayerID, reason string) {
	s := em.s

	switch e.Op {
	case "darkness":
		raiseDarkness(em, c, e.Amount, reason)
		checkEnd(em, c)

	case "spawn_monster":
		for range e.Amount {
			spawnMonster(em, c, rng, reason)
		}

	case "heal_all":
		for i := range s.Players {
			p := &s.Players[i]
			if p.Health >= c.Rules.MaxHealth {
				continue
			}
			heal := min(e.Amount, c.Rules.MaxHealth-p.Health)
			em.emit(Event{Kind: EvHealed, Player: p.ID, Amount: heal, Reason: reason})
		}

	case "damage_all":
		for i := range s.Players {
			damagePlayer(em, c, s.Players[i].ID, e.Amount, reason)
		}

	case "grant_all":
		var rs ResourceSet
		rs[*e.Resource] = e.Amount
		for i := range s.Players {
			grantResources(em, c, s.Players[i].ID, rs, reason)
		}

	case "block_gather":
		// Berlaku sampai akhir ronde; dibersihkan saat ronde berikutnya dimulai.
		for i := range s.Board.Locations {
			loc := &s.Board.Locations[i]
			if locationHasTag(c, loc, e.Tag) && !loc.GatherBlocked {
				em.emit(Event{Kind: EvGatherBlocked, From: loc.ID, Value: 1, Reason: reason})
			}
		}

	case "damage_at_tag":
		for i := range s.Players {
			p := &s.Players[i]
			loc := s.Board.Location(p.At)
			if loc != nil && locationHasTag(c, loc, e.Tag) {
				damagePlayer(em, c, p.ID, e.Amount, reason)
			}
		}

	case "restock":
		for i := range s.Board.Locations {
			loc := &s.Board.Locations[i]
			if !locationHasTag(c, loc, e.Tag) {
				continue
			}
			lt := c.LocationType(loc.Type)
			if lt == nil || lt.Yields.IsEmpty() {
				continue
			}
			var add ResourceSet
			maxPer := c.Rules.LocationMaxAvailablePerResource
			for r := range ResourceCount {
				if lt.Yields[r] == 0 || loc.Available[r] >= maxPer {
					continue
				}
				add[r] = min(e.Amount, maxPer-loc.Available[r])
			}
			if !add.IsEmpty() {
				em.emit(Event{Kind: EvLocationRegen, From: loc.ID, Resources: add, Reason: reason})
			}
		}

	// --- efek yang menyasar pemain pemicu ---

	case "gain_resource":
		if actor == "" {
			return
		}
		var rs ResourceSet
		rs[*e.Resource] = e.Amount
		grantResources(em, c, actor, rs, reason)

	case "pay_resource":
		if actor == "" {
			return
		}
		p := s.Player(actor)
		if p == nil {
			return
		}
		var rs ResourceSet
		// Bayar sebanyak yang mampu. Pilihan Mystery sudah divalidasi mampu bayar
		// sebelum ditawarkan, jadi ini hanya jaring pengaman.
		rs[*e.Resource] = min(e.Amount, p.Inventory[*e.Resource])
		if !rs.IsEmpty() {
			em.emit(Event{Kind: EvResourceSpent, Player: actor, Resources: rs, Reason: reason})
		}

	case "gain_artifact":
		if actor != "" {
			drawArtifact(em, c, rng, actor, reason)
		}

	case "gain_vp":
		if actor != "" {
			em.emit(Event{Kind: EvVPAwarded, Player: actor, Amount: e.Amount, Reason: reason})
		}

	case "damage":
		if actor != "" {
			damagePlayer(em, c, actor, e.Amount, reason)
		}

	case "reveal_tile":
		// Mengungkap satu lokasi belum tereksplorasi secara gratis, tanpa biaya AP.
		if actor != "" {
			for i := range s.Board.Locations {
				if !s.Board.Locations[i].Explored {
					revealLocation(em, c, rng, s.Board.Locations[i].ID, actor, reason)
					return
				}
			}
		}
	}
}

func locationHasTag(c *Content, loc *Location, tag string) bool {
	lt := c.LocationType(loc.Type)
	return lt != nil && lt.HasTag(tag)
}

// grantResources memberi resource ke pemain, dipotong kapasitas inventory.
//
// Pemotongan terjadi di sini, bukan di pemanggil, supaya setiap sumber resource
// (gather, kartu, mystery, trade) menghormati batas GDD 9.1 tanpa harus
// mengingatnya masing-masing.
func grantResources(em *emitter, c *Content, pid PlayerID, rs ResourceSet, reason string) {
	p := em.s.Player(pid)
	if p == nil {
		return
	}
	room := playerCapacity(c, p) - p.Inventory.Total()
	if room <= 0 {
		return
	}

	var give ResourceSet
	for r := range ResourceCount {
		if rs[r] <= 0 || room <= 0 {
			continue
		}
		n := min(rs[r], room)
		give[r] = n
		room -= n
	}
	if give.IsEmpty() {
		return
	}
	em.emit(Event{Kind: EvResourceGained, Player: pid, Resources: give, Reason: reason})
}

// playerCapacity menghitung kapasitas inventory efektif (GDD 9.1, 17, 21).
func playerCapacity(c *Content, p *Player) int {
	if p.Exhausted {
		return c.Rules.ExhaustedInventoryCapacity
	}
	cap := c.Rules.InventoryCapacity
	if hasArtifactEffect(c, p, "capacity_bonus") {
		cap += 2
	}
	return cap
}

func hasArtifactEffect(c *Content, p *Player, effect string) bool {
	for _, id := range p.Artifacts {
		if def := c.Artifact(id); def != nil && def.Effect == effect {
			return true
		}
	}
	return false
}

// damagePlayer menerapkan luka dan menandai Exhausted saat health mencapai nol.
//
// GDD 17 tegas: tidak ada pemain yang tersingkir permanen. Exhausted membatasi,
// bukan menghapus -- itu yang membedakan game ini dari eliminasi pemain.
func damagePlayer(em *emitter, c *Content, pid PlayerID, amount int, reason string) {
	p := em.s.Player(pid)
	if p == nil || amount <= 0 {
		return
	}
	actual := min(amount, p.Health)
	if actual > 0 {
		em.emit(Event{Kind: EvDamaged, Player: pid, Amount: actual, Reason: reason})
	}
	if p.Health <= 0 && !p.Exhausted {
		em.emit(Event{Kind: EvExhausted, Player: pid, Value: 1, Reason: reason})
		// Kapasitas turun jadi 3 saat Exhausted; kelebihan resource dibuang.
		dropExcess(em, c, pid)
	}
}

func dropExcess(em *emitter, c *Content, pid PlayerID) {
	p := em.s.Player(pid)
	if p == nil {
		return
	}
	excess := p.Inventory.Total() - playerCapacity(c, p)
	if excess <= 0 {
		return
	}
	var drop ResourceSet
	// Buang dari resource paling melimpah lebih dulu, dengan urutan enum tetap
	// supaya deterministik.
	for excess > 0 {
		best, bestN := -1, 0
		for r := range ResourceCount {
			if have := p.Inventory[r] - drop[r]; have > bestN {
				best, bestN = r, have
			}
		}
		if best < 0 {
			break
		}
		drop[best]++
		excess--
	}
	if !drop.IsEmpty() {
		em.emit(Event{Kind: EvResourceSpent, Player: pid, Resources: drop,
			Reason: "kapasitas turun karena kelelahan"})
	}
}
