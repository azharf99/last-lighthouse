package core

// State adalah state otoritatif satu match. Ia HANYA ada di server (mode online)
// atau di device (mode offline). Client online tidak pernah menerima tipe ini —
// ia menerima PlayerView. Lihat project.go dan ADR-006.
type State struct {
	MatchID     string `json:"matchId"`
	ContentHash string `json:"contentHash"` // ADR-005: match dikunci ke versi konten
	RNGState    int64  `json:"rngState"`    // di-snapshot agar restart bisa lanjut persis

	Status Status `json:"status"`
	Round  int    `json:"round"`
	Phase  Phase  `json:"phase"`

	// TurnOrder adalah slice, bukan map: urutan giliran signifikan dan harus
	// stabil di seluruh replay.
	TurnOrder  []PlayerID `json:"turnOrder"`
	ActiveIdx  int        `json:"activeIdx"`  // indeks ke TurnOrder
	FirstIdx   int        `json:"firstIdx"`   // token First Player, bergeser tiap ronde (GDD 27)
	TurnsTaken int        `json:"turnsTaken"` // giliran yang sudah selesai di ronde ini

	Players    []Player    `json:"players"`
	Board      Board       `json:"board"`
	Darkness   int         `json:"darkness"`   // GDD 22, kalah di DarknessMax
	Lighthouse []Component `json:"lighthouse"` // urut menaik menurut Order (GDD 7.1)

	// Deck. Urutan kartu yang belum ditarik bersifat RAHASIA dari semua pemain
	// (GDD 13, 20, 21) -- mengetahuinya menghapus seluruh ketegangan dan
	// memungkinkan permainan optimal sempurna. Project menggantinya dengan
	// kartu tertutup sejumlah yang sama (ADR-006).
	EventDeck    Deck `json:"eventDeck"`
	MysteryDeck  Deck `json:"mysteryDeck"`
	ArtifactDeck Deck `json:"artifactDeck"`

	// TileStack adalah tile yang belum diletakkan (GDD 18). Juga rahasia:
	// pemain tidak boleh tahu apa yang menunggu di balik tanda tanya.
	TileStack []LocationTypeID `json:"tileStack"`

	// Pending menahan permainan sampai satu pemain menjawab sebuah pilihan
	// (GDD 20). Selama tidak nil, hanya command CmdChoose yang diterima.
	Pending *PendingChoice `json:"pending,omitempty"`
}

// CardID adalah identitas kartu di dalam deck, terlepas dari jenis decknya.
// Satu tipe untuk semua deck membuat mesin deck cukup ditulis sekali.
type CardID string

// Deck adalah tumpukan tarik plus buangannya.
//
// Keduanya slice, bukan map, karena urutan justru INTI dari sebuah deck: kalau
// urutannya tidak deterministik, replay dan simulator kehilangan maknanya.
type Deck struct {
	Draw    []CardID `json:"draw"`
	Discard []CardID `json:"discard"`
}

func (d *Deck) Len() int { return len(d.Draw) }

// PendingChoice adalah keputusan yang sedang ditunggu dari seorang pemain.
//
// Mystery butuh dua langkah -- pemain harus MELIHAT kartunya dulu sebelum
// memutuskan (GDD 20: "pemain harus memutuskan apakah imbalannya sepadan").
// Menggabungkannya jadi satu command akan menghapus justru bagian yang membuat
// mekanik ini menarik.
type PendingChoice struct {
	Kind   string   `json:"kind"` // "mystery_option" | "mystery_card"
	Player PlayerID `json:"player"`

	// Card adalah kartu yang sedang diresolusikan (kind mystery_option).
	Card EventCardID `json:"card,omitempty"`

	// Cards adalah pilihan kartu untuk kemampuan Scholar (GDD 10.4):
	// tarik 2 Mystery, pilih 1.
	Cards []EventCardID `json:"cards,omitempty"`

	// Options adalah id pilihan yang boleh diambil. Pilihan yang tidak
	// terjangkau (mis. tidak mampu membayar) sudah disaring lebih dulu.
	Options []string `json:"options,omitempty"`
}

