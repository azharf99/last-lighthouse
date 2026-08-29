package core

// PlayerView adalah SATU-SATUNYA bentuk state yang boleh meninggalkan server.
//
// Ia membungkus State yang field rahasianya sudah dikosongkan, bukan tipe yang
// sama sekali berbeda. Alasannya: client menjalankan Apply yang sama persis
// dengan server. Kalau view punya tipe sendiri, kita butuh Apply kedua, dan
// keduanya akan menyimpang -- persis masalah yang dihindari ADR-002.
//
// Konsekuensinya, menambah field rahasia ke State berarti WAJIB mengosongkannya
// di Project. Yang menegakkan ini bukan niat baik melainkan TestNoSecretLeak,
// yang mem-fuzz state dan menolak kalau ada ID rahasia muncul di view
// terserialisasi.
type PlayerView struct {
	Viewer PlayerID `json:"viewer"`
	State  State    `json:"state"`

	// MyObjective adalah objective milik Viewer, satu-satunya yang boleh ia lihat.
	MyObjective ObjectiveID `json:"myObjective,omitempty"`

	// HasObjective menandai pemain lain sudah punya objective tanpa
	// mengungkapkan apa objective-nya (GDD 24).
	HasObjective []PlayerID `json:"hasObjective"`
}

// Project menghasilkan tampilan state untuk satu pemain.
//
// Ini titik penegakan ADR-006: game ini semi-koperatif dengan pemenang tunggal,
// dan contoh skor akhir di GDD 26 hanya berselisih 1 VP sementara personal
// objective bernilai 7-8 VP. Mengetahui objective lawan praktis setara
// memenangkan pertandingan, jadi datanya tidak boleh sampai ke device siapa pun
// selain pemiliknya.
func Project(s *State, viewer PlayerID) *PlayerView {
	v := &PlayerView{Viewer: viewer}
	clone := s.Clone()

	for i := range clone.Players {
		p := &clone.Players[i]
		if p.ID == viewer {
			v.MyObjective = p.Objective
			continue
		}
		if p.Objective != "" {
			v.HasObjective = append(v.HasObjective, p.ID)
		}
		// Redaksi. Setiap field rahasia yang ditambahkan ke Player harus
		// dikosongkan di sini.
		p.Objective = ""
	}

	// RNGState adalah rahasia operasional: membocorkannya memungkinkan pemain
	// memprediksi lemparan dadu dan kocokan deck di masa depan (GDD 16, 20).
	clone.RNGState = 0

	// Deck dan tumpukan tile: identitas kartu diganti penanda tertutup, tapi
	// JUMLAHNYA dipertahankan.
	//
	// Mempertahankan panjangnya penting karena dua alasan. Pertama, jumlah kartu
	// tersisa memang informasi publik di board game fisik -- tumpukannya
	// kelihatan. Kedua, Apply di client memakai operasi yang sama dengan server
	// (pop kartu teratas), jadi tumpukan berisi kartu tertutup membuat satu
	// implementasi Apply cukup untuk kedua sisi (ADR-002).
	redactDeck(&clone.EventDeck)
	redactDeck(&clone.MysteryDeck)
	redactDeck(&clone.ArtifactDeck)
	for i := range clone.TileStack {
		clone.TileStack[i] = hiddenTile
	}

	// Pilihan tertunda milik pemain lain: yang terlihat hanya bahwa seseorang
	// sedang memutuskan, bukan kartu apa yang ia lihat (kemampuan Scholar).
	if clone.Pending != nil && clone.Pending.Player != viewer {
		pc := *clone.Pending
		pc.Cards = nil
		pc.Options = nil
		clone.Pending = &pc
	}

	v.State = *clone
	return v
}

// ProjectEvent menyaring satu event untuk satu penerima.
//
// Mengembalikan nil kalau penerima tidak boleh melihat apa pun dari event itu.
// Server memanggil ini per koneksi -- tidak pernah melakukan broadcast mentah.
func ProjectEvent(e Event, viewer PlayerID) *Event {
	if e.Private == "" {
		out := e
		out.PublicVariant = nil
		return &out
	}
	if e.Private == viewer {
		out := e
		out.Private = ""
		out.PublicVariant = nil
		return &out
	}
	if e.PublicVariant != nil {
		out := *e.PublicVariant
		out.Private = ""
		out.PublicVariant = nil
		return &out
	}
	return nil
}

// ProjectEvents menyaring sederet event untuk satu penerima.
func ProjectEvents(events []Event, viewer PlayerID) []Event {
	out := make([]Event, 0, len(events))
	for _, e := range events {
		if pe := ProjectEvent(e, viewer); pe != nil {
			out = append(out, *pe)
		}
	}
	return out
}

// hiddenCard dan hiddenTile adalah penanda untuk sesuatu yang ada tapi belum
// boleh diketahui isinya.
const (
	hiddenCard CardID         = "?"
	hiddenTile LocationTypeID = "?"
)

// serverOnly menandai event yang TIDAK BOLEH dilihat siapa pun -- bukan hanya
// dirahasiakan dari lawan.
//
// Kocokan deck adalah contohnya: kalau seorang pemain tahu urutan kartu
// berikutnya, keputusan "apakah imbalannya sepadan dengan risikonya" (GDD 20)
// berubah jadi perhitungan tanpa risiko. Karena tidak ada pemain yang ID-nya
// bisa cocok dengan nilai ini, ProjectEvent selalu mengembalikan varian
// publiknya.
const serverOnly PlayerID = "!server-only!"

// hideCards mengganti daftar kartu dengan penanda tertutup sebanyak yang sama,
// untuk dipakai sebagai varian publik.
func hideCards(n int) []EventCardID {
	out := make([]EventCardID, n)
	for i := range out {
		out[i] = EventCardID(hiddenCard)
	}
	return out
}

func redactDeck(d *Deck) {
	for i := range d.Draw {
		d.Draw[i] = hiddenCard
	}
	// Tumpukan buangan TERBUKA: kartu yang sudah dimainkan memang sudah dilihat
	// semua orang, dan menyembunyikannya justru menghapus informasi sah yang
	// dipakai pemain untuk memperkirakan sisa deck.
}
