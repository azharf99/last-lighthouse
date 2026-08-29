import { useState, useEffect } from 'react';
import { sfx } from '../audio/sfx';

export interface GlobalTelemetryStats {
  totalMatches: number;
  wins: number;
  losses: number;
  winRatePercent: number;
  avgRounds: number;
  avgDarkness: number;
  totalMonstersSlain: number;
  roleDistribution: Record<string, number>;
  lastUpdated: string;
}

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export function TelemetryModal({ isOpen, onClose }: Props) {
  const [stats, setStats] = useState<GlobalTelemetryStats | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setLoading(true);
    fetch('/api/telemetry/stats')
      .then((res) => res.json())
      .then((json: GlobalTelemetryStats) => {
        setStats(json);
      })
      .catch((err) => {
        console.warn('Failed to fetch telemetry stats:', err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [isOpen]);

  if (!isOpen) return null;

  const totalRoles = stats
    ? Object.values(stats.roleDistribution).reduce((a, b) => a + b, 0)
    : 0;

  return (
    <div className="overlay" style={{ zIndex: 1100 }} data-testid="telemetry-modal">
      <div className="overlay__card" style={{ maxWidth: 640, width: '92%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <h2 className="overlay__title" style={{ margin: 0 }}>
            📊 Dasbor Telemetri & Keseimbangan Game (M6)
          </h2>
          <button
            className="action action--ghost"
            onClick={() => {
              sfx.playClick();
              onClose();
            }}
            aria-label="Tutup Telemetri"
          >
            ✕
          </button>
        </div>

        {loading && <p className="tiny muted">Mengambil data telemetri global...</p>}

        {stats && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {/* Top Cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 10 }}>
              <div
                style={{
                  padding: 12,
                  background: 'var(--bg-card)',
                  borderRadius: 8,
                  border: '1px solid var(--stone)',
                  textAlign: 'center',
                }}
              >
                <div className="tiny muted">Tingkat Kemenangan Tim</div>
                <div
                  style={{
                    fontSize: 22,
                    fontWeight: 700,
                    color: stats.winRatePercent >= 50 ? 'var(--ok)' : 'var(--dread)',
                    marginTop: 4,
                  }}
                >
                  {stats.winRatePercent.toFixed(1)}%
                </div>
                <div className="tiny muted" style={{ fontSize: 11 }}>
                  {stats.wins} Menang / {stats.losses} Kalah
                </div>
              </div>

              <div
                style={{
                  padding: 12,
                  background: 'var(--bg-card)',
                  borderRadius: 8,
                  border: '1px solid var(--stone)',
                  textAlign: 'center',
                }}
              >
                <div className="tiny muted">Rata-rata Ronde</div>
                <div style={{ fontSize: 22, fontWeight: 700, color: 'var(--beacon)', marginTop: 4 }}>
                  {stats.avgRounds.toFixed(1)}
                </div>
                <div className="tiny muted" style={{ fontSize: 11 }}>
                  Durasi Match
                </div>
              </div>

              <div
                style={{
                  padding: 12,
                  background: 'var(--bg-card)',
                  borderRadius: 8,
                  border: '1px solid var(--stone)',
                  textAlign: 'center',
                }}
              >
                <div className="tiny muted">Monster Dikalahkan</div>
                <div style={{ fontSize: 22, fontWeight: 700, color: 'var(--dread)', marginTop: 4 }}>
                  {stats.totalMonstersSlain}
                </div>
                <div className="tiny muted" style={{ fontSize: 11 }}>
                  Total Kill
                </div>
              </div>
            </div>

            {/* Character Popularity Breakdown */}
            <div
              style={{
                padding: 14,
                background: 'var(--bg-card)',
                borderRadius: 8,
                border: '1px solid var(--stone)',
              }}
            >
              <b style={{ fontSize: 14 }}>🧭 Distribusi Pemilihan Karakter</b>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 8, marginTop: 10 }}>
                {Object.entries(stats.roleDistribution).map(([role, count]) => {
                  const percent = totalRoles > 0 ? Math.round((count / totalRoles) * 100) : 0;
                  const roleName =
                    role === 'navigator'
                      ? '🧭 The Navigator'
                      : role === 'engineer'
                      ? '⚙️ The Engineer'
                      : role === 'hunter'
                      ? '🏹 The Hunter'
                      : '📜 The Scholar';
                  return (
                    <div key={role}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, marginBottom: 3 }}>
                        <span>{roleName}</span>
                        <span>{count} kali ({percent}%)</span>
                      </div>
                      <div
                        style={{
                          height: 6,
                          width: '100%',
                          background: 'rgba(255,255,255,0.08)',
                          borderRadius: 3,
                          overflow: 'hidden',
                        }}
                      >
                        <div
                          style={{
                            height: '100%',
                            width: `${percent}%`,
                            background: 'var(--beacon)',
                            borderRadius: 3,
                          }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            <p className="tiny muted" style={{ margin: 0, textAlign: 'center' }}>
              🔒 Data dikumpulkan secara anonim tanpa menyimpan informasi pribadi (Zero-PII Telemetry).
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
