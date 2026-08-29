import { useEffect, useRef, useState } from 'react';
import { Application, Container, Graphics, Text, TextStyle } from 'pixi.js';
import type { Command, GameState, LocationID, PlayerID } from '../../types';
import { RESOURCES } from '../../types';
import { locationName, RESOURCE_GLYPH } from '../labels';
import { sfx } from '../../audio/sfx';

const LAYOUT: Record<string, { x: number; y: number }> = {
  lighthouse: { x: 350, y: 65 },
  harbor: { x: 150, y: 155 },
  forest: { x: 550, y: 155 },
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
const NODE_R = 32;

const PAWN_COLORS = [0xffc76b, 0x7ad7e8, 0x8fc07a, 0xe58fa9];

const TYPE_ICONS: Record<string, string> = {
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

export function IslandMapCanvas({ state, legal, activePlayer, onMove, onExplore }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<Application | null>(null);
  const [zoom, setZoom] = useState<number>(1);

  const reachable = new Set(
    legal.filter((c) => c.kind === 'move' && c.to).map((c) => c.to as LocationID),
  );
  const explorableSet = new Set(
    legal.filter((c) => c.kind === 'explore' && c.to).map((c) => c.to as LocationID),
  );

  const pos = (id: LocationID) => LAYOUT[id] ?? FALLBACK;

  // Render PixiJS WebGL Canvas
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

      // Master World Stage Container (for Zoom / Pan)
      const world = new Container();
      world.x = VIEW_W / 2;
      world.y = VIEW_H / 2;
      world.pivot.x = VIEW_W / 2;
      world.pivot.y = VIEW_H / 2;
      world.scale.set(zoom);
      app.stage.addChild(world);

      // 1. Edges Layer
      const edgesGfx = new Graphics();
      world.addChild(edgesGfx);

      const drawnEdges = new Set<string>();
      for (const loc of state.board.locations) {
        for (const adj of loc.adjacent) {
          const key = loc.id < adj ? `${loc.id}-${adj}` : `${adj}-${loc.id}`;
          if (!drawnEdges.has(key)) {
            drawnEdges.add(key);
            const pa = pos(loc.id);
            const pb = pos(adj);

            const isPathActive =
              (reachable.has(loc.id) && loc.adjacent.includes(adj)) ||
              (reachable.has(adj) && loc.adjacent.includes(loc.id));

            edgesGfx
              .moveTo(pa.x, pa.y)
              .lineTo(pb.x, pb.y)
              .stroke({
                width: isPathActive ? 3.5 : 2,
                color: isPathActive ? 0xffc76b : 0x2b3d54,
                alpha: isPathActive ? 0.9 : 0.45,
              });
          }
        }
      }

      // 2. Lighthouse Beam Aura Layer
      const repairedCount = state.lighthouse.filter((c) => c.repaired).length;
      if (repairedCount > 0) {
        const lhPos = pos('lighthouse');
        const beam = new Graphics();
        beam
          .circle(lhPos.x, lhPos.y, 45 + repairedCount * 14)
          .fill({ color: 0xffc76b, alpha: 0.08 + repairedCount * 0.03 });
        world.addChild(beam);
      }

      // 3. Nodes Layer
      state.board.locations.forEach((loc) => {
        const p = pos(loc.id);
        const isReachable = reachable.has(loc.id);
        const explorable = explorableSet.has(loc.id);
        const isLighthouse = loc.type === 'lighthouse';

        const nodeContainer = new Container();
        nodeContainer.x = p.x;
        nodeContainer.y = p.y;
        world.addChild(nodeContainer);

        // Node Circle Graphic
        const nodeGfx = new Graphics();

        let fillColor = 0x131f30;
        let borderColor = 0x2b3d54;

        if (isLighthouse) {
          fillColor = 0x1f2e44;
          borderColor = 0xffc76b;
        } else if (!loc.explored) {
          fillColor = 0x090f19;
          borderColor = 0x1a2a3f;
        } else if (isReachable) {
          fillColor = 0x1a2e40;
          borderColor = 0x7ad7e8;
        }

        if (explorable) {
          borderColor = 0xffc76b;
        }

        nodeGfx.circle(0, 0, NODE_R).fill({ color: fillColor }).stroke({
          width: isReachable || explorable ? 3 : 2,
          color: borderColor,
        });

        // Pulsing glow for interactive reachable / explorable nodes
        if (isReachable || explorable) {
          nodeGfx.circle(0, 0, NODE_R + 4).stroke({
            width: 1.5,
            color: explorable ? 0xffc76b : 0x7ad7e8,
            alpha: 0.6,
          });
        }

        nodeContainer.addChild(nodeGfx);

        // Biome Icon
        const icon = loc.explored ? TYPE_ICONS[loc.type] || '📍' : '?';
        const iconStyle = new TextStyle({
          fontSize: 22,
          fill: 0xffffff,
        });
        const iconText = new Text({ text: icon, style: iconStyle });
        iconText.anchor.set(0.5);
        nodeContainer.addChild(iconText);

        // Node Name Label
        if (loc.explored) {
          const name = locationName(loc.type, loc.name);
          const nameStyle = new TextStyle({
            fontSize: 11,
            fontWeight: 'bold',
            fill: isReachable ? 0xffc76b : 0xe8eef6,
            fontFamily: 'Segoe UI, sans-serif',
          });
          const nameText = new Text({ text: name, style: nameStyle });
          nameText.anchor.set(0.5, 0);
          nameText.y = NODE_R + 6;
          nodeContainer.addChild(nameText);
        }

        // Available Resources Stock Badge
        if (loc.explored && !loc.gatherBlocked) {
          const stock = RESOURCES.filter((r) => (loc.available[r] ?? 0) > 0)
            .map((r) => `${RESOURCE_GLYPH[r]}${loc.available[r]}`)
            .join(' ');

          if (stock) {
            const stockStyle = new TextStyle({
              fontSize: 10,
              fill: 0x93a6bd,
              fontFamily: 'Segoe UI, sans-serif',
            });
            const stockText = new Text({ text: stock, style: stockStyle });
            stockText.anchor.set(0.5, 0);
            stockText.y = NODE_R + 22;
            nodeContainer.addChild(stockText);
          }
        }

        // Monster Token Badge
        if (loc.monsters > 0) {
          const monsterGfx = new Graphics();
          monsterGfx.circle(22, -22, 12).fill({ color: 0xb5476b }).stroke({ width: 2, color: 0xffdb9c });
          nodeContainer.addChild(monsterGfx);

          const mText = new Text({
            text: `👾${loc.monsters > 1 ? loc.monsters : ''}`,
            style: new TextStyle({ fontSize: 11 }),
          });
          mText.anchor.set(0.5);
          mText.x = 22;
          mText.y = -22;
          nodeContainer.addChild(mText);
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

      // 4. Player Pawns Layer
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
          const color = PAWN_COLORS[pIndex % PAWN_COLORS.length];
          const isActive = pid === activePlayer;

          const pawnContainer = new Container();
          const offsetX = (idx - (pids.length - 1) / 2) * 18;
          pawnContainer.x = pLoc.x + offsetX;
          pawnContainer.y = pLoc.y - NODE_R - 8;
          world.addChild(pawnContainer);

          const pawnGfx = new Graphics();
          pawnGfx
            .circle(0, 0, isActive ? 9 : 7)
            .fill({ color })
            .stroke({ width: isActive ? 2.5 : 1.5, color: 0xffffff });
          pawnContainer.addChild(pawnGfx);

          if (isActive) {
            const activeHalo = new Graphics();
            activeHalo.circle(0, 0, 13).stroke({ width: 1.5, color: 0xffc76b, alpha: 0.85 });
            pawnContainer.addChild(activeHalo);
          }
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
          🗺️ Peta Pulau (Canvas WebGL)
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
          >
            100%
          </button>
        </div>
      </div>

      {/* PixiJS Mount Target */}
      <div
        ref={containerRef}
        style={{
          width: '100%',
          height: VIEW_H,
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          background: 'radial-gradient(ellipse at center, rgba(26, 42, 63, 0.45) 0%, rgba(7, 13, 22, 0.8) 100%)',
          borderRadius: 'var(--radius)',
          overflow: 'hidden',
        }}
      />
    </section>
  );
}
