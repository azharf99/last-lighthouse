import { useCallback, useEffect, useRef, useState } from 'react';
import { api } from './api';
import type { GameSession, SessionPhase } from './useGameSession';
import type { Command, GameEvent, PlayerID, PlayerView } from '../types';

interface InEnvelope {
  v: number;
  type: string;
  matchId?: string;
  clientSeq?: number;
  payload?: any;
}

interface OutEnvelope {
  v: number;
  type: string;
  matchId?: string;
  eventSeq?: number;
  payload?: any;
}

export function useOnlineSession(
  matchId: string,
  playerId: PlayerID,
  token: string,
): GameSession {
  const [phase, setPhase] = useState<SessionPhase>('loading');
  const [error, setError] = useState<string | null>(null);
  const [view, setView] = useState<PlayerView | null>(null);
  const [legal, setLegal] = useState<Command[]>([]);
  const [log, setLog] = useState<GameEvent[]>([]);
  const [rejection, setRejection] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const clientSeqRef = useRef<number>(1);
  const lastEventSeqRef = useRef<number>(0);
  const reconnectTimerRef = useRef<number | null>(null);

  const connect = useCallback(() => {
    if (!matchId || !playerId || !token) return;

    try {
      const url = api.getWebSocketURL(token);
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        setPhase('ready');
        setError(null);

        // Join match
        const joinEnv: InEnvelope = {
          v: 1,
          type: 'join',
          matchId,
          payload: {
            player: playerId,
            lastEventSeq: lastEventSeqRef.current,
          },
        };
        ws.send(JSON.stringify(joinEnv));
      };

      ws.onmessage = (event) => {
        try {
          const msg: OutEnvelope = JSON.parse(event.data);

          if (msg.eventSeq) {
            lastEventSeqRef.current = msg.eventSeq;
          }

          if (msg.type === 'snapshot') {
            const v: PlayerView = msg.payload;
            setView(v);
            computeLocalLegal(v);
          } else if (msg.type === 'events') {
            const events: GameEvent[] = msg.payload || [];
            setLog((prev) => [...prev, ...events]);

            // Request fresh snapshot / resync after new events
            if (wsRef.current?.readyState === WebSocket.OPEN) {
              wsRef.current.send(
                JSON.stringify({
                  v: 1,
                  type: 'resync',
                  matchId,
                  clientSeq: 0,
                }),
              );
            }
          } else if (msg.type === 'error') {
            const errPayload = msg.payload;
            if (errPayload?.code === 'RULE_REJECTION') {
              setRejection(errPayload.message || 'Aksi ditolak aturan.');
            } else {
              setError(errPayload?.message || 'Terjadi kesalahan server.');
            }
          }
        } catch (e) {
          console.error('Failed to parse WS message:', e);
        }
      };

      ws.onerror = () => {
        setError('Koneksi WebSocket terputus.');
      };

      ws.onclose = () => {
        wsRef.current = null;
        // Exponential backoff reconnect
        reconnectTimerRef.current = window.setTimeout(() => {
          connect();
        }, 2000);
      };
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setPhase('error');
    }
  }, [matchId, playerId, token]);

  const computeLocalLegal = (v: PlayerView) => {
    // Generates conservative legal commands based on projected PlayerView
    const st = v.state;

    // Handle pending choices first (GDD §20 / ADR-006)
    if (st.pending) {
      if (st.pending.player === v.viewer) {
        const pendingCmds: Command[] = [];
        if (st.pending.kind === 'mystery_option' && st.pending.options) {
          for (const opt of st.pending.options) {
            pendingCmds.push({ kind: 'choose', player: v.viewer, option: opt });
          }
        } else if (st.pending.kind === 'mystery_card' && st.pending.cards) {
          for (const card of st.pending.cards) {
            pendingCmds.push({ kind: 'choose', player: v.viewer, card: card });
          }
        }
        setLegal(pendingCmds);
      } else {
        setLegal([]);
      }
      return;
    }

    const activeId = st.status === 'active' ? st.turnOrder[st.activeIdx] : null;
    if (activeId !== v.viewer) {
      setLegal([]);
      return;
    }

    const p = st.players.find((pl) => pl.id === v.viewer);
    if (!p) {
      setLegal([]);
      return;
    }

    const cmds: Command[] = [];
    const loc = st.board.locations.find((l) => l.id === p.at);

    if (loc) {
      // Move to explored adjacents
      for (const adj of loc.adjacent) {
        const target = st.board.locations.find((l) => l.id === adj);
        if (target && target.explored) {
          cmds.push({ kind: 'move', player: p.id, to: adj });
        } else if (target && !target.explored) {
          cmds.push({ kind: 'explore', player: p.id, to: adj });
        }
      }

      // Gather
      if (!loc.gatherBlocked && loc.available) {
        for (const [res, count] of Object.entries(loc.available)) {
          if (count && count > 0) {
            cmds.push({ kind: 'gather', player: p.id, resource: res as any });
          }
        }
      }

      // Repair
      const nextComp = st.lighthouse.find((c) => !c.repaired);
      if (loc.type === 'lighthouse' && nextComp) {
        cmds.push({
          kind: 'repair',
          player: p.id,
          component: nextComp.id,
          pay: p.inventory,
        });
      }

      // Fight
      if (loc.monsters > 0 && !p.exhausted) {
        cmds.push({ kind: 'fight', player: p.id });
      }

      // Investigate
      if (!loc.investigated && (loc.type === 'ruins' || loc.type === 'temple' || true)) {
        cmds.push({ kind: 'investigate', player: p.id });
      }

      // Rest
      if (loc.monsters === 0 && p.health < 3) {
        cmds.push({ kind: 'rest', player: p.id });
      }
    }

    // End Turn
    cmds.push({ kind: 'end_turn', player: p.id });
    setLegal(cmds);
  };

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect]);

  const send = useCallback(
    (cmd: Command) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        setRejection('Koneksi server terputus. Menghubungkan kembali...');
        return;
      }

      setBusy(true);
      const seq = clientSeqRef.current++;
      const env: InEnvelope = {
        v: 1,
        type: 'cmd',
        matchId,
        clientSeq: seq,
        payload: cmd,
      };

      wsRef.current.send(JSON.stringify(env));
      setTimeout(() => setBusy(false), 150);
    },
    [matchId],
  );

  const restart = useCallback(() => {
    // In online mode, restart is handled by lobby
  }, []);

  const confirmHandoff = useCallback(() => {}, []);
  const dismissRejection = useCallback(() => setRejection(null), []);

  return {
    phase,
    error,
    view,
    legal,
    log,
    rejection,
    dismissRejection,
    awaitingHandoff: false,
    handoffTo: null,
    confirmHandoff,
    send,
    restart,
    busy,
  };
}
