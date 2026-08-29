// Facade sesi permainan.
//
// UI berbicara HANYA ke lapisan ini, tidak pernah langsung ke WASM. Itu yang
// membuat mode online nanti (ADR-003) bisa masuk sebagai implementasi kedua di
// balik antarmuka yang sama, tanpa satu pun komponen UI berubah:
//
//   offline (M0) : send() -> core.wasm Decide  -> event
//   online  (M2) : send() -> WebSocket ke server -> event terproyeksi
//
// Di kedua kasus UI menerima hal yang sama: PlayerView, daftar command legal,
// dan aliran event.

import { useCallback, useEffect, useRef, useState } from 'react';
import { coreApi, RuleRejection } from '../wasm/loader';
import type { Command, GameEvent, PlayerID, PlayerView } from '../types';

export interface SeatConfig {
  id: string;
  name: string;
  character: string;
}

export type SessionPhase = 'loading' | 'ready' | 'error';

export interface GameSession {
  phase: SessionPhase;
  error: string | null;
  /** View milik pemain yang sedang bergiliran. */
  view: PlayerView | null;
  legal: Command[];
  log: GameEvent[];
  /** Penolakan aturan terakhir, ditampilkan sebagai toast lalu hilang. */
  rejection: string | null;
  dismissRejection(): void;
  /** True saat perangkat harus dioper ke pemain berikutnya (hotseat). */
  awaitingHandoff: boolean;
  handoffTo: PlayerID | null;
  confirmHandoff(): void;
  send(cmd: Command): void;
  restart(seed?: number): void;
  busy: boolean;
}

/**
 * Menjalankan satu match hotseat lokal di atas core.wasm.
 *
 * Catatan soal informasi tersembunyi: di hotseat seluruh state ada di perangkat
 * ini, dan itu memang diterima (ADR-006) -- para pemain duduk di meja yang sama.
 * Yang tetap dijaga adalah agar layar tidak menampilkan objective pemain lain,
 * lewat gerbang oper-perangkat di antara giliran. Tanpa itu, kartu rahasia
 * GDD 24 jadi tidak rahasia sama sekali dalam praktik.
 */
export function useGameSession(seats: SeatConfig[], initialSeed: number): GameSession {
  const [phase, setPhase] = useState<SessionPhase>('loading');
  const [error, setError] = useState<string | null>(null);
  const [view, setView] = useState<PlayerView | null>(null);
  const [legal, setLegal] = useState<Command[]>([]);
  const [log, setLog] = useState<GameEvent[]>([]);
  const [rejection, setRejection] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [awaitingHandoff, setAwaitingHandoff] = useState(false);
  const [handoffTo, setHandoffTo] = useState<PlayerID | null>(null);

  const handleRef = useRef<number | null>(null);
  const lastActiveRef = useRef<PlayerID | null>(null);
  const seedRef = useRef(initialSeed);

  /**
   * Menyegarkan view dan daftar aksi legal.
   *
   * Dua panggilan: yang pertama tanpa viewer untuk membaca state publik dan tahu
   * siapa yang sedang bergiliran, yang kedua atas nama pemain itu. Panggilan
   * pertama sekaligus berfungsi sebagai spectator view -- Project mengosongkan
   * semua rahasia karena tidak ada pemain yang cocok.
   */
  const refresh = useCallback(async (handle: number, silentHandoff: boolean) => {
    const publicView = await coreApi.view(handle, '');
    const st = publicView.state;
    const activeId = st.turnOrder[st.activeIdx] ?? null;

    if (st.status !== 'active' || !activeId) {
      setView(publicView);
      setLegal([]);
      setLog(await coreApi.events(handle, ''));
      setAwaitingHandoff(false);
      return;
    }

    const seatView = await coreApi.view(handle, activeId);
    const seatLegal = await coreApi.legal(handle, activeId);
    setView(seatView);
    setLegal(seatLegal);

    // Log diambil ulang dari core, bukan diakumulasi lokal, karena isinya
    // berbeda per pemain: baris yang boleh dilihat pemain yang sedang memegang
    // perangkat ditentukan core lewat ProjectEvents, bukan oleh UI.
    setLog(await coreApi.events(handle, activeId));

    // Giliran berpindah ke orang lain -> tutup layar sampai perangkat dioper.
    const changed = lastActiveRef.current !== null && lastActiveRef.current !== activeId;
    lastActiveRef.current = activeId;
    if (changed && !silentHandoff && seats.length > 1) {
      setHandoffTo(activeId);
      setAwaitingHandoff(true);
    }
  }, [seats.length]);

  const start = useCallback(
    async (seed: number) => {
      setPhase('loading');
      setError(null);
      setRejection(null);
      setAwaitingHandoff(false);
      lastActiveRef.current = null;

      try {
        if (handleRef.current !== null) {
          await coreApi.dispose(handleRef.current).catch(() => undefined);
          handleRef.current = null;
        }

        const { handle } = await coreApi.newGame({
          matchId: `hotseat-${seed}`,
          seed,
          players: seats,
        });
        handleRef.current = handle;
        await refresh(handle, true);
        setPhase('ready');
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
        setPhase('error');
      }
    },
    [refresh, seats],
  );

  useEffect(() => {
    void start(seedRef.current);
    return () => {
      if (handleRef.current !== null) {
        void coreApi.dispose(handleRef.current).catch(() => undefined);
      }
    };
    // Sengaja hanya sekali: seed berikutnya masuk lewat restart().
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const send = useCallback(
    (cmd: Command) => {
      const handle = handleRef.current;
      if (handle === null || busy || awaitingHandoff) return;

      setBusy(true);
      void (async () => {
        try {
          await coreApi.decide(handle, cmd);
          await refresh(handle, false);
        } catch (e) {
          if (e instanceof RuleRejection) {
            // Penolakan aturan adalah hasil yang wajar, bukan kerusakan:
            // prediksi legalitas di client memang konservatif (lihat legal.go).
            setRejection(e.message);
          } else {
            setError(e instanceof Error ? e.message : String(e));
            setPhase('error');
          }
        } finally {
          setBusy(false);
        }
      })();
    },
    [busy, awaitingHandoff, refresh],
  );

  const restart = useCallback(
    (seed?: number) => {
      seedRef.current = seed ?? Math.floor(Math.random() * 2_000_000_000);
      void start(seedRef.current);
    },
    [start],
  );

  const confirmHandoff = useCallback(() => setAwaitingHandoff(false), []);
  const dismissRejection = useCallback(() => setRejection(null), []);

  return {
    phase,
    error,
    view,
    legal,
    log,
    rejection,
    dismissRejection,
    awaitingHandoff,
    handoffTo,
    confirmHandoff,
    send,
    restart,
    busy,
  };
}
