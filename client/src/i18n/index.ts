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

    // Achievements & Leaderboard (M7)
    'ach.modal_title': 'Pencapaian & Papan Peringkat',
    'ach.tab_achievements': 'Pencapaian',
    'ach.tab_leaderboard': 'Papan Peringkat',
    'ach.tab_stats': 'Rekor Pribadi',
    'ach.progress_title': 'Progres Pencapaian',
    'ach.unlocked': 'Terbuka',
    'ach.unlocked_badge': 'Pencapaian Terbuka!',
    'ach.cat.all': 'Semua',
    'ach.cat.victory': 'Kemenangan',
    'ach.cat.combat': 'Tempur',
    'ach.cat.exploration': 'Eksplorasi',
    'ach.cat.team': 'Kerja Sama',
    'ach.cat.mastery': 'Masteri',

    'ach.first_win.title': 'Cahaya Harapan',
    'ach.first_win.desc': 'Nyalakan kelima komponen mercusuar dan menangkan permainan.',
    'ach.clutch_win.title': 'Di Ambang Kehancuran',
    'ach.clutch_win.desc': 'Menangkan match saat tingkat Darkness mencapai 7.',
    'ach.speed_run.title': 'Kilat di Tengah Badai',
    'ach.speed_run.desc': 'Menangkan match dalam 6 ronde atau lebih singkat.',
    'ach.unbroken.title': 'Tak Terkalahkan',
    'ach.unbroken.desc': 'Menangkan match tanpa pernah pingsan / kehabisan HP (Exhausted).',
    'ach.slayer_first.title': 'Pembasmi Pertama',
    'ach.slayer_first.desc': 'Kalahkan monster pertama dalam pertempuran dadu 1D6.',
    'ach.slayer_master.title': 'Penjaga Malam',
    'ach.slayer_master.desc': 'Kalahkan setidaknya 3 monster dalam satu match.',
    'ach.explorer_island.title': 'Pionir Pulau',
    'ach.explorer_island.desc': 'Kunjungi dan jelajahi minimal 6 lokasi di pulau.',
    'ach.cartographer.title': 'Pemeta Ulung',
    'ach.cartographer.desc': 'Buka dan petakan seluruh tile tersembunyi di pulau.',
    'ach.mystery_solver.title': 'Penyingkap Rahasia',
    'ach.mystery_solver.desc': 'Selesaikan dan pecahkan teka-teki kartu Mystery.',
    'ach.relic_collector.title': 'Kolektor Relik',
    'ach.relic_collector.desc': 'Kumpulkan 2 atau lebih Artefak Kuno dalam satu match.',
    'ach.master_builder.title': 'Arsitek Mercusuar',
    'ach.master_builder.desc': 'Berkontribusi pada 4 atau lebih komponen mercusuar.',
    'ach.team_player.title': 'Semangat Gotong Royong',
    'ach.team_player.desc': 'Berbagi atau melakukan trade resource dengan pemain lain.',
    'ach.secret_objective.title': 'Misi Rahasia Selesai',
    'ach.secret_objective.desc': 'Tuntaskan Personal Objective rahasia di akhir permainan.',
    'ach.high_scorer.title': 'Pahlawan Legendaris',
    'ach.high_scorer.desc': 'Raih skor 20 VP atau lebih dalam satu match.',
    'ach.four_pillars.title': 'Empat Pilar',
    'ach.four_pillars.desc': 'Raih kemenangan menggunakan keempat peran karakter berbeda.',

    // Leaderboard & Stats
    'lb.cat_vp': 'Skor Tertinggi (VP)',
    'lb.cat_speed': 'Kemenangan Tercepat',
    'lb.cat_monsters': 'Pemburu Terhebat',
    'lb.refresh': 'Segarkan',
    'lb.loading': 'Memuat data papan peringkat...',
    'lb.rank': 'Posisi',
    'lb.player': 'Pemain',
    'lb.role': 'Peran',
    'lb.status': 'Hasil',
    'lb.rounds': 'Ronde & Kegelapan',
    'lb.date': 'Tanggal',
    'lb.no_entries': 'Belum ada catatan skor pada kategori ini.',
    'lb.won': 'Menang',
    'lb.lost': 'Gugur',

    'stats.total_matches': 'Total Pertandingan',
    'stats.wins': 'Menang',
    'stats.losses': 'Kalah',
    'stats.highest_vp': 'Rekor Skor Tertinggi',
    'stats.personal_best': 'Personal High Score',
    'stats.monsters_slain': 'Monster Dikalahkan',
    'stats.repairs_joined': 'Komponen Diperbaiki',
    'stats.character_mastery': 'Masteri Karakter',
    'stats.character_mastery_sub': 'Menangkan match dengan tiap karakter untuk membuka pencapaian Empat Pilar.',
    'stats.victorious': 'Pernah Menang',
    'stats.untested': 'Belum Menang',
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

    // Achievements & Leaderboard (M7)
    'ach.modal_title': 'Achievements & Leaderboard',
    'ach.tab_achievements': 'Achievements',
    'ach.tab_leaderboard': 'Leaderboard',
    'ach.tab_stats': 'Career Stats',
    'ach.progress_title': 'Achievement Progress',
    'ach.unlocked': 'Unlocked',
    'ach.unlocked_badge': 'Achievement Unlocked!',
    'ach.cat.all': 'All',
    'ach.cat.victory': 'Victory',
    'ach.cat.combat': 'Combat',
    'ach.cat.exploration': 'Exploration',
    'ach.cat.team': 'Co-op & Team',
    'ach.cat.mastery': 'Mastery',

    'ach.first_win.title': 'Beacon of Hope',
    'ach.first_win.desc': 'Ignite all 5 lighthouse components and win the game.',
    'ach.clutch_win.title': 'Brink of Ruin',
    'ach.clutch_win.desc': 'Win a match while Darkness is at threshold 7.',
    'ach.speed_run.title': 'Storm Runner',
    'ach.speed_run.desc': 'Win a match in 6 rounds or fewer.',
    'ach.unbroken.title': 'Unbroken',
    'ach.unbroken.desc': 'Win a match without ever being exhausted / knocked down.',
    'ach.slayer_first.title': 'First Blood',
    'ach.slayer_first.desc': 'Defeat your first monster in 1D6 dice combat.',
    'ach.slayer_master.title': 'Night Warden',
    'ach.slayer_master.desc': 'Defeat at least 3 monsters in a single match.',
    'ach.explorer_island.title': 'Island Pioneer',
    'ach.explorer_island.desc': 'Visit and explore at least 6 locations on the island.',
    'ach.cartographer.title': 'Grand Cartographer',
    'ach.cartographer.desc': 'Reveal and map every hidden tile on the island.',
    'ach.mystery_solver.title': 'Mystery Solver',
    'ach.mystery_solver.desc': 'Investigate and resolve an ancient Mystery dilemma card.',
    'ach.relic_collector.title': 'Relic Hoarder',
    'ach.relic_collector.desc': 'Acquire 2 or more Ancient Artifacts in a single match.',
    'ach.master_builder.title': 'Master Architect',
    'ach.master_builder.desc': 'Contribute to 4 or more lighthouse components.',
    'ach.team_player.title': 'Generous Ally',
    'ach.team_player.desc': 'Share or trade resources with another player.',
    'ach.secret_objective.title': 'Secret Fulfilled',
    'ach.secret_objective.desc': 'Complete your secret personal objective at game end.',
    'ach.high_scorer.title': 'Legend of the Beacon',
    'ach.high_scorer.desc': 'Achieve a score of 20 VP or higher in a single match.',
    'ach.four_pillars.title': 'Four Pillars',
    'ach.four_pillars.desc': 'Achieve victory with all 4 distinct character roles.',

    // Leaderboard & Stats
    'lb.cat_vp': 'Top VP Score',
    'lb.cat_speed': 'Fastest Victory',
    'lb.cat_monsters': 'Top Slayers',
    'lb.refresh': 'Refresh',
    'lb.loading': 'Loading leaderboard rankings...',
    'lb.rank': 'Rank',
    'lb.player': 'Player',
    'lb.role': 'Role',
    'lb.status': 'Result',
    'lb.rounds': 'Rounds & Darkness',
    'lb.date': 'Date',
    'lb.no_entries': 'No entries recorded for this category yet.',
    'lb.won': 'Won',
    'lb.lost': 'Fallen',

    'stats.total_matches': 'Total Matches',
    'stats.wins': 'Wins',
    'stats.losses': 'Losses',
    'stats.highest_vp': 'Highest VP Score',
    'stats.personal_best': 'Personal High Score',
    'stats.monsters_slain': 'Monsters Slain',
    'stats.repairs_joined': 'Repairs Joined',
    'stats.character_mastery': 'Character Mastery',
    'stats.character_mastery_sub': 'Win matches with every character to unlock the Four Pillars achievement.',
    'stats.victorious': 'Victorious',
    'stats.untested': 'Untested',
  },
};