// Player adalah state satu pemain, termasuk field rahasia.
//
// PERINGATAN: menambah field ke sini berarti memutuskan visibilitasnya di
// project.go. Field baru tidak otomatis muncul di PlayerView — itu disengaja,
// supaya rahasia tidak bisa bocor karena lupa (ADR-006).
type Player struct {
	ID        PlayerID    `json:"id"`
	Name      string      `json:"name"`
	Character CharacterID `json:"character"`
	At        LocationID  `json:"at"`
	Health    int         `json:"health"`
	AP        int         `json:"ap"`
	Inventory ResourceSet `json:"inventory"`
	VP        int          `json:"vp"`
	Exhausted bool         `json:"exhausted"` // GDD 17: 0 Health, tidak pernah tersingkir
	Artifacts []ArtifactID `json:"artifacts"`

	// Penghitung untuk personal objective (GDD 24). Disimpan di state, bukan
	// dihitung ulang dari event log, supaya scoring tidak perlu memutar ulang
	// seluruh riwayat pertandingan.
	Explored        int `json:"explored"`
	MonstersSlain   int `json:"monstersSlain"`
	VillagesRescued int `json:"villagesRescued"`
	RepairsJoined   int `json:"repairsJoined"`
	ResourcesGiven  int `json:"resourcesGiven"`
	WasExhausted    bool `json:"wasExhausted"`

	// Objective RAHASIA dari pemain lain (GDD 24). Bernilai 7-8 VP, dan contoh
	// skor akhir di GDD 26 hanya berselisih 1 VP -- mengetahui objective lawan
	// setara memenangkan pertandingan.
	Objective ObjectiveID `json:"objective"`

	// Turn-scoped: di-reset di awal giliran pemain.
	ActedThisTurn bool `json:"actedThisTurn"`
	FreeMoveUsed  bool `json:"freeMoveUsed"`  // artifact Kompas Kuno
	AbilityUsedTurn bool `json:"abilityUsedTurn"` // kemampuan Navigator

	// Round-scoped: di-reset di awal ronde.
	RepairDiscountUsed bool `json:"repairDiscountUsed"` // Insinyur + Perkakas Terlupakan
}

func (p *Player) InventoryFull(capacity int) bool { return p.Inventory.Total() >= capacity }

// Board menyimpan lokasi sebagai slice supaya urutannya deterministik.
// Lookup lewat Location(); jumlah lokasi kecil (belasan), jadi pencarian linear
// lebih murah daripada menjaga map tetap sinkron.
type Board struct {
	Locations []Location `json:"locations"`
}

func (b *Board) Location(id LocationID) *Location {
	for i := range b.Locations {
		if b.Locations[i].ID == id {
			return &b.Locations[i]
		}
	}
	return nil
}

func (b *Board) Adjacent(from, to LocationID) bool {
	loc := b.Location(from)
	if loc == nil {
		return false
	}
	for _, id := range loc.Adjacent {
		if id == to {
			return true
		}
	}
	return false
}

// Location adalah satu petak di pulau (GDD 18, 19).
type Location struct {
	ID       LocationID     `json:"id"`
	Type     LocationTypeID `json:"type"`
	Name     string         `json:"name"`
	Explored bool           `json:"explored"`
	Adjacent []LocationID   `json:"adjacent"`

	// Available adalah resource yang masih bisa di-Gather di sini. Berkurang
	// saat diambil; sebagian dipulihkan tiap ronde (lihat phase.go).
	Available ResourceSet `json:"available"`

	// Monsters adalah jumlah monster di lokasi ini (GDD 15).
	Monsters int `json:"monsters"`

	// GatherBlocked disetel kartu Event (mis. Badai Besar) dan dibersihkan di
	// awal ronde berikutnya.
	GatherBlocked bool `json:"gatherBlocked"`

	// Rescued menandai desa yang sudah diselamatkan (objective GDD 24).
	Rescued bool `json:"rescued"`

	// Investigated menandai lokasi yang sudah diselidiki RONDE INI; direset di
	// awal ronde berikutnya.
	//
	// Versi pertama menandainya permanen, dan itu menutup mekaniknya hampir
	// seluruhnya: peta dasar hanya punya satu lokasi yang bisa diselidiki, jadi
	// deck 30 kartu Mystery (GDD 8.3) tereduksi menjadi paling banyak satu atau
	// dua tarikan per partai. Terukur: 0,40 investigate per partai, dan bahkan
	// permainan ACAK hanya mencapai 0,65 -- artinya aksinya memang jarang
	// tersedia, bukan sekadar kurang menarik.
	//
	// GDD 19 juga menyebut Temple sebagai "Mystery-heavy location", yang
	// mengandaikan pemakaian berulang.
	Investigated bool `json:"investigated"`
}

