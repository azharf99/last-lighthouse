import type { PlayerView, ResourceSet } from '../types';
import { RESOURCES, totalResources } from '../types';
import {
  CHARACTER_NAMES,
  locationLabel,
  OBJECTIVE_LABELS,
  RESOURCE_GLYPH as GLYPH,
} from './labels';

/** Inventory ditampilkan sebagai slot terbatas (GDD 9.1), bukan sekadar angka:
 *  keterbatasannya adalah bagian dari keputusan sulit yang ingin diciptakan
 *  desainnya ("bawa lebih banyak resource atau peralatan?"). */
function Inventory({ inv, capacity }: { inv: ResourceSet; capacity: number }) {
  const filled: string[] = [];
  for (const r of RESOURCES) {
    for (let i = 0; i < (inv[r] ?? 0); i++) filled.push(GLYPH[r]);
  }
  const slots = Math.max(capacity, filled.length);

  return (
    <div className="inventory" role="group" aria-label={`Inventory ${filled.length} dari ${capacity}`}>
      {Array.from({ length: slots }, (_, i) => (
        <div key={i} className={`slot ${i < filled.length ? 'slot--filled' : ''}`}>
          {filled[i] ?? ''}
        </div>
      ))}
    </div>
  );
}

function Pips({ n, max, kind }: { n: number; max: number; kind: 'ap' | 'hp' }) {
  return (
    <span className="pips">
      {Array.from({ length: max }, (_, i) => (
        <span key={i} className={`pip ${i < n ? `pip--on-${kind}` : ''}`} />
      ))}
    </span>
  );
}

interface Props {
  view: PlayerView;
  maxHealth: number;
  maxAP: number;
  inventoryCapacity: number;
}

export function PlayerPanel({ view, maxHealth, maxAP, inventoryCapacity }: Props) {
  const st = view.state;
  const activeId = st.turnOrder[st.activeIdx];

  return (
    <section className="panel" aria-label="Area pemain">
      <h2 className="panel__title">Pemain</h2>

      {st.players.map((p) => {
        const isActive = p.id === activeId;
        const isViewer = p.id === view.viewer;
        const capacity = p.exhausted ? 3 : inventoryCapacity;

        return (
          <div key={p.id} className={`seat ${isActive ? 'seat--active' : ''}`}>
            <div className="seat__head">
              <span className="seat__name">
                {p.name}
                {isActive && <span className="muted tiny"> · giliran</span>}
              </span>
              <span className="seat__role">{CHARACTER_NAMES[p.character] ?? p.character}</span>
            </div>

            <div className="seat__stats">
              <span>
                HP <Pips n={p.health} max={maxHealth} kind="hp" />
              </span>
              <span>
                AP <Pips n={p.ap} max={maxAP} kind="ap" />
              </span>
              <span>VP {p.vp}</span>
              <span className="muted">{locationLabel(st.board, p.at)}</span>
            </div>

            {p.exhausted && (
              <div className="tiny" style={{ color: '#e58fa9' }}>
                Kelelahan — tidak bisa bertarung, kapasitas turun jadi 3 (GDD 17)
              </div>
            )}

            {/* Objective hanya muncul untuk pemain yang sedang memegang perangkat.
                Objective pemain lain memang tidak ada di data ini: core sudah
                mengosongkannya saat projection (ADR-006). */}
            {isViewer && view.myObjective && (
              <div className="tiny" style={{ marginTop: 6, color: 'var(--beacon)' }}>
                🎯 {OBJECTIVE_LABELS[view.myObjective] ?? view.myObjective}
              </div>
            )}
            {!isViewer && (
              <div className="tiny muted" style={{ marginTop: 6 }}>
                🎯 objective rahasia
              </div>
            )}

            {isViewer ? (
              <Inventory inv={p.inventory} capacity={capacity} />
            ) : (
              <div className="seat__stats" style={{ marginTop: 6 }}>
                <span className="res-chip">
                  membawa {totalResources(p.inventory)} / {capacity}
                </span>
              </div>
            )}
          </div>
        );
      })}
    </section>
  );
}
