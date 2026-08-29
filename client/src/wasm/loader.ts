// Loader untuk core.wasm.
//
// Modul ini adalah satu-satunya tempat di client yang tahu tentang WebAssembly.
// Semua lapisan di atasnya bicara ke session facade (session/), sehingga
// nantinya transport online (WebSocket, ADR-003) bisa menggantikan WASM tanpa
// menyentuh satu pun komponen UI.

import type { Command, GameEvent, PlayerView } from '../types';

/** Bentuk hasil seragam dari binding Go (lihat cmd/wasm/main.go). */
interface WasmResult {
  ok: boolean;
  data?: string;
  error?: string;
}

interface WasmAPI {
  newGame(configJSON: string): WasmResult;
  decide(handle: number, commandJSON: string): WasmResult;
  view(handle: number, playerId: string): WasmResult;
  legal(handle: number, playerId: string): WasmResult;
  events(handle: number, playerId: string): WasmResult;
  dispose(handle: number): WasmResult;
  contentHash(): WasmResult;
}

declare global {
  interface Window {
    Go: new () => {
      importObject: WebAssembly.Imports;
      run(instance: WebAssembly.Instance): Promise<void>;
    };
    LastLighthouseCore?: WasmAPI;
  }
}

/** Error yang berasal dari penolakan aturan, bukan dari kegagalan teknis. */
export class RuleRejection extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'RuleRejection';
  }
}

let loading: Promise<WasmAPI> | null = null;

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const el = document.createElement('script');
    el.src = src;
    el.onload = () => resolve();
    el.onerror = () => reject(new Error(`gagal memuat ${src}`));
    document.head.appendChild(el);
  });
}

/**
 * Memuat core.wasm sekali dan mengembalikan binding-nya.
 *
 * Pemanggilan berulang berbagi promise yang sama: rules engine bersifat global
 * untuk seluruh aplikasi, dan memuatnya dua kali akan menggandakan ~4 MB memori
 * tanpa manfaat.
 */
export function loadCore(): Promise<WasmAPI> {
  if (loading) return loading;

  loading = (async () => {
    await loadScript(`${import.meta.env.BASE_URL}wasm/wasm_exec.js`);

    const go = new window.Go();
    const response = await fetch(`${import.meta.env.BASE_URL}wasm/core.wasm`);
    if (!response.ok) {
      throw new Error(`core.wasm tidak bisa diambil: HTTP ${response.status}`);
    }

    // instantiateStreaming men-stream kompilasi bersamaan dengan unduhan, yang
    // penting karena payload-nya ~1 MB gzip. Fallback dipakai kalau server
    // tidak mengirim Content-Type: application/wasm.
    let result: WebAssembly.WebAssemblyInstantiatedSource;
    try {
      result = await WebAssembly.instantiateStreaming(response.clone(), go.importObject);
    } catch {
      const buf = await response.arrayBuffer();
      result = await WebAssembly.instantiate(buf, go.importObject);
    }

    // main() Go memblokir di select{} agar callback tetap hidup, jadi run()
    // sengaja TIDAK di-await -- ia hanya selesai kalau program Go berhenti.
    void go.run(result.instance);

    const api = window.LastLighthouseCore;
    if (!api) {
      throw new Error('core.wasm dimuat tapi tidak mendaftarkan binding-nya');
    }
    return api;
  })();

  return loading;
}

/** Membuka hasil WasmResult, mengubah kegagalan jadi exception bertipe. */
function unwrap<T>(res: WasmResult): T {
  if (!res.ok) throw new RuleRejection(res.error ?? 'penolakan tanpa keterangan');
  return JSON.parse(res.data ?? 'null') as T;
}

export interface NewGameConfig {
  matchId: string;
  seed: number;
  players: { id: string; name: string; character: string }[];
}

export interface CoreHandle {
  handle: number;
  setupEvents: GameEvent[];
}

export const coreApi = {
  async contentHash(): Promise<string> {
    const api = await loadCore();
    return unwrap<{ hash: string }>(api.contentHash()).hash;
  },

  async newGame(cfg: NewGameConfig): Promise<CoreHandle> {
    const api = await loadCore();
    const res = unwrap<{ handle: number; events: GameEvent[] }>(
      api.newGame(JSON.stringify(cfg)),
    );
    return { handle: res.handle, setupEvents: res.events };
  },

  async decide(handle: number, cmd: Command): Promise<GameEvent[]> {
    const api = await loadCore();
    return unwrap<{ events: GameEvent[] }>(api.decide(handle, JSON.stringify(cmd))).events;
  },

  async view(handle: number, player: string): Promise<PlayerView> {
    const api = await loadCore();
    return unwrap<PlayerView>(api.view(handle, player));
  },

  async legal(handle: number, player: string): Promise<Command[]> {
    const api = await loadCore();
    return unwrap<Command[] | null>(api.legal(handle, player)) ?? [];
  },

  /** Log yang sudah diproyeksikan untuk pemain itu; '' = tampilan penonton. */
  async events(handle: number, player: string): Promise<GameEvent[]> {
    const api = await loadCore();
    return unwrap<GameEvent[] | null>(api.events(handle, player)) ?? [];
  },

  async dispose(handle: number): Promise<void> {
    const api = await loadCore();
    unwrap<unknown>(api.dispose(handle));
  },
};
