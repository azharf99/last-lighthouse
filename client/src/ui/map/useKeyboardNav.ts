import { useEffect, useCallback } from 'react';
import type { LocationID, GameState, Command } from '../../types';

// The map layout coordinates for directional calculation
const LAYOUT: Record<string, { x: number; y: number }> = {
  lighthouse: { x: 640, y: 120 },
  harbor: { x: 280, y: 260 },
  forest: { x: 1000, y: 260 },
  village: { x: 540, y: 420 },
  cave: { x: 1060, y: 500 },
  ruins: { x: 700, y: 640 },
  site_c: { x: 180, y: 520 },
  site_a: { x: 420, y: 720 },
  site_b: { x: 1100, y: 720 },
};

const FALLBACK = { x: 640, y: 400 };

type Direction = 'up' | 'down' | 'left' | 'right';

function keyToDirection(key: string): Direction | null {
  switch (key) {
    case 'w': case 'W': case 'ArrowUp': return 'up';
    case 's': case 'S': case 'ArrowDown': return 'down';
    case 'a': case 'A': case 'ArrowLeft': return 'left';
    case 'd': case 'D': case 'ArrowRight': return 'right';
    default: return null;
  }
}

/**
 * Given the current location and a direction, find the best adjacent location
 * in that direction based on geometric positions.
 */
function findBestTarget(
  currentLocId: LocationID,
  direction: Direction,
  adjacentIds: LocationID[],
): LocationID | null {
  if (adjacentIds.length === 0) return null;

  const current = LAYOUT[currentLocId] ?? FALLBACK;

  // Filter candidates that are actually in the correct general direction
  const candidates = adjacentIds
    .map((id) => {
      const pos = LAYOUT[id] ?? FALLBACK;
      const dx = pos.x - current.x;
      const dy = pos.y - current.y;
      return { id, dx, dy, dist: Math.sqrt(dx * dx + dy * dy) };
    })
    .filter((c) => {
      // Allow some tolerance (45 degree cone + fallback)
      switch (direction) {
        case 'up': return c.dy < 0; // target is above
        case 'down': return c.dy > 0;
        case 'left': return c.dx < 0;
        case 'right': return c.dx > 0;
      }
    });

  if (candidates.length === 0) {
    // If no candidate in the exact direction, pick the closest adjacent
    // that has the best directional component
    const scored = adjacentIds.map((id) => {
      const pos = LAYOUT[id] ?? FALLBACK;
      const dx = pos.x - current.x;
      const dy = pos.y - current.y;
      let score = 0;
      switch (direction) {
        case 'up': score = -dy; break;
        case 'down': score = dy; break;
        case 'left': score = -dx; break;
        case 'right': score = dx; break;
      }
      return { id, score };
    });
    scored.sort((a, b) => b.score - a.score);
    return scored[0]?.id ?? null;
  }

  // Among filtered candidates, pick the closest
  candidates.sort((a, b) => a.dist - b.dist);
  return candidates[0]?.id ?? null;
}

export interface KeyboardNavCallbacks {
  onMove(to: LocationID): void;
  onExplore(to: LocationID): void;
  onOpenActions(): void;
  onCloseModal(): void;
}

/**
 * Hook that listens for WASD/Arrow keys and triggers movement commands.
 * Also handles Space/Enter for action menu and Escape for closing modals.
 */
export function useKeyboardNav(
  state: GameState | null,
  legal: Command[],
  activePlayer: string | null,
  enabled: boolean,
  callbacks: KeyboardNavCallbacks,
) {
  const reachableSet = new Set(
    legal.filter((c) => c.kind === 'move' && c.to).map((c) => c.to as LocationID),
  );
  const explorableSet = new Set(
    legal.filter((c) => c.kind === 'explore' && c.to).map((c) => c.to as LocationID),
  );

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!enabled || !state || !activePlayer) return;
      // Don't intercept if user is typing in an input
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;

      const direction = keyToDirection(e.key);

      if (direction) {
        e.preventDefault();
        const player = state.players.find((p) => p.id === activePlayer);
        if (!player) return;

        const currentLoc = state.board.locations.find((l) => l.id === player.at);
        if (!currentLoc) return;

        // Combine reachable and explorable as possible targets
        const possibleTargets = currentLoc.adjacent.filter(
          (adj) => reachableSet.has(adj) || explorableSet.has(adj),
        );

        const target = findBestTarget(player.at, direction, possibleTargets);
        if (target) {
          if (explorableSet.has(target)) {
            callbacks.onExplore(target);
          } else if (reachableSet.has(target)) {
            callbacks.onMove(target);
          }
        }
        return;
      }

      // Space or Enter: open action menu
      if (e.key === ' ' || e.key === 'Enter') {
        e.preventDefault();
        callbacks.onOpenActions();
        return;
      }

      // Escape: close modal
      if (e.key === 'Escape') {
        e.preventDefault();
        callbacks.onCloseModal();
        return;
      }
    },
    [enabled, state, activePlayer, reachableSet, explorableSet, callbacks],
  );

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);
}
