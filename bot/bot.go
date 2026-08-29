// Package bot adalah AI heuristik untuk The Last Lighthouse.
//
// Bot ini dipakai di tiga tempat sekaligus: mengisi kursi kosong di match
// online, jadi lawan di mode solo, dan menggerakkan simulator balance. Karena
// ketiganya memakai bot yang sama, hasil simulator benar-benar mencerminkan
// perilaku yang akan dihadapi pemain.
//
// Bot bekerja HANYA dari PlayerView dan LegalCommands -- persis seperti client
// manusia. Ia tidak pernah membaca State penuh, jadi ia tidak bisa curang dan
// tidak bisa "melihat" isi deck. Itu juga berarti bot adalah uji tekan alami
// untuk projection (ADR-006): kalau view kekurangan informasi yang dibutuhkan
// untuk bermain, bot akan bermain buruk dan itu terlihat di angka simulator.
//
// Sama seperti core, package ini harus tetap deterministik: keputusan yang sama
// pada state yang sama, tanpa iterasi map yang memengaruhi hasil.
package bot

import (
	"github.com/lastlighthouse/lastlighthouse/core"
)

// Difficulty mengatur seberapa tajam bot bermain.
type Difficulty int

const (
	// Careless memilih hampir acak. Berguna sebagai garis dasar simulator:
	// kalau bot ceroboh pun bisa menang, game-nya terlalu mudah.
	Careless Difficulty = iota
	// Standard bermain dengan heuristik penuh.
	Standard
)

// Bot memilih satu command dari daftar aksi legal.
type Bot struct {
	Difficulty Difficulty
}

func New(d Difficulty) *Bot { return &Bot{Difficulty: d} }

// Choose memilih aksi berikutnya.
//
// rng dipakai untuk memecah seri, sehingga dua bot dengan situasi identik tidak
// selalu bergerak sama persis -- tanpa itu, ribuan simulasi hanya mengulang satu
// permainan yang sama dan tidak mengukur apa pun.
func (b *Bot) Choose(v *core.PlayerView, legal []core.Command, rng *core.RNG) (core.Command, bool) {
	if len(legal) == 0 {
		return core.Command{}, false
	}
	if b.Difficulty == Careless {
		return legal[rng.Intn(len(legal))], true
	}

	best := make([]core.Command, 0, 4)
	bestScore := -1 << 30
	for _, cmd := range legal {
		sc := b.score(v, cmd)
		switch {
		case sc > bestScore:
			bestScore = sc
			best = append(best[:0], cmd)
		case sc == bestScore:
			best = append(best, cmd)
		}
	}
	if len(best) == 0 {
		return legal[0], true
	}
	return best[rng.Intn(len(best))], true
}

