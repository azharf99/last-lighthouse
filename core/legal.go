package core

// LegalCommands mengembalikan aksi yang tampak legal bagi pemain, untuk
// menyalakan/mematikan tombol di UI.
//
// Ia beroperasi pada PlayerView, bukan State, karena client online memang tidak
// pernah punya state penuh. Konsekuensinya hasilnya bisa TIDAK AKURAT: view
// tidak punya informasi rahasia, jadi ada aksi yang tampak legal tapi ditolak
// server.
//
// Karena itu fungsi ini sengaja KONSERVATIF -- kalau ragu, tampilkan sebagai
// legal dan biarkan server yang menolak. Kebalikannya jauh lebih buruk: tombol
// yang mati padahal aksinya sah membuat pemain mengira game-nya rusak. UI
// menampilkan penolakan server sebagai toast, bukan error (ADR-006).
//
// Server SELALU memvalidasi ulang lewat Decide. Fungsi ini murni kosmetik.
//
// TestLegalCommandsAreAllAccepted menegakkan arah sebaliknya: apa pun yang
// ditawarkan di sini HARUS diterima Decide pada state yang sama. Kalau tidak,
// UI menyalakan tombol yang lalu ditolak, dan pemain menyimpulkan game-nya rusak.
func LegalCommands(v *PlayerView, c *Content) []Command {
	if c == nil {
		c = DefaultContent()
	}
	s := &v.State
	if s.Over() || s.Phase != PhasePlayer {
		return nil
	}
	p := s.Player(v.Viewer)
	if p == nil {
		return nil
	}

	// Pilihan tertunda mendahului segalanya: selama belum dijawab, tidak ada
	// aksi lain yang diterima (GDD 20).
	if s.Pending != nil {
		if s.Pending.Player != v.Viewer {
			return nil
		}
		return pendingCommands(s.Pending, v.Viewer)
	}

	active := s.ActivePlayer()
	if active == nil || active.ID != v.Viewer {
		return nil
	}

	// EndTurn selalu tersedia di giliran sendiri, bahkan tanpa AP.
	out := []Command{{Kind: CmdEndTurn, Player: p.ID}}

	loc := s.Board.Location(p.At)
	if loc == nil {
		return out
	}

	// Dua kemampuan tetap bisa dipakai walau AP habis: Kompas Kuno (GDD 21)
	// untuk berpindah, dan Pathfinder milik Navigator (GDD 10.1) untuk menjelajah.
	freeMove := !p.FreeMoveUsed && hasArtifactEffect(c, p, "free_move_per_turn")
	freeExplore := !p.AbilityUsedTurn && repairOrExploreAbility(c, p, "pathfinder")
	if p.AP < 1 && !freeMove && !freeExplore {
		return out
	}

	for _, adj := range loc.Adjacent {
		dest := s.Board.Location(adj)
		if dest == nil {
			continue
		}
		if p.AP < 1 && !(freeMove && dest.Explored) {
			continue
		}
		out = append(out, Command{Kind: CmdMove, Player: p.ID, To: adj})
	}

	// Explore: slot "?" yang bersebelahan (GDD 18). Navigator boleh melakukannya
	// walau AP-nya sudah habis, sekali per giliran.
	if p.AP >= 1 || freeExplore {
		for _, adj := range loc.Adjacent {
			if dest := s.Board.Location(adj); dest != nil && !dest.Explored {
				out = append(out, Command{Kind: CmdExplore, Player: p.ID, To: adj})
			}
		}
	}

	if p.AP < 1 {
		return out
	}

	// Gather
	if !loc.GatherBlocked && p.Inventory.Total() < playerCapacity(c, p) {
		for r := range ResourceCount {
			if loc.Available[r] > 0 {
				out = append(out, Command{Kind: CmdGather, Player: p.ID, Resource: Resource(r)})
			}
		}
	}

	// Repair
	if lt := c.LocationType(loc.Type); lt != nil && lt.CanRepair {
		if next := s.NextComponent(); next != nil {
			need := next.Progress.Missing(next.Cost)
			var pay ResourceSet
			for r := range ResourceCount {
				pay[r] = min(need[r], p.Inventory[r])
			}
			// Diskon Insinyur / Perkakas Terlupakan membuat aksi ini tetap
			// berguna walau pemain tidak membawa resource apa pun.
			discount := !p.RepairDiscountUsed && repairDiscountAvailable(c, p)
			if !pay.IsEmpty() || discount {
				out = append(out, Command{Kind: CmdRepair, Player: p.ID,
					Component: next.ID, Pay: pay})
			}
		}
	}

	// Fight (GDD 16). Pemain kelelahan tidak bisa bertarung (GDD 17).
	if loc.Monsters > 0 && !p.Exhausted {
		out = append(out, Command{Kind: CmdFight, Player: p.ID})
	}

	// Investigate (GDD 20)
	// Arah 2 (BALANCE-M1.md): InvestigateAnywhere membuka semua lokasi tereksplorasi.
	canInvestigate := false
	if lt := c.LocationType(loc.Type); lt != nil && lt.CanInvestigate {
		canInvestigate = true
	}
	if c.Rules.InvestigateAnywhere && loc.Explored {
		canInvestigate = true
	}
	if canInvestigate && !loc.Investigated {
		// Jumlah kartu tersisa terlihat di view (hanya identitasnya yang
		// tertutup), jadi prediksi ini akurat.
		if s.MysteryDeck.Len() > 0 || len(s.MysteryDeck.Discard) > 0 {
			out = append(out, Command{Kind: CmdInvestigate, Player: p.ID})
		}
	}

	// Rest
	if loc.Monsters == 0 && p.Health < c.Rules.MaxHealth {
		out = append(out, Command{Kind: CmdRest, Player: p.ID})
	}

	// Trade (GDD 11, 28): memberi satu resource ke pemain lain di lokasi sama.
	//
	// Hanya satu unit per resource yang ditawarkan. Menawarkan setiap kombinasi
	// jumlah akan meledakkan daftar aksi tanpa menambah keputusan yang berarti;
	// UI bisa menyediakan pengatur jumlah dan mengirim Command sendiri.
	for i := range s.Players {
		other := &s.Players[i]
		if other.ID == p.ID || other.At != p.At {
			continue
		}
		if other.Inventory.Total() >= playerCapacity(c, other) {
			continue
		}
		for r := range ResourceCount {
			if p.Inventory[r] <= 0 {
				continue
			}
			var give ResourceSet
			give[r] = 1
			out = append(out, Command{Kind: CmdTrade, Player: p.ID, Target: other.ID, Give: give})
		}
	}

	return out
}

// repairOrExploreAbility memeriksa kemampuan karakter berdasarkan id-nya.
func repairOrExploreAbility(c *Content, p *Player, abilityID string) bool {
	def := c.Character(p.Character)
	return def != nil && def.Ability.ID == abilityID
}

func repairDiscountAvailable(c *Content, p *Player) bool {
	if def := c.Character(p.Character); def != nil && def.Ability.ID == "efficient_repair" {
		return true
	}
	return hasArtifactEffect(c, p, "repair_discount_per_round")
}

// pendingCommands mengembalikan jawaban yang tersedia untuk pilihan tertunda.
func pendingCommands(pc *PendingChoice, viewer PlayerID) []Command {
	var out []Command
	switch pc.Kind {
	case "mystery_option":
		for _, id := range pc.Options {
			out = append(out, Command{Kind: CmdChoose, Player: viewer, Option: id})
		}
	case "mystery_card":
		for _, card := range pc.Cards {
			out = append(out, Command{Kind: CmdChoose, Player: viewer, Card: card})
		}
	}
	return out
}
