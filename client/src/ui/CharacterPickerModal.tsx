import { useState } from 'react';
import type { SeatConfig } from '../session/useGameSession';
import { i18n } from '../i18n';
import { sfx } from '../audio/sfx';

interface Props {
  initialSeats: SeatConfig[];
  isOpen: boolean;
  onConfirm(seats: SeatConfig[]): void;
  onCancel?(): void;
}

const AVAILABLE_CHARACTERS = [
  {
    id: 'navigator',
    nameKey: 'char.navigator',
    descKey: 'char.navigator.desc',
    icon: '🧭',
    color: '#ffc76b',
  },
  {
    id: 'engineer',
    nameKey: 'char.engineer',
    descKey: 'char.engineer.desc',
    icon: '⚙️',
    color: '#7ad7e8',
  },
  {
    id: 'hunter',
    nameKey: 'char.hunter',
    descKey: 'char.hunter.desc',
    icon: '🏹',
    color: '#e58fa9',
  },
  {
    id: 'scholar',
    nameKey: 'char.scholar',
    descKey: 'char.scholar.desc',
    icon: '📜',
    color: '#8fc07a',
  },
];

export function CharacterPickerModal({ initialSeats, isOpen, onConfirm, onCancel }: Props) {
  const [playerCount, setPlayerCount] = useState<number>(initialSeats.length || 3);
  const [seats, setSeats] = useState<SeatConfig[]>(() => {
    const list: SeatConfig[] = [
      { id: 'p1', name: initialSeats[0]?.name || 'Pemain 1', character: 'navigator' },
      { id: 'p2', name: initialSeats[1]?.name || 'Pemain 2', character: 'engineer' },
      { id: 'p3', name: initialSeats[2]?.name || 'Pemain 3', character: 'hunter' },
      { id: 'p4', name: initialSeats[3]?.name || 'Pemain 4', character: 'scholar' },
    ];
    return list;
  });

  if (!isOpen) return null;

  const currentSeats = seats.slice(0, playerCount);

  // Set of already selected characters in active seats
  const selectedChars = new Set(currentSeats.map((s) => s.character));

  const handleNameChange = (index: number, name: string) => {
    const next = [...seats];
    next[index] = { ...next[index], name };
    setSeats(next);
  };

  const handleCharacterChange = (index: number, character: string) => {
    sfx.playClick();
    const next = [...seats];
    next[index] = { ...next[index], character };
    setSeats(next);
  };

  const handleStart = () => {
    sfx.playLighthouseRepair(playerCount);
    onConfirm(currentSeats);
  };

  return (
    <div className="overlay">
      <div className="overlay__card overlay__card--wide" style={{ maxWidth: 780, textAlign: 'left' }}>
        <h2 className="overlay__title" style={{ textAlign: 'center', marginBottom: 4 }}>
          {i18n.t('setup.title')}
        </h2>
        <p className="overlay__text" style={{ textAlign: 'center', marginBottom: 20 }}>
          {i18n.t('setup.subtitle')}
        </p>

        {/* Player count selector */}
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 12, marginBottom: 24 }}>
          <span style={{ fontWeight: 600 }}>Jumlah Pemain:</span>
          {[2, 3, 4].map((count) => (
            <button
              key={count}
              className={`action ${playerCount === count ? 'action--primary' : 'action--ghost'}`}
              style={{ padding: '6px 16px' }}
              onClick={() => {
                sfx.playClick();
                setPlayerCount(count);
              }}
            >
              {count} Pemain
            </button>
          ))}
        </div>

        {/* Seats configuration */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16, marginBottom: 24 }}>
          {currentSeats.map((seat, index) => {
            const charInfo = AVAILABLE_CHARACTERS.find((c) => c.id === seat.character);
            return (
              <div
                key={seat.id}
                style={{
                  background: 'var(--panel)',
                  border: `1px solid ${charInfo?.color || 'var(--stone)'}`,
                  borderRadius: 'var(--radius)',
                  padding: 14,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 10,
                }}
              >
                <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                  <span
                    style={{
                      background: charInfo?.color || 'var(--stone)',
                      color: 'var(--abyss)',
                      fontWeight: 'bold',
                      padding: '4px 10px',
                      borderRadius: 6,
                    }}
                  >
                    {seat.id.toUpperCase()}
                  </span>
                  <input
                    type="text"
                    className="action"
                    style={{ flex: 1, textAlign: 'left', background: 'var(--panel-raised)', padding: '6px 12px' }}
                    value={seat.name}
                    onChange={(e) => handleNameChange(index, e.target.value)}
                    placeholder={`Nama Pemain ${index + 1}`}
                  />
                </div>

                {/* Character options */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))', gap: 8 }}>
                  {AVAILABLE_CHARACTERS.map((char) => {
                    const isSelected = seat.character === char.id;
                    const isTaken = selectedChars.has(char.id) && !isSelected;

                    return (
                      <button
                        key={char.id}
                        className={`action ${isSelected ? 'action--primary' : 'action--ghost'}`}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: 6,
                          padding: '6px 10px',
                          opacity: isTaken ? 0.35 : 1,
                          cursor: isTaken ? 'not-allowed' : 'pointer',
                        }}
                        disabled={isTaken}
                        onClick={() => !isTaken && handleCharacterChange(index, char.id)}
                        title={isTaken ? 'Karakter sudah dipilih pemain lain' : i18n.t(char.descKey)}
                      >
                        <span>{char.icon}</span>
                        <span style={{ fontSize: 13, fontWeight: isSelected ? 'bold' : 'normal' }}>
                          {i18n.t(char.nameKey)}
                        </span>
                      </button>
                    );
                  })}
                </div>

                {/* Selected character ability preview */}
                {charInfo && (
                  <div className="tiny" style={{ color: 'var(--ink-dim)', fontStyle: 'italic', paddingLeft: 4 }}>
                    ✦ {i18n.t(charInfo.descKey)}
                  </div>
                )}
              </div>
            );
          })}
        </div>

        {/* Buttons */}
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12 }}>
          {onCancel && (
            <button className="action action--ghost" onClick={onCancel}>
              Batal
            </button>
          )}
          <button className="action action--primary" style={{ padding: '10px 24px' }} onClick={handleStart}>
            🚀 {i18n.t('setup.start_game')}
          </button>
        </div>
      </div>
    </div>
  );
}
