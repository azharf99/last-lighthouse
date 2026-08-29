package core

// Mekanik M1: eksplorasi, monster, combat, dan penyelamatan desa.

// --- Eksplorasi (GDD 18) ---

// revealLocation membuka satu slot "?" dengan menarik tile dari tumpukan.
//
// Tumpukan diacak tiap match dan hanya sebagian terpakai, sehingga bentuk pulau
// berbeda antar permainan (GDD 31). Kalau tumpukan habis, slot dibuka sebagai
// lokasi kosong alih-alih menggantung selamanya.
func revealLocation(em *emitter, c *Content, rng *RNG, id LocationID, by PlayerID, reason string) {
	s := em.s
	loc := s.Board.Location(id)
	if loc == nil || loc.Explored {
		return
	}

	tile := LocationTypeID("")
	if len(s.TileStack) > 0 {
		tile = s.TileStack[0]
	}

	lt := c.LocationType(tile)
	var stock ResourceSet
	name := "Unknown"
	if lt != nil {
		name = lt.Name.Get("en")
		if c.Rules.ExploreRevealResources {
			stock = lt.Yields
		}
	}

	em.emit(Event{
		Kind:      EvLocationRevealed,
		Player:    by,
		From:      id,
		Tile:      tile,
		Resources: stock,
		Reason:    name,
	})

	// Arah 1 (BALANCE-M1.md): VP per eksplorasi. Nol berarti perilaku asli.
	if c.Rules.ExploreVP > 0 && by != "" {
		em.emit(Event{Kind: EvVPAwarded, Player: by, Amount: c.Rules.ExploreVP,
			Reason: "eksplorasi lokasi baru"})
	}

	// Lokasi berbahaya bisa langsung memunculkan monster (GDD 18).
	if lt != nil && monsterCanSpawnAt(c, lt.ID) && c.Monsters.SpawnOnExploreChance > 0 {
		if rng.D6() <= c.Monsters.SpawnOnExploreChance {
			em.emit(Event{Kind: EvMonsterSpawned, From: id, Amount: 1,
				Reason: "terbangun saat " + name + " dibuka"})
		}
	}
}

func monsterCanSpawnAt(c *Content, t LocationTypeID) bool {
	for _, id := range c.Monsters.SpawnLocationTypes {
		if id == t {
			return true
		}
	}
	return false
}

// --- Monster (GDD 15) ---

// spawnMonster menempatkan monster di lokasi berbahaya yang sudah tereksplorasi.
//
// Kalau tidak ada lokasi yang cocok, tidak terjadi apa-apa: memaksa monster
// muncul di lokasi aman akan membuat efek kartu terasa sewenang-wenang.
func spawnMonster(em *emitter, c *Content, rng *RNG, reason string) {
	s := em.s
	var candidates []LocationID
	for i := range s.Board.Locations {
		loc := &s.Board.Locations[i]
		if loc.Explored && monsterCanSpawnAt(c, loc.Type) {
			candidates = append(candidates, loc.ID)
		}
	}
	if len(candidates) == 0 {
		return
	}
	at := candidates[rng.Intn(len(candidates))]
	em.emit(Event{Kind: EvMonsterSpawned, From: at, Amount: 1, Reason: reason})
}

// runMonsterPhase menjalankan seluruh monster (GDD 15).
//
// Aturannya: kalau ada pemain di lokasi yang sama, serang. Kalau tidak, bergerak
// satu langkah (dua saat Darkness >= 2) menuju pemain terdekat. Kalau tidak ada
// pemain sama sekali, bergerak menuju Mercusuar.
//
// Ini yang mencegah pemain mengabaikan monster begitu saja, yang oleh GDD 38
// disebut sebagai risiko nyata: kalau menghindar selalu lebih baik daripada
// bertarung, aksi Fight tidak akan pernah dipilih.
func runMonsterPhase(em *emitter, c *Content, rng *RNG) {
	s := em.s
	moveBudget := 1 + c.Darkness.amountFor("monster_move_bonus", s.Darkness)

	// Salin daftar lokasi bermonster lebih dulu: memutasi papan sambil
	// mengiterasinya akan membuat monster yang baru pindah ikut bergerak lagi.
	type mob struct {
		at    LocationID
		count int
	}
	var mobs []mob
	for i := range s.Board.Locations {
		if n := s.Board.Locations[i].Monsters; n > 0 {
			mobs = append(mobs, mob{s.Board.Locations[i].ID, n})
		}
	}

	for _, m := range mobs {
		for range m.count {
			resolveOneMonster(em, c, m.at, moveBudget)
			if s.Over() {
				return
			}
		}
	}
	_ = rng
}

