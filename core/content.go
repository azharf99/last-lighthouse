package core

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed content/*.json
var contentFS embed.FS

// Content adalah seluruh data game yang bisa dituning. Ia dimuat sekali saat
// init dari JSON yang di-embed, sehingga ikut masuk ke binary server maupun ke
// core.wasm tanpa I/O runtime (ADR-005).
type Content struct {
	Rules      RulesConfig
	Darkness   DarknessConfig
	Lighthouse []ComponentDef
	Locations  []LocationTypeDef
	Map        MapDef
	Characters []CharacterDef
	Objectives []ObjectiveDef
	Events     []EventCardDef
	Mysteries  []MysteryCardDef
	Artifacts  []ArtifactDef
	Monsters   MonsterConfig

	// Hash mengidentifikasi versi konten. Match menyimpannya (ADR-004) sehingga
	// match yang sedang berjalan tidak rusak saat konten di-tuning, dan client
	// dengan konten berbeda ditolak saat join (ADR-003).
	Hash string
}

type RulesConfig struct {
	ActionPointsPerTurn             int `json:"action_points_per_turn"`
	MaxHealth                       int `json:"max_health"`
	InventoryCapacity               int `json:"inventory_capacity"`
	ExhaustedInventoryCapacity      int `json:"exhausted_inventory_capacity"`
	GatherBaseAmount                int `json:"gather_base_amount"`
	RestHealAmount                  int `json:"rest_heal_amount"`
	LocationRegenPerRound           int `json:"location_regen_per_round"`
	LocationMaxAvailablePerResource int `json:"location_max_available_per_resource"`
	ExploreRevealResources          bool `json:"explore_reveal_resources"`
	TradeMaxPerAction               int  `json:"trade_max_per_action"`
	RescueVillageVP                 int  `json:"rescue_village_vp"`
	MonsterDefeatVP                 int  `json:"monster_defeat_vp"`
	ArtifactBaseVP                  int  `json:"artifact_base_vp"`

	// GuaranteeCrystalTile memastikan pulau selalu menyimpan sumber crystal.
	//
	// Tanpa ini, sekitar 1 dari 5 partai tidak bisa dimenangkan sejak kartu
	// dibagikan: kelima komponen mercusuar menuntut crystal (GDD 7) sementara
	// tidak ada satu pun lokasi AWAL yang menghasilkannya. Pemain tidak akan
	// pernah tahu penyebabnya -- mereka hanya kalah tanpa alasan yang terlihat,
	// yang persis jenis kegagalan yang diperingatkan GDD 38.
	GuaranteeCrystalTile bool `json:"guarantee_crystal_tile"`

	// ExtraComponentCostPerExtraPlayer menaikkan biaya mercusuar seiring
	// bertambahnya pemain.
	//
	// Tanpa penskalaan, jumlah pemain mengubah tingkat kesulitan secara drastis:
	// Darkness naik per RONDE sementara jumlah aksi bertambah linear dengan
	// jumlah pemain, sehingga 4 pemain punya sekitar 1,7x anggaran aksi
	// dibanding 2 pemain untuk tujuan yang persis sama. Terukur: menang 14%
	// (2 pemain) vs 68% (4 pemain) pada biaya yang sama.
	//
	// Yang diskalakan adalah BIAYA, bukan panjang Darkness track: GDD 7
	// menyebut biayanya "provisional and should be balanced during testing",
	// sedangkan GDD 22 menggambar track-nya sebagai 0-8 yang tetap.
	ExtraComponentCostPerExtraPlayer int `json:"extra_component_cost_per_extra_player"`

	// --- Arah desain eksperimental (BALANCE-M1.md) ---

	// ExploreVP memberi VP kepada pemain yang menjelajahi lokasi baru (Arah 1).
	// Nol berarti tidak ada VP eksplorasi (perilaku asli GDD 25).
	//
	// GDD 25 tidak mencantumkan eksplorasi sebagai sumber VP sama sekali,
	// sehingga Navigator tidak punya sumber skor dari keahliannya dan pilar
	// desain §2.1 tidak berbayar. Ini eksperimen untuk mengukur dampaknya.
	ExploreVP int `json:"explore_vp,omitempty"`

	// InvestigateAnywhere membolehkan Investigate di semua lokasi yang sudah
	// tereksplorasi (Arah 2). False berarti hanya lokasi bertag
	// can_investigate (perilaku asli).
	//
	// Saat ini Artifact hanya bisa diperoleh lewat Mystery, dan Mystery butuh
	// berdiri di satu dari sedikit lokasi yang tepat. Membuka semua lokasi
	// untuk investigate memutus hambatan ketersediaan itu.
	InvestigateAnywhere bool `json:"investigate_anywhere,omitempty"`
}

type DarknessConfig struct {
	Start        int                 `json:"start"`
	Max          int                 `json:"max"`
	RisePerRound int                 `json:"rise_per_round"`
	Thresholds   []DarknessThreshold `json:"thresholds"`
}

type DarknessThreshold struct {
	At     int    `json:"at"`
	Effect string `json:"effect"`
	Amount int    `json:"amount"`
	Note   string `json:"note"`
}

// amountFor mengembalikan besaran efek kalau ambangnya sudah tercapai.
func (d DarknessConfig) amountFor(effect string, darkness int) int {
	for _, t := range d.Thresholds {
		if t.Effect == effect && darkness >= t.At {
			return t.Amount
		}
	}
	return 0
}

type LocalizedText map[string]string

func (t LocalizedText) Get(lang string) string {
	if v, ok := t[lang]; ok {
		return v
	}
	return t["en"]
}

type ComponentDef struct {
	ID    ComponentID   `json:"id"`
	Order int           `json:"order"`
	VP    int           `json:"vp"`
	Cost  ResourceSet   `json:"cost"`
	Name  LocalizedText `json:"name"`
}

type LocationTypeDef struct {
	ID               LocationTypeID `json:"id"`
	Yields           ResourceSet    `json:"yields"`
	Name             LocalizedText  `json:"name"`
	CanRepair        bool           `json:"can_repair"`
	CanHeal          bool           `json:"can_heal"`
	CanInvestigate   bool           `json:"can_investigate"`
	DarknessOnGather int            `json:"darkness_on_gather"`
	Rescuable        bool           `json:"rescuable"`
	Tags             []string       `json:"tags"`
}

func (l *LocationTypeDef) HasTag(tag string) bool {
	for _, t := range l.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

type MapDef struct {
	StartLocation LocationID       `json:"start_location"`
	Locations     []MapLocationDef `json:"locations"`

	// TileStack adalah tile yang bisa muncul di slot belum tereksplorasi.
	// Diacak tiap match dan hanya sebagian terpakai, sehingga layout pulau
	// berbeda antar permainan (GDD 18, 31).
	TileStack []LocationTypeID `json:"tile_stack"`
}

type MapLocationDef struct {
	ID       LocationID     `json:"id"`
	Type     LocationTypeID `json:"type"`
	Explored bool           `json:"explored"`
	Adjacent []LocationID   `json:"adjacent"`
}

type CharacterDef struct {
	ID      CharacterID   `json:"id"`
	Role    string        `json:"role"`
	Name    LocalizedText `json:"name"`
	Ability struct {
		ID   string `json:"id"`
		Note string `json:"note"`
	} `json:"ability"`
}

type ObjectiveDef struct {
	ID          ObjectiveID   `json:"id"`
	VP          int           `json:"vp"`
	Name        LocalizedText `json:"name"`
	Requirement struct {
		Kind  string `json:"kind"`
		Count int    `json:"count"`
	} `json:"requirement"`
}

// EffectDef adalah satu efek dari kartu Event atau pilihan Mystery.
//
// Op berasal dari himpunan TERTUTUP yang diimplementasikan di effect.go. Ini
// batas yang dijaga ADR-005: JSON mendeskripsikan APA yang terjadi, Go
// mengimplementasikan BAGAIMANA. Menambah kartu = edit JSON; menambah jenis
// efek baru = tambah satu op di Go, dengan sadar.
type EffectDef struct {
	Op  string `json:"op"`
	Tag string `json:"tag,omitempty"`

	// Pointer, bukan nilai: Wood adalah nilai nol dari Resource, jadi field yang
	// tidak diisi akan terbaca sebagai "kayu" alih-alih "tidak ada resource".
	// Ini kesalahan yang sama dengan bug omitempty di Command (lihat wire_test.go).
	Resource *Resource `json:"resource,omitempty"`

	Amount int `json:"amount,omitempty"`
}

type EventCardDef struct {
	ID      EventCardID   `json:"id"`
	Name    LocalizedText `json:"name"`
	Text    LocalizedText `json:"text"`
	Effects []EffectDef   `json:"effects"`
}

type MysteryCardDef struct {
	ID      EventCardID          `json:"id"`
	Name    LocalizedText        `json:"name"`
	Text    LocalizedText        `json:"text"`
	Options []MysteryOptionDef   `json:"options"`
}

type MysteryOptionDef struct {
	ID      string        `json:"id"`
	Text    LocalizedText `json:"text"`
	Effects []EffectDef   `json:"effects"`
}

type ArtifactDef struct {
	ID   ArtifactID    `json:"id"`
	Name LocalizedText `json:"name"`
	Text LocalizedText `json:"text"`

	// Effect adalah kemampuan pasif; "none" berarti kartu itu murni VP.
	Effect string `json:"effect"`
	VP     int    `json:"vp"`
}

type MonsterConfig struct {
	Base struct {
		ID     string        `json:"id"`
		Name   LocalizedText `json:"name"`
		Damage int           `json:"damage"`
		VP     int           `json:"vp"`
	} `json:"base"`

	// Rentang hasil lemparan 1D6 (GDD 16). Ini angka pertama yang perlu
	// dituning kalau combat terasa tidak sepadan dengan risikonya (GDD 38).
	Combat struct {
		PlayerDamagedMax   int `json:"player_damaged_max"`
		StandoffMax        int `json:"standoff_max"`
		MonsterDefeatedMin int `json:"monster_defeated_min"`
	} `json:"combat"`

	SpawnOnExploreChance int              `json:"spawn_on_explore_chance"`
	SpawnLocationTypes   []LocationTypeID `json:"spawn_location_types"`
}

var defaultContent *Content

func init() {
	c, err := LoadContent()
	if err != nil {
		// Konten di-embed di binary, jadi kegagalan di sini berarti file JSON
		// rusak saat build -- bukan kondisi yang bisa dipulihkan saat runtime.
		panic("core: konten tertanam tidak valid: " + err.Error())
	}
	defaultContent = c
}

// DefaultContent mengembalikan konten yang di-embed di binary.
func DefaultContent() *Content { return defaultContent }

func LoadContent() (*Content, error) {
	c := &Content{}
	load := func(name string, dst any) error {
		b, err := contentFS.ReadFile("content/" + name)
		if err != nil {
			return fmt.Errorf("baca %s: %w", name, err)
		}
		if err := json.Unmarshal(b, dst); err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}
		return nil
	}

	for _, step := range []struct {
		name string
		dst  any
	}{
		{"rules.json", &c.Rules},
		{"darkness.json", &c.Darkness},
		{"lighthouse.json", &c.Lighthouse},
		{"location_types.json", &c.Locations},
		{"map_prototype.json", &c.Map},
		{"characters.json", &c.Characters},
		{"objectives.json", &c.Objectives},
		{"events.json", &c.Events},
		{"mysteries.json", &c.Mysteries},
		{"artifacts.json", &c.Artifacts},
		{"monsters.json", &c.Monsters},
	} {
		if err := load(step.name, step.dst); err != nil {
			return nil, err
		}
	}

	// Komponen mercusuar harus terurut menaik: GDD 7.1 mewajibkan perbaikan
	// berurutan, dan sisa kode mengandalkan Lighthouse[0] sebagai yang berikutnya.
	sort.SliceStable(c.Lighthouse, func(i, j int) bool {
		return c.Lighthouse[i].Order < c.Lighthouse[j].Order
	})

	h, err := hashContent()
	if err != nil {
		return nil, err
	}
	c.Hash = h

	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// hashContent membaca file dalam urutan nama sehingga hash-nya stabil.
func hashContent() (string, error) {
	entries, err := fs.Glob(contentFS, "content/*.json")
	if err != nil {
		return "", err
	}
	sort.Strings(entries)

	h := sha256.New()
	for _, name := range entries {
		b, err := contentFS.ReadFile(name)
		if err != nil {
			return "", err
		}
		h.Write([]byte(name))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:16], nil
}

func (c *Content) LocationType(id LocationTypeID) *LocationTypeDef {
	for i := range c.Locations {
		if c.Locations[i].ID == id {
			return &c.Locations[i]
		}
	}
	return nil
}

func (c *Content) Character(id CharacterID) *CharacterDef {
	for i := range c.Characters {
		if c.Characters[i].ID == id {
			return &c.Characters[i]
		}
	}
	return nil
}

func (c *Content) EventCard(id EventCardID) *EventCardDef {
	for i := range c.Events {
		if c.Events[i].ID == id {
			return &c.Events[i]
		}
	}
	return nil
}

func (c *Content) MysteryCard(id EventCardID) *MysteryCardDef {
	for i := range c.Mysteries {
		if c.Mysteries[i].ID == id {
			return &c.Mysteries[i]
		}
	}
	return nil
}

func (c *Content) Artifact(id ArtifactID) *ArtifactDef {
	for i := range c.Artifacts {
		if c.Artifacts[i].ID == id {
			return &c.Artifacts[i]
		}
	}
	return nil
}

func (c *Content) Objective(id ObjectiveID) *ObjectiveDef {
	for i := range c.Objectives {
		if c.Objectives[i].ID == id {
			return &c.Objectives[i]
		}
	}
	return nil
}

// Validate menangkap konten rusak saat build/test, bukan saat pemain sedang main.
// Kesalahan ketik di JSON adalah mode kegagalan yang diperkirakan dari ADR-005,
// jadi pemeriksaannya dijalankan di CI.
func (c *Content) Validate() error {
	if c.Rules.ActionPointsPerTurn <= 0 {
		return fmt.Errorf("rules: action_points_per_turn harus > 0")
	}
	if c.Rules.MaxHealth <= 0 {
		return fmt.Errorf("rules: max_health harus > 0")
	}
	if c.Rules.InventoryCapacity <= 0 {
		return fmt.Errorf("rules: inventory_capacity harus > 0")
	}
	if c.Darkness.Max <= c.Darkness.Start {
		return fmt.Errorf("darkness: max (%d) harus > start (%d)", c.Darkness.Max, c.Darkness.Start)
	}
	if len(c.Lighthouse) == 0 {
		return fmt.Errorf("lighthouse: tidak ada komponen")
	}

	seenOrder := map[int]bool{}
	for _, comp := range c.Lighthouse {
		if comp.Cost.IsEmpty() {
			return fmt.Errorf("lighthouse %q: cost kosong", comp.ID)
		}
		if comp.Cost.HasNegative() {
			return fmt.Errorf("lighthouse %q: cost negatif", comp.ID)
		}
		if seenOrder[comp.Order] {
			return fmt.Errorf("lighthouse %q: order %d duplikat", comp.ID, comp.Order)
		}
		seenOrder[comp.Order] = true
	}

	if len(c.Map.Locations) == 0 {
		return fmt.Errorf("map: tidak ada lokasi")
	}
	known := map[LocationID]bool{}
	unexplored := 0
	for _, l := range c.Map.Locations {
		if known[l.ID] {
			return fmt.Errorf("map: lokasi %q duplikat", l.ID)
		}
		known[l.ID] = true

		// Slot yang belum tereksplorasi memang belum punya tipe -- tipenya baru
		// ditentukan saat aksi Explore menarik tile (GDD 18).
		if !l.Explored {
			if l.Type != "" {
				return fmt.Errorf("map: lokasi %q belum tereksplorasi tapi sudah punya tipe %q",
					l.ID, l.Type)
			}
			unexplored++
			continue
		}
		if c.LocationType(l.Type) == nil {
			return fmt.Errorf("map: lokasi %q memakai tipe tidak dikenal %q", l.ID, l.Type)
		}
	}

	// Tumpukan tile harus cukup untuk semua slot, kalau tidak ada lokasi yang
	// tidak akan pernah bisa dieksplorasi dan objective "jelajahi N lokasi"
	// (GDD 24) bisa jadi mustahil dipenuhi.
	if len(c.Map.TileStack) < unexplored {
		return fmt.Errorf("map: %d slot belum tereksplorasi tapi tile_stack hanya %d",
			unexplored, len(c.Map.TileStack))
	}
	for _, tid := range c.Map.TileStack {
		if c.LocationType(tid) == nil {
			return fmt.Errorf("map: tile_stack memuat tipe tidak dikenal %q", tid)
		}
	}
	if !known[c.Map.StartLocation] {
		return fmt.Errorf("map: start_location %q tidak ada", c.Map.StartLocation)
	}
	// Adjacency harus dua arah, kalau tidak Move jadi satu arah tanpa disengaja.
	for _, l := range c.Map.Locations {
		for _, adj := range l.Adjacent {
			if !known[adj] {
				return fmt.Errorf("map: %q bersebelahan dengan %q yang tidak ada", l.ID, adj)
			}
			if !c.Map.adjacentBothWays(l.ID, adj) {
				return fmt.Errorf("map: adjacency %q -> %q tidak timbal balik", l.ID, adj)
			}
		}
	}

	if len(c.Objectives) < 4 {
		return fmt.Errorf("objectives: butuh minimal 4 untuk 4 pemain, ada %d", len(c.Objectives))
	}
	if len(c.Characters) < 4 {
		return fmt.Errorf("characters: butuh minimal 4, ada %d", len(c.Characters))
	}
	if len(c.Events) == 0 {
		return fmt.Errorf("events: deck kosong; fase Event tidak punya kartu")
	}
	if len(c.Mysteries) == 0 {
		return fmt.Errorf("mysteries: deck kosong")
	}
	if len(c.Artifacts) == 0 {
		return fmt.Errorf("artifacts: deck kosong")
	}

	// Setiap op harus dikenali. Kesalahan ketik di JSON tidak boleh lolos ke
	// runtime dan diam-diam jadi efek yang tidak terjadi (ADR-005).
	for _, ev := range c.Events {
		for _, e := range ev.Effects {
			if err := validateEffect(ev.ID, e); err != nil {
				return err
			}
		}
	}
	for _, my := range c.Mysteries {
		if len(my.Options) < 2 {
			return fmt.Errorf("mystery %q: butuh minimal 2 pilihan agar ada keputusan", my.ID)
		}
		seen := map[string]bool{}
		for _, opt := range my.Options {
			if opt.ID == "" || seen[opt.ID] {
				return fmt.Errorf("mystery %q: id pilihan kosong atau duplikat (%q)", my.ID, opt.ID)
			}
			seen[opt.ID] = true
			for _, e := range opt.Effects {
				if err := validateEffect(my.ID, e); err != nil {
					return err
				}
			}
		}
	}
	for _, ar := range c.Artifacts {
		if !knownArtifactEffects[ar.Effect] {
			return fmt.Errorf("artifact %q: effect tidak dikenal %q", ar.ID, ar.Effect)
		}
	}

	mc := c.Monsters.Combat
	if mc.PlayerDamagedMax < 1 || mc.StandoffMax < mc.PlayerDamagedMax ||
		mc.MonsterDefeatedMin <= mc.StandoffMax || mc.MonsterDefeatedMin > 6 {
		return fmt.Errorf("monsters: rentang combat tidak masuk akal untuk 1D6: %+v", mc)
	}

	return nil
}

func validateEffect(owner EventCardID, e EffectDef) error {
	spec, found := knownEffectOps[e.Op]
	if !found {
		return fmt.Errorf("%q: op tidak dikenal %q", owner, e.Op)
	}
	if spec.needsResource && e.Resource == nil {
		return fmt.Errorf("%q: op %q butuh field resource", owner, e.Op)
	}
	if spec.needsAmount && e.Amount <= 0 {
		return fmt.Errorf("%q: op %q butuh amount > 0", owner, e.Op)
	}
	if spec.needsTag && e.Tag == "" {
		return fmt.Errorf("%q: op %q butuh field tag", owner, e.Op)
	}
	return nil
}

func (m MapDef) adjacentBothWays(a, b LocationID) bool {
	for _, l := range m.Locations {
		if l.ID != b {
			continue
		}
		for _, adj := range l.Adjacent {
			if adj == a {
				return true
			}
		}
	}
	return false
}
