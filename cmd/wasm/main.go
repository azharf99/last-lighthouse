//go:build js && wasm

// Command wasm mengekspos rules engine core ke client TypeScript.
//
// Binary ini dimuat di ketiga pembungkus client (browser, Capacitor WebView,
// Tauri WebView) dan menjalankan aturan yang PERSIS sama dengan server. Lihat
// ADR-002.
//
// Desain API-nya berbasis handle, bukan oper-state: state tinggal di memori Go
// dan JS memegang sebuah id. Alternatifnya adalah mengirim seluruh state bolak-
// balik sebagai JSON tiap aksi, yang berarti serialisasi seluruh papan puluhan
// kali per giliran tanpa alasan.
//
// Mode offline (hotseat, solo vs bot) memakai binding ini. Mode online tidak:
// di sana server yang otoritatif dan client hanya menerapkan event yang sudah
// diproyeksikan (ADR-006).
package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/lastlighthouse/lastlighthouse/core"
)

type session struct {
	state *core.State
	rng   *core.RNG
	// log menyimpan seluruh event supaya client bisa merekonstruksi atau
	// mengekspor replay tanpa server.
	log []core.Event
}

var (
	sessions = map[int]*session{}
	nextID   = 1
)

func main() {
	api := js.Global().Get("Object").New()
	api.Set("newGame", js.FuncOf(newGame))
	api.Set("decide", js.FuncOf(decide))
	api.Set("view", js.FuncOf(view))
	api.Set("legal", js.FuncOf(legal))
	api.Set("events", js.FuncOf(eventLog))
	api.Set("dispose", js.FuncOf(dispose))
	api.Set("contentHash", js.FuncOf(contentHash))
	js.Global().Set("LastLighthouseCore", api)

	// Beri tahu sisi JS bahwa binding sudah siap dipakai.
	if cb := js.Global().Get("onLastLighthouseCoreReady"); cb.Type() == js.TypeFunction {
		cb.Invoke()
	}

	// WASM di Go harus tetap hidup agar callback yang terdaftar bisa dipanggil.
	select {}
}

// ok dan fail membungkus hasil dalam bentuk yang seragam, supaya sisi TS tidak
// perlu membedakan exception dari nilai kembalian.
func ok(payload any) any {
	b, err := json.Marshal(payload)
	if err != nil {
		return fail(err)
	}
	res := js.Global().Get("Object").New()
	res.Set("ok", true)
	res.Set("data", string(b))
	return res
}

func fail(err error) any {
	res := js.Global().Get("Object").New()
	res.Set("ok", false)
	res.Set("error", err.Error())
	return res
}

func session0(args []js.Value, idx int) (*session, error) {
	if len(args) <= idx {
		return nil, fmt.Errorf("argumen handle tidak ada")
	}
	s, found := sessions[args[idx].Int()]
	if !found {
		return nil, fmt.Errorf("handle sesi tidak dikenal")
	}
	return s, nil
}

// newGame(configJSON) -> {handle, events}
//
// configJSON: {"matchId":"...","seed":123,
//
//	"players":[{"id":"p1","name":"Ana","character":"navigator"}]}
func newGame(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return fail(fmt.Errorf("newGame butuh satu argumen config JSON"))
	}

	var cfg struct {
		MatchID string `json:"matchId"`
		Seed    int64  `json:"seed"`
		Players []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Character string `json:"character"`
		} `json:"players"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &cfg); err != nil {
		return fail(fmt.Errorf("config tidak valid: %w", err))
	}

	setups := make([]core.PlayerSetup, 0, len(cfg.Players))
	for _, p := range cfg.Players {
		setups = append(setups, core.PlayerSetup{
			ID:        core.PlayerID(p.ID),
			Name:      p.Name,
			Character: core.CharacterID(p.Character),
		})
	}

	st, events, err := core.NewGame(cfg.MatchID, cfg.Seed, setups, core.DefaultContent())
	if err != nil {
		return fail(err)
	}

	id := nextID
	nextID++
	sessions[id] = &session{state: st, rng: core.NewRNG(st.RNGState), log: events}

	return ok(struct {
		Handle int          `json:"handle"`
		Events []core.Event `json:"events"`
	}{id, events})
}

// decide(handle, commandJSON) -> {events}
func decide(_ js.Value, args []js.Value) any {
	s, err := session0(args, 0)
	if err != nil {
		return fail(err)
	}
	if len(args) < 2 {
		return fail(fmt.Errorf("decide butuh command JSON"))
	}

	var cmd core.Command
	if err := json.Unmarshal([]byte(args[1].String()), &cmd); err != nil {
		return fail(fmt.Errorf("command tidak valid: %w", err))
	}

	events, err := core.Decide(s.state, cmd, core.DefaultContent(), s.rng)
	if err != nil {
		// Penolakan aturan bukan crash: UI menampilkannya sebagai toast.
		return fail(err)
	}

	core.ApplyAll(s.state, events)
	s.state.RNGState = s.rng.Seed()
	s.log = append(s.log, events...)

	return ok(struct {
		Events []core.Event `json:"events"`
	}{events})
}

// view(handle, playerId) -> PlayerView
func view(_ js.Value, args []js.Value) any {
	s, err := session0(args, 0)
	if err != nil {
		return fail(err)
	}
	if len(args) < 2 {
		return fail(fmt.Errorf("view butuh player id"))
	}
	return ok(core.Project(s.state, core.PlayerID(args[1].String())))
}

// legal(handle, playerId) -> []Command
func legal(_ js.Value, args []js.Value) any {
	s, err := session0(args, 0)
	if err != nil {
		return fail(err)
	}
	if len(args) < 2 {
		return fail(fmt.Errorf("legal butuh player id"))
	}
	v := core.Project(s.state, core.PlayerID(args[1].String()))
	return ok(core.LegalCommands(v, core.DefaultContent()))
}

// events(handle, playerId) -> event log yang SUDAH diproyeksikan untuk pemain itu.
//
// Proyeksi dilakukan di sini, bukan disaring di UI, supaya mode offline
// memperlakukan kerahasiaan dengan aturan yang sama persis dengan mode online
// (ADR-006). Kalau client yang menyaring, akan ada dua definisi "apa yang boleh
// dilihat pemain ini" dan keduanya pasti menyimpang.
//
// playerId kosong menghasilkan tampilan penonton: semua rahasia tersunting.
func eventLog(_ js.Value, args []js.Value) any {
	s, err := session0(args, 0)
	if err != nil {
		return fail(err)
	}
	viewer := ""
	if len(args) > 1 {
		viewer = args[1].String()
	}
	return ok(core.ProjectEvents(s.log, core.PlayerID(viewer)))
}

func dispose(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return fail(fmt.Errorf("dispose butuh handle"))
	}
	delete(sessions, args[0].Int())
	return ok(struct{}{})
}

// contentHash dipakai client untuk memastikan versinya cocok dengan server
// sebelum join match online (ADR-005).
func contentHash(_ js.Value, _ []js.Value) any {
	return ok(struct {
		Hash string `json:"hash"`
	}{core.DefaultContent().Hash})
}
