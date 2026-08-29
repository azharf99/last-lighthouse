import type { Command, GameState, ResourceSet } from '../types';
import { RESOURCES } from '../types';
import {
  componentName,
  locationLabel,
  RESOURCE_GLYPH as GLYPH,
  RESOURCE_NAMES,
} from './labels';

function summarise(rs: ResourceSet | undefined): string {
  if (!rs) return '';
  return RESOURCES.filter((r) => (rs[r] ?? 0) > 0)
    .map((r) => `${GLYPH[r]}${rs[r]}`)
    .join(' ');
}

/**
 * Memberi label aksi dalam bahasa yang bisa dibaca pemain.
 *
 * Daftar aksinya sendiri datang dari core (LegalCommands), bukan disusun di
 * sini. Client tidak pernah memutuskan apa yang legal -- ia hanya memberi nama
 * pada pilihan yang sudah diberikan core. Itu batas yang menjaga ADR-002 tetap
 * bermakna di lapisan UI.
 */
function label(cmd: Command, state: GameState): { text: string; hint?: string } {
  switch (cmd.kind) {
    case 'move':
      return { text: `Pindah ke ${locationLabel(state.board, cmd.to ?? '')}` };
    case 'gather': {
      const loc = state.board.locations.find((l) => l.id === state.players.find((p) => p.id === cmd.player)?.at);
      const stok = loc?.available[cmd.resource!] ?? 0;
      return {
        text: `Ambil ${GLYPH[cmd.resource!]} ${RESOURCE_NAMES[cmd.resource!] ?? cmd.resource}`,
        hint: `tersedia ${stok}`,
      };
    }
    case 'repair': {
      const comp = state.lighthouse.find((c) => c.id === cmd.component);
      return {
        text: `Perbaiki ${componentName(cmd.component!, comp?.name ?? cmd.component!)}`,
        hint: summarise(cmd.pay),
      };
    }
    case 'explore':
      return { text: 'Jelajahi', hint: 'wilayah belum dipetakan' };
    case 'fight': {
      const me = state.players.find((p) => p.id === cmd.player);
      const loc = state.board.locations.find((l) => l.id === me?.at);
      return { text: 'Lawan monster', hint: `${loc?.monsters ?? 0} di sini` };
    }
    case 'investigate':
      return { text: 'Selidiki', hint: `${state.mysteryDeck.draw.length} kartu tersisa` };
    case 'trade': {
      const to = state.players.find((p) => p.id === cmd.target);
      return { text: `Beri ${summarise(cmd.give)}`, hint: `ke ${to?.name ?? cmd.target}` };
    }
    case 'rest':
      return { text: 'Istirahat', hint: '+1 HP' };
    case 'end_turn':
      return { text: 'Akhiri giliran' };
    default:
      return { text: cmd.kind };
  }
}

import { sfx } from '../audio/sfx';

interface Props {
  state: GameState;
  legal: Command[];
  disabled: boolean;
  onSend(cmd: Command): void;
  onFight?(): void;
}

export function ActionBar({ state, legal, disabled, onSend, onFight }: Props) {
  // Move dan Explore sudah bisa dilakukan lewat klik di peta, jadi tidak diulang
  // sebagai tombol: dua jalan menuju aksi yang sama membuat panel ini penuh
  // tanpa menambah pilihan.
  const actions = legal.filter((c) => c.kind !== 'move' && c.kind !== 'explore');
  const endTurn = actions.find((c) => c.kind === 'end_turn');
  const rest = actions.filter((c) => c.kind !== 'end_turn');

  const handleClick = (cmd: Command) => {
    if (cmd.kind === 'gather') {
      sfx.playGather();
    } else if (cmd.kind === 'repair') {
      const comp = state.lighthouse.filter((c) => c.repaired).length + 1;
      sfx.playLighthouseRepair(comp);
    } else if (cmd.kind === 'fight' && onFight) {
      onFight();
      return;
    } else {
      sfx.playClick();
    }
    onSend(cmd);
  };

  return (
    <div>
      <div className="actions">
        {rest.length === 0 && (
          <span className="muted tiny">
            Tidak ada aksi selain berpindah, menjelajah, atau mengakhiri giliran.
          </span>
        )}

        {rest.map((cmd, i) => {
          const { text, hint } = label(cmd, state);
          return (
            <button
              key={`${cmd.kind}-${cmd.resource ?? cmd.component ?? cmd.target ?? i}-${i}`}
              className={`action ${cmd.kind === 'repair' ? 'action--primary' : ''}${
                cmd.kind === 'fight' ? ' action--danger' : ''
              }`}
              disabled={disabled}
              onClick={() => handleClick(cmd)}
            >
              <span>{text}</span>
              {hint && <span className="muted tiny">{hint}</span>}
              <span className="action__cost">1 AP</span>
            </button>
          );
        })}

        {endTurn && (
          <button
            className="action action--ghost"
            disabled={disabled}
            onClick={() => {
              sfx.playClick();
              onSend(endTurn);
            }}
          >
            {label(endTurn, state).text}
          </button>
        )}
      </div>
    </div>
  );
}
