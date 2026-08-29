import type { Command, GameState, LocationID, PlayerID } from '../types';
import { RESOURCES } from '../types';
import { locationName, RESOURCE_GLYPH } from './labels';

// Tata letak peta prototype (GDD 32: Lighthouse, Harbor, Village, Forest, Cave,
// Ruins). Posisinya dipilih agar tidak ada garis adjacency yang saling memotong,
// sehingga hubungan antar lokasi terbaca sekali lihat.
//
// CATATAN ARSITEKTUR: ADR-001 menetapkan PixiJS untuk peta pulau. Di M0 peta
// masih berupa 6 lokasi tetap tanpa pan/zoom, jadi SVG dipakai lebih dulu --
// ia DOM, sehingga klik, fokus keyboard, dan screen reader jalan tanpa kerja
// tambahan. PixiJS masuk di M3 saat peta jadi modular dengan tile eksplorasi
// dan benar-benar butuh kanvas.
//
// Komposisinya sengaja lebar (rasio ~1.5:1). Dengan viewBox yang hampir persegi,
// SVG yang tingginya dibatasi hanya mengisi sekitar 40% lebar panel dan sisanya
// tampak seperti kotak kosong; rasio lebar memakai ruang horizontal layar besar
// tanpa membuat peta jadi terlalu tinggi.
const LAYOUT: Record<string, { x: number; y: number }> = {
  lighthouse: { x: 350, y: 58 },
  harbor: { x: 150, y: 150 },
  forest: { x: 550, y: 150 },
  village: { x: 310, y: 265 },
  cave: { x: 585, y: 285 },
  ruins: { x: 395, y: 385 },
  site_c: { x: 105, y: 300 },
  site_a: { x: 240, y: 430 },
  site_b: { x: 620, y: 420 },
};

const FALLBACK = { x: 350, y: 240 };

const VIEW_W = 700;
const VIEW_H = 500;
const NODE_R = 30;

// Geometri label. Nama lokasi ditaruh DI BAWAH lingkaran, bukan di dalamnya:
// "Reruntuhan" berukuran 64px sementara diameter lingkaran hanya 60px, sehingga
// teksnya menyembul keluar. Menaruhnya di luar melepaskan panjang nama dari
// ukuran lingkaran -- penting karena konten bisa menambah lokasi baru dengan
// nama sepanjang apa pun tanpa menyentuh kode (ADR-005).
const LABEL_DY = 48;
const STOCK_DY = 64;
const PAWN_DY = -36;

const PAWN_COLORS = ['#ffc76b', '#7ad7e8', '#8fc07a', '#e58fa9'];

// Ikon per tipe lokasi (GDD 19). Lingkaran sekarang menampung ikon, bukan teks.
const UNEXPLORED_ICON = '?';

const TYPE_ICON: Record<string, string> = {
  lighthouse: '🗼',
  harbor: '⚓',
  village: '🏘',
  forest: '🌲',
  cave: '⛏',
  crystal_cavern: '💠',
  ruins: '🏛',
  mountain: '⛰',
  temple: '🛕',
};

interface Props {
  state: GameState;
  legal: Command[];
  activePlayer: PlayerID | null;
  onMove(to: LocationID): void;
  onExplore(to: LocationID): void;
}

