import { useEffect, useRef, useState } from 'react';
import { Application, Container, Graphics } from 'pixi.js';
import type { Command, GameState, LocationID, PlayerID } from '../../types';
import { RESOURCES } from '../../types';
import { locationName, RESOURCE_GLYPH } from '../labels';
import { sfx } from '../../audio/sfx';
import { createBiomeTile, HALF_BIOME } from './BiomeRenderer';
import { createHumanoidStandee } from './CharacterRenderer';
import { createMonsterStandee } from './MonsterRenderer';
import { AnimationEngine } from './AnimationEngine';
import { createPixelText } from './PixelFont';

// Expanded world dimensions for fullscreen retro overworld
const WORLD_W = 1400;
const WORLD_H = 900;

// Expanded map layout — locations spread wider like a Harvest Moon overworld
export const LAYOUT: Record<string, { x: number; y: number }> = {
  lighthouse: { x: 640, y: 120 },
  harbor:     { x: 280, y: 260 },
  forest:     { x: 1000, y: 260 },
  village:    { x: 540, y: 420 },
  cave:       { x: 1060, y: 500 },
  ruins:      { x: 700, y: 640 },
  site_c:     { x: 180, y: 520 },
  site_a:     { x: 420, y: 720 },
  site_b:     { x: 1100, y: 720 },
};

const FALLBACK = { x: 640, y: 400 };

const PAWN_COLORS = [0xffc76b, 0x7ad7e8, 0x8fc07a, 0xe58fa9];

// Retro ground color palette
const DIRT_COLOR = 0x73402e;
const WATER_COLOR = 0x394778;
const WATER_LIGHT = 0x3978a8;

interface Props {
  state: GameState;
  legal: Command[];
  activePlayer: PlayerID | null;
  onMove(to: LocationID): void;
  onExplore(to: LocationID): void;
}

/**
 * Draws a pixel-art ground tile pattern across the world background.
 * Creates a Harvest Moon overworld grass/earth feel using small rect blocks.
 */
function drawPixelBackground(gfx: Graphics, w: number, h: number) {
  const TILE = 16;

  // Fill base ocean
  gfx.rect(0, 0, w, h).fill({ color: WATER_COLOR });

  // Draw island landmass shape
  const islandPoly = [
    { x: 100, y: 80 },
    { x: 540, y: 40 },
    { x: 900, y: 50 },
    { x: 1200, y: 120 },
    { x: 1320, y: 300 },
    { x: 1300, y: 550 },
    { x: 1200, y: 750 },
    { x: 900, y: 820 },
    { x: 500, y: 830 },
    { x: 200, y: 780 },
    { x: 80, y: 600 },
    { x: 60, y: 350 },
  ];
  gfx.poly(islandPoly).fill({ color: 0x3b7d4f });

  // Pixel grass tiles on the island
  // Use a simple seeded pattern for variety
  for (let y = 0; y < h; y += TILE) {
    for (let x = 0; x < w; x += TILE) {
      // Check if inside island roughly
      const cx = x - w / 2;
      const cy = y - h / 2;
      const dist = Math.sqrt((cx * cx) / (500 * 500) + (cy * cy) / (360 * 360));
      if (dist > 1.05) continue;

      const hash = ((x * 7 + y * 13) % 97);
      if (hash < 20) {
        // Dark grass patches
        gfx.rect(x, y, TILE, TILE).fill({ color: 0x257179 });
      } else if (hash < 35) {
        // Dirt patches
        gfx.rect(x, y, TILE, TILE).fill({ color: DIRT_COLOR, alpha: 0.3 });
      } else if (hash < 40) {
        // Flower pixel (single colored dot)
        const flowerColors = [0xf7e26b, 0xe58fa9, 0x7ad7e8];
        const fColor = flowerColors[hash % 3];
        gfx.rect(x + 6, y + 6, 3, 3).fill({ color: fColor });
      }
    }
  }

  // Shoreline water edge pixels
  const shorePoints = [
    { x: 80, y: 80 }, { x: 540, y: 20 }, { x: 900, y: 30 },
    { x: 1220, y: 100 }, { x: 1340, y: 300 }, { x: 1320, y: 560 },
    { x: 1220, y: 770 }, { x: 900, y: 840 }, { x: 500, y: 850 },
    { x: 180, y: 800 }, { x: 60, y: 620 }, { x: 40, y: 360 },
  ];
  gfx.poly(shorePoints).stroke({ width: 6, color: WATER_LIGHT, alpha: 0.5 });
}

