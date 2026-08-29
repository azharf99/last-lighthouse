// Label bahasa Indonesia untuk id yang datang dari core.
//
// Konten di core/content/*.json sudah menyimpan nama dalam dua bahasa, tapi core
// membakukan nama Inggris ke dalam State saat setup karena simulator dan log
// server tidak punya konsep bahasa pengguna. Penerjemahan ke bahasa tampilan
// adalah urusan client, dan di sinilah tempatnya.
//
// M1: tabel ini digenerate dari file konten lewat cmd/contentgen (ADR-005),
// sehingga menambah lokasi atau objective baru tidak bisa lupa menambah label.

export const LOCATION_NAMES: Record<string, string> = {
  lighthouse: 'Mercusuar',
  harbor: 'Pelabuhan',
  village: 'Desa',
  forest: 'Hutan',
  cave: 'Gua',
  crystal_cavern: 'Gua Kristal',
  ruins: 'Reruntuhan',
  mountain: 'Gunung',
  temple: 'Kuil',
};

export const RESOURCE_NAMES: Record<string, string> = {
  wood: 'Kayu',
  metal: 'Logam',
  crystal: 'Kristal',
  food: 'Makanan',
};

export const COMPONENT_NAMES: Record<string, string> = {
  foundation: 'Fondasi',
  power_core: 'Inti Daya',
  lens: 'Lensa',
  mirror_array: 'Susunan Cermin',
  beacon: 'Suar',
};

export const CHARACTER_NAMES: Record<string, string> = {
  navigator: 'Navigator',
  engineer: 'Insinyur',
  hunter: 'Pemburu',
  scholar: 'Cendekia',
};

export const OBJECTIVE_LABELS: Record<string, string> = {
  the_savior: 'Sang Penyelamat — selamatkan 2 desa',
  the_collector: 'Sang Kolektor — kumpulkan 3 artifact',
  the_explorer: 'Sang Penjelajah — jelajahi 6 lokasi',
  the_guardian: 'Sang Penjaga — kalahkan 4 monster',
  the_builder: 'Sang Pembangun — sumbang 3 perbaikan',
  the_hoarder: 'Sang Penimbun — pegang 6 resource di akhir',
  the_survivor: 'Sang Penyintas — jangan pernah kelelahan',
  the_pillar: 'Sang Pilar — sumbang 10 resource total',
};

export const RESOURCE_GLYPH: Record<string, string> = {
  wood: '🪵',
  metal: '⛓',
  crystal: '💎',
  food: '🍖',
};

/**
 * Nama lokasi untuk ditampilkan.
 *
 * Pencariannya lewat TIPE lokasi, bukan id-nya. Lokasi awal kebetulan punya id
 * yang sama dengan tipenya (harbor, forest, ...), tapi petak hasil eksplorasi
 * tidak: id-nya "site_a" sementara tipenya bisa "crystal_cavern". Mencari lewat
 * id membuat semua tile hasil eksplorasi jatuh ke nama Inggris bawaan core.
 */
export function locationName(type: string, fallback: string): string {
  return LOCATION_NAMES[type] ?? fallback;
}

/** Label lokasi berdasarkan id, dengan mencari tipenya di papan lebih dulu. */
export function locationLabel(
  board: { locations: { id: string; type: string; name: string }[] },
  id: string,
): string {
  const loc = board.locations.find((l) => l.id === id);
  if (!loc) return id;
  return locationName(loc.type, loc.name);
}

export function componentName(id: string, fallback: string): string {
  return COMPONENT_NAMES[id] ?? fallback;
}

// --- Konten kartu (M1) ---
//
// Tabel di bawah ini adalah SALINAN PRESENTASI dari core/content/*.json: hanya
// teks yang ditampilkan, tanpa satu pun angka efek. Efeknya tetap dihitung core
// (ADR-002) -- kalau nilainya juga disalin ke sini, dua sumber kebenaran akan
// menyimpang persis seperti yang dihindari arsitekturnya.
//
// M1 lanjutan: digenerate lewat cmd/contentgen (ADR-005) supaya kartu baru tidak
// bisa lupa diberi label.

export interface MysteryOptionText {
  id: string;
  text: string;
}

export interface MysteryCardText {
  name: string;
  text: string;
  options: MysteryOptionText[];
}

