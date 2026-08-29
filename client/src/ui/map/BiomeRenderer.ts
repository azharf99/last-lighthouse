import { Container, Graphics, Text, TextStyle } from 'pixi.js';

export const BIOME_SIZE = 88;
export const HALF_BIOME = BIOME_SIZE / 2;

/**
 * Renders a rich thematic biome terrain plaza tile for a location node.
 */
export function createBiomeTile(
  type: string,
  explored: boolean,
  isReachable: boolean,
  isExplorable: boolean,
  repairedLighthouseCount: number = 0,
): Container {
  const container = new Container();

  // 1. Terrain Plaza Base Card (88x88 px rounded rectangle)
  const base = new Graphics();

  let bgColor = 0x131f30;
  let borderColor = 0x2b3d54;
  let borderWidth = 2;

  if (!explored) {
    bgColor = 0x090f19;
    borderColor = 0x1d2c3f;
  } else if (type === 'lighthouse') {
    bgColor = 0x1a273b;
    borderColor = 0xffc76b;
    borderWidth = 2.5;
  } else if (isReachable) {
    bgColor = 0x182c40;
    borderColor = 0x7ad7e8;
    borderWidth = 2.5;
  }

  if (isExplorable) {
    borderColor = 0xffc76b;
    borderWidth = 2.5;
  }

  // Drop shadow
  base
    .roundRect(-HALF_BIOME + 2, -HALF_BIOME + 4, BIOME_SIZE, BIOME_SIZE, 14)
    .fill({ color: 0x04080e, alpha: 0.7 });

  // Plaza Base
  base
    .roundRect(-HALF_BIOME, -HALF_BIOME, BIOME_SIZE, BIOME_SIZE, 14)
    .fill({ color: bgColor })
    .stroke({ width: borderWidth, color: borderColor });

  // Active / Explorable interactive outer glow
  if (isReachable || isExplorable) {
    base
      .roundRect(-HALF_BIOME - 3, -HALF_BIOME - 3, BIOME_SIZE + 6, BIOME_SIZE + 6, 16)
      .stroke({ width: 1.5, color: isExplorable ? 0xffc76b : 0x7ad7e8, alpha: 0.65 });
  }

  container.addChild(base);

  // 2. Thematic Biome Illustrations & Landmarks
  const art = new Graphics();

  if (!explored) {
    // Fog of war cloud swirls
    art.circle(-12, -8, 14).fill({ color: 0x162233, alpha: 0.6 });
    art.circle(12, -6, 16).fill({ color: 0x1a293e, alpha: 0.65 });
    art.circle(0, 10, 15).fill({ color: 0x131f30, alpha: 0.7 });

    const qText = new Text({
      text: '?',
      style: new TextStyle({
        fontSize: 32,
        fontWeight: 'bold',
        fill: isExplorable ? 0xffc76b : 0x5f7591,
        fontFamily: 'Segoe UI, sans-serif',
      }),
    });
    qText.anchor.set(0.5);
    qText.y = -2;
    container.addChild(art);
    container.addChild(qText);
    return container;
  }

  switch (type) {
    case 'lighthouse': {
      // Beacon tower base & stone stairs
      art.roundRect(-16, 8, 32, 18, 2).fill({ color: 0x3d5470 });
      art.poly([
        { x: -14, y: 8 },
        { x: 14, y: 8 },
        { x: 9, y: -24 },
        { x: -9, y: -24 },
      ]).fill({ color: 0x8fa3bc });

      // Lantern room & Golden Beacon Dome
      art.rect(-8, -32, 16, 8).fill({ color: 0xffdb9c });
      art.arc(0, -32, 8, Math.PI, 0).fill({ color: 0xffc76b });

      // Dynamic light beam rays
      if (repairedLighthouseCount > 0) {
        art.poly([
          { x: 0, y: -30 },
          { x: -70, y: -70 },
          { x: -40, y: -80 },
        ]).fill({ color: 0xffc76b, alpha: 0.15 + repairedLighthouseCount * 0.04 });

        art.poly([
          { x: 0, y: -30 },
          { x: 70, y: -70 },
          { x: 40, y: -80 },
        ]).fill({ color: 0xffc76b, alpha: 0.15 + repairedLighthouseCount * 0.04 });
      }
      break;
    }

    case 'forest': {
      // Deep pine trees with tiered triangular canopies
      // Tree 1 (Left)
      art.rect(-22, 10, 4, 8).fill({ color: 0x4a3728 });
      art.poly([{ x: -20, y: -6 }, { x: -30, y: 10 }, { x: -10, y: 10 }]).fill({ color: 0x1b4332 });
      art.poly([{ x: -20, y: -16 }, { x: -28, y: -4 }, { x: -12, y: -4 }]).fill({ color: 0x2d6a4f });

      // Tree 2 (Center High)
      art.rect(-3, 8, 6, 12).fill({ color: 0x4a3728 });
      art.poly([{ x: 0, y: -14 }, { x: -16, y: 8 }, { x: 16, y: 8 }]).fill({ color: 0x1b4332 });
      art.poly([{ x: 0, y: -26 }, { x: -13, y: -10 }, { x: 13, y: -10 }]).fill({ color: 0x2d6a4f });
      art.poly([{ x: 0, y: -34 }, { x: -9, y: -22 }, { x: 9, y: -22 }]).fill({ color: 0x40916c });

      // Tree 3 (Right)
      art.rect(18, 10, 4, 8).fill({ color: 0x4a3728 });
      art.poly([{ x: 20, y: -6 }, { x: 10, y: 10 }, { x: 30, y: 10 }]).fill({ color: 0x1b4332 });
      art.poly([{ x: 20, y: -16 }, { x: 12, y: -4 }, { x: 28, y: -4 }]).fill({ color: 0x2d6a4f });
      break;
    }

    case 'harbor': {
      // Pier wooden deck over coastal water
      art.roundRect(-36, 6, 72, 22, 4).fill({ color: 0x1d3557, alpha: 0.7 }); // Sea water
      art.rect(-24, -14, 48, 16).fill({ color: 0x6c584c }); // Wood pier
      art.rect(-20, -12, 40, 2).fill({ color: 0x8a705e });
      art.rect(-20, -4, 40, 2).fill({ color: 0x8a705e });

      // Anchor & Cargo Barrel
      art.circle(-12, -22, 5).stroke({ width: 2, color: 0x9fb3c8 }); // Anchor ring
      art.rect(-13, -18, 2, 8).fill({ color: 0x9fb3c8 });
      art.roundRect(10, -22, 10, 12, 2).fill({ color: 0xb07f4a }); // Barrel
      break;
    }

    case 'village': {
      // 2 Thatched cottages & stone path
      // Left cottage
      art.rect(-26, -4, 22, 18).fill({ color: 0x4a5568 });
      art.poly([{ x: -15, y: -18 }, { x: -30, y: -4 }, { x: 0, y: -4 }]).fill({ color: 0xb07f4a });
      art.rect(-18, 4, 6, 10).fill({ color: 0x1a202c }); // Door

      // Right cottage
      art.rect(4, -10, 24, 24).fill({ color: 0x5a6578 });
      art.poly([{ x: 16, y: -26 }, { x: 0, y: -10 }, { x: 32, y: -10 }]).fill({ color: 0xb07f4a });
      art.rect(13, 2, 7, 12).fill({ color: 0x1a202c }); // Door
      art.rect(22, -28, 4, 8).fill({ color: 0x2d3748 }); // Chimney
      break;
    }

    case 'cave': {
      // Dark rock cavern cliff & ore cracks
      art.poly([
        { x: -34, y: 22 },
        { x: -30, y: -18 },
        { x: 0, y: -30 },
        { x: 30, y: -18 },
        { x: 34, y: 22 },
      ]).fill({ color: 0x2b3d54 });

      // Cavern entrance
      art.arc(0, 10, 18, Math.PI, 0).fill({ color: 0x070d16 });

      // Metal ore sparkling glints
      art.circle(-14, -8, 2.5).fill({ color: 0x9fb3c8 });
      art.circle(12, -12, 3).fill({ color: 0x9fb3c8 });
      art.circle(-6, -20, 2).fill({ color: 0x9fb3c8 });
      break;
    }

    case 'crystal_cavern': {
      // Luminous dark crystal cave with glowing prisma
      art.arc(0, 12, 22, Math.PI, 0).fill({ color: 0x07111e });

      // 3 Glowing Crystal Shards
      art.poly([{ x: -14, y: 12 }, { x: -10, y: -16 }, { x: -6, y: 12 }]).fill({ color: 0x7ad7e8 });
      art.poly([{ x: -4, y: 12 }, { x: 2, y: -26 }, { x: 8, y: 12 }]).fill({ color: 0x90e0ef });
      art.poly([{ x: 10, y: 12 }, { x: 15, y: -14 }, { x: 20, y: 12 }]).fill({ color: 0x00b4d8 });
      break;
    }

    case 'ruins': {
      // Ancient cracked marble columns & stone dais
      art.roundRect(-30, 12, 60, 8, 2).fill({ color: 0x3d4a58 });
      // Left pillar (standing)
      art.rect(-22, -22, 8, 34).fill({ color: 0xd8e2dc });
      art.rect(-24, -26, 12, 4).fill({ color: 0xd8e2dc });
      // Right pillar (broken)
      art.rect(14, -8, 8, 20).fill({ color: 0xd8e2dc });
      art.poly([{ x: 14, y: -8 }, { x: 22, y: -4 }, { x: 22, y: 12 }]).fill({ color: 0x9fa8a3 });
      break;
    }

    case 'mountain': {
      // Tiered jagged mountain peaks
      art.poly([{ x: -20, y: 18 }, { x: -10, y: -18 }, { x: 10, y: 18 }]).fill({ color: 0x3d5470 });
      art.poly([{ x: -6, y: 18 }, { x: 12, y: -28 }, { x: 30, y: 18 }]).fill({ color: 0x5a6f87 });
      art.poly([{ x: 12, y: -28 }, { x: 7, y: -18 }, { x: 17, y: -18 }]).fill({ color: 0xe8eef6 }); // Snow peak
      break;
    }

    case 'temple': {
      // Ancient stepped pyramid altar & ceremonial braziers
      art.rect(-28, 8, 56, 8).fill({ color: 0x3d5470 });
      art.rect(-20, 0, 40, 8).fill({ color: 0x4a6280 });
      art.rect(-12, -8, 24, 8).fill({ color: 0x5f7a9e });
      art.rect(-4, -16, 8, 8).fill({ color: 0xffc76b }); // Golden idol
      break;
    }
  }

  container.addChild(art);
  return container;
}
