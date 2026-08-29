import type { GameState } from '../types';
import { EVENT_CARDS } from './labels';

interface Props {
  state: GameState;
}

export function CardDeckPanel({ state }: Props) {
  // Active/latest drawn event card
  const lastEvent = state.eventDeck.discard.length > 0 ? state.eventDeck.discard[state.eventDeck.discard.length - 1] : null;
  const eventInfo = lastEvent ? EVENT_CARDS[lastEvent] : null;

  return (
    <div className="panel" style={{ gap: 12 }}>
      <h2 className="panel__title">🃏 Deck Kartu Pulau (GDD §8.3)</h2>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 10 }}>
        {/* 1. Event Deck */}
        <div
          style={{
            background: 'var(--panel-raised)',
            border: '1px solid var(--stone)',
            borderRadius: 'var(--radius)',
            padding: 10,
            display: 'flex',
            flexDirection: 'column',
            gap: 6,
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <b style={{ color: 'var(--beacon)' }}>🌪️ Event Deck</b>
            <span className="tiny muted">{state.eventDeck.draw.length} sisa</span>
          </div>
          {eventInfo ? (
            <div
              style={{
                background: 'rgba(255, 199, 107, 0.1)',
                border: '1px solid var(--beacon-glow)',
                borderRadius: 6,
                padding: '6px 8px',
              }}
            >
              <div style={{ fontWeight: 'bold', fontSize: 12 }}>{eventInfo.name}</div>
              <div className="tiny" style={{ color: 'var(--ink-dim)' }}>
                {eventInfo.text}
              </div>
            </div>
          ) : (
            <div className="tiny muted" style={{ fontStyle: 'italic', padding: '6px 0' }}>
              Belum ada event ronde.
            </div>
          )}
        </div>

        {/* 2. Mystery Deck */}
        <div
          style={{
            background: 'var(--panel-raised)',
            border: '1px solid var(--stone)',
            borderRadius: 'var(--radius)',
            padding: 10,
            display: 'flex',
            flexDirection: 'column',
            gap: 6,
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <b style={{ color: 'var(--crystal)' }}>📜 Mystery Deck</b>
            <span className="tiny muted">{state.mysteryDeck.draw.length} sisa</span>
          </div>
          <div className="tiny" style={{ color: 'var(--ink-dim)', padding: '4px 0' }}>
            Tersedia di reruntuhan dan lokasi pulau. Selidiki untuk mendapatkan relic dan lore.
          </div>
        </div>

        {/* 3. Artifact Deck */}
        <div
          style={{
            background: 'var(--panel-raised)',
            border: '1px solid var(--stone)',
            borderRadius: 'var(--radius)',
            padding: 10,
            display: 'flex',
            flexDirection: 'column',
            gap: 6,
          }}
        >
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <b style={{ color: 'var(--food)' }}>🔮 Artifact Deck</b>
            <span className="tiny muted">{state.artifactDeck.draw.length} sisa</span>
          </div>
          <div className="tiny" style={{ color: 'var(--ink-dim)', padding: '4px 0' }}>
            Pusaka kuno pemberi efek pasif permanen dan nilai VP di akhir permainan.
          </div>
        </div>
      </div>
    </div>
  );
}
