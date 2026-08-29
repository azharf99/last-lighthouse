package core

// CommandKind adalah aksi yang bisa diminta pemain (GDD 11).
type CommandKind string

const (
	CmdMove        CommandKind = "move"        // 1 AP - pindah ke lokasi bersebelahan
	CmdGather      CommandKind = "gather"      // 1 AP - ambil resource di lokasi ini
	CmdRepair      CommandKind = "repair"      // 1 AP - setor resource ke mercusuar
	CmdRest        CommandKind = "rest"        // 1 AP - pulih 1 Health
	CmdEndTurn     CommandKind = "end_turn"    // 0 AP - akhiri giliran lebih awal
	CmdExplore     CommandKind = "explore"     // 1 AP - M1
	CmdFight       CommandKind = "fight"       // 1 AP - M1
	CmdInvestigate CommandKind = "investigate" // 1 AP - tarik & resolusikan Mystery
	CmdTrade       CommandKind = "trade"       // 1 AP - tukar resource dengan pemain di lokasi sama
	CmdChoose      CommandKind = "choose"      // 0 AP - jawab pilihan yang tertunda (GDD 20)
)

// Command adalah intent pemain. Bentuknya struct datar bertag, bukan interface,
// supaya bisa langsung di-marshal ke JSON envelope di ADR-003 tanpa
// UnmarshalJSON custom.
//
// Client mengirim ini; server memvalidasinya di Decide. Tidak ada isi Command
// yang boleh dipercaya begitu saja -- ia datang dari mesin pemain.
type Command struct {
	Kind   CommandKind `json:"kind"`
	Player PlayerID    `json:"player"`

	// Move
	To LocationID `json:"to,omitempty"`

	// Gather: resource mana yang diambil dari lokasi sekarang.
	//
	// TANPA omitempty, dan itu disengaja: Wood adalah nilai nol dari Resource,
	// sehingga omitempty akan MENGHILANGKAN field ini setiap kali pemain
	// mengambil kayu. Command yang terkirim lalu terlihat seperti "gather tanpa
	// resource", dan sisi penerima menebaknya sebagai Wood -- kebetulan benar
	// sekarang, tapi salah begitu urutan enum berubah, dan sudah pasti salah
	// saat ditampilkan di UI.
	Resource Resource `json:"resource"`

	// Repair: komponen mana, dan resource apa yang disetor.
	Component ComponentID `json:"component,omitempty"`
	Pay       ResourceSet `json:"pay,omitempty"`

	// Trade: kepada siapa, dan apa yang diberikan.
	Target PlayerID `json:"target,omitempty"`
	Give   ResourceSet `json:"give,omitempty"`

	// Choose: id pilihan Mystery, atau kartu yang dipilih (kemampuan Scholar).
	Option string      `json:"option,omitempty"`
	Card   EventCardID `json:"card,omitempty"`
}

// APCost mengembalikan biaya Action Point dasar sebelum modifier karakter
// atau artifact (GDD 11).
func (c Command) APCost() int {
	switch c.Kind {
	case CmdEndTurn, CmdChoose:
		// Menjawab pilihan yang tertunda bukan aksi baru: AP-nya sudah dibayar
		// oleh aksi yang memunculkan pilihan itu (mis. Investigate).
		return 0
	}
	return 1
}
