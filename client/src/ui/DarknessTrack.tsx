import type { GameState } from '../types';

// Ambang Darkness (GDD 23). Ini duplikasi PRESENTASI dari core/content/darkness.json:
// yang ditampilkan di sini hanya teks penjelas, sedangkan efek sesungguhnya
// dihitung core. Kalau angkanya di-tuning nanti, ini ikut diperbarui lewat
// cmd/contentgen di M1 (ADR-005).
const THRESHOLDS: Record<number, string> = {
  2: 'Monster jadi lebih agresif',
  4: 'Gathering jadi kurang efisien',
  6: 'Monster lebih sering muncul',
  7: 'Semua pemain kehilangan 1 AP',
};

interface Props {
  state: GameState;
  max: number;
}

/**
 * Darkness Track adalah ancaman global dan sumber utama tekanan waktu
 * (GDD 2.2, 22). Ia diletakkan paling atas dan selalu terlihat karena GDD 34
 * meminta informasi penting tidak disembunyikan di balik menu -- dan tidak ada
 * informasi yang lebih penting dari "berapa lama lagi kita punya waktu".
 */
export function DarknessTrack({ state, max }: Props) {
  const steps = Array.from({ length: max + 1 }, (_, i) => i);
  const active = Object.entries(THRESHOLDS)
    .filter(([at]) => state.darkness >= Number(at))
    .map(([, text]) => text);

  const remaining = max - state.darkness;

  return (
    <section className="darkness" aria-label="Darkness track">
      <div className="darkness__head">
        <span className="darkness__label">Darkness</span>
        <span className="darkness__meta">
          Ronde {state.round} · {state.darkness} / {max}
          {remaining <= 2 && state.status === 'active' && (
            <strong style={{ color: '#ff9db8' }}> · {remaining} langkah lagi</strong>
          )}
        </span>
      </div>

      <div className="darkness__track">
        {steps.map((i) => {
          const classes = ['darkness__step'];
          if (i <= state.darkness) classes.push('darkness__step--filled');
          if (i === max) classes.push('darkness__step--final');
          if (THRESHOLDS[i]) classes.push('darkness__step--threshold');
          return (
            <div
              key={i}
              className={classes.join(' ')}
              title={THRESHOLDS[i] ?? (i === max ? 'KEKALAHAN' : undefined)}
            >
              {i === max ? '☠' : i}
            </div>
          );
        })}
      </div>

      {active.length > 0 && (
        <p className="darkness__notes">Berlaku: {active.join(' · ')}</p>
      )}
    </section>
  );
}
