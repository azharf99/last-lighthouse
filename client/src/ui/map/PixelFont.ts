import { Text, TextStyle } from 'pixi.js';

export const PIXEL_FONT = 'Press Start 2P, monospace';

export function createPixelText(
  text: string,
  fontSize: number = 8,
  color: number = 0xf4f4f4,
  bold: boolean = false,
): Text {
  // Create text with pixel font, no anti-aliasing feel
  // Use dropShadow for retro text shadow
  const style = new TextStyle({
    fontSize,
    fontFamily: PIXEL_FONT,
    fill: color,
    fontWeight: bold ? 'bold' : 'normal',
    dropShadow: {
      alpha: 0.9,
      blur: 0,
      color: 0x181425,
      distance: 1,
    },
  });
  return new Text({ text, style });
}

export function createPixelLabel(
  text: string,
  color: number = 0xf4f4f4,
): Text {
  const t = createPixelText(text, 7, color);
  t.anchor.set(0.5, 0);
  return t;
}
