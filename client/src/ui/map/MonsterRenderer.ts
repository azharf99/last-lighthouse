import { Container, Graphics, Text, TextStyle } from 'pixi.js';

/**
 * Creates an illustrated Abyssal Dread Beast monster standee (GDD §15 / §16).
 * Replaces the simple emoji with a fearsome shadowy horned creature with glowing crimson eyes.
 */
export function createMonsterStandee(count: number = 1): Container {
  const container = new Container();

  // 1. Dread Tentacle Base / Shadow Pool
  const shadowBase = new Graphics();
  shadowBase
    .ellipse(0, 4, 18, 7)
    .fill({ color: 0x240712, alpha: 0.85 })
    .stroke({ width: 1.5, color: 0x800e13 });
  container.addChild(shadowBase);

  // 2. Abyssal Monster Body
  const monsterBody = new Graphics();

  // Shadowy body mantle
  monsterBody
    .poly([
      { x: -14, y: 2 },
      { x: -16, y: -16 },
      { x: -8, y: -26 },
      { x: 0, y: -30 },
      { x: 8, y: -26 },
      { x: 16, y: -16 },
      { x: 14, y: 2 },
    ])
    .fill({ color: 0x1a060d })
    .stroke({ width: 1.5, color: 0x6d1f38 });

  // Jagged Obsidian Horns
  monsterBody
    .poly([
      { x: -10, y: -24 },
      { x: -18, y: -38 },
      { x: -6, y: -28 },
    ])
    .fill({ color: 0x3a0818 })
    .stroke({ width: 1, color: 0xb5476b });

  monsterBody
    .poly([
      { x: 10, y: -24 },
      { x: 18, y: -38 },
      { x: 6, y: -28 },
    ])
    .fill({ color: 0x3a0818 })
    .stroke({ width: 1, color: 0xb5476b });

  // Sharp Claws
  monsterBody.poly([{ x: -15, y: -4 }, { x: -22, y: 0 }, { x: -13, y: 2 }]).fill({ color: 0x800e13 });
  monsterBody.poly([{ x: 15, y: -4 }, { x: 22, y: 0 }, { x: 13, y: 2 }]).fill({ color: 0x800e13 });

  // Glowing Crimson Eyes
  monsterBody.circle(-4.5, -16, 2.5).fill({ color: 0xff1e56 });
  monsterBody.circle(4.5, -16, 2.5).fill({ color: 0xff1e56 });
  monsterBody.circle(-4.5, -16, 1).fill({ color: 0xffffff });
  monsterBody.circle(4.5, -16, 1).fill({ color: 0xffffff });

  container.addChild(monsterBody);

  // 3. Threat Count Badge (if > 1 monster present)
  if (count > 1) {
    const badge = new Graphics();
    badge.circle(12, -26, 8).fill({ color: 0xb5476b }).stroke({ width: 1, color: 0xffdb9c });
    container.addChild(badge);

    const countText = new Text({
      text: String(count),
      style: new TextStyle({
        fontSize: 10,
        fontWeight: 'bold',
        fill: 0xffffff,
        fontFamily: 'Segoe UI, sans-serif',
      }),
    });
    countText.anchor.set(0.5);
    countText.x = 12;
    countText.y = -26;
    container.addChild(countText);
  }

  return container;
}

/**
 * Updates idle monster animation (breathing pulse and subtle aura oscillation).
 */
export function updateMonsterIdle(monster: Container, time: number) {
  const scaleMod = 1 + Math.sin(time * 3) * 0.04;
  monster.scale.set(scaleMod, 2 - scaleMod);
}
