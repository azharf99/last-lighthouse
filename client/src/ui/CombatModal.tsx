import { useState } from 'react';
import type { Command, PlayerID } from '../types';
import { i18n } from '../i18n';
import { sfx } from '../audio/sfx';

interface Props {
  isOpen: boolean;
  player: PlayerID;
  character: string;
  onSend(cmd: Command): void;
  onClose(): void;
}

const DICE_FACES = ['⚀', '⚁', '⚂', '⚃', '⚄', '⚅'];

export function CombatModal({ isOpen, player, character, onSend, onClose }: Props) {
  const [rolling, setRolling] = useState(false);
  const [diceResult, setDiceResult] = useState<number | null>(null);

  if (!isOpen) return null;

  const isHunter = character === 'hunter';

  const handleRoll = () => {
    setRolling(true);
    sfx.playDiceRoll();

    // Roll animation ticker
    let count = 0;
    const interval = setInterval(() => {
      setDiceResult(Math.floor(Math.random() * 6) + 1);
      count++;
      if (count >= 8) {
        clearInterval(interval);
        setRolling(false);
        // Execute fight command through game engine
        onSend({ kind: 'fight', player });
        setTimeout(() => {
          onClose();
        }, 1200);
      }
    }, 80);
  };

  return (
    <div className="overlay">
      <div className="overlay__card" style={{ maxWidth: 440, textAlign: 'center' }}>
        <h2 className="overlay__title">⚔️ {i18n.t('combat.title')}</h2>
        <p className="overlay__text">{i18n.t('combat.subtitle')}</p>

        {/* Dice Visual */}
        <div
          style={{
            fontSize: 72,
            margin: '16px 0',
            lineHeight: 1,
            color: 'var(--beacon)',
            textShadow: '0 0 20px rgba(255, 199, 107, 0.4)',
            transition: 'transform 0.1s ease',
            transform: rolling ? 'scale(1.15) rotate(12deg)' : 'scale(1)',
          }}
        >
          {diceResult ? DICE_FACES[diceResult - 1] : '🎲'}
        </div>

        {/* Modifiers */}
        {isHunter && (
          <div
            style={{
              background: 'rgba(229, 143, 169, 0.15)',
              border: '1px solid #e58fa9',
              color: '#e58fa9',
              borderRadius: 6,
              padding: '4px 10px',
              fontSize: 12,
              fontWeight: 'bold',
              marginBottom: 16,
              display: 'inline-block',
            }}
          >
            ✦ {i18n.t('combat.bonus_hunter')}
          </div>
        )}

        {/* Combat Outcome Rules (GDD §16) */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: 6,
            background: 'var(--panel)',
            padding: 10,
            borderRadius: 'var(--radius)',
            marginBottom: 20,
            textAlign: 'left',
            fontSize: 12,
          }}
        >
          <div style={{ color: 'var(--dread)' }}>• <b>1–2:</b> {i18n.t('combat.result_wound')}</div>
          <div style={{ color: 'var(--ink-dim)' }}>• <b>3–4:</b> {i18n.t('combat.result_standoff')}</div>
          <div style={{ color: 'var(--ok)' }}>• <b>5–6:</b> {i18n.t('combat.result_victory')}</div>
        </div>

        {/* Actions */}
        <div style={{ display: 'flex', justifyContent: 'center', gap: 12 }}>
          <button className="action action--ghost" onClick={onClose} disabled={rolling}>
            Mundur
          </button>
          <button className="action action--primary" onClick={handleRoll} disabled={rolling}>
            {rolling ? i18n.t('combat.rolling') : i18n.t('combat.roll_button')}
          </button>
        </div>
      </div>
    </div>
  );
}