// score memberi nilai pada satu aksi. Angkanya relatif, bukan absolut.
//
// Bobotnya sengaja sederhana dan mudah dibaca: tujuannya bukan bermain optimal,
// melainkan bermain MASUK AKAL, supaya simulator mengukur keseimbangan game --
// bukan kepintaran bot.
func (b *Bot) score(v *core.PlayerView, cmd core.Command) int {
	s := &v.State
	me := s.Player(v.Viewer)
	if me == nil {
		return 0
	}
	c := core.DefaultContent()

	// Urgensi naik seiring Darkness: makin dekat kekalahan, makin besar bobot
	// memperbaiki mercusuar dibanding mengejar kepentingan pribadi. Inilah
	// ketegangan inti GDD 2.4 yang dimodelkan bot.
	urgency := s.Darkness * 12

	switch cmd.Kind {
	case core.CmdRepair:
		if cmd.Pay.IsEmpty() {
			// Perbaikan tanpa setoran hanya memakai jatah diskon sekali per ronde
			// (GDD 10.2). Berguna, tapi jangan sampai dihabiskan di ronde pertama
			// saat tas masih kosong -- simpan sampai benar-benar mempercepat.
			return 120 + urgency
		}
		// Selain itu selalu prioritas tertinggi: ini satu-satunya jalan menang.
		return 300 + urgency + cmd.Pay.Total()*10

	case core.CmdChoose:
		return b.scoreChoice(v, cmd)

	case core.CmdGather:
		need := neededForNextComponent(s)
		// Hanya kumpulkan yang MASIH kurang setelah dihitung isi tas sendiri.
		// Tanpa ini bot terus memanen kayu walau sudah membawa lebih dari cukup,
		// dan kehabisan AP sebelum sempat pulang.
		short := need[cmd.Resource] - me.Inventory[cmd.Resource]
		if short > 0 {
			return 190 + short*15
		}
		// Resource yang tidak dibutuhkan tetap ada gunanya, tapi jauh lebih kecil.
		if me.Inventory.Total() < 3 {
			return 60
		}
		return 20

	case core.CmdExplore:
		// Eksplorasi membuka resource baru dan memenuhi objective penjelajah,
		// tapi tidak langsung memajukan kemenangan.
		return 120 - s.Darkness*8

	case core.CmdFight:
		// Bertarung hanya kalau cukup sehat. Ini sekaligus yang membuat angka
		// "berapa sering Fight dipilih" di simulator bermakna: kalau tetap
		// jarang walau bot mau bertarung, imbalannya memang terlalu kecil
		// dibanding risikonya (GDD 38).
		if me.Health <= 1 {
			return -50
		}
		// Monster yang menghalangi memang harus dilawan, tapi bertarung tidak
		// boleh mengalahkan tugas mengantar resource: itu yang membuat bot
		// berputar-putar di satu petak sepanjang ronde.
		if carryingUseful(s, me) > 0 {
			return 45
		}
		return 90 + me.Health*10

	case core.CmdInvestigate:
		return 100

	case core.CmdRest:
		if me.Health <= 1 {
			return 200
		}
		return 30

	case core.CmdTrade:
		// Memberi resource yang tidak dibutuhkan sendiri tapi dibutuhkan
		// mercusuar: kerja sama yang murah.
		need := neededForNextComponent(s)
		if need[firstResource(cmd.Give)] > 0 {
			return 70
		}
		return 10

	case core.CmdMove:
		return b.scoreMove(v, cmd, c)

	case core.CmdEndTurn:
		// Selalu paling rendah: hanya dipilih kalau tidak ada yang lebih baik.
		return -100
	}
	return 0
}

// scoreMove menilai perpindahan memakai JARAK, bukan hanya isi petak tetangga.
//
// Versi pertama bot ini hanya melihat lokasi yang persis bersebelahan. Akibatnya
// ia mengumpulkan resource lalu tidak pernah membawanya pulang: dari Reruntuhan,
// mercusuar berjarak dua langkah, jadi tidak ada satu pun tetangga yang terlihat
// "seperti mercusuar", dan bertarung selalu menang skor. Simulator lalu
// melaporkan win rate 0% -- yang mengukur bot buta arah, bukan game yang sulit.
//
// Karena itu bot sekarang menghitung jarak ke tujuan dan memberi nilai pada
// langkah yang MENDEKATKAN, bukan hanya yang sudah sampai.
func (b *Bot) scoreMove(v *core.PlayerView, cmd core.Command, c *core.Content) int {
	s := &v.State
	me := s.Player(v.Viewer)
	dest := s.Board.Location(cmd.To)
	if dest == nil || me == nil {
		return 0
	}
	need := neededForNextComponent(s)

	// Berapa banyak muatan yang benar-benar berguna untuk komponen berikutnya.
	carrying := 0
	for r := range core.ResourceCount {
		if need[r] > 0 {
			carrying += min(me.Inventory[r], need[r])
		}
	}

	here := s.Board.DistancesFrom(me.At)
	toDest, reachable := here[cmd.To]
	if !reachable {
		toDest = 1 // tetangga yang belum tereksplorasi tidak muncul di BFS
	}

	// Sudah membawa yang dibutuhkan -> pulang ke mercusuar.
	if carrying > 0 {
		if lh := lighthouseID(s, c); lh != "" {
			fromDest := s.Board.DistancesFrom(cmd.To)
			dNow, okNow := here[lh]
			dNext, okNext := fromDest[lh]
			if okNow && okNext {
				if dNext < dNow {
					// Makin banyak muatan berguna, makin kuat dorongan pulang.
					return 200 + carrying*20 - dNext*5
				}
				return 30 // menjauh dari mercusuar padahal sedang membawa muatan
			}
		}
	}

	// Belum membawa apa-apa -> cari sumber resource yang dibutuhkan.
	score := 40
	for r := range core.ResourceCount {
		if need[r] > 0 && dest.Available[r] > 0 {
			score += 45
		}
	}
	if !dest.Explored {
		score += 25
	}
	if dest.Monsters > 0 && me.Health <= 1 {
		score -= 80
	}
	return score - toDest
}

