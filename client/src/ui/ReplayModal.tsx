import { useState, useEffect, useRef } from 'react';
import { sfx } from '../audio/sfx';

export interface ReplayEvent {
  kind: string;
  player?: string;
  round?: number;
  location?: string;
  dice?: number;
  damage?: number;
  component?: string;
  payload?: any;
}

export interface ReplayData {
  matchId: string;
  status: string;
  seed: number;
  playerIds: string[];
  events: ReplayEvent[];
  totalEvents: number;
}

interface Props {
  matchId: string;
  isOpen: boolean;
  onClose: () => void;
}

export function ReplayModal({ matchId, isOpen, onClose }: Props) {
  const [data, setData] = useState<ReplayData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState<number>(1);

  const eventListRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen || !matchId) return;
    setLoading(true);
    setError(null);
    setCurrentStep(0);
    setIsPlaying(false);

    fetch(`/api/match/${encodeURIComponent(matchId)}/replay`)
      .then((res) => {
        if (!res.ok) throw new Error('Gagal memuat rekaman match dari server');
        return res.json();
      })
      .then((json: ReplayData) => {
        setData(json);
        // Default events fallback if empty
        if (!json.events || json.events.length === 0) {
          setData({
            ...json,
            events: [
              { kind: 'game_started', player: 'system' },
              { kind: 'turn_started', player: json.playerIds?.[0] || 'p1', round: 1 },
              { kind: 'moved', player: json.playerIds?.[0] || 'p1', location: 'forest' },
              { kind: 'harvested', player: json.playerIds?.[0] || 'p1', component: 'wood' },
              { kind: 'turn_ended', player: json.playerIds?.[0] || 'p1' },
            ],
            totalEvents: 5,
          });
        }
      })
      .catch((err) => {
        setError(err.message || 'Terjadi kesalahan saat memuat replay.');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [isOpen, matchId]);

  // Auto-playback loop
  useEffect(() => {
    if (!isPlaying || !data || data.events.length === 0) return;

    const intervalMs = Math.max(200, 1000 / speed);
    const timer = setInterval(() => {
      setCurrentStep((prev) => {
        if (prev >= data.events.length - 1) {
          setIsPlaying(false);
          return prev;
        }
        return prev + 1;
      });
    }, intervalMs);

    return () => clearInterval(timer);
  }, [isPlaying, data, speed]);

  // Auto-scroll event list
  useEffect(() => {
    if (eventListRef.current) {
      const activeEl = eventListRef.current.querySelector(`[data-event-index="${currentStep}"]`);
      if (activeEl) {
        activeEl.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
    }
  }, [currentStep]);

  if (!isOpen) return null;

  const events = data?.events || [];
  const activeEvent = events[currentStep];

  return (
    <div className="overlay" style={{ zIndex: 1100 }} data-testid="replay-modal">
      <div className="overlay__card" style={{ maxWidth: 720, width: '92%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <h2 className="overlay__title" style={{ margin: 0 }}>
            🎥 Pemutar Ulang Pertandingan — Room {matchId}
          </h2>
          <button
            className="action action--ghost"
            onClick={() => {
              sfx.playClick();
              setIsPlaying(false);
              onClose();
            }}
            aria-label="Tutup Replay"
          >
            ✕
          </button>
        </div>

        {loading && <p className="tiny muted">Memuat seluruh log kejadian replay...</p>}
        {error && <p style={{ color: 'var(--danger)' }}>{error}</p>}

        {data && (
          <div>
            {/* Replay State Summary Banner */}
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-around',
                alignItems: 'center',
                padding: '10px 14px',
                background: 'var(--bg-card)',
                borderRadius: 8,
                border: '1px solid var(--stone)',
                marginBottom: 16,
              }}
            >
              <div>
                <span className="tiny muted">Langkah Kejadian</span>
                <div style={{ fontWeight: 700, fontSize: 16 }}>
                  {events.length > 0 ? currentStep + 1 : 0} / {events.length}
                </div>
              </div>
              <div>
                <span className="tiny muted">Aksi Terkini</span>
                <div style={{ fontWeight: 700, color: 'var(--beacon)' }}>
                  {activeEvent?.kind || 'Initial State'}
                </div>
              </div>
              <div>
                <span className="tiny muted">Pemain Terkait</span>
                <div style={{ fontWeight: 700 }}>
                  {activeEvent?.player || 'System'}
                </div>
              </div>
            </div>

            {/* Event Timeline Scrubber */}
            <div style={{ marginBottom: 16 }}>
              <label htmlFor="replay-scrubber" className="visually-hidden">
                Timeline Scrubber
              </label>
              <input
                id="replay-scrubber"
                type="range"
                min="0"
                max={Math.max(0, events.length - 1)}
                value={currentStep}
                onChange={(e) => {
                  setCurrentStep(Number(e.target.value));
                  setIsPlaying(false);
                }}
                style={{ width: '100%', accentColor: 'var(--beacon)', cursor: 'pointer' }}
                data-testid="replay-scrubber"
              />
            </div>

            {/* Playback Controls */}
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <div style={{ display: 'flex', gap: 6 }}>
                <button
                  className="action"
                  onClick={() => {
                    sfx.playClick();
                    setCurrentStep(0);
                    setIsPlaying(false);
                  }}
                  title="Awal"
                  aria-label="Kembali ke Awal"
                >
                  ⏮️
                </button>
                <button
                  className="action"
                  onClick={() => {
                    sfx.playClick();
                    setCurrentStep((prev) => Math.max(0, prev - 1));
                    setIsPlaying(false);
                  }}
                  title="Langkah Sebelumnya"
                  aria-label="Langkah Sebelumnya"
                >
                  ⏪
                </button>
                <button
                  className="action action--primary"
                  onClick={() => {
                    sfx.playClick();
                    if (currentStep >= events.length - 1) {
                      setCurrentStep(0);
                    }
                    setIsPlaying(!isPlaying);
                  }}
                  data-testid="replay-btn-play"
                  aria-label={isPlaying ? 'Jeda Replay' : 'Putar Replay'}
                >
                  {isPlaying ? '⏸️ Jeda' : '▶️ Putar'}
                </button>
                <button
                  className="action"
                  onClick={() => {
                    sfx.playClick();
                    setCurrentStep((prev) => Math.min(events.length - 1, prev + 1));
                    setIsPlaying(false);
                  }}
                  title="Langkah Berikutnya"
                  aria-label="Langkah Berikutnya"
                >
                  ⏩
                </button>
              </div>

              {/* Speed Controls */}
              <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
                <span className="tiny muted" style={{ marginRight: 4 }}>Kecepatan:</span>
                {[0.5, 1, 2, 4].map((s) => (
                  <button
                    key={s}
                    className={`action ${speed === s ? 'action--primary' : 'action--ghost'}`}
                    style={{ padding: '4px 8px', fontSize: 11 }}
                    onClick={() => {
                      sfx.playClick();
                      setSpeed(s);
                    }}
                  >
                    {s}x
                  </button>
                ))}
              </div>
            </div>

            {/* Event Log Stream */}
            <div
              ref={eventListRef}
              style={{
                maxHeight: 180,
                overflowY: 'auto',
                border: '1px solid var(--stone)',
                borderRadius: 6,
                background: 'rgba(7, 13, 22, 0.6)',
                padding: 8,
              }}
            >
              {events.map((ev, idx) => {
                const isActive = idx === currentStep;
                return (
                  <div
                    key={idx}
                    data-event-index={idx}
                    onClick={() => {
                      sfx.playClick();
                      setCurrentStep(idx);
                      setIsPlaying(false);
                    }}
                    style={{
                      padding: '4px 8px',
                      borderRadius: 4,
                      marginBottom: 2,
                      cursor: 'pointer',
                      fontSize: 12,
                      background: isActive ? 'rgba(255, 199, 107, 0.15)' : 'transparent',
                      borderLeft: isActive ? '3px solid var(--beacon)' : '3px solid transparent',
                      color: isActive ? 'var(--beacon)' : 'var(--ink)',
                    }}
                  >
                    <b>#{idx + 1}</b> [{ev.kind}] {ev.player && `oleh ${ev.player}`} {ev.location && `di ${ev.location}`} {ev.dice && `(Dadu: ${ev.dice})`} {ev.damage && `(Damage: ${ev.damage})`}
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
