import type { GameEvent, GameState, PlayerID } from '../types';
import type { Achievement, PlayerCareerStats } from './types';

export const ACHIEVEMENTS: Achievement[] = [
  {
    id: 'first_win',
    titleKey: 'ach.first_win.title',
    descKey: 'ach.first_win.desc',
    icon: '🏰',
    tier: 'gold',
    category: 'victory',
  },
  {
    id: 'clutch_win',
    titleKey: 'ach.clutch_win.title',
    descKey: 'ach.clutch_win.desc',
    icon: '⚡',
    tier: 'platinum',
    category: 'victory',
  },
  {
    id: 'speed_run',
    titleKey: 'ach.speed_run.title',
    descKey: 'ach.speed_run.desc',
    icon: '⏱️',
    tier: 'gold',
    category: 'victory',
  },
  {
    id: 'unbroken',
    titleKey: 'ach.unbroken.title',
    descKey: 'ach.unbroken.desc',
    icon: '🛡️',
    tier: 'gold',
    category: 'victory',
  },
  {
    id: 'slayer_first',
    titleKey: 'ach.slayer_first.title',
    descKey: 'ach.slayer_first.desc',
    icon: '⚔️',
    tier: 'bronze',
    category: 'combat',
  },
  {
    id: 'slayer_master',
    titleKey: 'ach.slayer_master.title',
    descKey: 'ach.slayer_master.desc',
    icon: '🗡️',
    tier: 'silver',
    category: 'combat',
  },
  {
    id: 'explorer_island',
    titleKey: 'ach.explorer_island.title',
    descKey: 'ach.explorer_island.desc',
    icon: '🗺️',
    tier: 'bronze',
    category: 'exploration',
  },
  {
    id: 'cartographer',
    titleKey: 'ach.cartographer.title',
    descKey: 'ach.cartographer.desc',
    icon: '📜',
    tier: 'silver',
    category: 'exploration',
  },
  {
    id: 'mystery_solver',
    titleKey: 'ach.mystery_solver.title',
    descKey: 'ach.mystery_solver.desc',
    icon: '🔍',
    tier: 'bronze',
    category: 'exploration',
  },
  {
    id: 'relic_collector',
    titleKey: 'ach.relic_collector.title',
    descKey: 'ach.relic_collector.desc',
    icon: '💎',
    tier: 'silver',
    category: 'exploration',
  },
  {
    id: 'master_builder',
    titleKey: 'ach.master_builder.title',
    descKey: 'ach.master_builder.desc',
    icon: '🛠️',
    tier: 'silver',
    category: 'team',
  },
  {
    id: 'team_player',
    titleKey: 'ach.team_player.title',
    descKey: 'ach.team_player.desc',
    icon: '🤝',
    tier: 'bronze',
    category: 'team',
  },
  {
    id: 'secret_objective',
    titleKey: 'ach.secret_objective.title',
    descKey: 'ach.secret_objective.desc',
    icon: '🎯',
    tier: 'gold',
    category: 'mastery',
  },
  {
    id: 'high_scorer',
    titleKey: 'ach.high_scorer.title',
    descKey: 'ach.high_scorer.desc',
    icon: '👑',
    tier: 'gold',
    category: 'mastery',
  },
  {
    id: 'four_pillars',
    titleKey: 'ach.four_pillars.title',
    descKey: 'ach.four_pillars.desc',
    icon: '🌟',
    tier: 'platinum',
    category: 'mastery',
  },
];

const STATS_KEY = 'llh_stats_v1';

const defaultStats: PlayerCareerStats = {
  totalMatches: 0,
  wins: 0,
  losses: 0,
  highestVP: 0,
  totalMonstersSlain: 0,
  totalRepairs: 0,
  charactersWon: [],
  unlockedAchievements: {},
};

type AchievementUnlockListener = (achievement: Achievement) => void;
const listeners: Set<AchievementUnlockListener> = new Set();