func resolveOneMonster(em *emitter, c *Content, at LocationID, moveBudget int) {
	s := em.s

	// Serang pemain di lokasi yang sama.
	if victims := playersAt(s, at); len(victims) > 0 {
		for _, pid := range victims {
			if immuneToMonsters(c, s, pid) {
				continue
			}
			em.emit(Event{Kind: EvMonsterAttacked, Player: pid, From: at})
			damagePlayer(em, c, pid, c.Monsters.Base.Damage, "diserang monster")
			break // satu monster menyerang satu pemain
		}
		return
	}

	// Bergerak menuju pemain terdekat, atau menuju Mercusuar kalau tidak ada.
	target := nearestPlayerLocation(s, at)
	if target == "" {
		target = lighthouseLocation(c, s)
	}
	if target == "" || target == at {
		return
	}

	cur := at
	for range moveBudget {
		next := stepToward(s, cur, target)
		if next == "" || next == cur {
			break
		}
		em.emit(Event{Kind: EvMonsterMoved, From: cur, To: next})
		cur = next
		if len(playersAt(s, cur)) > 0 {
			break // berhenti begitu bertemu pemain; serangan menyusul ronde depan
		}
	}
}

// immuneToMonsters mengecek artifact Lentera Eter (GDD 21).
func immuneToMonsters(c *Content, s *State, pid PlayerID) bool {
	p := s.Player(pid)
	if p == nil || !hasArtifactEffect(c, p, "safe_at_lighthouse") {
		return false
	}
	loc := s.Board.Location(p.At)
	if loc == nil {
		return false
	}
	lt := c.LocationType(loc.Type)
	return lt != nil && lt.CanRepair // mercusuar adalah satu-satunya lokasi repair
}

func playersAt(s *State, id LocationID) []PlayerID {
	var out []PlayerID
	for i := range s.Players {
		if s.Players[i].At == id {
			out = append(out, s.Players[i].ID)
		}
	}
	return out
}

func lighthouseLocation(c *Content, s *State) LocationID {
	for i := range s.Board.Locations {
		lt := c.LocationType(s.Board.Locations[i].Type)
		if lt != nil && lt.CanRepair {
			return s.Board.Locations[i].ID
		}
	}
	return ""
}

// DistancesFrom mengembalikan jarak langkah dari sebuah lokasi ke setiap lokasi
// lain yang terjangkau.
//
// Diekspor karena bot membutuhkannya: tanpa jarak, bot hanya bisa mengenali
// tujuan yang PERSIS bersebelahan, dan akibatnya ia tidak pernah membawa pulang
// resource yang sudah dikumpulkannya. Itu bukan sekadar bot yang bermain buruk
// -- ia membuat hasil simulator mengukur ketidakmampuan bot, bukan keseimbangan
// game.
//
// Hanya melewati lokasi yang sudah tereksplorasi: baik monster maupun pemain
// tidak bisa menembus bagian pulau yang belum tersingkap.
func (b *Board) DistancesFrom(from LocationID) map[LocationID]int {
	return bfsBoard(b, from)
}

func bfs(s *State, from LocationID) map[LocationID]int {
	return bfsBoard(&s.Board, from)
}

func bfsBoard(board *Board, from LocationID) map[LocationID]int {
	dist := map[LocationID]int{from: 0}
	queue := []LocationID{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		loc := board.Location(cur)
		if loc == nil {
			continue
		}
		for _, adj := range loc.Adjacent {
			if _, seen := dist[adj]; seen {
				continue
			}
			next := board.Location(adj)
			if next == nil || !next.Explored {
				continue
			}
			dist[adj] = dist[cur] + 1
			queue = append(queue, adj)
		}
	}
	return dist
}

