import { Container, Graphics, Text, TextStyle } from 'pixi.js';
import type { CharacterID } from '../../types';

export const CHARACTER_THEMES: Record<
  CharacterID | string,
  {
    name: string;
    primaryColor: number;
    secondaryColor: number;
    accentColor: number;
    icon: string;
  }
> = {
  navigator: {
    name: 'Navigator',
    primaryColor: 0x1d3557, // Navy coat
    secondaryColor: 0x457b9d, // Ocean blue
    accentColor: 0xffc76b, // Golden brass
    icon: '🧭',
  },
  engineer: {
    name: 'Insinyur',
    primaryColor: 0x6c584c, // Leather apron
    secondaryColor: 0x7ad7e8, // Cyan energy
    accentColor: 0xb07f4a, // Copper bronze
    icon: '⚙️',
  },
  hunter: {
    name: 'Pemburu',
    primaryColor: 0x800e13, // Deep crimson cloak
    secondaryColor: 0x283618, // Forest green
    accentColor: 0xe58fa9, // Rose leather
    icon: '🏹',
  },
  scholar: {
    name: 'Cendekia',
    primaryColor: 0x1b4332, // Sage emerald robe
    secondaryColor: 0x2d6a4f, // Jade sash
    accentColor: 0x8fc07a, // Light rune
    icon: '📜',
  },
};

/**
 * Creates an illustrated tabletop standee miniature container for a humanoid character.
 */
export function createHumanoidStandee(
  characterId: string,
  playerColor: number,
  isActive: boolean,
  isExhausted: boolean,
  playerName: string,
): Container {
  const container = new Container();
  const theme = CHARACTER_THEMES[characterId] || CHARACTER_THEMES.navigator;

  // 1. Pedestal Base (Wooden Mini Standee Base)
  const base = new Graphics();
  base.ellipse(0, 0, 18, 7).fill({ color: 0x182435 }).stroke({ width: 1.5, color: playerColor });
  container.addChild(base);

  // Active Player Glowing Halo Ring
  if (isActive) {
    const halo = new Graphics();
    halo.ellipse(0, 0, 22, 9).stroke({ width: 2, color: 0xffc76b, alpha: 0.95 });
    container.addChild(halo);
  }

  // 2. Humanoid Body Group (Height ~44px from base Y: 0 to top Y: -44)
  const body = new Graphics();

  // Shadow under standee
  body.ellipse(0, -1, 12, 4).fill({ color: 0x070d16, alpha: 0.6 });

  // Legs / Boots
  body.roundRect(-7, -12, 5, 12, 2).fill({ color: 0x111927 });
  body.roundRect(2, -12, 5, 12, 2).fill({ color: 0x111927 });

  // Torso / Tunic
  body
    .roundRect(-9, -28, 18, 18, 4)
    .fill({ color: isExhausted ? 0x3d4a58 : theme.primaryColor })
    .stroke({ width: 1, color: theme.secondaryColor });

  // Belt & Buckle
  body.rect(-9, -15, 18, 3).fill({ color: 0x2b1d0c });
  body.rect(-2, -16, 4, 5).fill({ color: 0xffc76b });

  // Head / Face
  body.circle(0, -34, 6.5).fill({ color: 0xfbd1a2 }); // Skin tone

  // Character-specific Gear, Hats, Cloaks & Props
  if (characterId === 'navigator') {
    // Navigator: Tricorn explorer hat + Brass Compass + Spyglass
    body
      .poly([
        { x: -11, y: -38 },
        { x: 11, y: -38 },
        { x: 0, y: -45 },
      ])
      .fill({ color: 0x0f1c2e })
      .stroke({ width: 1, color: 0xffc76b });

    // Compass in right hand
    body.circle(9, -20, 4).fill({ color: 0xffc76b }).stroke({ width: 1, color: 0xffffff });
    // Spyglass on left hip
    body.rect(-12, -22, 3, 10).fill({ color: 0xcaa363 });
  } else if (characterId === 'engineer') {
    // Engineer: Copper goggles on forehead + Iron wrench
    body.roundRect(-7, -40, 14, 5, 2).fill({ color: 0xb07f4a });
    body.circle(-3, -38, 2.5).fill({ color: 0x7ad7e8 });
    body.circle(3, -38, 2.5).fill({ color: 0x7ad7e8 });

    // Big wrench in hand
    body.rect(8, -26, 3, 14).fill({ color: 0x8f9ba8 });
    body.circle(9.5, -28, 4).fill({ color: 0x8f9ba8 });
    body.circle(9.5, -28, 2).fill({ color: theme.primaryColor });
  } else if (characterId === 'hunter') {
    // Hunter: Hooded ranger cloak + Bow across back + Dagger
    body.circle(0, -35, 8.5).fill({ color: theme.primaryColor }); // Hood
    body.circle(0, -34, 5.5).fill({ color: 0xfbd1a2 }); // Face inside hood

    // Bow curve on back
    body
      .arc(-7, -26, 12, -Math.PI / 2, Math.PI / 2)
      .stroke({ width: 2, color: 0x6f4e37 });
    // Hunting Dagger in hand
    body.rect(8, -18, 2.5, 9).fill({ color: 0xd8e2dc });
    body.rect(7, -19, 4.5, 2).fill({ color: 0x4a5568 });
  } else if (characterId === 'scholar') {
    // Scholar: Robe cowl + Ancient parchment scroll + glowing flask
    body.poly([
      { x: -8, y: -41 },
      { x: 8, y: -41 },
      { x: 0, y: -46 },
    ]).fill({ color: 0x1b4332 });

    // Scroll in right hand
    body.roundRect(7, -23, 6, 11, 2).fill({ color: 0xf4f1de }).stroke({ width: 1, color: 0xb07f4a });
    // Glowing Crystal vial in left hand
    body.circle(-9, -20, 3.5).fill({ color: 0x7ad7e8, alpha: 0.9 });
  }

  container.addChild(body);

  // 3. Floating Player Name Tag (under miniature base)
  const nameStyle = new TextStyle({
    fontSize: 9.5,
    fontWeight: isActive ? 'bold' : 'normal',
    fill: isActive ? 0xffc76b : 0xe8eef6,
    fontFamily: 'Segoe UI, sans-serif',
  });
  const nameText = new Text({ text: playerName, style: nameStyle });
  nameText.anchor.set(0.5, 0);
  nameText.y = 7;
  container.addChild(nameText);

  // Exhausted indicator icon
  if (isExhausted) {
    const exText = new Text({
      text: '💤',
      style: new TextStyle({ fontSize: 10 }),
    });
    exText.x = 8;
    exText.y = -44;
    container.addChild(exText);
  }

  return container;
}
