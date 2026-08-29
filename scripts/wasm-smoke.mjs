// Smoke test untuk core.wasm.
//
// Ini memverifikasi rantai penuh ADR-002: aturan yang sama yang dijalankan
// server juga berjalan di dalam webview, lewat binding JS, dan menghasilkan
// permainan yang bisa dimainkan. Kalau ini lolos, mode hotseat offline punya
// fondasi yang benar.
//
// Jalankan: node scripts/wasm-smoke.mjs

import fs from 'node:fs';
import { webcrypto } from 'node:crypto';
import { performance } from 'node:perf_hooks';

globalThis.crypto ??= webcrypto;
globalThis.performance ??= performance;

const WASM_DIR = 'client/public/wasm';

// wasm_exec.js adalah skrip klasik (bukan modul), jadi ia dievaluasi manual
// agar mendaftarkan globalThis.Go.
const shim = fs.readFileSync(`${WASM_DIR}/wasm_exec.js`, 'utf8');
new Function(shim).call(globalThis);

const go = new globalThis.Go();
const bytes = fs.readFileSync(`${WASM_DIR}/core.wasm`);
const { instance } = await WebAssembly.instantiate(bytes, go.importObject);

// main() Go memblokir di select{} supaya callback tetap hidup, jadi run() tidak
// boleh di-await -- ia baru selesai saat program Go keluar.
go.run(instance);

const core = globalThis.LastLighthouseCore;
if (!core) {
  console.error('GAGAL: LastLighthouseCore tidak terdaftar di global');
  process.exit(1);
}

let failures = 0;
const check = (label, cond, detail = '') => {
  if (cond) {
    console.log(`  ok   ${label}`);
  } else {
    console.error(`  FAIL ${label} ${detail}`);
    failures++;
  }
};

const unwrap = (res, what) => {
  if (!res.ok) throw new Error(`${what}: ${res.error}`);
  return JSON.parse(res.data);
};

console.log('core.wasm smoke test\n');

// --- content hash ---
const { hash } = unwrap(core.contentHash(), 'contentHash');
check('content hash tersedia', typeof hash === 'string' && hash.length > 0, `got ${hash}`);

// --- buat game ---
const started = unwrap(
  core.newGame(
    JSON.stringify({
      matchId: 'm_smoke',
      seed: 20260829,
      players: [
        { id: 'p1', name: 'Ana', character: 'navigator' },
        { id: 'p2', name: 'Budi', character: 'engineer' },
      ],
    }),
  ),
  'newGame',
);
const handle = started.handle;
check('game dibuat', Number.isInteger(handle) && handle > 0, `handle=${handle}`);
check('setup menghasilkan event', started.events.length > 0, `n=${started.events.length}`);

// --- projection: p1 tidak boleh melihat objective p2 ---
const v1 = unwrap(core.view(handle, 'p1'), 'view p1');
const v2 = unwrap(core.view(handle, 'p2'), 'view p2');

check('p1 melihat objective miliknya', !!v1.myObjective, `got ${v1.myObjective}`);
check(
  'objective p2 tidak bocor ke view p1',
  !JSON.stringify(v1).includes(v2.myObjective),
  `objective p2 = ${v2.myObjective}`,
);
check('RNG state tidak bocor', v1.state.rngState === 0 || v1.state.rngState === undefined);

// --- mainkan sampai selesai memakai bot acak ---
let steps = 0;
let lastError = null;
const MAX_STEPS = 500;

while (steps < MAX_STEPS) {
  const view = unwrap(core.view(handle, 'p1'), 'view');
  const st = view.state;
  if (st.status === 'won' || st.status === 'lost') break;

  const activeId = st.turnOrder[st.activeIdx];
  const legal = unwrap(core.legal(handle, activeId), 'legal');
  if (!legal || legal.length === 0) {
    lastError = `tidak ada aksi legal untuk ${activeId} di ronde ${st.round}`;
    break;
  }

  // Pilihan deterministik supaya smoke test tidak flaky.
  const cmd = legal[steps % legal.length];
  const res = core.decide(handle, JSON.stringify(cmd));
  if (!res.ok) {
    lastError = `decide menolak ${cmd.kind}: ${res.error}`;
    break;
  }
  steps++;
}

const finalView = unwrap(core.view(handle, 'p1'), 'view akhir');
const final = finalView.state;

check('tidak ada error selama permainan', lastError === null, lastError ?? '');
check('permainan berjalan maju', steps > 10, `steps=${steps}`);
check(
  'permainan mencapai kondisi akhir',
  final.status === 'won' || final.status === 'lost',
  `status=${final.status}, round=${final.round}, darkness=${final.darkness}`,
);

const log = unwrap(core.events(handle), 'events');
check('event log terkumpul', log.length > steps, `n=${log.length}`);

unwrap(core.dispose(handle), 'dispose');
const afterDispose = core.view(handle, 'p1');
check('handle yang sudah dibuang ditolak', afterDispose.ok === false);

console.log(
  `\nRingkasan: ${steps} aksi, status akhir "${final.status}", ` +
    `ronde ${final.round}, darkness ${final.darkness}, ${log.length} event`,
);

if (failures > 0) {
  console.error(`\n${failures} pemeriksaan GAGAL`);
  process.exit(1);
}
console.log('\nSemua pemeriksaan lolos.');
process.exit(0);
