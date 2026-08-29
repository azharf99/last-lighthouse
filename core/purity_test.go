package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// allowedImports adalah daftar putih import untuk file non-test di core/.
//
// Daftarnya sengaja pendek. Setiap penambahan harus dipertimbangkan sadar:
// core dikompilasi ke WASM dan dikirim ke setiap pemain, jadi dependensi yang
// masuk ke sini menambah ukuran unduhan bagi semua orang; dan core harus tetap
// deterministik, jadi apa pun yang menyentuh waktu, jaringan, atau keacakan
// global akan merusak replay dan simulator balance.
var allowedImports = map[string]bool{
	"crypto/sha256": true,
	"embed":         true,
	"encoding/hex":  true,
	"encoding/json": true,
	"errors":        true,
	"fmt":           true,
	"io/fs":         true,
	"sort":          true,
}

// forbiddenSelectors adalah pemanggilan yang merusak determinisme meski
// package-nya sendiri terlihat tidak berbahaya.
var forbiddenSelectors = map[string]string{
	"time.Now":     "waktu harus diinjeksikan lewat parameter, bukan dibaca dari jam sistem",
	"time.Since":   "waktu harus diinjeksikan lewat parameter",
	"rand.Intn":    "gunakan *RNG milik core, bukan math/rand global",
	"rand.Int":     "gunakan *RNG milik core",
	"rand.Float64": "gunakan *RNG milik core",
	"os.Getenv":    "core tidak boleh membaca environment",
}

func coreSourceFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("baca direktori core: %v", err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		out = append(out, filepath.Join(".", name))
	}
	if len(out) == 0 {
		t.Fatal("tidak ada file sumber core yang ditemukan")
	}
	return out
}

// TestCoreImportsStayPure menegakkan kontrak kemurnian yang didokumentasikan di
// doc.go dan ADR-002.
//
// Ini ditulis sebagai test, bukan sekadar catatan di dokumen, karena kontrak
// yang hanya hidup di dokumentasi akan dilanggar diam-diam saat seseorang
// menambahkan "cuma satu import kecil" di bawah tekanan tenggat.
func TestCoreImportsStayPure(t *testing.T) {
	fset := token.NewFileSet()

	for _, path := range coreSourceFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		for _, imp := range file.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: path import tidak valid %s", path, imp.Path.Value)
			}
			if !allowedImports[p] {
				t.Errorf("%s mengimpor %q, yang tidak ada di daftar putih core.\n"+
					"core dikompilasi ke WASM dan harus tetap deterministik "+
					"(lihat doc.go). Kalau import ini benar-benar dibutuhkan, "+
					"tambahkan ke allowedImports secara sadar.",
					path, p)
			}
		}
	}
}

// TestCoreHasNoConcurrency memastikan core bebas goroutine dan channel.
//
// Konkurensi di dalam core akan membuat urutan eksekusi tidak deterministik,
// yang langsung merusak replay dan simulator. Konkurensi adalah urusan lapisan
// server (Match Actor, ADR-004), bukan urusan aturan game.
func TestCoreHasNoConcurrency(t *testing.T) {
	fset := token.NewFileSet()

	for _, path := range coreSourceFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.GoStmt:
				t.Errorf("%s: ada goroutine di %s. Konkurensi adalah urusan "+
					"server (ADR-004), bukan core.",
					path, fset.Position(node.Pos()))
			case *ast.ChanType:
				t.Errorf("%s: ada channel di %s. Core harus sinkron dan deterministik.",
					path, fset.Position(node.Pos()))
			case *ast.SelectStmt:
				t.Errorf("%s: ada select di %s.", path, fset.Position(node.Pos()))
			}
			return true
		})
	}
}

// TestCoreAvoidsNondeterministicCalls menangkap pemanggilan yang lolos dari
// pemeriksaan import, misalnya lewat alias package.
func TestCoreAvoidsNondeterministicCalls(t *testing.T) {
	fset := token.NewFileSet()

	for _, path := range coreSourceFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			sel, isSel := n.(*ast.SelectorExpr)
			if !isSel {
				return true
			}
			pkg, isIdent := sel.X.(*ast.Ident)
			if !isIdent {
				return true
			}
			key := pkg.Name + "." + sel.Sel.Name
			if why, bad := forbiddenSelectors[key]; bad {
				t.Errorf("%s: %s dipanggil di %s -- %s",
					path, key, fset.Position(sel.Pos()), why)
			}
			return true
		})
	}
}

// TestStateHasNoMaps menjaga State bebas dari map.
//
// Iterasi map di Go acak secara sengaja. Kalau ada map masuk ke State, hashing
// state jadi tidak stabil, perbandingan replay mulai gagal secara acak, dan
// simulator balance kehilangan reprodusibilitasnya. Slice dan array adalah
// harga yang harus dibayar untuk determinisme.
func TestStateHasNoMaps(t *testing.T) {
	fset := token.NewFileSet()
	guarded := map[string]bool{
		"State": true, "Player": true, "Board": true, "Location": true,
		"Component": true, "Contribution": true, "MatchSetup": true,
		"Event": true, "Command": true, "PlayerView": true,
	}

	for _, path := range coreSourceFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}

		ast.Inspect(file, func(n ast.Node) bool {
			ts, isType := n.(*ast.TypeSpec)
			if !isType || !guarded[ts.Name.Name] {
				return true
			}
			st, isStruct := ts.Type.(*ast.StructType)
			if !isStruct {
				return true
			}
			for _, field := range st.Fields.List {
				if _, isMap := field.Type.(*ast.MapType); isMap {
					names := ""
					for _, nm := range field.Names {
						names += nm.Name + " "
					}
					t.Errorf("%s: %s punya field map (%s) di %s.\n"+
						"Iterasi map tidak deterministik dan akan merusak "+
						"replay serta simulator. Pakai slice atau array.",
						path, ts.Name.Name, strings.TrimSpace(names),
						fset.Position(field.Pos()))
				}
			}
			return true
		})
	}
}
