import { Container, Graphics } from 'pixi.js';
import { createPixelLabel } from './PixelFont';

function drawPixelRect(g: Graphics, x: number, y: number, w: number, h: number, color: number) {
  g.rect(x * 3, y * 3, w * 3, h * 3).fill(color);
}

export function createMonsterStandee(count: number): Container {
  const container = new Container();
  const g = new Graphics();
  container.addChild(g);

  const offsetX = -4;
  const offsetY = -8;

  // Shadow
  g.ellipse(0, 3, 10, 5).fill(0x181425);

  // Body
  drawPixelRect(g, offsetX + 1, offsetY + 4, 6, 4, 0x3b1443);
  drawPixelRect(g, offsetX + 2, offsetY + 2, 4, 2, 0x3b1443);
  drawPixelRect(g, offsetX, offsetY + 5, 8, 3, 0xb13e53);

  // Horns
  drawPixelRect(g, offsetX + 1, offsetY, 1, 2, 0x3b1443);
  drawPixelRect(g, offsetX + 6, offsetY, 1, 2, 0x3b1443);

  // Eyes (glowing red)
  drawPixelRect(g, offsetX + 1, offsetY + 3, 2, 2, 0xff1e56);
  drawPixelRect(g, offsetX + 5, offsetY + 3, 2, 2, 0xff1e56);
  // White center
  drawPixelRect(g, offsetX + 1, offsetY + 3, 1, 1, 0xf4f4f4);
  drawPixelRect(g, offsetX + 5, offsetY + 3, 1, 1, 0xf4f4f4);

  if (count > 1) {
    const badge = new Graphics();
    badge.circle(12, -12, 10).fill(0x181425);
    badge.circle(12, -12, 8).fill(0x3a4466);
    container.addChild(badge);

    const text = createPixelLabel(count.toString(), 0xf4f4f4);
    text.position.set(12, -18);
    container.addChild(text);
  }

  // Animation metadata
  (container as any).animTimer = Math.random() * 100;
  (container as any).baseY = 0;

  return container;
}

export function updateMonsterIdle(monster: Container, time: number): void {
  const t = (monster as any).animTimer + time * 0.005;
  (monster as any).animTimer = t;
  // Pixel hop idle
  monster.y = (monster as any).baseY + (Math.sin(t) > 0.5 ? -3 : 0);
}
