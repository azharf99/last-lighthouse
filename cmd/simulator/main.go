// Command simulator menjalankan ribuan partai bot-vs-bot dan melaporkan
// statistik balance.
//
// Ini alasan terkuat kenapa rules engine ditulis di Go dan bukan di TypeScript
// (ADR-002): karena core murni dan deterministik, seluruh pertanyaan balance di
// GDD §33 bisa diukur dalam hitungan detik alih-alih ditebak dari 20-50 playtest
// manual (GDD §40 fase 7).
//
// Yang dijalankan simulator adalah KODE PRODUKSI yang sama persis -- core dan
// bot yang sama yang dipakai server dan client. Kalau ia menjalankan salinan
// aturan yang berbeda, angkanya tidak menjamin apa pun.
//
// Contoh:
//
//	go run ./cmd/simulator -games 10000
//	go run ./cmd/simulator -games 2000 -players 4 -csv hasil.csv
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/lastlighthouse/lastlighthouse/bot"
	"github.com/lastlighthouse/lastlighthouse/core"
)

const maxStepsPerGame = 4000

type gameResult struct {
	Seed       int64
	Won        bool
	Rounds     int
	Darkness   int
	Components int
	Steps      int

	ActionCounts map[core.CommandKind]int
	PlayerVP     map[core.CharacterID]int
	TopCharacter core.CharacterID
	Exhaustions  int
	MonstersSlain int
	Explored     int
	ArtifactsHeld int

	// Diagnostik ketersediaan resource. Komponen mercusuar menuntut 5 crystal
	// total (GDD §7), sementara peta awal tidak punya SATU PUN sumber crystal --
	// semuanya harus ditemukan lewat eksplorasi. Kalau tile-nya tidak terbuka,
	// partainya tidak bisa dimenangkan berapa pun AP-nya, dan itu harus terukur
	// alih-alih tersamar sebagai "win rate rendah".
	CrystalSourceFound bool
	CrystalGathered    int

	// VPBySource menjawab pertanyaan yang tidak bisa dijawab total VP saja:
	// apakah keenam sumber VP di GDD 25 benar-benar terpakai, atau skor akhir
	// sebenarnya hanya cerminan satu aktivitas?
	VPBySource map[string]int
}

func main() {
	games := flag.Int("games", 2000, "jumlah partai yang disimulasikan")
	players := flag.Int("players", 3, "jumlah pemain per partai (2-4)")
	seed0 := flag.Int64("seed", 1, "seed awal; partai ke-n memakai seed+n")
	careless := flag.Bool("careless", false, "pakai bot ceroboh sebagai garis dasar")
	csvPath := flag.String("csv", "", "tulis hasil per partai ke file CSV")

	// Override tuning. Ini yang mengubah simulator dari laporan sekali jalan
	// menjadi alat eksperimen: pertanyaan seperti "apakah 3 AP cukup?" (GDD §33)
	// dijawab dengan menjalankan kedua nilainya dan membandingkan hasilnya,
	// bukan dengan berdebat.
	ap := flag.Int("ap", 0, "override action point per giliran (0 = pakai konten)")
	darkMax := flag.Int("darkness-max", 0, "override batas Darkness")
	rise := flag.Int("rise", -1, "override kenaikan Darkness per ronde")
	gather := flag.Int("gather", 0, "override jumlah hasil gather")
	scale := flag.Int("scale", -1, "override tambahan biaya komponen per pemain di atas 2")
	exploreVP := flag.Int("explore-vp", 0, "VP per eksplorasi lokasi baru (Arah 1, default 0)")
	investigateAll := flag.Bool("investigate-all", false, "bolehkan Investigate di semua lokasi (Arah 2)")
	trace := flag.Bool("trace", false, "cetak jalannya satu partai lalu berhenti")
	flag.Parse()

	if *players < 2 || *players > 4 {
		fmt.Fprintln(os.Stderr, "players harus 2-4")
		os.Exit(1)
	}

	level := bot.Standard
	if *careless {
		level = bot.Careless
	}

	// LoadContent, bukan DefaultContent: override di bawah memutasi konten, dan
	// DefaultContent dipakai bersama seluruh proses.
	c, err := core.LoadContent()
	if err != nil {
		fmt.Fprintf(os.Stderr, "muat konten: %v\n", err)
		os.Exit(1)
	}
	var tweaks []string
	if *ap > 0 {
		c.Rules.ActionPointsPerTurn = *ap
		tweaks = append(tweaks, fmt.Sprintf("ap=%d", *ap))
	}
	if *darkMax > 0 {
		c.Darkness.Max = *darkMax
		tweaks = append(tweaks, fmt.Sprintf("darkness-max=%d", *darkMax))
	}
	if *rise >= 0 {
		c.Darkness.RisePerRound = *rise
		tweaks = append(tweaks, fmt.Sprintf("rise=%d", *rise))
	}
	if *gather > 0 {
		c.Rules.GatherBaseAmount = *gather
		tweaks = append(tweaks, fmt.Sprintf("gather=%d", *gather))
	}
	if *scale >= 0 {
		c.Rules.ExtraComponentCostPerExtraPlayer = *scale
		tweaks = append(tweaks, fmt.Sprintf("scale=%d", *scale))
	}
	if *exploreVP > 0 {
		c.Rules.ExploreVP = *exploreVP
		tweaks = append(tweaks, fmt.Sprintf("explore-vp=%d", *exploreVP))
	}
	if *investigateAll {
		c.Rules.InvestigateAnywhere = true
		tweaks = append(tweaks, "investigate-all")
	}

	if *trace {
		traceGame(*seed0, *players, level, c)
		return
	}

	results := make([]gameResult, 0, *games)

	for i := range *games {
		r, err := playGame(*seed0+int64(i), *players, level, c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "partai seed %d gagal: %v\n", *seed0+int64(i), err)
			os.Exit(1)
		}
		results = append(results, r)
	}

	report(results, *players, level, c, tweaks)

	if *csvPath != "" {
		if err := writeCSV(*csvPath, results); err != nil {
			fmt.Fprintf(os.Stderr, "tulis CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nHasil per partai ditulis ke %s\n", *csvPath)
	}
}

