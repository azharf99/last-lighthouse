import { useState, useMemo, useEffect } from 'react';
import { useGameSession, type SeatConfig, type GameSession } from './session/useGameSession';
import { useOnlineSession } from './session/useOnlineSession';
import { api, type MatchSummary } from './session/api';
import { DarknessTrack } from './ui/DarknessTrack';
import { IslandMapCanvas } from './ui/map/IslandMapCanvas';
import { IslandMap } from './ui/IslandMap';
import { PlayerPanel } from './ui/PlayerPanel';
import { LighthousePanel } from './ui/LighthousePanel';
import { ActionBar } from './ui/ActionBar';
import { EventLog } from './ui/EventLog';
import { MysteryDialog } from './ui/MysteryDialog';
import { CardDeckPanel } from './ui/CardDeckPanel';
import { CombatModal } from './ui/CombatModal';
import { CharacterPickerModal } from './ui/CharacterPickerModal';
import { sfx } from './audio/sfx';
import { i18n } from './i18n';
import type { Command, PlayerID } from './types';

const MAX_HEALTH = 3;
const MAX_AP = 3;
const INVENTORY_CAPACITY = 6;
const DARKNESS_MAX = 8;

const DEFAULT_SEATS: SeatConfig[] = [
  { id: 'p1', name: 'Ana', character: 'navigator' },
  { id: 'p2', name: 'Budi', character: 'engineer' },
  { id: 'p3', name: 'Citra', character: 'hunter' },
];

