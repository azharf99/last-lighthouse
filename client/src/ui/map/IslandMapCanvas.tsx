import { useEffect, useRef, useState } from 'react';
import { Application, Container, Graphics, Text, TextStyle } from 'pixi.js';
import type { Command, GameState, LocationID, PlayerID } from '../../types';
import { RESOURCES } from '../../types';
import { locationName, RESOURCE_GLYPH } from '../labels';
import { sfx } from '../../audio/sfx';
import { createBiomeTile, HALF_BIOME } from './BiomeRenderer';
import { createHumanoidStandee } from './CharacterRenderer';
import { createMonsterStandee } from './MonsterRenderer';
import { AnimationEngine } from './AnimationEngine';

// Expanded cinematic 16:10 resolution layout
const VIEW_W = 920;
const VIEW_H = 580;

const LAYOUT: Record<string, { x: number; y: number }> = {
  lighthouse: { x: 460, y: 75 },
  harbor: { x: 200, y: 175 },
  forest: { x: 720, y: 175 },
  village: { x: 390, y: 285 },
  cave: { x: 760, y: 340 },
  ruins: { x: 500, y: 440 },
  site_c: { x: 130, y: 360 },
  site_a: { x: 300, y: 500 },
  site_b: { x: 790, y: 495 },
};

const FALLBACK = { x: 460, y: 290 };

const PAWN_COLORS = [0xffc76b, 0x7ad7e8, 0x8fc07a, 0xe58fa9];

interface Props {
  state: GameState;
  legal: Command[];
  activePlayer: PlayerID | null;
  onMove(to: LocationID): void;
  onExplore(to: LocationID): void;
}

