import { useState, useEffect, useCallback } from 'react';
import { ACHIEVEMENTS, achievementManager } from '../achievements/achievements';
import type { AchievementCategory } from '../achievements/types';
import { leaderboardApi, type LeaderboardEntry } from '../session/leaderboard';
import { CHARACTER_NAMES } from './labels';
import { sfx } from '../audio/sfx';
import { i18n } from '../i18n';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  defaultTab?: 'achievements' | 'leaderboard' | 'stats';
}

type TabType = 'achievements' | 'leaderboard' | 'stats';

export function AchievementsModal({ isOpen, onClose, defaultTab = 'achievements' }: Props) {
  const [activeTab, setActiveTab] = useState<TabType>(defaultTab);
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [leaderboardCategory, setLeaderboardCategory] = useState<string>('vp');
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  const stats = achievementManager.getStats();
  const unlockedCount = Object.keys(stats.unlockedAchievements).length;
  const totalCount = ACHIEVEMENTS.length;
  const percentUnlocked = Math.round((unlockedCount / totalCount) * 100);

  const loadLeaderboard = useCallback(async (cat: string) => {
    setLoading(true);
    try {
      const data = await leaderboardApi.fetchLeaderboard(cat, 25);
      setLeaderboard(data);
    } catch (e) {
      console.warn('Failed to load leaderboard:', e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      setActiveTab(defaultTab);
      loadLeaderboard(leaderboardCategory);
    }
  }, [isOpen, defaultTab, leaderboardCategory, loadLeaderboard]);

  if (!isOpen) return null;

  const categories: { id: string; label: string }[] = [
    { id: 'all', label: i18n.t('ach.cat.all') },
    { id: 'victory', label: i18n.t('ach.cat.victory') },
    { id: 'combat', label: i18n.t('ach.cat.combat') },
    { id: 'exploration', label: i18n.t('ach.cat.exploration') },
    { id: 'team', label: i18n.t('ach.cat.team') },
    { id: 'mastery', label: i18n.t('ach.cat.mastery') },
  ];

  const filteredAchievements = ACHIEVEMENTS.filter((a) => {
    if (selectedCategory === 'all') return true;
    return a.category === (selectedCategory as AchievementCategory);
  });

  const tierColors: Record<string, string> = {
    bronze: '#cd7f32',
    silver: '#c0c0c0',
    gold: '#ffd700',
    platinum: '#00ffff',
  };

  const getRankBadge = (idx: number) => {
    if (idx === 0) return '🥇 1';
    if (idx === 1) return '🥈 2';
    if (idx === 2) return '🥉 3';
    return `#${idx + 1}`;
  };

  const winRate =
    stats.totalMatches > 0 ? Math.round((stats.wins / stats.totalMatches) * 100) : 0;

  return (
    <div className="overlay" style={{ zIndex: 1200 }} data-testid="achievements-modal">
      <div className="overlay__card" style={{ maxWidth: 760, width: '95%', maxHeight: '90vh', overflowY: 'auto' }}>
        {/* Header */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
            <h2 className="overlay__title" style={{ margin: 0 }}>
              🏆 {i18n.t('ach.modal_title')}
            </h2>
          </div>
          <button
            className="action action--ghost"
            onClick={() => {
              sfx.playClick();
              onClose();
            }}
            aria-label="Tutup Modal"
          >
            ✕
          </button>
        </div>

        {/* Tab Navigation */}
        <div style={{ display: 'flex', gap: 6, marginBottom: 14, borderBottom: '2px solid var(--pixel-gray)', paddingBottom: 8 }}>
          <button
            className={`action ${activeTab === 'achievements' ? 'action--primary' : 'action--ghost'}`}
            style={{ fontSize: 11 }}
            onClick={() => {
              sfx.playClick();
              setActiveTab('achievements');
            }}
          >
            🏆 {i18n.t('ach.tab_achievements')} ({unlockedCount}/{totalCount})
          </button>
          <button
            className={`action ${activeTab === 'leaderboard' ? 'action--primary' : 'action--ghost'}`}
            style={{ fontSize: 11 }}
            onClick={() => {
              sfx.playClick();
              setActiveTab('leaderboard');
              loadLeaderboard(leaderboardCategory);
            }}
          >
            🥇 {i18n.t('ach.tab_leaderboard')}
          </button>
          <button
            className={`action ${activeTab === 'stats' ? 'action--primary' : 'action--ghost'}`}
            style={{ fontSize: 11 }}
            onClick={() => {
              sfx.playClick();
              setActiveTab('stats');
            }}
          >
            👤 {i18n.t('ach.tab_stats')}
          </button>
        </div>

        {/* ========================================================== */}
        {/* TAB 1: ACHIEVEMENTS GRID */}
        {/* ========================================================== */}
        {activeTab === 'achievements' && (
          <div>
            {/* Progress Bar Summary */}
            <div
              style={{
                background: 'var(--pixel-dark)',
                padding: 12,
                borderRadius: 6,
                border: '1px solid var(--pixel-gray)',
                marginBottom: 12,
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 6 }}>
                <span>
                  <b>{i18n.t('ach.progress_title')}</b>: {unlockedCount} / {totalCount} {i18n.t('ach.unlocked')}
                </span>
                <span style={{ color: 'var(--pixel-beacon)', fontWeight: 700 }}>{percentUnlocked}%</span>
              </div>
              <div
                style={{
                  height: 8,
                  width: '100%',
                  background: 'rgba(255,255,255,0.1)',
                  borderRadius: 4,
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${percentUnlocked}%`,
                    background: 'var(--pixel-beacon)',
                    borderRadius: 4,
                    transition: 'width 0.4s ease',
                  }}
                />
              </div>
            </div>

            {/* Category Filter Pills */}
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 14 }}>
              {categories.map((c) => (
                <button
                  key={c.id}
                  className={`action ${selectedCategory === c.id ? 'action--primary' : 'action--ghost'}`}
                  style={{ fontSize: 9, padding: '4px 8px' }}
                  onClick={() => {
                    sfx.playClick();
                    setSelectedCategory(c.id);
                  }}
                >
                  {c.label}
                </button>
              ))}
            </div>

            {/* Achievement Cards Grid */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 10 }}>
              {filteredAchievements.map((ach) => {
                const unlockedAt = stats.unlockedAchievements[ach.id];
                const isUnlocked = Boolean(unlockedAt);
                const tierColor = tierColors[ach.tier] || '#ffd700';

                return (
                  <div
                    key={ach.id}
                    className={`achievement-card ${isUnlocked ? 'achievement-card--unlocked' : 'achievement-card--locked'}`}
                    style={{
                      borderLeft: `4px solid ${isUnlocked ? tierColor : 'var(--pixel-gray)'}`,
                      padding: 10,
                      background: isUnlocked ? 'rgba(255, 215, 0, 0.04)' : 'var(--pixel-dark)',
                      borderRadius: 4,
                      border: '1px solid var(--pixel-gray)',
                      display: 'flex',
                      gap: 10,
                      alignItems: 'flex-start',
                    }}
                  >
                    <div
                      style={{
                        fontSize: 24,
                        filter: isUnlocked ? 'none' : 'grayscale(100%) opacity(40%)',
                        lineHeight: 1,
                      }}
                    >
                      {ach.icon}
                    </div>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span
                          style={{
                            fontSize: 11,
                            fontWeight: 700,
                            color: isUnlocked ? 'var(--pixel-beacon)' : 'var(--pixel-text-muted)',
                          }}
                        >
                          {i18n.t(ach.titleKey)}
                        </span>
                        <span
                          style={{
                            fontSize: 8,
                            padding: '1px 4px',
                            borderRadius: 2,
                            background: isUnlocked ? tierColor : '#333',
                            color: isUnlocked ? '#000' : '#888',
                            fontWeight: 700,
                            textTransform: 'uppercase',
                          }}
                        >
                          {ach.tier}
                        </span>
                      </div>
                      <p
                        className="tiny"
                        style={{
                          margin: '4px 0 0 0',
                          color: isUnlocked ? 'var(--pixel-text)' : 'var(--pixel-text-muted)',
                          fontSize: 10,
                          lineHeight: 1.3,
                        }}
                      >
                        {i18n.t(ach.descKey)}
                      </p>
                      {isUnlocked && (
                        <div className="tiny muted" style={{ fontSize: 8, marginTop: 4, color: tierColor }}>
                          ✓ {i18n.t('ach.unlocked')} · {new Date(unlockedAt).toLocaleDateString()}
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* ========================================================== */}
        {/* TAB 2: LEADERBOARD */}
        {/* ========================================================== */}
        {activeTab === 'leaderboard' && (
          <div>
            {/* Category Filter & Refresh */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12, flexWrap: 'wrap', gap: 6 }}>
              <div style={{ display: 'flex', gap: 6 }}>
                <button
                  className={`action ${leaderboardCategory === 'vp' ? 'action--primary' : 'action--ghost'}`}
                  style={{ fontSize: 9, padding: '4px 8px' }}
                  onClick={() => {
                    sfx.playClick();
                    setLeaderboardCategory('vp');
                    loadLeaderboard('vp');
                  }}
                >
                  👑 {i18n.t('lb.cat_vp')}
                </button>
                <button
                  className={`action ${leaderboardCategory === 'speed' ? 'action--primary' : 'action--ghost'}`}
                  style={{ fontSize: 9, padding: '4px 8px' }}
                  onClick={() => {
                    sfx.playClick();
                    setLeaderboardCategory('speed');
                    loadLeaderboard('speed');
                  }}
                >
                  ⚡ {i18n.t('lb.cat_speed')}
                </button>
                <button
                  className={`action ${leaderboardCategory === 'monsters' ? 'action--primary' : 'action--ghost'}`}
                  style={{ fontSize: 9, padding: '4px 8px' }}
                  onClick={() => {
                    sfx.playClick();
                    setLeaderboardCategory('monsters');
                    loadLeaderboard('monsters');
                  }}
                >
                  ⚔️ {i18n.t('lb.cat_monsters')}
                </button>
              </div>

              <button
                className="action action--ghost"
                style={{ fontSize: 9, padding: '4px 8px' }}
                onClick={() => {
                  sfx.playClick();
                  loadLeaderboard(leaderboardCategory);
                }}
                disabled={loading}
              >
                🔄 {i18n.t('lb.refresh')}
              </button>
            </div>

            {loading && <p className="tiny muted">{i18n.t('lb.loading')}</p>}

            {/* Leaderboard Table */}
            <div style={{ overflowX: 'auto', background: 'var(--pixel-dark)', border: '1px solid var(--pixel-gray)', borderRadius: 4 }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11, textAlign: 'left' }}>
                <thead>
                  <tr style={{ borderBottom: '2px solid var(--pixel-gray)', background: 'rgba(255,255,255,0.03)' }}>
                    <th style={{ padding: '8px 10px' }}>{i18n.t('lb.rank')}</th>
                    <th style={{ padding: '8px 10px' }}>{i18n.t('lb.player')}</th>
                    <th style={{ padding: '8px 10px' }}>{i18n.t('lb.role')}</th>
                    <th style={{ padding: '8px 10px', textAlign: 'center' }}>VP</th>
                    <th style={{ padding: '8px 10px', textAlign: 'center' }}>{i18n.t('lb.status')}</th>
                    <th style={{ padding: '8px 10px', textAlign: 'center' }}>{i18n.t('lb.rounds')}</th>
                    <th style={{ padding: '8px 10px', textAlign: 'right' }}>{i18n.t('lb.date')}</th>
                  </tr>
                </thead>
                <tbody>
                  {leaderboard.length === 0 ? (
                    <tr>
                      <td colSpan={7} style={{ padding: 14, textAlign: 'center', color: 'var(--pixel-text-muted)' }}>
                        {i18n.t('lb.no_entries')}
                      </td>
                    </tr>
                  ) : (
                    leaderboard.map((row, idx) => {
                      const isTop3 = idx < 3;
                      const charName = CHARACTER_NAMES[row.character] || row.character;

                      return (
                        <tr
                          key={row.id || idx}
                          style={{
                            borderBottom: '1px solid rgba(255,255,255,0.05)',
                            background: isTop3 ? 'rgba(255,215,0,0.03)' : 'transparent',
                          }}
                        >
                          <td style={{ padding: '6px 10px', fontWeight: isTop3 ? 700 : 400 }}>
                            {getRankBadge(idx)}
                          </td>
                          <td style={{ padding: '6px 10px', fontWeight: 700 }}>
                            {row.playerName}
                          </td>
                          <td style={{ padding: '6px 10px' }}>
                            <span className="tiny muted">{charName}</span>
                          </td>
                          <td style={{ padding: '6px 10px', textAlign: 'center', fontWeight: 700, color: 'var(--pixel-beacon)' }}>
                            {row.vp} VP
                          </td>
                          <td style={{ padding: '6px 10px', textAlign: 'center' }}>
                            {row.won ? (
                              <span style={{ color: 'var(--pixel-green)', fontSize: 10 }}>★ {i18n.t('lb.won')}</span>
                            ) : (
                              <span style={{ color: 'var(--pixel-red)', fontSize: 10 }}>☠ {i18n.t('lb.lost')}</span>
                            )}
                          </td>
                          <td style={{ padding: '6px 10px', textAlign: 'center', fontSize: 10 }}>
                            R{row.rounds} · D{row.darkness}
                          </td>
                          <td style={{ padding: '6px 10px', textAlign: 'right', fontSize: 9, color: 'var(--pixel-text-muted)' }}>
                            {new Date(row.createdAt).toLocaleDateString()}
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* ========================================================== */}
        {/* TAB 3: PLAYER CAREER STATS */}
        {/* ========================================================== */}
        {activeTab === 'stats' && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* Top Metric Cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
              <div
                style={{
                  padding: 12,
                  background: 'var(--pixel-dark)',
                  borderRadius: 6,
                  border: '1px solid var(--pixel-gray)',
                  textAlign: 'center',
                }}
              >
                <div className="tiny muted">{i18n.t('stats.total_matches')}</div>
                <div style={{ fontSize: 22, fontWeight: 700, color: 'var(--pixel-beacon)', marginTop: 4 }}>
                  {stats.totalMatches}
                </div>
                <div className="tiny muted" style={{ fontSize: 10 }}>
                  {stats.wins} {i18n.t('stats.wins')} / {stats.losses} {i18n.t('stats.losses')} ({winRate}%)
                </div>
              </div>

              <div
                style={{
                  padding: 12,
                  background: 'var(--pixel-dark)',
                  borderRadius: 6,
                  border: '1px solid var(--pixel-gray)',
                  textAlign: 'center',
                }}
              >
                <div className="tiny muted">{i18n.t('stats.highest_vp')}</div>
                <div style={{ fontSize: 22, fontWeight: 700, color: '#ffd700', marginTop: 4 }}>
                  {stats.highestVP} VP
                </div>
                <div className="tiny muted" style={{ fontSize: 10 }}>
                  {i18n.t('stats.personal_best')}
                </div>
              </div>

              <div
                style={{
                  padding: 12,
                  background: 'var(--pixel-dark)',
                  borderRadius: 6,
                  border: '1px solid var(--pixel-gray)',
                  textAlign: 'center',
                }}
              >
                <div className="tiny muted">{i18n.t('stats.monsters_slain')}</div>
                <div style={{ fontSize: 22, fontWeight: 700, color: 'var(--pixel-red)', marginTop: 4 }}>
                  {stats.totalMonstersSlain}
                </div>
                <div className="tiny muted" style={{ fontSize: 10 }}>
                  {stats.totalRepairs} {i18n.t('stats.repairs_joined')}
                </div>
              </div>
            </div>

            {/* Character Mastery Progress */}
            <div
              style={{
                padding: 14,
                background: 'var(--pixel-dark)',
                borderRadius: 6,
                border: '1px solid var(--pixel-gray)',
              }}
            >
              <b style={{ fontSize: 13 }}>🧭 {i18n.t('stats.character_mastery')}</b>
              <div className="tiny muted" style={{ marginBottom: 10, marginTop: 2 }}>
                {i18n.t('stats.character_mastery_sub')}
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 10 }}>
                {[
                  { id: 'navigator', name: 'Sang Navigator', icon: '🧭' },
                  { id: 'engineer', name: 'Sang Insinyur', icon: '⚙️' },
                  { id: 'hunter', name: 'Sang Pemburu', icon: '🏹' },
                  { id: 'scholar', name: 'Sang Cendekia', icon: '📜' },
                ].map((c) => {
                  const hasWon = stats.charactersWon.includes(c.id);
                  return (
                    <div
                      key={c.id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        padding: 8,
                        borderRadius: 4,
                        background: hasWon ? 'rgba(255,215,0,0.06)' : 'rgba(255,255,255,0.02)',
                        border: `1px solid ${hasWon ? '#ffd700' : 'var(--pixel-gray)'}`,
                      }}
                    >
                      <span style={{ fontSize: 20 }}>{c.icon}</span>
                      <div style={{ flex: 1 }}>
                        <div style={{ fontSize: 11, fontWeight: 700 }}>{c.name}</div>
                        <div className="tiny" style={{ color: hasWon ? 'var(--pixel-green)' : 'var(--pixel-text-muted)' }}>
                          {hasWon ? `✓ ${i18n.t('stats.victorious')}` : `○ ${i18n.t('stats.untested')}`}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