export function IslandMap({ state, legal, activePlayer, onMove, onExplore }: Props) {
  const reachable = new Set(
    legal.filter((c) => c.kind === 'move' && c.to).map((c) => c.to as LocationID),
  );
  // Slot "?" diklik untuk MENJELAJAHI, bukan berpindah -- kalau keduanya
  // dipetakan ke klik yang sama, pemain akan mengira petaknya rusak.
  const explorableSet = new Set(
    legal.filter((c) => c.kind === 'explore' && c.to).map((c) => c.to as LocationID),
  );

  const pos = (id: LocationID) => LAYOUT[id] ?? FALLBACK;

  // Setiap sisi digambar sekali saja: adjacency di core bersifat timbal balik
  // (divalidasi Content.Validate), jadi tanpa penyaringan ini tiap garis akan
  // tergambar dua kali.
  const edges: [LocationID, LocationID][] = [];
  for (const loc of state.board.locations) {
    for (const adj of loc.adjacent) {
      if (loc.id < adj) edges.push([loc.id, adj]);
    }
  }

  // Kelompokkan pemain per lokasi agar bidaknya tidak saling menimpa.
  const occupants = new Map<LocationID, PlayerID[]>();
  for (const p of state.players) {
    const list = occupants.get(p.at) ?? [];
    list.push(p.id);
    occupants.set(p.at, list);
  }

  const repairedCount = state.lighthouse.filter((c) => c.repaired).length;

  return (
    <section className="panel map" aria-label="Peta pulau">
      <h2 className="panel__title">Pulau</h2>
      <svg className="map__svg" viewBox={`0 0 ${VIEW_W} ${VIEW_H}`} role="img">
        <title>Peta pulau dengan {state.board.locations.length} lokasi</title>

        {edges.map(([a, b]) => {
          const pa = pos(a);
          const pb = pos(b);
          return (
            <line key={`${a}-${b}`} className="edge" x1={pa.x} y1={pa.y} x2={pb.x} y2={pb.y} />
          );
        })}

        {state.board.locations.map((loc) => {
          const p = pos(loc.id);
          const isReachable = reachable.has(loc.id);
          const isLighthouse = loc.type === 'lighthouse';
          const here = occupants.get(loc.id) ?? [];
          const explorable = explorableSet.has(loc.id);
          // Slot yang belum dibuka tidak diberi label teks: ikon "?" dan
          // lingkaran putus-putus sudah menyampaikannya, sementara tiga label
          // "Belum dijelajahi" yang identik hanya menambah derau. Pembaca layar
          // tetap terlayani lewat aria-label di grup.
          const name = loc.explored ? locationName(loc.type, loc.name) : '';

          const stock = !loc.explored
            ? ''
            : RESOURCES.filter((r) => (loc.available[r] ?? 0) > 0)
                .map((r) => `${RESOURCE_GLYPH[r]}${loc.available[r]}`)
                .join(' ');

          const classes = ['node'];
          if (isReachable || explorable) classes.push('node--reachable');
          if (isLighthouse) classes.push('node--lighthouse');
          if (!loc.explored) classes.push('node--unexplored');
          if (loc.gatherBlocked) classes.push('node--blocked');

          const action = explorable
            ? () => onExplore(loc.id)
            : isReachable
              ? () => onMove(loc.id)
              : undefined;
          const actionLabel = explorable
            ? 'Jelajahi wilayah yang belum dipetakan'
            : isReachable
              ? `Pindah ke ${name}`
              : `${name}, tidak bisa dijangkau`;

          return (
            <g
              key={loc.id}
              className={classes.join(' ')}
              onClick={action}
              role={action ? 'button' : undefined}
              tabIndex={action ? 0 : undefined}
              onKeyDown={
                action
                  ? (e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        action();
                      }
                    }
                  : undefined
              }
              aria-label={actionLabel}
            >
              {/* Cahaya mercusuar menguat tiap komponen selesai: mercusuar
                  adalah representasi visual harapan (GDD 35). */}
              {isLighthouse && repairedCount > 0 && (
                <circle className="beacon-halo" cx={p.x} cy={p.y} r={NODE_R + repairedCount * 7} />
              )}

              <circle className="node__disc" cx={p.x} cy={p.y} r={NODE_R} />

              <text className="node__icon" x={p.x} y={p.y + 9}>
                {loc.explored ? (TYPE_ICON[loc.type] ?? '◇') : UNEXPLORED_ICON}
              </text>

              {/* Monster digambar sebagai lencana di sudut, bukan menimpa ikon:
                  keduanya adalah informasi berbeda dan harus terbaca bersamaan. */}
              {loc.monsters > 0 && (
                <>
                  <circle className="monster-badge" cx={p.x + 22} cy={p.y - 20} r={11} />
                  <text className="monster-badge__text" x={p.x + 22} y={p.y - 16}>
                    {loc.monsters > 1 ? loc.monsters : '☠'}
                  </text>
                </>
              )}

              {name && (
                <text className="node__label" x={p.x} y={p.y + LABEL_DY}>
                  {name}
                </text>
              )}

              {stock && (
                <text className="node__sub" x={p.x} y={p.y + STOCK_DY}>
                  {stock}
                </text>
              )}

              {here.map((pid, i) => {
                const idx = state.turnOrder.indexOf(pid);
                const spread = (i - (here.length - 1) / 2) * 15;
                return (
                  <circle
                    key={pid}
                    className={`pawn ${pid === activePlayer ? 'pawn--active' : ''}`}
                    cx={p.x + spread}
                    cy={p.y + PAWN_DY}
                    r={6}
                    fill={PAWN_COLORS[idx % PAWN_COLORS.length]}
                  />
                );
              })}
            </g>
          );
        })}
      </svg>
    </section>
  );
}
