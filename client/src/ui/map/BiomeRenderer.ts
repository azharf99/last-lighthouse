import { Container, Graphics, Text, TextStyle } from 'pixi.js';

export const BIOME_SIZE = 64;
export const HALF_BIOME = 32;

function drawPixelRect(g: Graphics, x: number, y: number, w: number, h: number, color: number) {
  g.rect(x * 3, y * 3, w * 3, h * 3).fill(color);
}

function drawPixel(g: Graphics, x: number, y: number, color: number) {
  g.rect(x * 3, y * 3, 3, 3).fill(color);
}

export function createBiomeTile(
  type: string,
  explored: boolean,
  isReachable: boolean,
  isExplorable: boolean,
  repairedLighthouseCount: number
): Container {
  const container = new Container();
  const g = new Graphics();
  container.addChild(g);

  // Colors
  const bgColor = explored ? 0x3a4466 : 0x181425;
  
  // Base tile
  g.rect(-HALF_BIOME, -HALF_BIOME, BIOME_SIZE, BIOME_SIZE).fill(bgColor);
  
  // Borders
  if (isExplorable) {
    g.rect(-HALF_BIOME, -HALF_BIOME, BIOME_SIZE, BIOME_SIZE).stroke({ width: 2, color: 0xffc76b });
  } else if (isReachable) {
    g.rect(-HALF_BIOME, -HALF_BIOME, BIOME_SIZE, BIOME_SIZE).stroke({ width: 2, color: 0x7ad7e8 });
  } else {
    g.rect(-HALF_BIOME, -HALF_BIOME, BIOME_SIZE, BIOME_SIZE).stroke({ width: 1, color: 0x3a4466 });
  }

  // Draw pixel art offset to center
  const offsetX = -10;
  const offsetY = -10;

  if (!explored) {
    // Unexplored pattern
    for (let i = 0; i < 5; i++) {
      drawPixel(g, offsetX + Math.random() * 20, offsetY + Math.random() * 20, 0x5a6988);
    }
    const qText = new Text({ text: '?', style: new TextStyle({ fontFamily: 'Press Start 2P, monospace', fontSize: 16, fill: 0x5a6988 }) });
    qText.anchor.set(0.5);
    container.addChild(qText);
    return container;
  }

  switch (type) {
    case 'lighthouse':
      // Stone tower
      drawPixelRect(g, offsetX + 7, offsetY + 15, 6, 5, 0x5a6988);
      drawPixelRect(g, offsetX + 8, offsetY + 10, 4, 5, 0x5a6988);
      drawPixelRect(g, offsetX + 9, offsetY + 6, 2, 4, 0x5a6988);
      // Beacon
      drawPixelRect(g, offsetX + 8, offsetY + 4, 4, 2, 0xffc76b);
      // Light rays
      if (repairedLighthouseCount > 0) {
        drawPixel(g, offsetX + 4, offsetY + 4, 0xf7e26b);
        drawPixel(g, offsetX + 14, offsetY + 4, 0xf7e26b);
        drawPixel(g, offsetX + 2, offsetY + 3, 0xf7e26b);
        drawPixel(g, offsetX + 16, offsetY + 3, 0xf7e26b);
      }
      break;
    case 'forest':
      // Trees
      drawPixelRect(g, offsetX + 4, offsetY + 16, 2, 3, 0x73402e);
      drawPixelRect(g, offsetX + 14, offsetY + 16, 2, 3, 0x73402e);
      drawPixelRect(g, offsetX + 9, offsetY + 12, 2, 3, 0x73402e);
      
      // Leaves
      drawPixelRect(g, offsetX + 3, offsetY + 13, 4, 3, 0x38b764);
      drawPixelRect(g, offsetX + 4, offsetY + 10, 2, 3, 0x38b764);
      
      drawPixelRect(g, offsetX + 13, offsetY + 13, 4, 3, 0x38b764);
      drawPixelRect(g, offsetX + 14, offsetY + 10, 2, 3, 0x38b764);
      
      drawPixelRect(g, offsetX + 8, offsetY + 9, 4, 3, 0x257179);
      drawPixelRect(g, offsetX + 9, offsetY + 6, 2, 3, 0x257179);
      break;
    case 'harbor':
      // Water
      drawPixelRect(g, offsetX + 2, offsetY + 14, 16, 6, 0x3978a8);
      drawPixelRect(g, offsetX + 4, offsetY + 16, 12, 2, 0x394778);
      // Pier
      drawPixelRect(g, offsetX + 2, offsetY + 10, 8, 4, 0x73402e);
      // Boat
      drawPixelRect(g, offsetX + 12, offsetY + 13, 6, 2, 0xb86f50);
      drawPixelRect(g, offsetX + 13, offsetY + 11, 4, 2, 0xb86f50);
      break;
    case 'village':
      // House 1
      drawPixelRect(g, offsetX + 3, offsetY + 12, 6, 6, 0xb86f50);
      drawPixelRect(g, offsetX + 2, offsetY + 9, 8, 3, 0xb13e53); // Roof
      drawPixelRect(g, offsetX + 5, offsetY + 15, 2, 3, 0x181425); // Door
      
      // House 2
      drawPixelRect(g, offsetX + 11, offsetY + 10, 6, 6, 0xb86f50);
      drawPixelRect(g, offsetX + 10, offsetY + 7, 8, 3, 0x73402e); // Roof
      drawPixelRect(g, offsetX + 13, offsetY + 13, 2, 3, 0x181425); // Door
      break;
    case 'cave':
      drawPixelRect(g, offsetX + 4, offsetY + 8, 12, 10, 0x3a4466);
      drawPixelRect(g, offsetX + 6, offsetY + 10, 8, 8, 0x181425);
      drawPixel(g, offsetX + 8, offsetY + 12, 0x5a6988);
      drawPixel(g, offsetX + 12, offsetY + 16, 0x5a6988);
      break;
    case 'crystal_cavern':
      drawPixelRect(g, offsetX + 4, offsetY + 8, 12, 10, 0x181425);
      drawPixelRect(g, offsetX + 6, offsetY + 14, 2, 4, 0x7ad7e8);
      drawPixelRect(g, offsetX + 10, offsetY + 12, 3, 6, 0x7ad7e8);
      drawPixelRect(g, offsetX + 14, offsetY + 15, 2, 3, 0x7ad7e8);
      break;
    case 'ruins':
      drawPixelRect(g, offsetX + 4, offsetY + 14, 4, 4, 0x5a6988);
      drawPixelRect(g, offsetX + 12, offsetY + 10, 4, 8, 0x5a6988);
      drawPixelRect(g, offsetX + 8, offsetY + 16, 4, 2, 0x5a6988);
      drawPixel(g, offsetX + 10, offsetY + 14, 0x3a4466);
      drawPixel(g, offsetX + 5, offsetY + 13, 0x3a4466);
      break;
    case 'mountain':
      // Mountain base
      drawPixelRect(g, offsetX + 2, offsetY + 12, 16, 8, 0x3a4466);
      drawPixelRect(g, offsetX + 4, offsetY + 8, 12, 4, 0x3a4466);
      drawPixelRect(g, offsetX + 6, offsetY + 4, 8, 4, 0x3a4466);
      // Snow caps
      drawPixelRect(g, offsetX + 6, offsetY + 4, 8, 2, 0xf4f4f4);
      drawPixelRect(g, offsetX + 8, offsetY + 6, 4, 2, 0xf4f4f4);
      break;
    case 'temple':
      drawPixelRect(g, offsetX + 4, offsetY + 16, 12, 2, 0x73402e);
      drawPixelRect(g, offsetX + 6, offsetY + 12, 8, 4, 0x73402e);
      drawPixelRect(g, offsetX + 8, offsetY + 8, 4, 4, 0x73402e);
      drawPixelRect(g, offsetX + 8, offsetY + 6, 4, 2, 0xf7e26b); // Gold top
      break;
    default:
      // Fallback
      drawPixelRect(g, offsetX + 8, offsetY + 8, 4, 4, 0x5a6988);
      break;
  }

  return container;
}