var characters = []core.CharacterID{"navigator", "engineer", "hunter", "scholar"}

func playGame(seed int64, players int, level bot.Difficulty, c *core.Content) (gameResult, error) {
	setups := make([]core.PlayerSetup, 0, players)
	for i := range players {
		setups = append(setups, core.PlayerSetup{
			ID:        core.PlayerID(fmt.Sprintf("p%d", i+1)),
			Name:      fmt.Sprintf("Bot %d", i+1),
			Character: characters[i%len(characters)],
		})
	}

	s, _, err := core.NewGame(fmt.Sprintf("sim-%d", seed), seed, setups, c)
	if err != nil {
		return gameResult{}, err
	}

	rng := core.NewRNG(s.RNGState)
	// RNG keputusan bot dipisah dari RNG game, supaya jumlah keputusan bot tidak
	// menggeser lemparan dadu dan kocokan deck. Tanpa pemisahan ini, mengubah
	// heuristik bot akan mengubah SELURUH keacakan permainan dan perbandingan
	// antar konfigurasi jadi tidak bermakna.
	pick := core.NewRNG(seed ^ 0x2545F4914F6CDD1D)
	ai := bot.New(level)

	res := gameResult{
		Seed:         seed,
		ActionCounts: map[core.CommandKind]int{},
		PlayerVP:     map[core.CharacterID]int{},
		VPBySource:   map[string]int{},
	}

	for step := range maxStepsPerGame {
		if s.Over() {
			break
		}

		actor := actingPlayer(s)
		if actor == "" {
			break
		}

		view := core.Project(s, actor)
		legal := core.LegalCommands(view, c)
		cmd, ok := ai.Choose(view, legal, pick)
		if !ok {
			break
		}

		events, err := core.Decide(s, cmd, c, rng)
		if err != nil {
			return res, fmt.Errorf("langkah %d: bot memilih %s tapi ditolak: %w", step, cmd.Kind, err)
		}
		core.ApplyAll(s, events)
		s.RNGState = rng.Seed()

		for _, e := range events {
			if e.Kind == core.EvVPAwarded {
				res.VPBySource[vpSource(e.Reason)] += e.Amount
			}
		}

		res.ActionCounts[cmd.Kind]++
		res.Steps++
	}

	res.Won = s.Status == core.StatusWon
	res.Rounds = s.Round
	res.Darkness = s.Darkness
	for _, comp := range s.Lighthouse {
		if comp.Repaired {
			res.Components++
		}
	}

	for i := range s.Board.Locations {
		loc := &s.Board.Locations[i]
		if !loc.Explored {
			continue
		}
		if lt := c.LocationType(loc.Type); lt != nil && lt.Yields[core.Crystal] > 0 {
			res.CrystalSourceFound = true
		}
	}

	topVP := -1
	for i := range s.Players {
		p := &s.Players[i]
		res.PlayerVP[p.Character] = p.VP
		res.MonstersSlain += p.MonstersSlain
		res.CrystalGathered += p.Inventory[core.Crystal]
		res.Explored += p.Explored
		res.ArtifactsHeld += len(p.Artifacts)
		if p.WasExhausted {
			res.Exhaustions++
		}
		if p.VP > topVP {
			topVP = p.VP
			res.TopCharacter = p.Character
		}
	}
	return res, nil
}

