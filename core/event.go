package core

// EventKind adalah fakta yang sudah terjadi. Event bersifat past tense dan tidak
// pernah gagal saat di-Apply -- seluruh validasi sudah terjadi di Decide.
type EventKind string

const (
	EvMatchStarted   EventKind = "match_started"
	EvObjectiveDealt EventKind = "objective_dealt" // RAHASIA - lihat Private
	EvTurnStarted    EventKind = "turn_started"
	EvTurnEnded      EventKind = "turn_ended"
	EvAPSpent        EventKind = "ap_spent"
	EvMoved          EventKind = "moved"
	EvResourceGained EventKind = "resource_gained"
	EvResourceSpent  EventKind = "resource_spent"
	EvLocationRegen  EventKind = "location_regen"
	EvRepaired       EventKind = "repaired"           // setoran parsial
	EvComponentDone  EventKind = "component_repaired" // komponen selesai
	EvVPAwarded      EventKind = "vp_awarded"
	EvHealed         EventKind = "healed"
	EvDamaged        EventKind = "damaged"
	EvExhausted      EventKind = "exhausted"
	EvPhaseChanged   EventKind = "phase_changed"
	EvRoundStarted   EventKind = "round_started"
	EvDarknessRose   EventKind = "darkness_rose"
	EvGameWon        EventKind = "game_won"
	EvGameLost       EventKind = "game_lost"

	// --- M1 ---
	EvDeckShuffled     EventKind = "deck_shuffled"
	EvDeckReshuffled   EventKind = "deck_reshuffled"
	EvCardDrawn        EventKind = "card_drawn"
	EvEventResolved    EventKind = "event_resolved"
	EvLocationRevealed EventKind = "location_revealed"
	EvGatherBlocked    EventKind = "gather_blocked"
	EvMonsterSpawned   EventKind = "monster_spawned"
	EvMonsterMoved     EventKind = "monster_moved"
	EvMonsterAttacked  EventKind = "monster_attacked"
	EvMonsterDefeated  EventKind = "monster_defeated"
	EvDiceRolled       EventKind = "dice_rolled"
	EvArtifactGained   EventKind = "artifact_gained"
	EvMysteryOffered   EventKind = "mystery_offered"
	EvMysteryResolved  EventKind = "mystery_resolved"
	EvChoiceCleared    EventKind = "choice_cleared"
	EvTraded           EventKind = "traded"
	EvVillageRescued   EventKind = "village_rescued"
	EvInvestigated     EventKind = "location_investigated"
	EvAbilityUsed      EventKind = "ability_used"
)

// DeckKind menamai deck mana yang disentuh sebuah event.
type DeckKind string

const (
	DeckEvent    DeckKind = "event"
	DeckMystery  DeckKind = "mystery"
	DeckArtifact DeckKind = "artifact"
)

// Event adalah satu perubahan atomik pada State.
//
// Sengaja struct datar, bukan interface: event ini di-serialisasi ke event log
// Postgres (ADR-004) dan ke JSON di WebSocket (ADR-003), dan ia harus bisa
// direplay bertahun-tahun setelah ditulis. Struct datar bertag jauh lebih
// mudah di-versi daripada hierarki tipe.
type Event struct {
	Kind EventKind `json:"kind"`
	V    int       `json:"v"` // versi skema event; dinaikkan kalau arti field berubah

	Player    PlayerID    `json:"player,omitempty"`
	From      LocationID  `json:"from,omitempty"`
	To        LocationID  `json:"to,omitempty"`
	Resources ResourceSet `json:"resources,omitempty"`
	Component ComponentID `json:"component,omitempty"`
	Objective ObjectiveID `json:"objective,omitempty"`
	Amount    int         `json:"amount,omitempty"`
	Value     int         `json:"value,omitempty"` // nilai absolut setelah perubahan (mis. Darkness)
	Phase     Phase       `json:"phase,omitempty"`
	Round     int         `json:"round,omitempty"`
	Reason    string      `json:"reason,omitempty"` // untuk log yang terbaca manusia

	// --- M1 ---
	Deck     DeckKind       `json:"deck,omitempty"`
	Card     EventCardID    `json:"card,omitempty"`
	Cards    []EventCardID  `json:"cards,omitempty"`
	Artifact ArtifactID     `json:"artifact,omitempty"`
	Option   string         `json:"option,omitempty"`
	Tile     LocationTypeID `json:"tile,omitempty"`
	Target   PlayerID       `json:"target,omitempty"`
	Choice   *PendingChoice `json:"choice,omitempty"`

	// Setup hanya diisi pada EvMatchStarted. Ia membawa seluruh struktur awal
	// match (papan, mercusuar, pemain, urutan giliran) supaya event log benar-
	// benar swasembada: state bisa direkonstruksi dari nol tanpa perlu tahu
	// konfigurasi awalnya. Tanpa ini, replay hanya bekerja kalau digabung
	// dengan snapshot, dan janji "kirim match ID-nya, saya replay persis
	// kejadiannya" di ADR-004 tidak berlaku.
	Setup *MatchSetup `json:"setup,omitempty"`

	// Private, kalau tidak kosong, berarti HANYA pemain itu yang boleh melihat
	// event ini. Pemain lain menerima PublicVariant sebagai gantinya (atau tidak
	// menerima apa-apa kalau nil). Lihat ProjectEvent di project.go dan ADR-006.
	Private PlayerID `json:"-"`

	// PublicVariant adalah versi tersunting yang dilihat pemain lain. Contoh:
	// "p2 menerima sebuah objective" tanpa menyebut objective apa.
	PublicVariant *Event `json:"-"`
}

// secret menandai event hanya untuk satu pemain, dengan varian publik opsional
// yang dilihat pemain lain.
func secret(owner PlayerID, e Event, publicVariant *Event) Event {
	e.V = eventSchemaVersion
	e.Private = owner
	if publicVariant != nil {
		pv := *publicVariant
		pv.V = eventSchemaVersion
		e.PublicVariant = &pv
	}
	return e
}

// MatchSetup adalah muatan EvMatchStarted: kondisi awal match.
//
// Objective pemain SENGAJA tidak ada di sini. EvMatchStarted bersifat publik,
// sementara objective dibagikan lewat event rahasia terpisah setelahnya (GDD 24,
// ADR-006).
type MatchSetup struct {
	MatchID     string      `json:"matchId"`
	ContentHash string      `json:"contentHash"`
	TurnOrder   []PlayerID  `json:"turnOrder"`
	Players     []Player    `json:"players"`
	Board       Board       `json:"board"`
	Lighthouse  []Component `json:"lighthouse"`
	Darkness    int         `json:"darkness"`
}

// eventSchemaVersion dinaikkan saat arti field yang ada berubah, supaya replay
// event lama tetap bisa diinterpretasikan dengan benar (ADR-004).
const eventSchemaVersion = 1
