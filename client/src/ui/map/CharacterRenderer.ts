import { Container, Graphics } from 'pixi.js';
import { createPixelLabel } from './PixelFont';

export const CHARACTER_THEMES: Record<string, { name: string; primaryColor: number; secondaryColor: number; accentColor: number; icon: string }> = {
  navigator: { name: 'Navigator', primaryColor: 0x3978a8, secondaryColor: 0x394778, accentColor: 0xf7e26b, icon: '🧭' },
  engineer: { name: 'Engineer', primaryColor: 0xb86f50, secondaryColor: 0x73402e, accentColor: 0xffc76b, icon: '🔧' },
  hunter: { name: 'Hunter', primaryColor: 0xb13e53, secondaryColor: 0x3b1443, accentColor: 0x38b764, icon: '🏹' },
  scholar: { name: 'Scholar', primaryColor: 0x38b764, secondaryColor: 0x257179, accentColor: 0xf4f4f4, icon: '📜' },
};

function drawPixelRect(g: Graphics, x: number, y: number, w: number, h: number, color: number) {
  g.rect(x * 3, y * 3, w * 3, h * 3).fill(color);
}

export function createHumanoidStandee(
  characterId: string,
  playerColor: number,
  isActive: boolean,
  isExhausted: boolean,
  playerName: string
): Container {
  const container = new Container();
  const theme = CHARACTER_THEMES[characterId] || CHARACTER_THEMES.navigator;
  
  const g = new Graphics();
  container.addChild(g);

  // Gray out if exhausted
  const primary = isExhausted ? 0x5a6988 : theme.primaryColor;
  const secondary = isExhausted ? 0x3a4466 : theme.secondaryColor;
  const accent = isExhausted ? 0xf4f4f4 : theme.accentColor;
  const skin = isExhausted ? 0x5a6988 : 0xf2c8a0;

  const offsetX = -3;
  const offsetY = -12;

  // Active glowing halo
  if (isActive) {
    g.ellipse(0, 5, 12, 6).stroke({ width: 2, color: 0xffc76b });
    // Bouncing arrow
    const t = Date.now() / 200;
    const bounce = Math.sin(t) * 2;
    drawPixelRect(g, offsetX + 1, offsetY - 6 + bounce, 4, 1, 0xffc76b);
    drawPixelRect(g, offsetX + 2, offsetY - 5 + bounce, 2, 1, 0xffc76b);
  }

  // Shadow
  g.ellipse(0, 3, 8, 4).fill(0x181425);

  // Feet
  drawPixelRect(g, offsetX + 1, offsetY + 11, 2, 1, 0x181425);
  drawPixelRect(g, offsetX + 3, offsetY + 11, 2, 1, 0x181425);

  // Body
  drawPixelRect(g, offsetX + 1, offsetY + 5, 4, 6, primary);
  
  // Details
  if (characterId === 'navigator') {
    drawPixelRect(g, offsetX + 2, offsetY + 6, 2, 2, accent);
    drawPixelRect(g, offsetX, offsetY + 1, 6, 2, secondary); // Hat
    drawPixelRect(g, offsetX + 1, offsetY, 4, 1, secondary);
  } else if (characterId === 'engineer') {
    drawPixelRect(g, offsetX + 1, offsetY + 5, 4, 3, secondary); // Apron
    drawPixelRect(g, offsetX + 2, offsetY + 2, 2, 1, accent); // Goggles
  } else if (characterId === 'hunter') {
    drawPixelRect(g, offsetX, offsetY + 1, 6, 4, primary); // Hood
    drawPixelRect(g, offsetX, offsetY + 5, 1, 6, secondary); // Bow
  } else if (characterId === 'scholar') {
    drawPixelRect(g, offsetX + 1, offsetY + 5, 4, 7, primary); // Robe
    drawPixelRect(g, offsetX + 4, offsetY + 8, 1, 2, accent); // Scroll
  }

  // Head
  drawPixelRect(g, offsetX + 1, offsetY + 2, 4, 3, skin);
  // Eyes
  drawPixelRect(g, offsetX + 1, offsetY + 3, 1, 1, 0x181425);
  drawPixelRect(g, offsetX + 3, offsetY + 3, 1, 1, 0x181425);

  // Name Label
  const nameLabel = createPixelLabel(playerName, playerColor);
  nameLabel.y = 12;
  container.addChild(nameLabel);

  if (isExhausted) {
    const zzz = createPixelLabel('zzZ', 0xf4f4f4);
    zzz.y = -40;
    container.addChild(zzz);
  }

  return container;
}