// actingPlayer menentukan siapa yang harus bertindak: pemilik pilihan tertunda
// kalau ada, kalau tidak pemain yang sedang bergiliran.
func actingPlayer(s *core.State) core.PlayerID {
	if s.Pending != nil {
		return s.Pending.Player
	}
	if p := s.ActivePlayer(); p != nil {
		return p.ID
	}
	return ""
}

func report(rs []gameResult, players int, level bot.Difficulty, c *core.Content, tweaks []string) {
	n := len(rs)
	if n == 0 {
		return
	}

	wins, totalRounds, totalSteps, totalDark := 0, 0, 0, 0
	compDist := map[int]int{}
	actionTotals := map[core.CommandKind]int{}
	vpByChar := map[core.CharacterID][]int{}
	topWins := map[core.CharacterID]int{}
	roundDist := map[int]int{}
	totalExhaust, totalSlain, totalExplored, totalArtifacts := 0, 0, 0, 0
	crystalGames, winsWithCrystal := 0, 0
	vpSources := map[string]int{}
	vpTotal := 0

	for _, r := range rs {
		if r.Won {
			wins++
			topWins[r.TopCharacter]++
		}
		totalRounds += r.Rounds
		totalSteps += r.Steps
		totalDark += r.Darkness
		compDist[r.Components]++
		roundDist[r.Rounds]++
		totalExhaust += r.Exhaustions
		totalSlain += r.MonstersSlain
		totalExplored += r.Explored
		totalArtifacts += r.ArtifactsHeld
		if r.CrystalSourceFound {
			crystalGames++
			if r.Won {
				winsWithCrystal++
			}
		}
		for k, v := range r.ActionCounts {
			actionTotals[k] += v
		}
		for ch, vp := range r.PlayerVP {
			vpByChar[ch] = append(vpByChar[ch], vp)
		}
		for src, vp := range r.VPBySource {
			vpSources[src] += vp
			vpTotal += vp
		}
	}

	levelName := "standard"
	if level == bot.Careless {
		levelName = "careless"
	}

	fmt.Printf("LAPORAN BALANCE — %d partai, %d pemain, bot %s, konten %s\n",
		n, players, levelName, c.Hash)
	fmt.Println("================================================================")

	// GDD §33: "Apakah Darkness naik terlalu cepat?" dan "Apakah game selesai
	// dalam 60 menit?"
	fmt.Printf("\nHASIL\n")
	fmt.Printf("  Menang (mercusuar menyala) : %5.1f%%\n", pct(wins, n))
	fmt.Printf("  Kalah (Darkness mencapai 8): %5.1f%%\n", pct(n-wins, n))
	fmt.Printf("  Ronde rata-rata            : %5.2f\n", float64(totalRounds)/float64(n))
	fmt.Printf("  Darkness akhir rata-rata   : %5.2f\n", float64(totalDark)/float64(n))
	fmt.Printf("  Aksi per partai            : %5.1f\n", float64(totalSteps)/float64(n))

	fmt.Printf("\nKEMAJUAN MERCUSUAR (berapa dari 5 komponen selesai)\n")
	for i := 0; i <= 5; i++ {
		fmt.Printf("  %d komponen : %5.1f%%  %s\n", i, pct(compDist[i], n), bar(compDist[i], n))
	}

	fmt.Printf("\nDISTRIBUSI RONDE SAAT BERAKHIR\n")
	var rounds []int
	for r := range roundDist {
		rounds = append(rounds, r)
	}
	sort.Ints(rounds)
	for _, r := range rounds {
		if roundDist[r]*100/n < 1 {
			continue
		}
		fmt.Printf("  ronde %2d : %5.1f%%  %s\n", r, pct(roundDist[r], n), bar(roundDist[r], n))
	}

	// GDD §33: "Apakah combat sepadan?" dan §38 "Unnecessary combat".
	fmt.Printf("\nPILIHAN AKSI (per partai)\n")
	var kinds []string
	for k := range actionTotals {
		kinds = append(kinds, string(k))
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		v := actionTotals[core.CommandKind(k)]
		fmt.Printf("  %-12s %6.2f  (%4.1f%% dari semua aksi)\n",
			k, float64(v)/float64(n), pct(v, totalSteps))
	}

	// GDD §33: "Apakah karakter seimbang?"
	fmt.Printf("\nVP RATA-RATA PER KARAKTER\n")
	var chars []string
	for ch := range vpByChar {
		chars = append(chars, string(ch))
	}
	sort.Strings(chars)
	for _, ch := range chars {
		vps := vpByChar[core.CharacterID(ch)]
		fmt.Printf("  %-10s %6.2f VP   menang terbanyak: %d partai\n",
			ch, mean(vps), topWins[core.CharacterID(ch)])
	}

	fmt.Printf("\nLAIN-LAIN (per partai)\n")
	fmt.Printf("  Monster dikalahkan  : %5.2f\n", float64(totalSlain)/float64(n))
	fmt.Printf("  Lokasi dieksplorasi : %5.2f\n", float64(totalExplored)/float64(n))
	fmt.Printf("  Artifact dipegang   : %5.2f\n", float64(totalArtifacts)/float64(n))
	fmt.Printf("  Pemain kelelahan    : %5.2f\n", float64(totalExhaust)/float64(n))

	// GDD 25 mendaftarkan enam sumber VP. Kalau skor akhir sebenarnya hanya
	// mencerminkan satu aktivitas, maka lima sisanya adalah desain yang tidak
	// pernah terpakai -- dan karakter yang unggul di aktivitas itu akan selalu
	// menang, berapa pun angkanya disetel.
	fmt.Printf("\nSUMBER VP (GDD 25)\n")
	var srcs []string
	for k := range vpSources {
		srcs = append(srcs, k)
	}
	sort.Strings(srcs)
	for _, k := range srcs {
		fmt.Printf("  %-22s %6.2f per partai  (%5.1f%% dari seluruh VP)\n",
			k, float64(vpSources[k])/float64(n), pct(vpSources[k], vpTotal))
	}

	// Diagnostik yang memisahkan "game terlalu sulit" dari "game tidak mungkin".
	fmt.Printf("\nKETERSEDIAAN CRYSTAL (mercusuar menuntut 5 crystal)\n")
	fmt.Printf("  Partai dengan sumber crystal terbuka : %5.1f%%\n", pct(crystalGames, n))
	fmt.Printf("  Menang di antara partai tersebut      : %5.1f%%\n", pct(winsWithCrystal, crystalGames))
	fmt.Printf("  Menang di antara partai TANPA sumber  : %5.1f%%\n", pct(wins-winsWithCrystal, n-crystalGames))

	fmt.Printf("\nCATATAN\n")
	fmt.Println("  Angka di atas mengukur PERMAINAN BOT, bukan permainan manusia.")
	fmt.Println("  Kegunaannya adalah membandingkan sebelum/sesudah perubahan konten,")
	fmt.Println("  bukan memprediksi win rate pemain sungguhan.")
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) * 100 / float64(b)
}