export function IslandMapCanvas({ state, legal, activePlayer, onMove, onExplore }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<Application | null>(null);
  const prevPosMapRef = useRef<Map<PlayerID, LocationID>>(new Map());
  const [zoom, setZoom] = useState<number>(1);

  const reachable = new Set(
    legal.filter((c) => c.kind === 'move' && c.to).map((c) => c.to as LocationID),
  );
  const explorableSet = new Set(
    legal.filter((c) => c.kind === 'explore' && c.to).map((c) => c.to as LocationID),
  );

  const pos = (id: LocationID) => LAYOUT[id] ?? FALLBACK;

  useEffect(() => {
    let isCancelled = false;
    const container = containerRef.current;
    if (!container) return;

    const app = new Application();

    const initPixi = async () => {
      await app.init({
        width: VIEW_W,
        height: VIEW_H,
        backgroundAlpha: 0,
        antialias: true,
        resolution: window.devicePixelRatio || 1,
        autoDensity: true,
      });

      if (isCancelled || !container) {
        app.destroy(true);
        return;
      }

      container.innerHTML = '';
      container.appendChild(app.canvas);
      appRef.current = app;

      // Master World Stage Container (supports smooth Zoom & Pan)
      const world = new Container();
      world.x = VIEW_W / 2;
      world.y = VIEW_H / 2;
      world.pivot.x = VIEW_W / 2;
      world.pivot.y = VIEW_H / 2;
      world.scale.set(zoom);
      app.stage.addChild(world);

      // 0. Ocean Background & Shorelines
      const oceanGfx = new Graphics();

      // Island Sandy Shore Contour (soft sandy peninsula underlay)
      oceanGfx
        .poly([
          { x: 460, y: 35 },
          { x: 770, y: 130 },
          { x: 860, y: 340 },
          { x: 840, y: 530 },
          { x: 260, y: 540 },
          { x: 70, y: 370 },
          { x: 130, y: 130 },
        ])
        .fill({ color: 0x0a1420, alpha: 0.85 })
        .stroke({ width: 3, color: 0x16283f, alpha: 0.6 });

      // Subtle ocean ripple lines
      oceanGfx.arc(460, 290, 420, 0, Math.PI * 2).stroke({ width: 1, color: 0x1d3557, alpha: 0.15 });
      oceanGfx.arc(460, 290, 360, 0, Math.PI * 2).stroke({ width: 1, color: 0x1d3557, alpha: 0.2 });

      world.addChild(oceanGfx);

      // 1. Cobblestone Trails / Path Edges Layer
      const edgesGfx = new Graphics();
      world.addChild(edgesGfx);

      const activePlayerObj = state.players.find((p) => p.id === activePlayer);
      const currentLoc = activePlayerObj?.at;

      const drawnEdges = new Set<string>();
      for (const loc of state.board.locations) {
        for (const adj of loc.adjacent) {
          const key = loc.id < adj ? `${loc.id}-${adj}` : `${adj}-${loc.id}`;
          if (!drawnEdges.has(key)) {
            drawnEdges.add(key);
            const pa = pos(loc.id);
            const pb = pos(adj);

            // Active path connects current player's location to a reachable/explorable target
            const isPathActive = Boolean(
              currentLoc &&
                ((loc.id === currentLoc && (reachable.has(adj) || explorableSet.has(adj))) ||
                  (adj === currentLoc && (reachable.has(loc.id) || explorableSet.has(loc.id)))),
            );

            // Path trail underlay
            edgesGfx
              .moveTo(pa.x, pa.y)
              .lineTo(pb.x, pb.y)
              .stroke({
                width: isPathActive ? 5 : 3,
                color: isPathActive ? 0xffc76b : 0x223245,
                alpha: isPathActive ? 0.95 : 0.45,
              });

            // Luminescent golden dash overlay for active trail
            if (isPathActive) {
              edgesGfx
                .moveTo(pa.x, pa.y)
                .lineTo(pb.x, pb.y)
                .stroke({
                  width: 2,
                  color: 0xfff0cc,
                  alpha: 0.8,
                });
            }
          }
        }
      }

      // 2. Thematic Biome Plaza Nodes Layer
      const repairedCount = state.lighthouse.filter((c) => c.repaired).length;

      state.board.locations.forEach((loc) => {
        const p = pos(loc.id);
        const isReachable = reachable.has(loc.id);
        const explorable = explorableSet.has(loc.id);

        const nodeContainer = new Container();
        nodeContainer.x = p.x;
        nodeContainer.y = p.y;
        world.addChild(nodeContainer);

        // Biome Plaza Card
        const biomeTile = createBiomeTile(
          loc.type,
          loc.explored,
          isReachable,
          explorable,
          repairedCount,
        );
        nodeContainer.addChild(biomeTile);

        // Location Name Label
        if (loc.explored) {
          const name = locationName(loc.type, loc.name);
          const nameStyle = new TextStyle({
            fontSize: 12,
            fontWeight: 'bold',
            fill: isReachable ? 0xffc76b : 0xe8eef6,
            fontFamily: 'Segoe UI, sans-serif',
            dropShadow: {
              alpha: 0.85,
              blur: 4,
              color: 0x070d16,
              distance: 1,
            },
          });
          const nameText = new Text({ text: name, style: nameStyle });
          nameText.anchor.set(0.5, 0);
          nameText.y = HALF_BIOME + 5;
          nodeContainer.addChild(nameText);
        }

        // Available Resources Stock Badge
        if (loc.explored && !loc.gatherBlocked) {
          const stock = RESOURCES.filter((r) => (loc.available[r] ?? 0) > 0)
            .map((r) => `${RESOURCE_GLYPH[r]}${loc.available[r]}`)
            .join(' ');

          if (stock) {
            const stockStyle = new TextStyle({
              fontSize: 11,
              fill: 0x93a6bd,
              fontFamily: 'Segoe UI, sans-serif',
            });
            const stockText = new Text({ text: stock, style: stockStyle });
            stockText.anchor.set(0.5, 0);
            stockText.y = HALF_BIOME + 22;
            nodeContainer.addChild(stockText);
          }
        }

        // Illustrated Abyssal Dread Beast Monster Standee
        if (loc.monsters > 0) {
          const monsterStandee = createMonsterStandee(loc.monsters);
          monsterStandee.x = HALF_BIOME - 12;
          monsterStandee.y = -HALF_BIOME + 18;
          nodeContainer.addChild(monsterStandee);
        }

        // Interactive Click Handling
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

      // 3. Illustrated Humanoid Character Miniatures Layer with Walk Animation
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

          // Auto-spacing side-by-side inside location plaza
          const offsetX = (idx - (pids.length - 1) / 2) * 28;
          const targetX = pLoc.x + offsetX;
          const targetY = pLoc.y + 12;

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
              { x: startP.x + offsetX, y: startP.y + 12 },
              { x: targetX, y: targetY },
              650,
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
  }, [state, legal, activePlayer, zoom, reachable, explorableSet, onMove, onExplore]);

  return (
    <section className="panel map" aria-label="Peta pulau PixiJS" style={{ position: 'relative' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <h2 className="panel__title" style={{ margin: 0 }}>
          🗺️ Bentang Alam Pulau (Modular Biomes & Miniatures)
        </h2>
        {/* Zoom Controls */}
        <div style={{ display: 'flex', gap: 6 }}>
          <button
            className="action action--ghost"
            style={{ padding: '2px 8px', fontSize: 13 }}
            onClick={() => {
              sfx.playClick();
              setZoom((z) => Math.min(1.4, z + 0.1));
            }}
            title="Zoom In"
            aria-label="Perbesar tampilan peta (Zoom In)"
          >
            +
          </button>
          <button
            className="action action--ghost"
            style={{ padding: '2px 8px', fontSize: 13 }}
            onClick={() => {
              sfx.playClick();
              setZoom((z) => Math.max(0.7, z - 0.1));
            }}
            title="Zoom Out"
            aria-label="Perkecil tampilan peta (Zoom Out)"
          >
            -
          </button>
          <button
            className="action action--ghost"
            style={{ padding: '2px 8px', fontSize: 11 }}
            onClick={() => {
              sfx.playClick();
              setZoom(1);
            }}
            title="Reset Zoom"
            aria-label="Reset zoom peta ke ukuran 100%"
          >
            100%
          </button>
        </div>
      </div>

      {/* PixiJS Canvas Mount */}
      <div
        ref={containerRef}
        style={{
          width: '100%',
          height: VIEW_H,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          background: 'radial-gradient(ellipse at center, rgba(16, 28, 44, 0.6) 0%, rgba(7, 13, 22, 0.95) 100%)',
          borderRadius: 'var(--radius)',
          overflow: 'hidden',
          border: '1px solid var(--stone)',
        }}
      />
    </section>
  );
}