/**
 * Draws pixel-art dirt paths between connected locations.
 */
function drawPixelPaths(
  gfx: Graphics,
  locations: GameState['board']['locations'],
  currentLoc: LocationID | undefined,
  reachable: Set<LocationID>,
  explorableSet: Set<LocationID>,
) {
  const drawnEdges = new Set<string>();
  const pos = (id: LocationID) => LAYOUT[id] ?? FALLBACK;

  for (const loc of locations) {
    for (const adj of loc.adjacent) {
      const key = loc.id < adj ? `${loc.id}-${adj}` : `${adj}-${loc.id}`;
      if (drawnEdges.has(key)) continue;
      drawnEdges.add(key);

      const pa = pos(loc.id);
      const pb = pos(adj);

      const isPathActive = Boolean(
        currentLoc &&
          ((loc.id === currentLoc && (reachable.has(adj) || explorableSet.has(adj))) ||
            (adj === currentLoc && (reachable.has(loc.id) || explorableSet.has(loc.id)))),
      );

      // Base dirt path (pixel blocks)
      const dx = pb.x - pa.x;
      const dy = pb.y - pa.y;
      const steps = Math.ceil(Math.sqrt(dx * dx + dy * dy) / 8);

      for (let i = 0; i <= steps; i++) {
        const t = i / steps;
        const px = pa.x + dx * t;
        const py = pa.y + dy * t;
        const size = isPathActive ? 6 : 4;
        const color = isPathActive ? 0xffc76b : DIRT_COLOR;
        const alpha = isPathActive ? 0.9 : 0.6;
        gfx.rect(px - size / 2, py - size / 2, size, size).fill({ color, alpha });
      }

      // Active path golden highlights
      if (isPathActive) {
        for (let i = 0; i <= steps; i += 3) {
          const t = i / steps;
          const px = pa.x + dx * t;
          const py = pa.y + dy * t;
          gfx.rect(px - 1, py - 1, 3, 3).fill({ color: 0xf7e26b, alpha: 0.7 });
        }
      }
    }
  }
}