export const MYSTERY_CARDS: Record<string, MysteryCardText> = {
  "my_abandoned_lab": { name: "Laboratorium Terbengkalai", text: "Di bawah reruntuhan kau menemukan laboratorium tua.", options: [{ id: "a", text: "Geledah menyeluruh" }, { id: "b", text: "Geledah hati-hati" }, { id: "c", text: "Tinggalkan saja" }] },
  "my_drowned_shrine": { name: "Kuil Tenggelam", text: "Air dingin menjilat altar terlupakan.", options: [{ id: "a", text: "Selami persembahannya" }, { id: "b", text: "Berdoa di tepinya" }, { id: "c", text: "Berbalik" }] },
  "my_keepers_journal": { name: "Jurnal Sang Penjaga", text: "Halamannya menjelaskan cara merawat cahaya itu.", options: [{ id: "a", text: "Baca seluruh halaman" }, { id: "b", text: "Baca sekilas" }, { id: "c", text: "Tutup bukunya" }] },
  "my_hollow_tree": { name: "Pohon Berongga", text: "Ada sesuatu yang tinggal di dalamnya.", options: [{ id: "a", text: "Raih ke dalam" }, { id: "b", text: "Tebang pohonnya" }, { id: "c", text: "Berjalan terus" }] },
  "my_stranded_survivor": { name: "Penyintas Terdampar", text: "Ada orang lain yang sampai ke pulau ini.", options: [{ id: "a", text: "Bagikan makananmu" }, { id: "b", text: "Tanya apa yang mereka lihat" }, { id: "c", text: "Diam saja" }] },
  "my_crystal_vein": { name: "Urat Kristal", text: "Batunya bercahaya samar.", options: [{ id: "a", text: "Bongkar semuanya" }, { id: "b", text: "Ambil satu serpih" }, { id: "c", text: "Tutup uratnya" }] },
  "my_watchers_cairn": { name: "Tugu Sang Pengawas", text: "Batu-batu ditumpuk dengan hati-hati.", options: [{ id: "a", text: "Bongkar tugunya" }, { id: "b", text: "Tambahkan satu batu" }, { id: "c", text: "Biarkan berdiri" }] },
  "my_signal_fire": { name: "Api Isyarat", text: "Kayu tua, masih kering.", options: [{ id: "a", text: "Nyalakan" }, { id: "b", text: "Ambil kayunya" }, { id: "c", text: "Tinggalkan" }] },
  "my_sunken_crate": { name: "Peti Karam", text: "Sesuatu terbawa arus.", options: [{ id: "a", text: "Congkel terbuka" }, { id: "b", text: "Ambil papannya" }, { id: "c", text: "Dorong kembali ke laut" }] },
  "my_whispering_dark": { name: "Bisikan Gelap", text: "Bayangan seolah menawarkan sesuatu.", options: [{ id: "a", text: "Dengarkan" }, { id: "b", text: "Menolak" }, { id: "c", text: "Lari" }] },
};

export const ARTIFACT_CARDS: Record<string, { name: string; text: string }> = {
  "ar_ancient_compass": { name: "Kompas Kuno", text: "Sekali per giliran, pindah ke lokasi tereksplorasi menjadi gratis." },
  "ar_aether_lantern": { name: "Lentera Eter", text: "Monster tidak bisa menyerangmu di Mercusuar." },
  "ar_forgotten_tool": { name: "Perkakas Terlupakan", text: "Sekali per ronde, satu perbaikan butuh 1 resource lebih sedikit." },
  "ar_black_pearl": { name: "Mutiara Hitam", text: "Bernilai 5 VP di akhir permainan." },
  "ar_tide_charm": { name: "Jimat Pasang", text: "Gathering menghasilkan 1 lebih banyak." },
  "ar_iron_ward": { name: "Tameng Besi", text: "+1 pada lemparan combat-mu." },
  "ar_pilgrims_pack": { name: "Ransel Peziarah", text: "Bisa membawa 2 resource lebih banyak." },
  "ar_keepers_seal": { name: "Segel Penjaga", text: "Bernilai 3 VP di akhir permainan." },
};

export const EVENT_CARDS: Record<string, { name: string; text: string }> = {
  "ev_heavy_storm": { name: "Badai Besar", text: "Lokasi pesisir menjadi berbahaya." },
  "ev_creeping_fog": { name: "Kabut Merayap", text: "Kegelapan menebal." },
  "ev_low_tide": { name: "Air Surut", text: "Pelabuhan menghasilkan lebih banyak." },
  "ev_night_howls": { name: "Lolongan Malam", text: "Sesuatu bergerak di tempat gelap." },
  "ev_scavenged_supplies": { name: "Perbekalan Temuan", text: "Peti-peti terdampar di pantai." },
  "ev_collapsed_tunnel": { name: "Terowongan Runtuh", text: "Tambang menjadi tidak stabil." },
  "ev_calm_night": { name: "Malam Tenang", text: "Jeda singkat." },
  "ev_the_light_flickers": { name: "Cahaya Berkedip", text: "Mercusuar menahan kegelapan." },
  "ev_forest_bounty": { name: "Limpahan Hutan", text: "Hutan sedang murah hati." },
  "ev_hollow_wind": { name: "Angin Hampa", text: "Angin dingin menguras tenaga." },
};
