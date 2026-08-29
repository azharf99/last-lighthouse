package core

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestCommandRoundTripsOverWire menjaga agar Command selamat melewati JSON.
//
// Command menyeberangi batas kepercayaan dua kali: dari client ke server lewat
// WebSocket (ADR-003), dan dari JS ke Go lewat binding WASM (ADR-002). Field
// yang hilang saat serialisasi tidak memunculkan error di mana pun -- ia hanya
// berubah jadi nilai nol di sisi seberang.
//
// Kasus yang memicu test ini: Resource(Wood) bernilai 0, dan field-nya sempat
// memakai `omitempty`. Akibatnya setiap perintah "ambil kayu" terkirim tanpa
// field resource sama sekali. Server tetap menebaknya Wood karena kebetulan itu
// nilai nol, jadi bug-nya tidak terlihat di server -- ia baru muncul di UI
// sebagai "Ambil undefined".
func TestCommandRoundTripsOverWire(t *testing.T) {
	cases := []Command{
		{Kind: CmdGather, Player: "p1", Resource: Wood}, // nilai nol: kasus kritis
		{Kind: CmdGather, Player: "p1", Resource: Metal},
		{Kind: CmdGather, Player: "p1", Resource: Crystal},
		{Kind: CmdGather, Player: "p1", Resource: Food},
		{Kind: CmdMove, Player: "p2", To: "forest"},
		{Kind: CmdRepair, Player: "p3", Component: "foundation",
			Pay: NewResourceSet(map[Resource]int{Wood: 1, Metal: 1})},
		{Kind: CmdRest, Player: "p1"},
		{Kind: CmdEndTurn, Player: "p1"},
	}

	for _, want := range cases {
		blob, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("marshal %+v: %v", want, err)
		}

		var got Command
		if err := json.Unmarshal(blob, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", blob, err)
		}

		if got != want {
			t.Errorf("command tidak bertahan melewati JSON\n  kirim  %+v\n  terima %+v\n  wire   %s",
				want, got, blob)
		}

		// Untuk gather, field resource harus benar-benar ADA di wire, bukan
		// sekadar kebetulan terisi benar setelah di-decode.
		if want.Kind == CmdGather && !strings.Contains(string(blob), `"resource"`) {
			t.Errorf("command gather terkirim tanpa field resource: %s", blob)
		}
	}
}

// TestLegalCommandsSerialiseCompletely memeriksa command yang benar-benar
// dihasilkan core, bukan hanya yang disusun manual di test.
//
// Ini yang menangkap bug versi aslinya: LegalCommands menawarkan "gather wood",
// UI menerimanya tanpa field resource, dan tombolnya tampil sebagai
// "Ambil undefined undefined".
func TestLegalCommandsSerialiseCompletely(t *testing.T) {
	c := DefaultContent()
	s, _, err := NewGame("m_wire", 3, testSetups(), c)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	rng := NewRNG(s.RNGState)
	pick := NewRNG(5)

	seenGather := false

	for step := 0; step < 200 && !s.Over(); step++ {
		active := s.ActivePlayer()
		if active == nil {
			break
		}
		legal := LegalCommands(Project(s, active.ID), c)

		for _, cmd := range legal {
			blob, err := json.Marshal(cmd)
			if err != nil {
				t.Fatalf("marshal %+v: %v", cmd, err)
			}
			var back Command
			if err := json.Unmarshal(blob, &back); err != nil {
				t.Fatalf("unmarshal %s: %v", blob, err)
			}
			if back != cmd {
				t.Fatalf("command dari LegalCommands berubah setelah JSON\n"+
					"  asli   %+v\n  balik  %+v\n  wire   %s", cmd, back, blob)
			}
			if cmd.Kind == CmdGather {
				seenGather = true
			}
		}

		evs, err := Decide(s, legal[pick.Intn(len(legal))], c, rng)
		if err != nil {
			t.Fatalf("langkah %d: %v", step, err)
		}
		ApplyAll(s, evs)
		s.RNGState = rng.Seed()
	}

	if !seenGather {
		t.Error("tidak ada command gather yang muncul; test tidak menguji apa pun")
	}
}

// TestEventRoundTripsOverWire menerapkan pemeriksaan yang sama ke Event, yang
// juga harus bertahan di event log Postgres (ADR-004) selama bertahun-tahun.
func TestEventRoundTripsOverWire(t *testing.T) {
	_, events, err := NewGame("m_ev_wire", 11, testSetups(), DefaultContent())
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}

	for _, want := range events {
		blob, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("marshal %+v: %v", want, err)
		}
		var got Event
		if err := json.Unmarshal(blob, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", blob, err)
		}

		// Private dan PublicVariant sengaja tidak ikut serialisasi (json:"-"):
		// keduanya adalah metadata visibilitas sisi server, dan justru TIDAK
		// boleh sampai ke client.
		want.Private = ""
		want.PublicVariant = nil

		if got.Kind != want.Kind || got.Player != want.Player ||
			got.Objective != want.Objective || got.Value != want.Value ||
			got.Amount != want.Amount || got.Resources != want.Resources {
			t.Errorf("event tidak bertahan melewati JSON\n  asli  %+v\n  balik %+v\n  wire  %s",
				want, got, blob)
		}
	}
}