// lighthouseID mencari lokasi mercusuar di papan.
func lighthouseID(s *core.State, c *core.Content) core.LocationID {
	for i := range s.Board.Locations {
		if lt := c.LocationType(s.Board.Locations[i].Type); lt != nil && lt.CanRepair {
			return s.Board.Locations[i].ID
		}
	}
	return ""
}

// scoreChoice menilai jawaban atas kartu Mystery (GDD 20).
//
// Bot menghindari pilihan yang menaikkan Darkness saat sudah mendekati batas.
// Ini persis dilema yang dirancang GDD 30: imbalan besar hampir selalu datang
// bersama Darkness, dan nilainya berubah tergantung sisa waktu.
func (b *Bot) scoreChoice(v *core.PlayerView, cmd core.Command) int {
	s := &v.State
	if s.Pending == nil {
		return 0
	}
	c := core.DefaultContent()

	// Pilihan kartu (kemampuan Scholar): ambil yang pertama; keduanya belum
	// bisa dibandingkan tanpa menilai seluruh pohon pilihannya.
	if s.Pending.Kind == "mystery_card" {
		return 100
	}

	def := c.MysteryCard(s.Pending.Card)
	if def == nil {
		return 0
	}
	var opt *core.MysteryOptionDef
	for i := range def.Options {
		if def.Options[i].ID == cmd.Option {
			opt = &def.Options[i]
			break
		}
	}
	if opt == nil {
		return 0
	}

	score := 50
	// Makin gelap, makin mahal harga satu poin Darkness.
	darknessCost := 20 + s.Darkness*25

	for _, e := range opt.Effects {
		switch e.Op {
		case "darkness":
			score -= e.Amount * darknessCost
		case "gain_vp":
			score += e.Amount * 12
		case "gain_artifact":
			score += 45
		case "gain_resource":
			score += e.Amount * 10
		case "pay_resource":
			score -= e.Amount * 8
		case "damage":
			score -= e.Amount * 30
		case "spawn_monster":
			score -= e.Amount * 25
		case "heal_all":
			score += e.Amount * 15
		case "reveal_tile":
			score += 20
		}
	}
	return score
}

// neededForNextComponent mengembalikan kekurangan komponen mercusuar berikutnya.
func neededForNextComponent(s *core.State) core.ResourceSet {
	next := s.NextComponent()
	if next == nil {
		return core.ResourceSet{}
	}
	return next.Progress.Missing(next.Cost)
}

func firstResource(rs core.ResourceSet) core.Resource {
	for r := range core.ResourceCount {
		if rs[r] > 0 {
			return core.Resource(r)
		}
	}
	return core.Wood
}

// carryingUseful menghitung muatan yang berguna untuk komponen berikutnya.
func carryingUseful(s *core.State, p *core.Player) int {
	need := neededForNextComponent(s)
	total := 0
	for r := range core.ResourceCount {
		if need[r] > 0 {
			total += min(p.Inventory[r], need[r])
		}
	}
	return total
}
