import { useState, useMemo, useEffect, useCallback, useRef } from 'react';
import { useGameSession, type SeatConfig, type GameSession } from './session/useGameSession';
import { useOnlineSession } from './session/useOnlineSession';
import { api, type MatchSummary } from './session/api';
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
import { MatchHubPanel } from './ui/MatchHubPanel';
import { ReplayModal } from './ui/ReplayModal';
import { TelemetryModal } from './ui/TelemetryModal';
import { TutorialModal } from './ui/TutorialModal';
import { AchievementsModal } from './ui/AchievementsModal';
import { AchievementToast } from './ui/AchievementToast';
import { achievementManager } from './achievements/achievements';
import { leaderboardApi } from './session/leaderboard';
import { FloatingHUD } from './ui/FloatingHUD';
import { PixelModal } from './ui/PixelModal';
import { useKeyboardNav } from './ui/map/useKeyboardNav';
import { registerServiceWorker } from './session/pushManager';
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
  const [showTutorialModal, setShowTutorialModal] = useState<boolean>(false);
  const [showTelemetryModal, setShowTelemetryModal] = useState<boolean>(false);
  const [showAchievementsModal, setShowAchievementsModal] = useState<boolean>(false);
  const [achievementsTab, setAchievementsTab] = useState<'achievements' | 'leaderboard' | 'stats'>('achievements');
  const [replayMatchId, setReplayMatchId] = useState<string | null>(null);

  const gameEndEvaluatedRef = useRef<string | null>(null);

  // Retro HUD modal states
  const [showActionsModal, setShowActionsModal] = useState<boolean>(false);
  const [showPlayersModal, setShowPlayersModal] = useState<boolean>(false);
  const [showLighthouseModal, setShowLighthouseModal] = useState<boolean>(false);
  const [showLogModal, setShowLogModal] = useState<boolean>(false);
  const [showCardsModal, setShowCardsModal] = useState<boolean>(false);
  const [showSettingsModal, setShowSettingsModal] = useState<boolean>(false);

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
    registerServiceWorker();
  }, []);

  useEffect(() => {
    if (playMode === 'online' && !onlineMatchId) {
      refreshLobbies();
    }
  }, [playMode, onlineMatchId]);

  // Game state shortcuts
  const isInGame = game.phase === 'ready' && game.view && !game.awaitingHandoff;
  const st = game.view?.state;
  const activeId = st?.status === 'active' ? (st.turnOrder[st.activeIdx] ?? null) : null;
  const over = st?.status === 'won' || st?.status === 'lost';

  // Listen to live events for achievement tracking
  useEffect(() => {
    if (game.log && game.log.length > 0 && st) {
      achievementManager.processEvents(game.log, st, myPlayerId || activeId);
    }
  }, [game.log, st, myPlayerId, activeId]);

  // Handle game end achievements & leaderboard submission
  useEffect(() => {
    if (!st) return;
    const isOver = st.status === 'won' || st.status === 'lost';
    const matchKey = `${st.matchId || 'match'}-${st.round}-${st.darkness}-${st.status}`;

    if (isOver && gameEndEvaluatedRef.current !== matchKey) {
      gameEndEvaluatedRef.current = matchKey;
      // Process game end achievements
      achievementManager.processGameEnd(st, myPlayerId || activeId);

      // Submit all player scores to leaderboard
      st.players.forEach((p) => {
        void leaderboardApi.submitEntry({
          playerName: p.name || displayName || 'Pemain',
          character: p.character,
          vp: p.vp,
          darkness: st.darkness,
          rounds: st.round,
          won: st.status === 'won',
          monstersSlain: p.monstersSlain || 0,
          componentsContributed: p.repairsJoined || 0,
          matchId: st.matchId || 'match',
        });
      });
    }
  }, [st, myPlayerId, activeId, displayName]);

  // Close all HUD modals helper
  const closeAllModals = useCallback(() => {
    setShowActionsModal(false);
    setShowPlayersModal(false);
    setShowLighthouseModal(false);
    setShowLogModal(false);
    setShowCardsModal(false);
    setShowSettingsModal(false);
    setShowAchievementsModal(false);
  }, []);

  const anyModalOpen =
    showActionsModal || showPlayersModal || showLighthouseModal ||
    showLogModal || showCardsModal || showSettingsModal ||
    showCombatModal || showSetupModal || showTutorialModal || showTelemetryModal || showAchievementsModal;

  const keyboardCallbacks = useMemo(
    () => ({
      onMove: (to: string) => {
        if (activeId) {
          sfx.playMove();
          game.send({ kind: 'move', player: activeId, to });
        }
      },
      onExplore: (to: string) => {
        if (activeId) {
          sfx.playMove();
          game.send({ kind: 'explore', player: activeId, to });
        }
      },
      onOpenActions: () => setShowActionsModal(true),
      onCloseModal: () => closeAllModals(),
    }),
    [activeId, game, closeAllModals],
  );

  useKeyboardNav(
    st ?? null,
    game.legal,
    activeId,
    Boolean(isInGame && !over && !anyModalOpen),
    keyboardCallbacks,
  );

  const send = (cmd: Command) => game.send(cmd);

  // ========================================
  // Online Lobby UI (uses app--lobby class)
  // ========================================
  if (playMode === 'online' && !onlineMatchId) {
    return (
      <div className="app app--lobby">
        <div className="panel banner">
          <span className="banner__turn">
            <b>{i18n.t('app.title')} — Lobby Online</b>
          </span>
          <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
            <button className="action action--ghost" onClick={() => { sfx.playClick(); setAchievementsTab('achievements'); setShowAchievementsModal(true); }} aria-label="🏆 Pencapaian" data-testid="btn-achievements">🏆</button>
            <button className="action action--ghost" onClick={() => { sfx.playClick(); setShowTutorialModal(true); }} aria-label="📖 Panduan" data-testid="btn-tutorial">📖</button>
            <button className="action action--ghost" onClick={() => { sfx.playClick(); setShowTelemetryModal(true); }} aria-label="📊 Statistik" data-testid="btn-telemetry">📊</button>
            <button className="action action--ghost" onClick={handleToggleLang} aria-label={lang === 'id' ? '🇮🇩 ID' : '🇬🇧 EN'}>{lang === 'id' ? '🇮🇩' : '🇬🇧'}</button>
            <button className="action action--ghost" onClick={handleToggleSound} aria-label={isMuted ? '🔇' : '🔊'}>{isMuted ? '🔇' : '🔊'}</button>
            <button className="action action--ghost" onClick={() => { sfx.playClick(); setPlayMode('hotseat'); }} aria-label={i18n.t('nav.hotseat')}>{i18n.t('nav.hotseat')}</button>
          </div>
        </div>

        <div className="panel" style={{ maxWidth: 680, margin: '16px auto' }} data-testid="lobby-panel">
          <h2 className="panel__title">👤 Profil Pemain</h2>
          <div style={{ display: 'flex', gap: 6, marginBottom: 12 }}>
            <label htmlFor="player-name-input" className="visually-hidden">Nama Profil Pemain</label>
            <input
              id="player-name-input"
              type="text"
              className="action"
              style={{ flex: 1, textAlign: 'left', background: 'var(--pixel-dark)' }}
              value={displayName}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="Nama Anda"
              data-testid="lobby-input-name"
            />
          </div>

          <div style={{ display: 'flex', gap: 6, marginBottom: 20, flexWrap: 'wrap' }}>
            <button className="action action--primary" onClick={() => { sfx.playClick(); setShowSetupModal(true); }} disabled={lobbyLoading} data-testid="lobby-btn-create">
              + Buat Match Baru
            </button>
            <button className="action" onClick={() => { sfx.playClick(); refreshLobbies(); }} disabled={lobbyLoading} data-testid="lobby-btn-refresh">
              🔄 Segarkan
            </button>
          </div>

          {lobbyError && <p style={{ color: 'var(--pixel-red)', marginBottom: 12 }}>{lobbyError}</p>}

          <h2 className="panel__title">⚔️ Match Aktif</h2>
          {lobbies.length === 0 ? (
            <p className="tiny muted">Belum ada match online.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              {lobbies.map((m) => (
                <div key={m.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: 10, background: 'var(--pixel-dark)', border: '2px solid var(--pixel-gray)' }}>
                  <div>
                    <b>Room {m.id}</b> · <span style={{ color: 'var(--pixel-beacon)' }}>{m.status}</span>
                    <div className="tiny muted">Pemain: {m.playerIds?.join(', ')}</div>
                  </div>
                  <div style={{ display: 'flex', gap: 4 }}>
                    {m.playerIds?.map((pid) => (
                      <button key={pid} className="action action--ghost" style={{ padding: '3px 6px', fontSize: 8 }} onClick={() => { sfx.playClick(); handleJoinOnlineMatch(m.id, pid); }}>
                        Join {pid}
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <MatchHubPanel playerId={myPlayerId} authToken={authToken} onJoinMatch={handleJoinOnlineMatch} onOpenReplay={(mid) => setReplayMatchId(mid)} />

        <CharacterPickerModal initialSeats={seats} isOpen={showSetupModal} onConfirm={handleStartWithCustomSeats} onCancel={() => setShowSetupModal(false)} />
        <AchievementsModal isOpen={showAchievementsModal} onClose={() => setShowAchievementsModal(false)} defaultTab={achievementsTab} />
        <TutorialModal isOpen={showTutorialModal} onClose={() => setShowTutorialModal(false)} />
        <TelemetryModal isOpen={showTelemetryModal} onClose={() => setShowTelemetryModal(false)} />
        <ReplayModal matchId={replayMatchId || ''} isOpen={!!replayMatchId} onClose={() => setReplayMatchId(null)} />
        <AchievementToast />
      </div>
    );
  }

  // ========================================
  // Loading Screen
  // ========================================
  if (game.phase === 'loading') {
    return (
      <div className="overlay">
        <div className="overlay__card">
          <h2 className="overlay__title">Menyalakan mercusuar…</h2>
          <p className="overlay__text">
            {playMode === 'online' ? 'Menghubungkan ke server...' : 'Memuat rules engine WASM...'}
          </p>
        </div>
      </div>
    );
  }

  // ========================================
  // Error Screen
  // ========================================
  if (game.phase === 'error' || !game.view) {
    return (
      <div className="overlay">
        <div className="overlay__card">
          <h2 className="overlay__title">Gagal memuat</h2>
          <p className="overlay__text">{game.error ?? 'Alasan tidak diketahui.'}</p>
          {playMode === 'online' ? (
            <button className="action action--primary" onClick={() => setOnlineMatchId(null)}>{i18n.t('nav.back_to_lobby')}</button>
          ) : (
            <button className="action" onClick={() => game.restart(seed)}>Coba lagi</button>
          )}
        </div>
      </div>
    );
  }

  // ========================================
  // MAIN GAMEPLAY — Fullscreen Map + Floating HUD
  // ========================================
  const activePlayer = st!.players.find((p) => p.id === activeId) ?? null;
  const isGameOver = over ?? false;

  return (
    <div className="app">
      <h1 className="visually-hidden">The Last Lighthouse — Menyalakan Mercusuar Terakhir</h1>

      {/* Fullscreen Pixel-Art Canvas Map */}
      {useCanvasMap ? (
        <IslandMapCanvas
          state={st!}
          legal={game.legal}
          activePlayer={activeId}
          onMove={(to) => activeId && send({ kind: 'move', player: activeId, to })}
          onExplore={(to) => activeId && send({ kind: 'explore', player: activeId, to })}
        />
      ) : (
        <IslandMap
          state={st!}
          legal={game.legal}
          activePlayer={activeId}
          onMove={(to) => activeId && send({ kind: 'move', player: activeId, to })}
          onExplore={(to) => activeId && send({ kind: 'explore', player: activeId, to })}
        />
      )}

      {/* Floating HUD Overlay */}
      <FloatingHUD
        state={st!}
        activePlayer={activeId}
        darknessMax={DARKNESS_MAX}
        isMuted={isMuted}
        lang={lang}
        gameOver={isGameOver}
        legal={game.legal}
        onToggleSound={handleToggleSound}
        onToggleLang={handleToggleLang}
        onOpenTutorial={() => setShowTutorialModal(true)}
        onOpenTelemetry={() => setShowTelemetryModal(true)}
        onOpenAchievements={() => { sfx.playClick(); setAchievementsTab('achievements'); setShowAchievementsModal(true); }}
        onOpenSettings={() => setShowSettingsModal(true)}
        onOpenActions={() => setShowActionsModal(true)}
        onOpenLighthouse={() => setShowLighthouseModal(true)}
        onOpenLog={() => setShowLogModal(true)}
        onOpenCards={() => setShowCardsModal(true)}
        onOpenPlayers={() => setShowPlayersModal(true)}
        onNewGame={() => { sfx.playClick(); game.restart(); }}
        onSwitchOnline={() => { sfx.playClick(); setPlayMode('online'); setOnlineMatchId(null); }}
      />

      {/* ===== PIXEL MODALS (opened from floating HUD buttons) ===== */}

      {/* Actions Modal */}
      <PixelModal isOpen={showActionsModal} onClose={() => setShowActionsModal(false)} title={`⚡ Aksi — ${activePlayer?.name ?? ''}`}>
        {!isGameOver && (
          <ActionBar
            state={st!}
            legal={game.legal}
            disabled={game.busy}
            onSend={(cmd) => { send(cmd); setShowActionsModal(false); }}
            onFight={() => { setShowActionsModal(false); setShowCombatModal(true); }}
          />
        )}
      </PixelModal>

      {/* Players Modal */}
      <PixelModal isOpen={showPlayersModal} onClose={() => setShowPlayersModal(false)} title="👤 Pemain" wide>
        <PlayerPanel view={game.view} maxHealth={MAX_HEALTH} maxAP={MAX_AP} inventoryCapacity={INVENTORY_CAPACITY} />
      </PixelModal>

      {/* Lighthouse Modal */}
      <PixelModal isOpen={showLighthouseModal} onClose={() => setShowLighthouseModal(false)} title="🏰 Mercusuar">
        <LighthousePanel state={st!} />
      </PixelModal>

      {/* Event Log Modal */}
      <PixelModal isOpen={showLogModal} onClose={() => setShowLogModal(false)} title="📜 Catatan" wide>
        <EventLog log={game.log} state={st!} />
      </PixelModal>

      {/* Card Decks Modal */}
      <PixelModal isOpen={showCardsModal} onClose={() => setShowCardsModal(false)} title="🎴 Kartu" wide>
        <CardDeckPanel state={st!} />
      </PixelModal>

      {/* Settings Modal */}
      <PixelModal isOpen={showSettingsModal} onClose={() => setShowSettingsModal(false)} title="⚙️ Pengaturan">
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <button className="action" onClick={() => { setUseCanvasMap((v) => !v); sfx.playClick(); }}>
            {useCanvasMap ? '🎨 Peta: WebGL Canvas' : '📐 Peta: SVG Vektor'}
          </button>
          <button className="action" onClick={() => { sfx.playClick(); setShowSettingsModal(false); setShowSetupModal(true); }}>
            ⚙️ Pilih Karakter
          </button>
          {playMode === 'hotseat' && (
            <>
              <button className="action" onClick={() => { sfx.playClick(); setPlayMode('online'); setOnlineMatchId(null); setShowSettingsModal(false); }}>
                🌐 {i18n.t('nav.online')}
              </button>
              <button className="action action--primary" onClick={() => { sfx.playClick(); game.restart(); setShowSettingsModal(false); }}>
                🔄 {i18n.t('nav.new_game')}
              </button>
            </>
          )}
          {playMode === 'online' && (
            <button className="action" onClick={() => { sfx.playClick(); setOnlineMatchId(null); setShowSettingsModal(false); }}>
              ← {i18n.t('nav.back_to_lobby')}
            </button>
          )}
        </div>
      </PixelModal>

      {/* ===== EXISTING GAME MODALS (unchanged logic) ===== */}

      {/* Mystery Dilemma Dialog */}
      {st!.pending && !isGameOver && !game.awaitingHandoff && (
        <MysteryDialog
          pending={st!.pending}
          viewer={game.view.viewer}
          playerName={st!.players.find((p) => p.id === st!.pending?.player)?.name ?? ''}
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
      {game.awaitingHandoff && !isGameOver && (
        <div className="overlay">
          <div className="overlay__card">
            <h2 className="overlay__title">{i18n.t('handoff.title')}</h2>
            <p className="overlay__text">
              {i18n.t('handoff.prompt')} <b>{st!.players.find((p) => p.id === game.handoffTo)?.name}</b>.
              <br />
              <span className="tiny">{i18n.t('handoff.subtext')}</span>
            </p>
            <button className="action action--primary" onClick={() => { sfx.playClick(); game.confirmHandoff(); }}>
              {i18n.t('handoff.confirm')}
            </button>
          </div>
        </div>
      )}

      {/* Game Over Screen */}
      {isGameOver && (
        <div className="overlay">
          <div className="overlay__card">
            <h2 className="overlay__title">
              {st!.status === 'won' ? i18n.t('status.won') : i18n.t('status.lost')}
            </h2>
            <p className="overlay__text">
              {st!.status === 'won'
                ? 'Kelima komponen mercusuar telah menyala sempurna!'
                : `Darkness mencapai ${DARKNESS_MAX} di ronde ${st!.round}.`}
            </p>

            {st!.status === 'won' && (
              <div style={{ textAlign: 'left', marginBottom: 14 }}>
                {[...st!.players]
                  .sort((a, b) => b.vp - a.vp)
                  .map((p, i) => (
                    <div key={p.id} className="seat__head" style={{ padding: '3px 0' }}>
                      <span>{i === 0 ? '👑 ' : ''}{p.name} ({p.character})</span>
                      <b style={{ color: 'var(--pixel-beacon)' }}>{p.vp} VP</b>
                    </div>
                  ))}
              </div>
            )}

            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <button
                className="action action--ghost"
                style={{ width: '100%' }}
                onClick={() => {
                  sfx.playClick();
                  setAchievementsTab('leaderboard');
                  setShowAchievementsModal(true);
                }}
              >
                🏆 {i18n.t('ach.tab_leaderboard')} & {i18n.t('ach.tab_achievements')}
              </button>

              {playMode === 'online' ? (
                <button className="action action--primary" onClick={() => { sfx.playClick(); setOnlineMatchId(null); }}>
                  {i18n.t('nav.back_to_lobby')}
                </button>
              ) : (
                <button className="action action--primary" onClick={() => { sfx.playClick(); game.restart(); }}>
                  Main Lagi
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Rejection Toast */}
      {game.rejection && (
        <div className="toast" role="alert" onClick={game.dismissRejection}>
          {game.rejection}
        </div>
      )}

      {/* Existing Modals */}
      <AchievementsModal isOpen={showAchievementsModal} onClose={() => setShowAchievementsModal(false)} defaultTab={achievementsTab} />
      <TutorialModal isOpen={showTutorialModal} onClose={() => setShowTutorialModal(false)} />
      <TelemetryModal isOpen={showTelemetryModal} onClose={() => setShowTelemetryModal(false)} />
      <ReplayModal matchId={replayMatchId || ''} isOpen={!!replayMatchId} onClose={() => setReplayMatchId(null)} />
      <AchievementToast />
    </div>
  );
}
