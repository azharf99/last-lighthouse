package bot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lastlighthouse/lastlighthouse/core"
)

// allowedImports untuk bot/ bahkan lebih ketat daripada core/.
//
// bot ikut dikompilasi ke WASM dan dikirim ke setiap pemain (mode solo), jadi
// dependensi apa pun di sini menambah ukuran unduhan bagi semua orang. Dan
// karena keputusan bot memengaruhi jalannya permainan, apa pun yang tidak
// deterministik di sini akan merusak reprodusibilitas simulator.
var allowedImports = map[string]bool{
	"github.com/lastlighthouse/lastlighthouse/core": true,
}

func TestBotStaysPure(t *testing.T) {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("baca direktori bot: %v", err)
	}
	checked := 0

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++

		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		for _, imp := range file.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if !allowedImports[p] {
				t.Errorf("%s mengimpor %q. bot ikut masuk ke core.wasm dan harus "+
					"tetap deterministik; tambahkan ke allowedImports hanya dengan sadar.",
					name, p)
			}
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.GoStmt, *ast.ChanType, *ast.SelectStmt:
				t.Errorf("%s: ada konkurensi di %s; keputusan bot harus sinkron.",
					name, fset.Position(n.Pos()))
			}
			return true
		})
	}

	if checked == 0 {
		t.Fatal("tidak ada file sumber bot yang diperiksa")
	}
}

// TestBotIsDeterministic memverifikasi bahwa bot dengan seed yang sama memilih
// aksi yang sama.
//
// Ini prasyarat agar simulator bisa dipakai membandingkan konfigurasi: kalau bot
// sendiri tidak reprodusibel, perbedaan win rate antara dua nilai konten tidak
// bisa dibedakan dari derau.
func TestBotIsDeterministic(t *testing.T) {
	c := core.DefaultContent()
	setups := []core.PlayerSetup{
		{ID: "b1", Name: "Bot 1", Character: "navigator"},
		{ID: "b2", Name: "Bot 2", Character: "engineer"},
	}

	run := func() []core.CommandKind {
		s, _, err := core.NewGame("det", 909, setups, c)
		if err != nil {
			t.Fatalf("NewGame: %v", err)
		}
		rng := core.NewRNG(s.RNGState)
		pick := core.NewRNG(4242)
		ai := New(Standard)

		var seq []core.CommandKind
		for range 250 {
			if s.Over() {
				break
			}
			actor := s.ActivePlayer()
			if s.Pending != nil {
				actor = s.Player(s.Pending.Player)
			}
			if actor == nil {
				break
			}
			view := core.Project(s, actor.ID)
			cmd, ok := ai.Choose(view, core.LegalCommands(view, c), pick)
			if !ok {
				break
			}
			evs, err := core.Decide(s, cmd, c, rng)
			if err != nil {
				t.Fatalf("bot memilih %s tapi ditolak: %v", cmd.Kind, err)
			}
			core.ApplyAll(s, evs)
			s.RNGState = rng.Seed()
			seq = append(seq, cmd.Kind)
		}
		return seq
	}

	a, b := run(), run()
	if len(a) != len(b) {
		t.Fatalf("panjang urutan aksi berbeda: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("aksi ke-%d berbeda antar run: %s vs %s", i, a[i], b[i])
		}
	}
	if len(a) < 20 {
		t.Errorf("hanya %d aksi tercatat; test tidak menguji banyak", len(a))
	}
}

// TestBotNeverProposesIllegalCommand adalah kontrak yang sama dengan yang
// dijaga core untuk client: apa pun yang dipilih bot harus diterima Decide.
func TestBotNeverProposesIllegalCommand(t *testing.T) {
	c := core.DefaultContent()
	setups := []core.PlayerSetup{
		{ID: "b1", Character: "navigator"},
		{ID: "b2", Character: "engineer"},
		{ID: "b3", Character: "hunter"},
		{ID: "b4", Character: "scholar"},
	}

	for seed := int64(1); seed <= 60; seed++ {
		s, _, err := core.NewGame("legal", seed, setups, c)
		if err != nil {
			t.Fatalf("NewGame: %v", err)
		}
		rng := core.NewRNG(s.RNGState)
		pick := core.NewRNG(seed * 31)
		ai := New(Standard)

		for range 400 {
			if s.Over() {
				break
			}
			actor := s.ActivePlayer()
			if s.Pending != nil {
				actor = s.Player(s.Pending.Player)
			}
			if actor == nil {
				break
			}
			view := core.Project(s, actor.ID)
			cmd, ok := ai.Choose(view, core.LegalCommands(view, c), pick)
			if !ok {
				break
			}
			evs, err := core.Decide(s, cmd, c, rng)
			if err != nil {
				t.Fatalf("seed %d: bot memilih %+v tapi Decide menolak: %v", seed, cmd, err)
			}
			core.ApplyAll(s, evs)
			s.RNGState = rng.Seed()
		}
	}
}