func mean(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	t := 0
	for _, x := range xs {
		t += x
	}
	return float64(t) / float64(len(xs))
}

func bar(a, b int) string {
	width := int(pct(a, b) / 2.5)
	out := make([]byte, 0, width)
	for range width {
		out = append(out, '#')
	}
	return string(out)
}

func writeCSV(path string, rs []gameResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"seed", "won", "rounds", "darkness", "components", "steps",
		"monsters_slain", "explored", "artifacts", "exhaustions", "top_character"}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, r := range rs {
		row := []string{
			strconv.FormatInt(r.Seed, 10),
			strconv.FormatBool(r.Won),
			strconv.Itoa(r.Rounds),
			strconv.Itoa(r.Darkness),
			strconv.Itoa(r.Components),
			strconv.Itoa(r.Steps),
			strconv.Itoa(r.MonstersSlain),
			strconv.Itoa(r.Explored),
			strconv.Itoa(r.ArtifactsHeld),
			strconv.Itoa(r.Exhaustions),
			string(r.TopCharacter),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// vpSource mengelompokkan alasan pemberian VP ke kategori GDD 25.
func vpSource(reason string) string {
	switch {
	case reason == "kontribusi perbaikan":
		return "perbaikan mercusuar"
	case reason == "personal objective":
		return "personal objective"
	case reason == "artifact":
		return "artifact"
	case reason == "mengalahkan monster":
		return "monster"
	case reason == "desa diselamatkan":
		return "desa diselamatkan"
	case reason == "eksplorasi lokasi baru":
		return "eksplorasi"
	case len(reason) >= 8 && reason[:8] == "mystery:":
		return "mystery"
	default:
		return "lainnya"
	}
}