export default function App() {
  const [playMode, setPlayMode] = useState<'hotseat' | 'online'>('hotseat');
  const [onlineMatchId, setOnlineMatchId] = useState<string | null>(null);
  const [myPlayerId, setMyPlayerId] = useState<PlayerID>('p1');
  const [authToken, setAuthToken] = useState<string>('');
  const [displayName, setDisplayName] = useState<string>(() => {
    return localStorage.getItem('llh_name') || 'Pemain 1';
  });

  // UI Settings & Modals
  const [useCanvasMap, setUseCanvasMap] = useState<boolean>(true);
  const [isMuted, setIsMuted] = useState<boolean>(sfx.isMuted());
  const [lang, setLang] = useState<'id' | 'en'>(i18n.getLang());
  const [showSetupModal, setShowSetupModal] = useState<boolean>(false);
  const [showCombatModal, setShowCombatModal] = useState<boolean>(false);

  // Active seats
  const [seats, setSeats] = useState<SeatConfig[]>(DEFAULT_SEATS);

  // Lobby state
  const [lobbies, setLobbies] = useState<MatchSummary[]>([]);
  const [lobbyLoading, setLobbyLoading] = useState(false);
  const [lobbyError, setLobbyError] = useState<string | null>(null);

  const seed = useMemo(() => 20260829, []);

  // Sessions
  const hotseatGame = useGameSession(seats, seed);
  const onlineGame = useOnlineSession(onlineMatchId || '', myPlayerId, authToken);

  const game: GameSession = playMode === 'online' && onlineMatchId ? onlineGame : hotseatGame;

  const handleToggleSound = () => {
    const muted = sfx.toggleMute();
    setIsMuted(muted);
    if (!muted) sfx.playClick();
  };

  const handleToggleLang = () => {
    sfx.playClick();
    const nextLang = i18n.toggleLang();
    setLang(nextLang);
  };

  const handleNameChange = (name: string) => {
    setDisplayName(name);
    localStorage.setItem('llh_name', name);
  };

  const refreshLobbies = async () => {
    setLobbyLoading(true);
    setLobbyError(null);
    try {
      const list = await api.listLobbies();
      setLobbies(list);
    } catch (e) {
      setLobbyError(e instanceof Error ? e.message : String(e));
    } finally {
      setLobbyLoading(false);
    }
  };

  const handleStartWithCustomSeats = async (customSeats: SeatConfig[]) => {
    setShowSetupModal(false);
    setSeats(customSeats);

    if (playMode === 'online') {
      setLobbyLoading(true);
      setLobbyError(null);
      try {
        const auth = await api.authGuest(customSeats[0]?.name || displayName || 'Pemain');
        setAuthToken(auth.token);

        const res = await api.createMatch({
          players: customSeats,
          seed: Math.floor(Math.random() * 1_000_000),
        });

        setMyPlayerId('p1');
        setOnlineMatchId(res.matchId);
      } catch (e) {
        setLobbyError(e instanceof Error ? e.message : String(e));
      } finally {
        setLobbyLoading(false);
      }
    } else {
      hotseatGame.restart();
    }
  };

  const handleJoinOnlineMatch = async (matchId: string, seatId: PlayerID) => {
    setLobbyLoading(true);
    setLobbyError(null);
    try {
      const auth = await api.authGuest(displayName || 'Pemain');
      setAuthToken(auth.token);
      setMyPlayerId(seatId);
      setOnlineMatchId(matchId);
    } catch (e) {
      setLobbyError(e instanceof Error ? e.message : String(e));
    } finally {
      setLobbyLoading(false);
    }
  };

  useEffect(() => {
    if (playMode === 'online' && !onlineMatchId) {
      refreshLobbies();
    }
  }, [playMode, onlineMatchId]);

  // Online Lobby UI Screen
  if (playMode === 'online' && !onlineMatchId) {
    return (
      <div className="app">
        <div className="panel banner">
          <span className="banner__turn">
            <b>{i18n.t('app.title')} — Lobby Online (M2/M3)</b>
          </span>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <button className="action action--ghost" onClick={handleToggleLang}>
              {lang === 'id' ? '🇮🇩 ID' : '🇬🇧 EN'}
            </button>
            <button className="action action--ghost" onClick={handleToggleSound}>
              {isMuted ? '🔇' : '🔊'}
            </button>
            <button
              className="action action--ghost"
              onClick={() => {
                sfx.playClick();
                setPlayMode('hotseat');
              }}
            >
              {i18n.t('nav.hotseat')}
            </button>
          </div>
        </div>

        <div className="panel" style={{ maxWidth: 680, margin: '20px auto' }}>
          <h2 className="panel__title">👤 Profil Pemain</h2>
          <div style={{ display: 'flex', gap: 8, marginBottom: 16 }}>
            <input
              type="text"
              className="action"
              style={{ flex: 1, textAlign: 'left', background: 'var(--bg-card)' }}
              value={displayName}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="Nama Anda"
            />
          </div>

          <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
            <button
              className="action action--primary"
              onClick={() => {
                sfx.playClick();
                setShowSetupModal(true);
              }}
              disabled={lobbyLoading}
            >
              + Buat Match Baru (Pilih Karakter)
            </button>
            <button
              className="action"
              onClick={() => {
                sfx.playClick();
                refreshLobbies();
              }}
              disabled={lobbyLoading}
            >
              🔄 Segarkan Daftar
            </button>
          </div>

          {lobbyError && <p style={{ color: 'var(--danger)', marginBottom: 16 }}>{lobbyError}</p>}

          <h2 className="panel__title">⚔️ Daftar Match Aktif</h2>
          {lobbies.length === 0 ? (
            <p className="tiny muted">Belum ada match online. Klik tombol di atas untuk membuat match.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {lobbies.map((m) => (
                <div
                  key={m.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: 12,
                    background: 'var(--bg-card)',
                    borderRadius: 6,
                  }}
                >
                  <div>
                    <b>Room {m.id}</b> · Status: <span style={{ color: 'var(--beacon)' }}>{m.status}</span>
                    <div className="tiny muted">Pemain: {m.playerIds?.join(', ')}</div>
                  </div>
                  <div style={{ display: 'flex', gap: 6 }}>
                    {m.playerIds?.map((pid) => (
                      <button
                        key={pid}
                        className="action action--ghost"
                        style={{ padding: '4px 8px', fontSize: 12 }}
                        onClick={() => {
                          sfx.playClick();
                          handleJoinOnlineMatch(m.id, pid);
                        }}
                      >
                        Join {pid}
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <CharacterPickerModal
          initialSeats={seats}
          isOpen={showSetupModal}
          onConfirm={handleStartWithCustomSeats}
          onCancel={() => setShowSetupModal(false)}
        />
      </div>
    );
  }

  // Loading Screen
  if (game.phase === 'loading') {
    return (
      <div className="overlay">
        <div className="overlay__card">
          <h2 className="overlay__title">Menyalakan mercusuar…</h2>
          <p className="overlay__text">
            {playMode === 'online' ? 'Menghubungkan ke server WebSocket...' : 'Memuat rules engine WASM...'}
          </p>
        </div>
      </div>
    );
  }

  // Error Screen
  if (game.phase === 'error' || !game.view) {
    return (
      <div className="overlay">
        <div className="overlay__card">
          <h2 className="overlay__title">Gagal memuat</h2>
          <p className="overlay__text">{game.error ?? 'Alasan tidak diketahui.'}</p>
          {playMode === 'online' ? (
            <button className="action action--primary" onClick={() => setOnlineMatchId(null)}>
              {i18n.t('nav.back_to_lobby')}
            </button>
          ) : (
            <button className="action" onClick={() => game.restart(seed)}>
              Coba lagi
            </button>
          )}
        </div>
      </div>
    );
  }

  const st = game.view.state;
  const activeId = st.status === 'active' ? (st.turnOrder[st.activeIdx] ?? null) : null;
  const activePlayer = st.players.find((p) => p.id === activeId) ?? null;
  const over = st.status === 'won' || st.status === 'lost';

  const send = (cmd: Command) => game.send(cmd);

  return (
    <div className="app">
      {/* Top Darkness Track Bar */}
      <DarknessTrack state={st} max={DARKNESS_MAX} />

      {/* Main Header / Banner */}
      <div className="panel banner">
        <span className="banner__turn">
          {over ? (
            <b>{st.status === 'won' ? i18n.t('status.won') : i18n.t('status.lost')}</b>
          ) : (
            <>
              {playMode === 'online' ? (
                <span>
                  Room <b>{onlineMatchId}</b> ({i18n.t('nav.online')}) ·{' '}
                </span>
              ) : null}
              {i18n.t('status.turn_of')} <b>{activePlayer?.name}</b> · {activePlayer?.ap ?? 0} {i18n.t('status.ap_left')}
            </>
          )}
        </span>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
          {/* Controls: Audio, Language, Canvas Mode */}
          <button className="action action--ghost" onClick={handleToggleLang} title="Ganti Bahasa">
            {lang === 'id' ? '🇮🇩 ID' : '🇬🇧 EN'}
          </button>
          <button className="action action--ghost" onClick={handleToggleSound} title="Toggle Audio">
            {isMuted ? '🔇' : '🔊'}
          </button>
          <button
            className="action action--ghost"
            onClick={() => {
              sfx.playClick();
              setUseCanvasMap((v) => !v);
            }}
            title="Toggle PixiJS Canvas / SVG Map"
          >
            {useCanvasMap ? '🎨 WebGL' : '📐 SVG'}
          </button>

          {playMode === 'online' ? (
            <button
              className="action action--ghost"
              onClick={() => {
                sfx.playClick();
                setOnlineMatchId(null);
              }}
            >
              {i18n.t('nav.back_to_lobby')}
            </button>
          ) : (
            <>
              <button
                className="action action--ghost"
                onClick={() => {
                  sfx.playClick();
                  setShowSetupModal(true);
                }}
              >
                ⚙️ Karakter
              </button>
              <button
                className="action action--ghost"
                onClick={() => {
                  sfx.playClick();
                  setPlayMode('online');
                  setOnlineMatchId(null);
                }}
              >
                {i18n.t('nav.online')}
              </button>
              <button
                className="action action--ghost"
                onClick={() => {
                  sfx.playClick();
                  game.restart();
                }}
              >
                {i18n.t('nav.new_game')}
              </button>
            </>
          )}
        </div>
      </div>

      {/* Center 2D Island Map (PixiJS Canvas or SVG fallback) */}
      {useCanvasMap ? (
        <IslandMapCanvas
          state={st}
          legal={game.legal}
          activePlayer={activeId}
          onMove={(to) => activeId && send({ kind: 'move', player: activeId, to })}
          onExplore={(to) => activeId && send({ kind: 'explore', player: activeId, to })}
        />
      ) : (
        <IslandMap
          state={st}
          legal={game.legal}
          activePlayer={activeId}
          onMove={(to) => activeId && send({ kind: 'move', player: activeId, to })}
          onExplore={(to) => activeId && send({ kind: 'explore', player: activeId, to })}
        />
      )}

      {/* Action Bar */}
      {!over && (
        <div className="panel">
          <h2 className="panel__title">⚡ {i18n.t('status.turn_of')} {activePlayer?.name} — Aksi</h2>
          <ActionBar
            state={st}
            legal={game.legal}
            disabled={game.busy}
            onSend={send}
            onFight={() => setShowCombatModal(true)}
          />
        </div>
      )}

      {/* Card Decks Section (GDD §8.3) */}
      <CardDeckPanel state={st} />

      {/* Player Inventory & Lighthouse Panels */}
      <div className="app__lower">
        <PlayerPanel
          view={game.view}
          maxHealth={MAX_HEALTH}
          maxAP={MAX_AP}
          inventoryCapacity={INVENTORY_CAPACITY}
        />
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--gap)', minHeight: 0 }}>
          <LighthousePanel state={st} />
          <EventLog log={game.log} state={st} />
        </div>
      </div>

      {/* Mystery Dilemma Dialog */}
      {st.pending && !over && !game.awaitingHandoff && (
        <MysteryDialog
          pending={st.pending}
          viewer={game.view.viewer}
          playerName={st.players.find((p) => p.id === st.pending?.player)?.name ?? ''}
          legal={game.legal}
          disabled={game.busy}
          onSend={send}
        />
      )}

      {/* 1D6 Monster Combat Modal */}
      {showCombatModal && activePlayer && (
        <CombatModal
          isOpen={showCombatModal}
          player={activePlayer.id}
          character={activePlayer.character}
          onSend={send}
          onClose={() => setShowCombatModal(false)}
        />
      )}

      {/* Character Picker & Setup Modal */}
      <CharacterPickerModal
        initialSeats={seats}
        isOpen={showSetupModal}
        onConfirm={handleStartWithCustomSeats}
        onCancel={() => setShowSetupModal(false)}
      />

      {/* Pass-and-Play Device Handoff Screen */}
      {game.awaitingHandoff && !over && (
        <div className="overlay">
          <div className="overlay__card">
            <h2 className="overlay__title">{i18n.t('handoff.title')}</h2>
            <p className="overlay__text">
              {i18n.t('handoff.prompt')} <b>{st.players.find((p) => p.id === game.handoffTo)?.name}</b>.
              <br />
              <span className="tiny">{i18n.t('handoff.subtext')}</span>
            </p>
            <button
              className="action action--primary"
              onClick={() => {
                sfx.playClick();
                game.confirmHandoff();
              }}
            >
              {i18n.t('handoff.confirm')}
            </button>
          </div>
        </div>
      )}

      {/* Game Over Screen */}
      {over && (
        <div className="overlay">
          <div className="overlay__card">
            <h2 className="overlay__title">
              {st.status === 'won' ? i18n.t('status.won') : i18n.t('status.lost')}
            </h2>
            <p className="overlay__text">
              {st.status === 'won'
                ? 'Kelima komponen mercusuar telah menyala sempurna. Skor akhir dihitung:'
                : `Darkness mencapai ${DARKNESS_MAX} di ronde ${st.round}. Tidak ada pemenang.`}
            </p>

            {st.status === 'won' && (
              <div style={{ textAlign: 'left', marginBottom: 16 }}>
                {[...st.players]
                  .sort((a, b) => b.vp - a.vp)
                  .map((p, i) => (
                    <div key={p.id} className="seat__head" style={{ padding: '4px 0' }}>
                      <span>
                        {i === 0 ? '👑 ' : ''}
                        {p.name} ({p.character})
                      </span>
                      <b style={{ color: 'var(--beacon)' }}>{p.vp} VP</b>
                    </div>
                  ))}
              </div>
            )}

            {playMode === 'online' ? (
              <button
                className="action action--primary"
                onClick={() => {
                  sfx.playClick();
                  setOnlineMatchId(null);
                }}
              >
                {i18n.t('nav.back_to_lobby')}
              </button>
            ) : (
              <button
                className="action action--primary"
                onClick={() => {
                  sfx.playClick();
                  game.restart();
                }}
              >
                Main Lagi
              </button>
            )}
          </div>
        </div>
      )}

      {/* Rejection Toast */}
      {game.rejection && (
        <div className="toast" role="alert" onClick={game.dismissRejection}>
          {game.rejection}
        </div>
      )}
    </div>
  );
}
