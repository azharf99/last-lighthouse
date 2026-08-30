// Leaderboard client & offline fallback store

export interface LeaderboardEntry {
  id: string;
  playerName: string;
  character: string;
  vp: number;
  darkness: number;
  rounds: number;
  won: boolean;
  monstersSlain: number;
  componentsContributed: number;
  matchId: string;
  createdAt: string;
}

const LOCAL_LEADERBOARD_KEY = 'llh_local_leaderboard_v1';

const BASELINE_ENTRIES: LeaderboardEntry[] = [
  {
    id: 'lb_001',
    playerName: 'Kapten Ana',
    character: 'navigator',
    vp: 24,
    darkness: 4,
    rounds: 6,
    won: true,
    monstersSlain: 2,
    componentsContributed: 3,
    matchId: 'm_solo_pro',
    createdAt: new Date(Date.now() - 3 * 86400000).toISOString(),
  },
  {
    id: 'lb_002',
    playerName: 'Mekanik Budi',
    character: 'engineer',
    vp: 22,
    darkness: 5,
    rounds: 7,
    won: true,
    monstersSlain: 1,
    componentsContributed: 4,
    matchId: 'm_coop_duo',
    createdAt: new Date(Date.now() - 2 * 86400000).toISOString(),
  },
  {
    id: 'lb_003',
    playerName: 'Ranger Citra',
    character: 'hunter',
    vp: 19,
    darkness: 6,
    rounds: 8,
    won: true,
    monstersSlain: 5,
    componentsContributed: 2,
    matchId: 'm_hunt_01',
    createdAt: new Date(Date.now() - 1 * 86400000).toISOString(),
  },
  {
    id: 'lb_004',
    playerName: 'Arsiparis Dewi',
    character: 'scholar',
    vp: 18,
    darkness: 7,
    rounds: 8,
    won: true,
    monstersSlain: 0,
    componentsContributed: 3,
    matchId: 'm_lore_99',
    createdAt: new Date(Date.now() - 12 * 3600000).toISOString(),
  },
  {
    id: 'lb_005',
    playerName: 'Petualang Eko',
    character: 'navigator',
    vp: 15,
    darkness: 8,
    rounds: 5,
    won: false,
    monstersSlain: 2,
    componentsContributed: 2,
    matchId: 'm_dread_01',
    createdAt: new Date(Date.now() - 6 * 3600000).toISOString(),
  },
];

export const leaderboardApi = {
  getLocalEntries(): LeaderboardEntry[] {
    try {
      const raw = localStorage.getItem(LOCAL_LEADERBOARD_KEY);
      if (!raw) {
        localStorage.setItem(LOCAL_LEADERBOARD_KEY, JSON.stringify(BASELINE_ENTRIES));
        return [...BASELINE_ENTRIES];
      }
      return JSON.parse(raw);
    } catch {
      return [...BASELINE_ENTRIES];
    }
  },

  saveLocalEntry(entry: LeaderboardEntry): void {
    try {
      const list = this.getLocalEntries();
      list.push(entry);
      localStorage.setItem(LOCAL_LEADERBOARD_KEY, JSON.stringify(list));
    } catch (e) {
      console.warn('Failed to save local leaderboard entry:', e);
    }
  },

  async fetchLeaderboard(category = 'vp', limit = 20): Promise<LeaderboardEntry[]> {
    try {
      const res = await fetch(`/api/leaderboard?category=${encodeURIComponent(category)}&limit=${limit}`);
      if (res.ok) {
        const remoteList: LeaderboardEntry[] = await res.json();
        if (remoteList && remoteList.length > 0) {
          return remoteList;
        }
      }
    } catch {
      // Offline fallback
    }

    // Fallback local sorting
    const local = this.getLocalEntries();
    let sorted = [...local];

    if (category === 'speed' || category === 'rounds') {
      sorted = sorted
        .filter((e) => e.won)
        .sort((a, b) => (a.rounds !== b.rounds ? a.rounds - b.rounds : b.vp - a.vp));
    } else if (category === 'monsters') {
      sorted.sort((a, b) =>
        a.monstersSlain !== b.monstersSlain ? b.monstersSlain - a.monstersSlain : b.vp - a.vp,
      );
    } else {
      sorted.sort((a, b) => (a.vp !== b.vp ? b.vp - a.vp : Number(b.won) - Number(a.won)));
    }

    return sorted.slice(0, limit);
  },

  async submitEntry(entry: Omit<LeaderboardEntry, 'id' | 'createdAt'>): Promise<void> {
    const fullEntry: LeaderboardEntry = {
      ...entry,
      id: 'lb_' + Math.random().toString(36).substring(2, 9),
      createdAt: new Date().toISOString(),
    };

    // Always record locally
    this.saveLocalEntry(fullEntry);

    // Try posting to online server
    try {
      await fetch('/api/leaderboard', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(fullEntry),
      });
    } catch {
      // Ignore network errors in hotseat/offline mode
    }
  },
};
