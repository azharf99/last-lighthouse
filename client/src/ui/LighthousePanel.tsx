import type { GameState, ResourceSet } from '../types';
import { RESOURCES } from '../types';
import { componentName, RESOURCE_GLYPH as GLYPH } from './labels';

/** Menampilkan biaya sebagai "terpenuhi / dibutuhkan" agar pemain langsung tahu
 *  apa yang masih kurang, bukan hanya total biayanya. Ini keputusan yang sama
 *  dengan yang membuat core memotong setoran berlebih: kekurangan adalah
 *  informasi yang dipakai untuk mengambil keputusan, bukan totalnya. */
function CostRow({ cost, progress }: { cost: ResourceSet; progress: ResourceSet }) {
  return (
    <div className="component__cost">
      {RESOURCES.filter((r) => (cost[r] ?? 0) > 0).map((r) => {
        const need = cost[r] ?? 0;
        const have = Math.min(progress[r] ?? 0, need);
        const done = have >= need;
        return (
          <span
            key={r}
            className="res-chip"
            style={done ? { borderColor: 'var(--beacon)', color: 'var(--beacon)' } : undefined}
          >
            {GLYPH[r]} {have}/{need}
          </span>
        );
      })}
    </div>
  );
}

export function LighthousePanel({ state }: { state: GameState }) {
  const nextIdx = state.lighthouse.findIndex((c) => !c.repaired);

  return (
    <section className="panel" aria-label="Mercusuar">
      <h2 className="panel__title">
        Mercusuar · {state.lighthouse.filter((c) => c.repaired).length} / {state.lighthouse.length}
      </h2>

      {state.lighthouse.map((comp, i) => {
        const classes = ['component'];
        if (comp.repaired) classes.push('component--done');
        else if (i === nextIdx) classes.push('component--next');

        return (
          <div key={comp.id} className={classes.join(' ')}>
            <div className="component__order">{comp.repaired ? '✓' : comp.order}</div>
            <div className="component__body">
              <div className="component__name">{componentName(comp.id, comp.name)}</div>
              {/* Komponen setelah yang berikutnya masih terkunci (GDD 7.1),
                  jadi biayanya diredam agar fokus tetap ke target sekarang. */}
              {!comp.repaired && (
                <div style={i === nextIdx ? undefined : { opacity: 0.45 }}>
                  <CostRow cost={comp.cost} progress={comp.progress} />
                </div>
              )}
            </div>
            <div className="component__vp">{comp.vp} VP</div>
          </div>
        );
      })}

      {nextIdx === -1 && (
        <p className="tiny" style={{ color: 'var(--beacon)', marginBottom: 0 }}>
          Mercusuar menyala kembali.
        </p>
      )}
    </section>
  );
}