export function IslandMapCanvas({ state, legal, activePlayer, onMove, onExplore }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<Application | null>(null);
  const prevPosMapRef = useRef<Map<PlayerID, LocationID>>(new Map());
  const [canvasSize, setCanvasSize] = useState({ w: window.innerWidth, h: window.innerHeight });

  const reachable = new Set(
    legal.filter((c) => c.kind === 'move' && c.to).map((c) => c.to as LocationID),
  );
  const explorableSet = new Set(
    legal.filter((c) => c.kind === 'explore' && c.to).map((c) => c.to as LocationID),
  );

  const pos = (id: LocationID) => LAYOUT[id] ?? FALLBACK;

  // Track viewport resize
  useEffect(() => {
    const onResize = () => setCanvasSize({ w: window.innerWidth, h: window.innerHeight });
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  useEffect(() => {
    let isCancelled = false;
    const container = containerRef.current;
    if (!container) return;

    const app = new Application();

    const initPixi = async () => {
      await app.init({
        width: canvasSize.w,
        height: canvasSize.h,
        backgroundAlpha: 1,
        backgroundColor: WATER_COLOR,
        antialias: false,
        resolution: 1,
        autoDensity: false,
      });

      if (isCancelled || !container) {
        app.destroy(true);
        return;
      }

      container.innerHTML = '';
      const canvas = app.canvas as HTMLCanvasElement;
      canvas.style.imageRendering = 'pixelated';
      canvas.style.width = '100%';
      canvas.style.height = '100%';
      container.appendChild(canvas);
      appRef.current = app;

      // --- World container with camera follow ---
      const world = new Container();
      app.stage.addChild(world);

      // Camera: center on active player
      const activePlayerObj = state.players.find((p) => p.id === activePlayer);
      const cameraTarget = activePlayerObj ? pos(activePlayerObj.at) : { x: WORLD_W / 2, y: WORLD_H / 2 };

      // Calculate camera offset to center the target in viewport
      const camX = canvasSize.w / 2 - cameraTarget.x;
      const camY = canvasSize.h / 2 - cameraTarget.y;

      // Clamp camera so we don't show too much outside the world
      world.x = Math.min(0, Math.max(canvasSize.w - WORLD_W, camX));
      world.y = Math.min(0, Math.max(canvasSize.h - WORLD_H, camY));

      // 0. Pixel art background — Harvest Moon overworld
      const bgGfx = new Graphics();
      drawPixelBackground(bgGfx, WORLD_W, WORLD_H);
      world.addChild(bgGfx);

      // 1. Pixel dirt paths between locations
      const pathsGfx = new Graphics();
      const currentLoc = activePlayerObj?.at;
      drawPixelPaths(pathsGfx, state.board.locations, currentLoc, reachable, explorableSet);
      world.addChild(pathsGfx);

      // 2. Biome tiles at each location
      const repairedCount = state.lighthouse.filter((c) => c.repaired).length;

      state.board.locations.forEach((loc) => {
        const p = pos(loc.id);
        const isReachable = reachable.has(loc.id);
        const explorable = explorableSet.has(loc.id);

        const nodeContainer = new Container();
        nodeContainer.x = p.x;
        nodeContainer.y = p.y;
        world.addChild(nodeContainer);

        // Biome pixel-art tile
        const biomeTile = createBiomeTile(
          loc.type,
          loc.explored,
          isReachable,
          explorable,
          repairedCount,
        );
        nodeContainer.addChild(biomeTile);

        // Location name label (pixel font)
        if (loc.explored) {
          const name = locationName(loc.type, loc.name);
          const nameText = createPixelText(name, 7, isReachable ? 0xffc76b : 0xf4f4f4);
          nameText.anchor.set(0.5, 0);
          nameText.y = HALF_BIOME + 6;
          nodeContainer.addChild(nameText);
        }

        // Resource stock badge
        if (loc.explored && !loc.gatherBlocked) {
          const stock = RESOURCES.filter((r) => (loc.available[r] ?? 0) > 0)
            .map((r) => `${RESOURCE_GLYPH[r]}${loc.available[r]}`)
            .join(' ');

          if (stock) {
            const stockText = createPixelText(stock, 7, 0x8b9bb4);
            stockText.anchor.set(0.5, 0);
            stockText.y = HALF_BIOME + 20;
            nodeContainer.addChild(stockText);
          }
        }

        // Monster standee
        if (loc.monsters > 0) {
          const monsterStandee = createMonsterStandee(loc.monsters);
          monsterStandee.x = HALF_BIOME - 8;
          monsterStandee.y = -HALF_BIOME + 14;
          nodeContainer.addChild(monsterStandee);
        }

        // Click handler
        if (isReachable || explorable) {
          nodeContainer.eventMode = 'static';
          nodeContainer.cursor = 'pointer';
          nodeContainer.on('pointerdown', () => {
            if (explorable) {
              sfx.playMove();
              onExplore(loc.id);
            } else if (isReachable) {
              sfx.playMove();
              onMove(loc.id);
            }
          });
        }
      });

      // 3. Character standees with pixel walk animation
      const occupants = new Map<LocationID, PlayerID[]>();
      state.players.forEach((p) => {
        const list = occupants.get(p.at) ?? [];
        list.push(p.id);
        occupants.set(p.at, list);
      });

      occupants.forEach((pids, locId) => {
        const pLoc = pos(locId);

        pids.forEach((pid, idx) => {
          const pIndex = state.players.findIndex((p) => p.id === pid);
          const playerObj = state.players[pIndex];
          const color = PAWN_COLORS[pIndex % PAWN_COLORS.length];
          const isActive = pid === activePlayer;

          const offsetX = (idx - (pids.length - 1) / 2) * 24;
          const targetX = pLoc.x + offsetX;
          const targetY = pLoc.y + 10;

          const standee = createHumanoidStandee(
            playerObj.character,
            color,
            isActive,
            playerObj.exhausted,
            playerObj.name,
          );

          world.addChild(standee);

          const prevLoc = prevPosMapRef.current.get(pid);
          if (prevLoc && prevLoc !== locId) {
            const startP = pos(prevLoc);
            AnimationEngine.animateWalk(
              standee,
              { x: startP.x + offsetX, y: startP.y + 10 },
              { x: targetX, y: targetY },
              500,
            );
          } else {
            standee.x = targetX;
            standee.y = targetY;
          }

          prevPosMapRef.current.set(pid, locId);
        });
      });
    };

    void initPixi();

    return () => {
      isCancelled = true;
      if (appRef.current) {
        appRef.current.destroy(true, { children: true });
        appRef.current = null;
      }
    };
  }, [state, legal, activePlayer, canvasSize, reachable, explorableSet, onMove, onExplore]);

  return (
    <div
      ref={containerRef}
      style={{
        position: 'fixed',
        inset: 0,
        width: '100vw',
        height: '100vh',
        zIndex: 0,
        background: '#394778',
      }}
      aria-label="Peta pulau pixel-art 8-bit"
    />
  );
}
