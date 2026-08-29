package core

import "errors"

// Error dari Decide bersifat "penolakan yang diharapkan", bukan bug. Client
// memprediksi legalitas dari PlayerView yang tidak punya informasi rahasia
// (ADR-006), jadi ia kadang mengirim command yang ditolak. UI menampilkannya
// sebagai toast, bukan error dialog.
var (
	ErrNotImplemented  = errors.New("aksi ini belum diimplementasikan (M1)")
	ErrMatchOver       = errors.New("match sudah selesai")
	ErrNotPlayerPhase  = errors.New("bukan fase pemain")
	ErrNotYourTurn     = errors.New("bukan giliranmu")
	ErrUnknownPlayer   = errors.New("pemain tidak dikenal")
	ErrUnknownLocation = errors.New("lokasi tidak dikenal")
	ErrNotAdjacent     = errors.New("lokasi tidak bersebelahan")
	ErrNoAP            = errors.New("action point tidak cukup")
	ErrNothingToGather = errors.New("tidak ada resource itu di lokasi ini")
	ErrInventoryFull   = errors.New("inventory penuh")
	ErrNotAtLighthouse = errors.New("harus berada di mercusuar untuk memperbaiki")
	ErrWrongComponent  = errors.New("komponen harus diperbaiki berurutan")
	ErrComponentDone   = errors.New("komponen sudah selesai")
	ErrNotEnoughRes    = errors.New("resource tidak cukup")
	ErrUselessPayment   = errors.New("setoran tidak mengurangi kebutuhan komponen")
	ErrMonsterPresent  = errors.New("tidak bisa istirahat saat ada monster")
	ErrHealthFull      = errors.New("health sudah penuh")
	ErrBadCommand      = errors.New("command tidak valid")

	// --- M1 ---
	ErrAlreadyExplored  = errors.New("lokasi sudah tereksplorasi")
	ErrNotExplored      = errors.New("lokasi belum tereksplorasi")
	ErrNoMonster        = errors.New("tidak ada monster di sini")
	ErrExhaustedNoFight = errors.New("pemain kelelahan tidak bisa bertarung")
	ErrCannotInvestigate = errors.New("lokasi ini tidak bisa diselidiki")
	ErrAlreadyInvestigated = errors.New("lokasi ini sudah diselidiki")
	ErrNoMysteryLeft    = errors.New("tidak ada kartu mystery tersisa")
	ErrChoicePending    = errors.New("selesaikan pilihan yang tertunda dulu")
	ErrNoChoicePending  = errors.New("tidak ada pilihan yang menunggu")
	ErrNotYourChoice    = errors.New("pilihan ini bukan milikmu")
	ErrBadOption        = errors.New("pilihan tidak tersedia")
	ErrTargetNotHere    = errors.New("pemain tujuan tidak berada di lokasi ini")
	ErrTradeEmpty       = errors.New("tidak ada yang ditukar")
	ErrTargetFull       = errors.New("inventory pemain tujuan penuh")
	ErrGatherBlocked    = errors.New("lokasi ini sedang tidak bisa dipanen")
)
