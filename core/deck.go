package core

// Mesin deck untuk kartu Event, Mystery, dan Artifact (GDD 8.3, 13, 20, 21).
//
// Urutan kartu yang belum ditarik bersifat rahasia dari SEMUA pemain -- bukan
// hanya dari lawan. Kalau urutannya diketahui, keputusan "apakah imbalannya
// sepadan dengan risikonya" (GDD 20) berubah jadi perhitungan tanpa risiko, dan
// mekanik intinya kehilangan makna. Project menggantinya dengan kartu tertutup
// sejumlah yang sama (ADR-006).

func (s *State) deck(kind DeckKind) *Deck {
	switch kind {
	case DeckEvent:
		return &s.EventDeck
	case DeckMystery:
		return &s.MysteryDeck
	case DeckArtifact:
		return &s.ArtifactDeck
	}
	return nil
}

// drawCard menarik kartu teratas, mengocok ulang buangan lebih dulu kalau
// tumpukan tarik habis.
//
// owner menentukan siapa yang boleh melihat kartunya: kosong berarti terbuka
// untuk semua (kartu Event dan Mystery memang dimainkan terbuka), sedangkan
// pemain tertentu berarti hanya dia yang melihatnya -- dipakai kemampuan Scholar
// yang menarik 2 kartu untuk dipilih sendiri (GDD 10.4).
//
// Mengembalikan string kosong kalau deck DAN buangannya sama-sama kosong --
// mungkin terjadi kalau semua artifact sudah dimiliki pemain. Pemanggil harus
// menangani kasus itu alih-alih mengasumsikan kartu selalu tersedia.
func drawCard(em *emitter, kind DeckKind, rng *RNG, owner PlayerID, reason string) CardID {
	d := em.s.deck(kind)
	if d == nil {
		return ""
	}

	if len(d.Draw) == 0 {
		if len(d.Discard) == 0 {
			return ""
		}
		// Kocok ulang buangan. Urutan hasil kocokan dibakukan ke dalam event,
		// supaya Apply tidak perlu RNG dan replay tetap deterministik (ADR-002).
		shuffled := append([]CardID(nil), d.Discard...)
		rng.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		// Urutan hasil kocokan RAHASIA dari semua pemain; yang publik hanya
		// berapa banyak kartu yang kembali ke tumpukan.
		em.emitSecret(serverOnly,
			Event{Kind: EvDeckReshuffled, Deck: kind, Cards: toEventCardIDs(shuffled), Reason: reason},
			&Event{Kind: EvDeckReshuffled, Deck: kind, Cards: hideCards(len(shuffled)), Reason: reason})
	}

	d = em.s.deck(kind)
	if len(d.Draw) == 0 {
		return ""
	}
	card := d.Draw[0]

	if owner == "" {
		em.emit(Event{Kind: EvCardDrawn, Deck: kind, Card: EventCardID(card), Reason: reason})
	} else {
		em.emitSecret(owner,
			Event{Kind: EvCardDrawn, Deck: kind, Card: EventCardID(card), Player: owner, Reason: reason},
			// Yang lain hanya tahu satu kartu ditarik, bukan kartu apa.
			&Event{Kind: EvCardDrawn, Deck: kind, Player: owner, Reason: reason})
	}
	return card
}

// discardCard mengembalikan kartu ke tumpukan buangan supaya bisa muncul lagi
// setelah dikocok ulang.
func discardCard(em *emitter, kind DeckKind, card CardID) {
	if card == "" {
		return
	}
	em.emit(Event{Kind: EvEventResolved, Deck: kind, Card: EventCardID(card)})
}

func toEventCardIDs(in []CardID) []EventCardID {
	out := make([]EventCardID, len(in))
	for i, c := range in {
		out[i] = EventCardID(c)
	}
	return out
}

func toCardIDs(in []EventCardID) []CardID {
	out := make([]CardID, len(in))
	for i, c := range in {
		out[i] = CardID(c)
	}
	return out
}

// buildDeck menyiapkan deck teracak dari daftar id konten.
//
// Hasil kocokan dikirim sebagai event, bukan diacak ulang di Apply: dengan
// begitu server dan replay memakai urutan yang sama tanpa harus mereproduksi
// urutan pemanggilan RNG.
func buildDeck(em *emitter, kind DeckKind, ids []CardID, rng *RNG) {
	shuffled := append([]CardID(nil), ids...)
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	em.emitSecret(serverOnly,
		Event{Kind: EvDeckShuffled, Deck: kind, Cards: toEventCardIDs(shuffled)},
		&Event{Kind: EvDeckShuffled, Deck: kind, Cards: hideCards(len(shuffled))})
}

// drawArtifact memberi satu artifact ke pemain (GDD 21).
func drawArtifact(em *emitter, c *Content, rng *RNG, pid PlayerID, reason string) {
	card := drawCard(em, DeckArtifact, rng, "", reason)
	if card == "" {
		return // deck artifact habis; bukan kesalahan, hanya tidak ada hadiah
	}
	if c.Artifact(ArtifactID(card)) == nil {
		return
	}
	em.emit(Event{
		Kind:     EvArtifactGained,
		Player:   pid,
		Artifact: ArtifactID(card),
		Reason:   reason,
	})
}