// nearestPlayerLocation memilih lokasi pemain terdekat.
//
// Seri diputus lewat urutan s.Players (urutan giliran), bukan lewat iterasi map,
// supaya hasilnya deterministik -- iterasi map di Go acak dan akan merusak replay.
func nearestPlayerLocation(s *State, from LocationID) LocationID {
	dist := bfs(s, from)
	best, bestD := LocationID(""), 1<<30
	for i := range s.Players {
		p := &s.Players[i]
		if d, ok := dist[p.At]; ok && d < bestD {
			best, bestD = p.At, d
		}
	}
	return best
}

// stepToward mengembalikan tetangga pertama yang mendekatkan ke tujuan.
func stepToward(s *State, from, to LocationID) LocationID {
	if from == to {
		return from
	}
	distFromTarget := bfs(s, to)
	loc := s.Board.Location(from)
	if loc == nil {
		return ""
	}
	cur, ok := distFromTarget[from]
	if !ok {
		return ""
	}
	// Tetangga diperiksa dalam urutan slice Adjacent (stabil dari konten),
	// sehingga seri selalu diputus dengan cara yang sama.
	for _, adj := range loc.Adjacent {
		next := s.Board.Location(adj)
		if next == nil || !next.Explored {
			continue
		}
		if d, ok := distFromTarget[adj]; ok && d < cur {
			return adj
		}
	}
	return ""
}

// --- Combat (GDD 16) ---

// resolveCombat menjalankan satu lemparan 1D6 melawan monster.
//
//	1-2 : pemain terluka
//	3-4 : keduanya selamat
//	5-6 : monster kalah
//
// Modifier berasal dari kemampuan Hunter (GDD 10.3) dan artifact Tameng Besi.
// Rentangnya sendiri ada di konten, karena inilah angka pertama yang perlu
// dituning kalau bertarung terasa tidak pernah sepadan (GDD 38).
func resolveCombat(em *emitter, c *Content, rng *RNG, pid PlayerID, at LocationID) {
	s := em.s
	p := s.Player(pid)
	if p == nil {
		return
	}

	roll := rng.D6()
	modifier := combatModifier(c, p)
	total := min(roll+modifier, 6)

	em.emit(Event{Kind: EvDiceRolled, Player: pid, Amount: roll, Value: total,
		Reason: "combat"})

	switch {
	case total >= c.Monsters.Combat.MonsterDefeatedMin:
		em.emit(Event{Kind: EvMonsterDefeated, Player: pid, From: at})
		em.emit(Event{Kind: EvVPAwarded, Player: pid, Amount: c.Rules.MonsterDefeatVP,
			Reason: "mengalahkan monster"})
	case total <= c.Monsters.Combat.PlayerDamagedMax:
		damagePlayer(em, c, pid, c.Monsters.Base.Damage, "kalah dalam pertarungan")
	default:
		// Seri: keduanya bertahan. Ini yang membuat Fight tetap berisiko tanpa
		// selalu menghukum.
	}
}

func combatModifier(c *Content, p *Player) int {
	mod := 0
	if def := c.Character(p.Character); def != nil && def.Ability.ID == "predator" {
		mod++
	}
	if hasArtifactEffect(c, p, "combat_bonus") {
		mod++
	}
	return mod
}

// --- Penyelamatan desa (GDD 24, 25) ---

// tryRescueVillage memberi VP saat pemain membersihkan monster dari desa.
//
// GDD menyebut "rescue 2 villages" sebagai objective (24) dan memberi 2 VP per
// desa (25), tetapi tidak menyatakan apa yang membuat sebuah desa "terselamatkan".
// Mengalahkan monster yang mengancamnya adalah tafsir yang paling langsung
// terhubung ke mekanik yang sudah ada. Ini keputusan desain terbuka.
func tryRescueVillage(em *emitter, c *Content, at LocationID, by PlayerID) {
	loc := em.s.Board.Location(at)
	if loc == nil || loc.Rescued || loc.Monsters > 0 {
		return
	}
	lt := c.LocationType(loc.Type)
	if lt == nil || !lt.Rescuable {
		return
	}
	em.emit(Event{Kind: EvVillageRescued, Player: by, From: at})
	em.emit(Event{Kind: EvVPAwarded, Player: by, Amount: c.Rules.RescueVillageVP,
		Reason: "desa diselamatkan"})
}
