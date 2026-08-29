import { Container, Graphics, Text, TextStyle, Ticker } from 'pixi.js';
import { RESOURCE_GLYPH } from '../labels';

interface ActiveTween {
  update(deltaMs: number): boolean; // returns true when finished
}

const activeTweens: Set<ActiveTween> = new Set();
let globalTickerStarted = false;

function ensureTicker() {
  if (!globalTickerStarted) {
    globalTickerStarted = true;
    Ticker.shared.add((ticker) => {
      const deltaMs = ticker.deltaMS;
      for (const tween of activeTweens) {
        if (tween.update(deltaMs)) {
          activeTweens.delete(tween);
        }
      }
    });
  }
}

export const AnimationEngine = {
  /**
   * Smoothly animates a character standee walking from startPos to endPos
   * with a natural hopping arc and footstep sway.
   */
  animateWalk(
    standee: Container,
    from: { x: number; y: number },
    to: { x: number; y: number },
    durationMs: number = 650,
    onComplete?: () => void,
  ) {
    ensureTicker();
    let elapsed = 0;

    const tween: ActiveTween = {
      update(deltaMs: number) {
        elapsed += deltaMs;
        const progress = Math.min(1, elapsed / durationMs);

        // Smooth cubic ease-in-out
        const t =
          progress < 0.5
            ? 4 * progress * progress * progress
            : 1 - Math.pow(-2 * progress + 2, 3) / 2;

        // Base linear transit
        const curX = from.x + (to.x - from.x) * t;
        const curY = from.y + (to.y - from.y) * t;

        // 3 Hopping arc bounces during the journey
        const hopOffset = -Math.abs(Math.sin(progress * Math.PI * 3)) * 14;

        standee.x = curX;
        standee.y = curY + hopOffset;

        // Footstep sway / tilt
        standee.rotation = Math.sin(progress * Math.PI * 6) * 0.12;

        if (progress >= 1) {
          standee.x = to.x;
          standee.y = to.y;
          standee.rotation = 0;
          if (onComplete) onComplete();
          return true;
        }
        return false;
      },
    };

    activeTweens.add(tween);
  },

  /**
   * Spawns a shimmering floating resource harvest burst when items are gathered.
   */
  animateGatherBurst(
    world: Container,
    origin: { x: number; y: number },
    resource: string,
    amount: number = 1,
  ) {
    ensureTicker();
    const burstContainer = new Container();
    burstContainer.x = origin.x;
    burstContainer.y = origin.y;
    world.addChild(burstContainer);

    // Floating Glyph & Amount Text
    const glyph = RESOURCE_GLYPH[resource] || '📦';
    const text = new Text({
      text: `+${amount} ${glyph}`,
      style: new TextStyle({
        fontSize: 16,
        fontWeight: 'bold',
        fill: 0xffdb9c,
        fontFamily: 'Segoe UI, sans-serif',
        dropShadow: {
          alpha: 0.9,
          blur: 4,
          color: 0x070d16,
          distance: 1,
        },
      }),
    });
    text.anchor.set(0.5);
    burstContainer.addChild(text);

    // Sparkle sparkles around text
    const sparkles = new Graphics();
    sparkles.circle(-18, -4, 2.5).fill({ color: 0xffffff });
    sparkles.circle(18, -6, 2).fill({ color: 0xffc76b });
    sparkles.circle(0, -18, 3).fill({ color: 0x7ad7e8 });
    burstContainer.addChild(sparkles);

    let elapsed = 0;
    const duration = 900;

    const tween: ActiveTween = {
      update(deltaMs: number) {
        elapsed += deltaMs;
        const progress = Math.min(1, elapsed / duration);

        // Float upward with decelerating velocity
        burstContainer.y = origin.y - Math.sin(progress * (Math.PI / 2)) * 48;
        burstContainer.alpha = 1 - progress * progress; // Fade out near top
        burstContainer.scale.set(1 + progress * 0.2);

        if (progress >= 1) {
          world.removeChild(burstContainer);
          burstContainer.destroy({ children: true });
          return true;
        }
        return false;
      },
    };

    activeTweens.add(tween);
  },

  /**
   * Animates a fierce combat clash (attacker lunges, slash arc FX flashes, monster shakes).
   */
  animateCombatClash(
    world: Container,
    attacker: Container,
    monsterPos: { x: number; y: number },
    onHit?: () => void,
    onComplete?: () => void,
  ) {
    ensureTicker();
    const origX = attacker.x;
    const origY = attacker.y;

    // Slash FX Graphics
    const slashGfx = new Graphics();
    slashGfx.x = monsterPos.x;
    slashGfx.y = monsterPos.y;
    slashGfx.alpha = 0;
    world.addChild(slashGfx);

    let elapsed = 0;
    const duration = 600;
    let hitTriggered = false;

    const tween: ActiveTween = {
      update(deltaMs: number) {
        elapsed += deltaMs;
        const progress = Math.min(1, elapsed / duration);

        // Phase 1: Lunge forward (0 to 0.4)
        if (progress < 0.4) {
          const lungeT = progress / 0.4;
          attacker.x = origX + (monsterPos.x - origX) * 0.45 * lungeT;
          attacker.y = origY + (monsterPos.y - origY) * 0.45 * lungeT;
          attacker.rotation = 0.2;
        }
        // Phase 2: Slash Impact & Screen Flash (0.4 to 0.7)
        else if (progress < 0.7) {
          if (!hitTriggered) {
            hitTriggered = true;
            if (onHit) onHit();
          }

          const slashT = (progress - 0.4) / 0.3;
          slashGfx.clear();
          slashGfx.alpha = 1 - slashT;

          // Arc slash line across monster
          slashGfx
            .moveTo(-28, -28)
            .lineTo(28, 20)
            .stroke({ width: 4.5, color: 0xffffff });
          slashGfx
            .moveTo(-20, -32)
            .lineTo(24, 14)
            .stroke({ width: 2, color: 0xffc76b });
        }
        // Phase 3: Return to original position (0.7 to 1.0)
        else {
          const returnT = (progress - 0.7) / 0.3;
          attacker.x = origX + (monsterPos.x - origX) * 0.45 * (1 - returnT);
          attacker.y = origY + (monsterPos.y - origY) * 0.45 * (1 - returnT);
          attacker.rotation = 0.2 * (1 - returnT);
        }

        if (progress >= 1) {
          attacker.x = origX;
          attacker.y = origY;
          attacker.rotation = 0;
          world.removeChild(slashGfx);
          slashGfx.destroy();
          if (onComplete) onComplete();
          return true;
        }
        return false;
      },
    };

    activeTweens.add(tween);
  },
};
