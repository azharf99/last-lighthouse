// Internationalization (i18n) bilingual system (ID & EN)
// Content strings are synced with core/content/*.json (ADR-005).

export type Lang = 'id' | 'en';

let currentLang: Lang = (localStorage.getItem('llh_lang') as Lang) || 'id';

export const i18n = {
  getLang(): Lang {
    return currentLang;
  },

  setLang(lang: Lang) {
    currentLang = lang;
    localStorage.setItem('llh_lang', lang);
  },

  toggleLang(): Lang {
    currentLang = currentLang === 'id' ? 'en' : 'id';
    localStorage.setItem('llh_lang', currentLang);
    return currentLang;
  },

  t(key: string, fallback?: string): string {
    const dict = TRANSLATIONS[currentLang] || TRANSLATIONS.id;
    return dict[key] || fallback || key;
  },
};

export const TRANSLATIONS: Record<Lang, Record<string, string>> = {
  id: {
    // Nav & Mode
    'app.title': 'The Last Lighthouse',
    'app.subtitle': 'Menyalakan Mercusuar Terakhir',
    'nav.hotseat': 'Mode Hotseat',
    'nav.online': 'Mode Online',
    'nav.new_game': 'Match Baru',
    'nav.lobby': 'Lobby',
    'nav.back_to_lobby': 'Kembali ke Lobby',
    'nav.sound_on': 'Suara: Aktif',
    'nav.sound_off': 'Suara: Hening',

    // Game Status
    'status.won': '★ Mercusuar Menyala! Pulau Selamat!',
    'status.lost': '☠ Kegelapan Menang. Pulau Tenggelam.',
    'status.turn_of': 'Giliran',
    'status.ap_left': 'AP tersisa',
    'status.waiting': 'Menunggu giliran...',

    // Actions
    'action.move': 'Pindah',
    'action.explore': 'Jelajahi',
    'action.gather': 'Ambil',
    'action.repair': 'Perbaiki',
    'action.fight': 'Lawan Monster',
    'action.investigate': 'Selidiki',
    'action.rest': 'Istirahat',
    'action.trade': 'Beri Resource',
    'action.end_turn': 'Akhiri Giliran',

    // Locations
    'loc.lighthouse': 'Mercusuar',
    'loc.harbor': 'Pelabuhan',
    'loc.village': 'Desa',
    'loc.forest': 'Hutan',
    'loc.cave': 'Gua',
    'loc.crystal_cavern': 'Gua Kristal',
    'loc.ruins': 'Reruntuhan',
    'loc.mountain': 'Gunung',
    'loc.temple': 'Kuil',
    'loc.unexplored': 'Wilayah Belum Dipetakan',

    // Resources
    'res.wood': 'Kayu',
    'res.metal': 'Logam',
    'res.crystal': 'Kristal',
    'res.food': 'Makanan',

    // Lighthouse Components
    'comp.foundation': 'Fondasi',
    'comp.power_core': 'Inti Daya',
    'comp.lens': 'Lensa',
    'comp.mirror_array': 'Susunan Cermin',
    'comp.beacon': 'Suar',

    // Characters
    'char.navigator': 'Sang Navigator',
    'char.engineer': 'Sang Insinyur',
    'char.hunter': 'Sang Pemburu',
    'char.scholar': 'Sang Cendekia',
    'char.navigator.desc': 'Ahli menjelajah pulau. Gratis eksplorasi 1x per giliran tanpa menghabiskan AP.',
    'char.engineer.desc': 'Pakar mekanikal. Diskon 1 resource per ronde saat memperbaiki mercusuar.',
    'char.hunter.desc': 'Petarung tangguh. Mendapat bonus +1 pada setiap lemparan dadu pertempuran.',
    'char.scholar.desc': 'Peneliti misteri kuno. Menarik 2 kartu misteri sekaligus dan memilih yang terbaik.',

    // Darkness & Thresholds
    'darkness.title': 'Tingkat Kegelapan (Darkness Track)',
    'darkness.t2': 'Ambang 2: Monster bergerak 2 petak',
    'darkness.t4': 'Ambang 4: Penalti pengumpulan resource (-1)',
    'darkness.t6': 'Ambang 6: Kemunculan monster baru dipercepat',
    'darkness.t7': 'Ambang 7: AP berkurang menjadi 2',
    'darkness.t8': 'Ambang 8: Kegelapan total — Kalah',

    // Combat
    'combat.title': 'Pertarungan Monster (1D6)',
    'combat.subtitle': 'Lempar dadu untuk menentukan nasib pertempuran:',
    'combat.roll_button': '🎲 Lempar Dadu (1 AP)',
    'combat.rolling': 'Menggelindingkan dadu...',
    'combat.result_wound': 'Kalah: Terluka (-1 HP)',
    'combat.result_standoff': 'Seri: Monster bertahan (Standoff)',
    'combat.result_victory': 'Menang: Monster dikalahkan (+1 VP)!',
    'combat.bonus_hunter': '+1 Bonus Pemburu Aktif',

    // Card Decks
    'deck.event': 'Event Deck (Peristiwa Ronde)',
    'deck.mystery': 'Mystery Deck (Misteri Pulau)',
    'deck.artifact': 'Artifact Deck (Relik & Pusaka)',
    'deck.cards_left': 'kartu tersisa',
    'deck.discarded': 'dibuang',

    // Handoff & Lobby
    'handoff.title': 'Oper Perangkat',
    'handoff.prompt': 'Berikan layar ke',
    'handoff.subtext': 'Objective rahasia pemain lain tidak boleh terlihat.',
    'handoff.confirm': 'Saya siap — lanjut',

    // Character Setup Modal
    'setup.title': 'Persiapan Karakter & Pemain',
    'setup.subtitle': 'Pilih peran unik untuk menjaga sinergi tim di pulau:',
    'setup.start_game': 'Mulai Menyalakan Mercusuar',
  },

  en: {
    // Nav & Mode
    'app.title': 'The Last Lighthouse',
    'app.subtitle': 'Igniting the Last Beacon',
    'nav.hotseat': 'Hotseat Mode',
    'nav.online': 'Online Mode',
    'nav.new_game': 'New Match',
    'nav.lobby': 'Lobby',
    'nav.back_to_lobby': 'Back to Lobby',
    'nav.sound_on': 'Audio: On',
    'nav.sound_off': 'Audio: Muted',

    // Game Status
    'status.won': '★ The Lighthouse is Lit! The Island is Saved!',
    'status.lost': '☠ Darkness Consumes All. The Island Sinks.',
    'status.turn_of': 'Turn of',
    'status.ap_left': 'AP remaining',
    'status.waiting': 'Waiting for turn...',

    // Actions
    'action.move': 'Move',
    'action.explore': 'Explore',
    'action.gather': 'Gather',
    'action.repair': 'Repair',
    'action.fight': 'Fight Monster',
    'action.investigate': 'Investigate',
    'action.rest': 'Rest',
    'action.trade': 'Trade Resource',
    'action.end_turn': 'End Turn',

    // Locations
    'loc.lighthouse': 'Lighthouse',
    'loc.harbor': 'Harbor',
    'loc.village': 'Village',
    'loc.forest': 'Forest',
    'loc.cave': 'Cave',
    'loc.crystal_cavern': 'Crystal Cavern',
    'loc.ruins': 'Ruins',
    'loc.mountain': 'Mountain',
    'loc.temple': 'Temple',
    'loc.unexplored': 'Unexplored Territory',

    // Resources
    'res.wood': 'Wood',
    'res.metal': 'Metal',
    'res.crystal': 'Crystal',
    'res.food': 'Food',

    // Lighthouse Components
    'comp.foundation': 'Foundation',
    'comp.power_core': 'Power Core',
    'comp.lens': 'Lens',
    'comp.mirror_array': 'Mirror Array',
    'comp.beacon': 'Beacon',

    // Characters
    'char.navigator': 'The Navigator',
    'char.engineer': 'The Engineer',
    'char.hunter': 'The Hunter',
    'char.scholar': 'The Scholar',
    'char.navigator.desc': 'Master explorer. Free explore action 1x per turn without spending AP.',
    'char.engineer.desc': 'Mechanical expert. 1 free resource discount per round when repairing the lighthouse.',
    'char.hunter.desc': 'Fierce combatant. Gains a permanent +1 bonus on all combat dice rolls.',
    'char.scholar.desc': 'Ancient lore researcher. Draws 2 mystery cards and chooses the best one.',

    // Darkness & Thresholds
    'darkness.title': 'Darkness Track',
    'darkness.t2': 'Threshold 2: Monster moves 2 spaces',
    'darkness.t4': 'Threshold 4: Gather resource penalty (-1)',
    'darkness.t6': 'Threshold 6: Monster spawn rate increases',
    'darkness.t7': 'Threshold 7: AP reduced to 2',
    'darkness.t8': 'Threshold 8: Total Darkness — Defeat',

    // Combat
    'combat.title': 'Monster Combat (1D6)',
    'combat.subtitle': 'Roll the dice to resolve battle:',
    'combat.roll_button': '🎲 Roll Dice (1 AP)',
    'combat.rolling': 'Rolling dice...',
    'combat.result_wound': 'Defeat: Wounded (-1 HP)',
    'combat.result_standoff': 'Standoff: Monster holds position',
    'combat.result_victory': 'Victory: Monster defeated (+1 VP)!',
    'combat.bonus_hunter': '+1 Hunter Bonus Active',

    // Card Decks
    'deck.event': 'Event Deck (Round Events)',
    'deck.mystery': 'Mystery Deck (Island Lore)',
    'deck.artifact': 'Artifact Deck (Ancient Relics)',
    'deck.cards_left': 'cards left',
    'deck.discarded': 'discarded',

    // Handoff & Lobby
    'handoff.title': 'Pass the Device',
    'handoff.prompt': 'Hand the screen to',
    'handoff.subtext': 'Secret personal objectives must remain hidden.',
    'handoff.confirm': 'I am ready — continue',

    // Character Setup Modal
    'setup.title': 'Character & Player Setup',
    'setup.subtitle': 'Select unique asymmetric roles to sustain team synergy:',
    'setup.start_game': 'Begin Igniting the Lighthouse',
  },
};