export const achievementManager = {
  getStats(): PlayerCareerStats {
    try {
      const raw = localStorage.getItem(STATS_KEY);
      if (!raw) return { ...defaultStats };
      return { ...defaultStats, ...JSON.parse(raw) };
    } catch {
      return { ...defaultStats };
    }
  },

  saveStats(stats: PlayerCareerStats): void {
    try {
      localStorage.setItem(STATS_KEY, JSON.stringify(stats));
    } catch (e) {
      console.warn('Failed to save career stats:', e);
    }
  },

  subscribe(listener: AchievementUnlockListener): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },

  unlock(id: string): Achievement | null {
    const stats = this.getStats();
    if (stats.unlockedAchievements[id]) {
      return null; // Already unlocked
    }

    const ach = ACHIEVEMENTS.find((a) => a.id === id);
    if (!ach) return null;

    stats.unlockedAchievements[id] = new Date().toISOString();
    this.saveStats(stats);

    // Notify listeners
    listeners.forEach((fn) => {
      try {
        fn(ach);
      } catch (err) {
        console.error('Error in achievement listener:', err);
      }
    });

    return ach;
  },

  isUnlocked(id: string): boolean {
    const stats = this.getStats();
    return Boolean(stats.unlockedAchievements[id]);
  },

  getUnlockedAt(id: string): string | null {
    const stats = this.getStats();
    return stats.unlockedAchievements[id] || null;
  },

  /**
   * Mengevaluasi aliran event saat permainan berlangsung.
   */
  processEvents(events: GameEvent[], state: GameState, activePid?: PlayerID | null): Achievement[] {
    const newlyUnlocked: Achievement[] = [];
    const trigger = (id: string) => {
      const u = this.unlock(id);
      if (u) newlyUnlocked.push(u);
    };

    const myPlayer = state.players.find((p) => p.id === activePid);

    for (const e of events) {
      // 1. Monster combat
      if (e.kind === 'monster_defeated') {
        trigger('slayer_first');
        if (myPlayer && myPlayer.monstersSlain >= 3) {
          trigger('slayer_master');
        }
      }

      // 2. Mystery resolved
      if (e.kind === 'mystery_resolved' || e.kind === 'location_investigated') {
        trigger('mystery_solver');
      }

      // 3. Trade resource
      if (e.kind === 'traded') {
        trigger('team_player');
      }

      // 4. Component repaired
      if (e.kind === 'component_repaired' || e.kind === 'repaired') {
        if (myPlayer && myPlayer.repairsJoined >= 4) {
          trigger('master_builder');
        }
      }
    }

    // State based evaluation during gameplay
    if (myPlayer) {
      if (myPlayer.explored >= 6) {
        trigger('explorer_island');
      }
      if (myPlayer.artifacts && myPlayer.artifacts.length >= 2) {
        trigger('relic_collector');
      }
      if (myPlayer.monstersSlain >= 3) {
        trigger('slayer_master');
      }
      if (myPlayer.repairsJoined >= 4) {
        trigger('master_builder');
      }
    }

    // Check if whole island is explored
    const unexploredLocs = state.board.locations.filter((l) => !l.explored);
    if (unexploredLocs.length === 0 && state.board.locations.length > 5) {
      trigger('cartographer');
    }

    return newlyUnlocked;
  },

  /**
   * Mengevaluasi kondisi saat permainan selesai (Victory / Defeat).
   */
  processGameEnd(state: GameState, pid?: PlayerID | null): Achievement[] {
    const newlyUnlocked: Achievement[] = [];
    const trigger = (id: string) => {
      const u = this.unlock(id);
      if (u) newlyUnlocked.push(u);
    };

    const stats = this.getStats();
    stats.totalMatches += 1;

    const myPlayer = state.players.find((p) => p.id === pid) || state.players[0];
    const isWon = state.status === 'won';

    if (myPlayer) {
      if (myPlayer.vp > stats.highestVP) {
        stats.highestVP = myPlayer.vp;
      }
      stats.totalMonstersSlain += myPlayer.monstersSlain || 0;
      stats.totalRepairs += myPlayer.repairsJoined || 0;

      if (myPlayer.vp >= 20) {
        trigger('high_scorer');
      }

      if (myPlayer.monstersSlain >= 3) {
        trigger('slayer_master');
      }

      if (myPlayer.repairsJoined >= 4) {
        trigger('master_builder');
      }
    }

    if (isWon) {
      stats.wins += 1;
      trigger('first_win');

      if (state.darkness >= 7) {
        trigger('clutch_win');
      }

      if (state.round <= 6) {
        trigger('speed_run');
      }

      if (myPlayer && !myPlayer.wasExhausted && myPlayer.health > 0) {
        trigger('unbroken');
      }

      // Check personal objective fulfillment: awarded VP at game end
      if (myPlayer) {
        if (!stats.charactersWon.includes(myPlayer.character)) {
          stats.charactersWon.push(myPlayer.character);
        }

        const charTypes = ['navigator', 'engineer', 'hunter', 'scholar'];
        const wonAll = charTypes.every((c) => stats.charactersWon.includes(c));
        if (wonAll) {
          trigger('four_pillars');
        }
      }
    } else {
      stats.losses += 1;
    }

    this.saveStats(stats);
    return newlyUnlocked;
  },
};
