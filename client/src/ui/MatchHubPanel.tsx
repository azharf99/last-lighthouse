import { useState, useEffect } from 'react';
import { sfx } from '../audio/sfx';
import { isPushSupported, getPushPermissionState, subscribeToPush } from '../session/pushManager';

export interface MyMatchItem {
  matchId: string;
  status: string;
  playerIDs: string[];
  activePlayer?: string;
  isMyTurn: boolean;
  round: number;
  darkness: number;
  deadlineAt?: string;
  createdAt: string;
  finishedAt?: string;
}

interface Props {
  playerId: string;
  authToken: string;
  onJoinMatch: (matchId: string, playerId: string) => void;
  onOpenReplay?: (matchId: string) => void;
}

export function MatchHubPanel({ playerId, authToken, onJoinMatch, onOpenReplay }: Props) {
  const [tab, setTab] = useState<'active' | 'history' | 'notifications'>('active');
  const [myMatches, setMyMatches] = useState<MyMatchItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [pushStatus, setPushStatus] = useState<string>('default');
  const [pushMessage, setPushMessage] = useState<string | null>(null);

  const fetchMyMatches = async () => {
    if (!playerId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/matches/my?playerId=${encodeURIComponent(playerId)}`, {
        headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
      });
      if (res.ok) {
        const data = await res.json();
        setMyMatches(data || []);
      }
    } catch (err) {
      console.warn('Failed to fetch player matches:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMyMatches();
    if (isPushSupported()) {
      getPushPermissionState().then(setPushStatus);
    }
  }, [playerId, authToken]);

  const handleEnablePush = async () => {
    sfx.playClick();
    const res = await subscribeToPush(playerId, authToken);
    setPushMessage(res.message);
    if (isPushSupported()) {
      getPushPermissionState().then(setPushStatus);
    }
    setTimeout(() => setPushMessage(null), 5000);
  };

  const activeMatches = myMatches.filter((m) => m.status === 'active' || m.status === 'lobby');
  const finishedMatches = myMatches.filter((m) => m.status === 'won' || m.status === 'lost');

  const formatTimeLeft = (deadlineAt?: string) => {
    if (!deadlineAt) return null;
    const diff = new Date(deadlineAt).getTime() - Date.now();
    if (diff <= 0) return '⏰ Batas waktu habis';
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const mins = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    if (hours > 0) return `⏰ Sisa ${hours}j ${mins}m`;
    return `⏰ Sisa ${mins} menit`;
  };

  return (
    <section className="panel" style={{ maxWidth: 680, margin: '20px auto' }} data-testid="match-hub-panel">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <h2 className="panel__title" style={{ margin: 0 }}>
          🎮 Hub Permainan Asinkron & Riwayat (M5)
        </h2>
        <button
          className="action action--ghost"
          style={{ padding: '4px 8px', fontSize: 12 }}
          onClick={() => {
            sfx.playClick();
            fetchMyMatches();
          }}
          disabled={loading}
          aria-label="Segarkan Daftar Match Saya"
        >
          🔄 Segarkan
        </button>
      </div>

      {/* Hub Tabs */}
      <div style={{ display: 'flex', gap: 6, marginBottom: 16 }}>
        <button
          className={`action ${tab === 'active' ? 'action--primary' : 'action--ghost'}`}
          style={{ flex: 1, padding: '8px 12px', fontSize: 13 }}
          onClick={() => setTab('active')}
          data-testid="tab-active-matches"
        >
          ⚔️ Match Aktif ({activeMatches.length})
        </button>
        <button
          className={`action ${tab === 'history' ? 'action--primary' : 'action--ghost'}`}
          style={{ flex: 1, padding: '8px 12px', fontSize: 13 }}
          onClick={() => setTab('history')}
          data-testid="tab-match-history"
        >
          📜 Riwayat ({finishedMatches.length})
        </button>
        <button
          className={`action ${tab === 'notifications' ? 'action--primary' : 'action--ghost'}`}
          style={{ flex: 1, padding: '8px 12px', fontSize: 13 }}
          onClick={() => setTab('notifications')}
          data-testid="tab-push-settings"
        >
          🔔 Notifikasi Push
        </button>
      </div>

      {/* Tab 1: Active Matches */}
      {tab === 'active' && (
        <div>
          {activeMatches.length === 0 ? (
            <p className="tiny muted" style={{ textAlign: 'center', padding: '16px 0' }}>
              Tidak ada match aktif yang sedang berjalan. Buat match baru di atas untuk mulai bermain.
            </p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {activeMatches.map((m) => {
                const timeLeft = formatTimeLeft(m.deadlineAt);
                return (
                  <div
                    key={m.matchId}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      padding: 12,
                      background: m.isMyTurn ? 'rgba(255, 199, 107, 0.08)' : 'var(--bg-card)',
                      border: m.isMyTurn ? '1px solid var(--beacon)' : '1px solid var(--stone)',
                      borderRadius: 8,
                    }}
                    data-testid={`active-match-${m.matchId}`}
                  >
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <b>Room {m.matchId}</b>
                        {m.isMyTurn ? (
                          <span
                            style={{
                              background: 'var(--beacon)',
                              color: '#070d16',
                              fontSize: 11,
                              fontWeight: 700,
                              padding: '2px 6px',
                              borderRadius: 4,
                            }}
                          >
                            ⚡ GILIRANMU
                          </span>
                        ) : (
                          <span className="tiny muted">
                            Menunggu: {m.activePlayer || 'Pemain lain'}
                          </span>
                        )}
                      </div>
                      <div className="tiny muted" style={{ marginTop: 4 }}>
                        Ronde {m.round || 1} · Darkness {m.darkness || 1}/8 · {m.playerIDs.length} Pemain
                        {timeLeft && <span style={{ color: 'var(--beacon)', marginLeft: 8 }}>{timeLeft}</span>}
                      </div>
                    </div>

                    <button
                      className={`action ${m.isMyTurn ? 'action--primary' : ''}`}
                      style={{ padding: '6px 14px' }}
                      onClick={() => {
                        sfx.playClick();
                        onJoinMatch(m.matchId, playerId);
                      }}
                      data-testid={`btn-resume-${m.matchId}`}
                    >
                      ▶️ Lanjutkan Match
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Tab 2: Match History */}
      {tab === 'history' && (
        <div>
          {finishedMatches.length === 0 ? (
            <p className="tiny muted" style={{ textAlign: 'center', padding: '16px 0' }}>
              Belum ada riwayat match selesai.
            </p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {finishedMatches.map((m) => {
                const isWon = m.status === 'won';
                return (
                  <div
                    key={m.matchId}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      padding: 12,
                      background: 'var(--bg-card)',
                      borderRadius: 8,
                      border: '1px solid var(--stone)',
                    }}
                  >
                    <div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <b>Room {m.matchId}</b>
                        <span
                          style={{
                            color: isWon ? 'var(--ok)' : 'var(--dread)',
                            fontWeight: 700,
                            fontSize: 12,
                          }}
                        >
                          {isWon ? '🏆 MENANG' : '💀 KALAH'}
                        </span>
                      </div>
                      <div className="tiny muted" style={{ marginTop: 4 }}>
                        Selesai pada Ronde {m.round} · Darkness Akhir: {m.darkness}/8
                      </div>
                    </div>

                    {onOpenReplay && (
                      <button
                        className="action action--ghost"
                        style={{ padding: '6px 12px', fontSize: 12 }}
                        onClick={() => {
                          sfx.playClick();
                          onOpenReplay(m.matchId);
                        }}
                        data-testid={`btn-replay-${m.matchId}`}
                      >
                        🎥 Replay
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Tab 3: Notification & Async Settings */}
      {tab === 'notifications' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <p className="tiny muted" style={{ margin: 0 }}>
            Mode Asinkron (ADR-007) memungkinkan Anda bermain lintas hari dengan batas waktu giliran 24 jam.
            Aktifkan notifikasi dorong (Web Push / FCM) agar Anda mendapat pemberitahuan saat giliran Anda tiba
            meskipun peramban ditutup.
          </p>

          <div
            style={{
              padding: 12,
              background: 'var(--bg-card)',
              borderRadius: 8,
              border: '1px solid var(--stone)',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <div>
              <b>Status Notifikasi Browser</b>
              <div className="tiny muted">
                Status saat ini:{' '}
                <span
                  style={{
                    color: pushStatus === 'granted' ? 'var(--ok)' : 'var(--beacon)',
                    fontWeight: 700,
                  }}
                >
                  {pushStatus === 'granted' ? 'Aktif (Diberikan)' : pushStatus === 'denied' ? 'Ditolak' : 'Belum Diaktifkan'}
                </span>
              </div>
            </div>

            <button
              className="action action--primary"
              onClick={handleEnablePush}
              disabled={pushStatus === 'granted'}
              data-testid="btn-enable-push"
            >
              {pushStatus === 'granted' ? '✅ Notifikasi Aktif' : '🔔 Aktifkan Notifikasi'}
            </button>
          </div>

          {pushMessage && (
            <div
              style={{
                padding: 10,
                borderRadius: 6,
                background: 'rgba(111, 207, 151, 0.15)',
                color: 'var(--ok)',
                fontSize: 13,
              }}
            >
              {pushMessage}
            </div>
          )}
        </div>
      )}
    </section>
  );
}
