import { sfx } from '../audio/sfx';
import type { GameState, PlayerID, Command } from '../types';
import { RESOURCES } from '../types';
import { CHARACTER_NAMES, locationLabel, RESOURCE_GLYPH } from './labels';

interface Props {
  state: GameState;
  activePlayer: PlayerID | null;
  darknessMax: number;
  isMuted: boolean;
  lang: 'id' | 'en';
  gameOver: boolean;
  legal: Command[];
  // Modal openers
  onToggleSound(): void;
  onToggleLang(): void;
  onOpenTutorial(): void;
  onOpenTelemetry(): void;
  onOpenAchievements(): void;
  onOpenSettings(): void;
  onOpenActions(): void;
  onOpenLighthouse(): void;
  onOpenLog(): void;
  onOpenCards(): void;
  onOpenPlayers(): void;
  onNewGame(): void;
  onSwitchOnline(): void;
}

export function FloatingHUD({
  state,
  activePlayer,
  darknessMax,
  isMuted,
  lang,
  gameOver,
  legal,
  onToggleSound,
  onToggleLang,
  onOpenTutorial,
  onOpenTelemetry,
  onOpenAchievements,
  onOpenSettings,
  onOpenActions,
  onOpenLighthouse,
  onOpenLog,
  onOpenCards,
  onOpenPlayers,
  onNewGame: _onNewGame,
  onSwitchOnline: _onSwitchOnline,
}: Props) {
  const player = state.players.find((p) => p.id === activePlayer);
  const repairedCount = state.lighthouse.filter((c) => c.repaired).length;
  const totalComponents = state.lighthouse.length;

  // Mini inventory summary
  const invSummary = player
    ? RESOURCES.filter((r) => (player.inventory[r] ?? 0) > 0)
        .map((r) => `${RESOURCE_GLYPH[r]}${player.inventory[r]}`)
        .join(' ')
    : '';

  // Count available non-move actions
  const actionCount = legal.filter(
    (c) => c.kind !== 'move' && c.kind !== 'explore',
  ).length;

  return (
    <div className="hud">
      {/* TOP-LEFT: Darkness & Round */}
      <div className="hud__top-left">
        <div className="hud__darkness">
          <span className="hud__darkness-label">☠ DARKNESS</span>
          <div className="hud__darkness-bar">
            {Array.from({ length: darknessMax + 1 }, (_, i) => (
              <div
                key={i}
                className={`hud__darkness-pip ${
                  i <= state.darkness ? 'hud__darkness-pip--filled' : ''
                } ${i === darknessMax ? 'hud__darkness-pip--final' : ''}`}
              />
            ))}
          </div>
          <span className="hud__round">
            R{state.round} · {state.darkness}/{darknessMax}
          </span>
        </div>
      </div>

      {/* TOP-RIGHT: Settings */}
      <div className="hud__top-right">
        <button className="hud__btn" onClick={() => { sfx.playClick(); onOpenAchievements(); }} title="Pencapaian & Papan Peringkat">🏆</button>
        <button className="hud__btn" onClick={() => { sfx.playClick(); onOpenTutorial(); }} title="Panduan">📖</button>
        <button className="hud__btn" onClick={() => { sfx.playClick(); onOpenTelemetry(); }} title="Statistik">📊</button>
        <button className="hud__btn" onClick={onToggleLang} title="Bahasa">
          {lang === 'id' ? '🇮🇩' : '🇬🇧'}
        </button>
        <button className="hud__btn" onClick={onToggleSound} title="Audio">
          {isMuted ? '🔇' : '🔊'}
        </button>
        <button className="hud__btn" onClick={() => { sfx.playClick(); onOpenSettings(); }} title="Pengaturan">⚙️</button>
      </div>

      {/* BOTTOM-LEFT: Active Player Status */}
      {player && !gameOver && (
        <div className="hud__bottom-left">
          <div className="hud__player-status">
            <div className="hud__player-name">
              {player.name}
              <span className="hud__player-role">
                {CHARACTER_NAMES[player.character] ?? player.character}
              </span>
            </div>
            <div className="hud__player-stats">
              <span>♥{'●'.repeat(player.health)}{'○'.repeat(3 - player.health)}</span>
              <span>⚡{'●'.repeat(player.ap)}{'○'.repeat(3 - player.ap)}</span>
              <span>★{player.vp}</span>
            </div>
            <div className="hud__player-loc">
              📍 {locationLabel(state.board, player.at)}
            </div>
            {invSummary && (
              <div className="hud__player-inv">{invSummary}</div>
            )}
          </div>
        </div>
      )}

      {/* BOTTOM-RIGHT: Action Buttons */}
      {!gameOver && (
        <div className="hud__bottom-right">
          <button className="hud__btn hud__btn--action" onClick={() => { sfx.playClick(); onOpenActions(); }} title="Aksi (Space)">
            ⚡ Aksi{actionCount > 0 ? ` (${actionCount})` : ''}
          </button>
          <button className="hud__btn hud__btn--action" onClick={() => { sfx.playClick(); onOpenPlayers(); }} title="Pemain">
            👤
          </button>
          <button className="hud__btn hud__btn--action" onClick={() => { sfx.playClick(); onOpenLighthouse(); }} title="Mercusuar">
            🏰 {repairedCount}/{totalComponents}
          </button>
          <button className="hud__btn hud__btn--action" onClick={() => { sfx.playClick(); onOpenLog(); }} title="Catatan">
            📜
          </button>
          <button className="hud__btn hud__btn--action" onClick={() => { sfx.playClick(); onOpenCards(); }} title="Kartu">
            🎴
          </button>
        </div>
      )}

      {/* BOTTOM-CENTER: Keyboard Hint */}
      {!gameOver && (
        <div className="hud__bottom-center">
          <span className="hud__hint">WASD / ↑←↓→ bergerak · SPACE aksi · ESC tutup</span>
        </div>
      )}
    </div>
  );
}
