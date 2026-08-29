import type { GameEvent, GameState, ResourceSet } from '../types';
import { RESOURCES } from '../types';
import {
  ARTIFACT_CARDS,
  componentName,
  EVENT_CARDS,
  locationLabel,
  locationName,
  MYSTERY_CARDS,
  RESOURCE_GLYPH as GLYPH,
} from './labels';

function res(rs: ResourceSet | undefined): string {
  if (!rs) return '';
  return RESOURCES.filter((r) => (rs[r] ?? 0) > 0)
    .map((r) => `${GLYPH[r]}${rs[r]}`)
    .join(' ');
}

interface Line {
  text: string;
  tone?: 'danger' | 'good';
}

/**
 * Menerjemahkan event jadi kalimat.
 *
 * Ini bukan sekadar hiasan. GDD 38 mencatat risiko nyata bahwa pemain tidak
 * paham kenapa Darkness naik, dan mitigasinya adalah mengikat setiap kenaikan
 * ke sebab yang terlihat. Karena core sudah membawa alasan di setiap event
 * Darkness, log ini bisa menampilkannya langsung.
 */
function describe(e: GameEvent, state: GameState): Line | null {
  const who = (id?: string) => state.players.find((p) => p.id === id)?.name ?? id ?? '';
  const place = (id?: string) => (id ? locationLabel(state.board, id) : '');

  switch (e.kind) {
    case 'round_started':
      return { text: `— Ronde ${e.round} —` };
    case 'moved':
      return { text: `${who(e.player)} pindah ke ${place(e.to)}` };
    case 'resource_gained': {
      if (!e.player) return null;
      // Resource dari kartu atau misteri tidak berasal dari lokasi mana pun,
      // jadi kalimat "mengambil X di ___" akan menggantung tanpa tempat.
      if (!e.from) {
        return { text: `${who(e.player)} memperoleh ${res(e.resources)}${e.reason ? ` (${e.reason})` : ''}` };
      }
      return { text: `${who(e.player)} mengambil ${res(e.resources)} di ${place(e.from)}` };
    }
    case 'repaired':
      return { text: `${who(e.player)} menyetor ${res(e.resources)} ke mercusuar` };
    case 'component_repaired': {
      const comp = state.lighthouse.find((c) => c.id === e.component);
      return {
        text: `✦ ${componentName(e.component!, comp?.name ?? e.component!)} selesai diperbaiki`,
        tone: 'good',
      };
    }
    case 'vp_awarded':
      return { text: `${who(e.player)} mendapat ${e.amount} VP (${e.reason ?? ''})`, tone: 'good' };
    case 'healed':
      return { text: `${who(e.player)} pulih ${e.amount} HP` };
    case 'damaged':
      return { text: `${who(e.player)} kehilangan ${e.amount} HP`, tone: 'danger' };
    case 'darkness_rose':
      return {
        text: `Darkness naik jadi ${e.value}${e.reason ? ` — ${e.reason}` : ''}`,
        tone: 'danger',
      };
    case 'game_won':
      return { text: '★ Mercusuar menyala. Kalian selamat.', tone: 'good' };
    case 'game_lost':
      return { text: '☠ Darkness menelan pulau. Semua kalah.', tone: 'danger' };
    case 'objective_dealt':
      return e.objective ? { text: 'Objective rahasiamu dibagikan' } : null;

    // --- M1 ---
    case 'card_drawn': {
      if (e.deck !== 'event' || !e.card) return null;
      const card = EVENT_CARDS[e.card];
      return card ? { text: `Peristiwa: ${card.name} — ${card.text}` } : null;
    }
    case 'location_revealed':
      return {
        text: `${who(e.player)} menyingkap ${locationName(e.tile ?? '', e.reason ?? '?')}`,
        tone: 'good',
      };
    case 'monster_spawned':
      return { text: `Monster muncul di ${place(e.from)}${e.reason ? ` — ${e.reason}` : ''}`, tone: 'danger' };
    case 'monster_moved':
      return { text: `Monster bergerak dari ${place(e.from)} ke ${place(e.to)}`, tone: 'danger' };
    case 'monster_attacked':
      return { text: `${who(e.player)} diserang monster di ${place(e.from)}`, tone: 'danger' };
    case 'monster_defeated':
      return { text: `${who(e.player)} mengalahkan monster di ${place(e.from)}`, tone: 'good' };
    case 'dice_rolled':
      return {
        text:
          e.amount === e.value
            ? `${who(e.player)} melempar ${e.amount}`
            : `${who(e.player)} melempar ${e.amount} (jadi ${e.value} dengan bonus)`,
      };
    case 'artifact_gained': {
      const art = e.artifact ? ARTIFACT_CARDS[e.artifact] : undefined;
      return { text: `${who(e.player)} memperoleh ${art?.name ?? e.artifact}`, tone: 'good' };
    }
    case 'mystery_offered': {
      const card = e.card ? MYSTERY_CARDS[e.card] : undefined;
      return { text: `${who(e.player)} menemukan misteri${card ? `: ${card.name}` : ''}` };
    }
    case 'mystery_resolved': {
      const card = e.card ? MYSTERY_CARDS[e.card] : undefined;
      const opt = card?.options.find((o) => o.id === e.option);
      return { text: `${who(e.player)} memilih: ${opt?.text ?? e.option}` };
    }
    case 'traded':
      return { text: `${who(e.player)} memberi ${res(e.resources)} kepada ${who(e.target)}` };
    case 'village_rescued':
      return { text: `${who(e.player)} menyelamatkan ${place(e.from)}`, tone: 'good' };
    case 'gather_blocked':
      return e.value ? { text: `${place(e.from)} tidak bisa dipanen ronde ini`, tone: 'danger' } : null;
    case 'exhausted':
      return e.value ? { text: `${who(e.player)} kelelahan`, tone: 'danger' } : null;
    default:
      // Event mekanis (ap_spent, phase_changed, turn_started, location_regen)
      // sengaja tidak ditampilkan: log yang penuh derau membuat baris yang
      // benar-benar penting jadi tenggelam.
      return null;
  }
}

export function EventLog({ log, state }: { log: GameEvent[]; state: GameState }) {
  const lines = log
    .map((e) => describe(e, state))
    .filter((l): l is Line => l !== null)
    .slice(-60)
    .reverse();

  return (
    <section className="panel" aria-label="Catatan kejadian" style={{ flex: 1 }}>
      <h2 className="panel__title">Catatan</h2>
      <div className="log">
        {lines.length === 0 && <p className="muted tiny">Belum ada kejadian.</p>}
        {lines.map((l, i) => (
          <div
            key={i}
            className={`log__line ${l.tone ? `log__line--${l.tone}` : ''}`}
          >
            {l.text}
          </div>
        ))}
      </div>
    </section>
  );
}