// Component adalah satu dari lima bagian mercusuar (GDD 7).
type Component struct {
	ID       ComponentID `json:"id"`
	Name     string      `json:"name"`
	Order    int         `json:"order"` // GDD 7.1: harus diperbaiki berurutan
	Cost     ResourceSet `json:"cost"`
	VP       int         `json:"vp"`
	Repaired bool        `json:"repaired"`

	// Progress adalah resource yang sudah disetor sejauh ini.
	Progress ResourceSet `json:"progress"`

	// Contributions mencatat siapa menyetor berapa, dipakai untuk membagi VP
	// saat komponen selesai. Slice (bukan map) demi pembagian VP yang
	// deterministik saat terjadi seri.
	Contributions []Contribution `json:"contributions"`
}

type Contribution struct {
	Player PlayerID `json:"player"`
	Amount int      `json:"amount"` // jumlah resource yang disetor
}

func (c *Component) Complete() bool { return c.Progress.Covers(c.Cost) }

func (c *Component) contribute(p PlayerID, n int) {
	for i := range c.Contributions {
		if c.Contributions[i].Player == p {
			c.Contributions[i].Amount += n
			return
		}
	}
	c.Contributions = append(c.Contributions, Contribution{Player: p, Amount: n})
}

// --- helper akses ---

func (s *State) Player(id PlayerID) *Player {
	for i := range s.Players {
		if s.Players[i].ID == id {
			return &s.Players[i]
		}
	}
	return nil
}

func (s *State) ActivePlayer() *Player {
	if s.ActiveIdx < 0 || s.ActiveIdx >= len(s.TurnOrder) {
		return nil
	}
	return s.Player(s.TurnOrder[s.ActiveIdx])
}

// NextComponent mengembalikan komponen mercusuar yang harus diperbaiki
// berikutnya, atau nil kalau kelimanya sudah selesai (GDD 7.1).
func (s *State) NextComponent() *Component {
	for i := range s.Lighthouse {
		if !s.Lighthouse[i].Repaired {
			return &s.Lighthouse[i]
		}
	}
	return nil
}

func (s *State) Component(id ComponentID) *Component {
	for i := range s.Lighthouse {
		if s.Lighthouse[i].ID == id {
			return &s.Lighthouse[i]
		}
	}
	return nil
}

func (s *State) Over() bool { return s.Status == StatusWon || s.Status == StatusLost }

// Clone menghasilkan salinan dalam (deep copy).
//
// Apply memutasi State di tempat demi menghindari alokasi per event dan bug
// aliasing slice, jadi pemanggil yang butuh state "sebelum" harus Clone dulu.
// Server memakai ini untuk snapshot; simulator memakainya untuk percabangan.
func (s *State) Clone() *State {
	out := *s

	out.TurnOrder = append([]PlayerID(nil), s.TurnOrder...)

	out.Players = make([]Player, len(s.Players))
	for i, p := range s.Players {
		p.Artifacts = append([]ArtifactID(nil), p.Artifacts...)
		out.Players[i] = p
	}

	out.Board.Locations = make([]Location, len(s.Board.Locations))
	for i, l := range s.Board.Locations {
		l.Adjacent = append([]LocationID(nil), l.Adjacent...)
		out.Board.Locations[i] = l
	}

	out.Lighthouse = make([]Component, len(s.Lighthouse))
	for i, c := range s.Lighthouse {
		c.Contributions = append([]Contribution(nil), c.Contributions...)
		out.Lighthouse[i] = c
	}

	out.EventDeck = s.EventDeck.clone()
	out.MysteryDeck = s.MysteryDeck.clone()
	out.ArtifactDeck = s.ArtifactDeck.clone()
	out.TileStack = append([]LocationTypeID(nil), s.TileStack...)

	if s.Pending != nil {
		pc := *s.Pending
		pc.Cards = append([]EventCardID(nil), s.Pending.Cards...)
		pc.Options = append([]string(nil), s.Pending.Options...)
		out.Pending = &pc
	}

	return &out
}

func (d Deck) clone() Deck {
	return Deck{
		Draw:    append([]CardID(nil), d.Draw...),
		Discard: append([]CardID(nil), d.Discard...),
	}
}
