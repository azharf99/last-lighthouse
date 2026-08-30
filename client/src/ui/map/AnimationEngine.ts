import { Container, Graphics, Ticker } from 'pixi.js';

interface ActiveTween {
  update(deltaMs: number): boolean; // returns true when finished
}

const activeTweens: Set<ActiveTween> = new Set();
let globalTickerStarted = false;

function ensureTicker() {
  if (!globalTickerStarted) {
    globalTickerStarted = true;
    Ticker.shared.add((ticker) => {
      const dt = ticker.deltaMS;
      for (const tween of activeTweens) {
        if (tween.update(dt)) {
          activeTweens.delete(tween);
        }
      }
    });
  }
}

export const AnimationEngine = {
  /**
   * Retro pixel-hop walk: 3 discrete hops from A to B.
   */
  animateWalk(
    standee: Container,
    from: { x: number; y: number },
    to: { x: number; y: number },
    durationMs: number = 500,
    onComplete?: () => void,
  ) {
    ensureTicker();
    let elapsed = 0;

    const tween: ActiveTween = {
      update(deltaMs: number) {
        elapsed += deltaMs;
        const progress = Math.min(1, elapsed / durationMs);

        // 3 discrete hops — snap between positions
        const steps = 3;
        const step = Math.min(steps, Math.floor(progress * (steps + 1)));
        const stepT = step / steps;

        standee.x = from.x + (to.x - from.x) * stepT;
        standee.y = from.y + (to.y - from.y) * stepT;

        // Pixel jump arc per hop
        const hopFrac = (progress * steps) % 1;
        if (progress < 1 && hopFrac < 0.5) {
          standee.y -= 10; // bounce up
        }

        if (progress >= 1) {
          standee.x = to.x;
          standee.y = to.y;
          if (onComplete) onComplete();
          return true;
        }
        return false;
      },
    };

    activeTweens.add(tween);
  },

  /**
   * Pixelated sparkle burst when gathering resources.
   */
  animateGatherBurst(
    world: Container,
    origin: { x: number; y: number },
    resource: string,
    _amount: number = 1,
  ) {
    ensureTicker();
    const burstContainer = new Container();
    burstContainer.x = origin.x;
    burstContainer.y = origin.y;
    world.addChild(burstContainer);

    // Spawn pixel sparkle particles
    const colors: Record<string, number> = {
      wood: 0xb86f50,
      metal: 0x8b9bb4,
      crystal: 0x7ad7e8,
      food: 0x38b764,
    };
    const color = colors[resource] ?? 0xf7e26b;

    for (let i = 0; i < 6; i++) {
      const spark = new Graphics();
      spark.rect(0, 0, 4, 4).fill({ color });
      spark.x = (Math.random() - 0.5) * 24;
      spark.y = (Math.random() - 0.5) * 16;
      burstContainer.addChild(spark);
    }

    let elapsed = 0;
    const duration = 600;

    const tween: ActiveTween = {
      update(deltaMs: number) {
        elapsed += deltaMs;
        const progress = Math.min(1, elapsed / duration);

        // Float upward in discrete pixel steps
        burstContainer.y = origin.y - Math.floor(progress * 40);
        burstContainer.alpha = 1 - progress;

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
   * Retro combat clash: screen flash + pixel X slash.
   */
  animateCombatClash(
    world: Container,
    _attackerPos: { x: number; y: number },
    monsterPos: { x: number; y: number },
    onHit?: () => void,
    onComplete?: () => void,
  ) {
    ensureTicker();

    // White flash overlay
    const flash = new Graphics();
    flash.rect(-1000, -1000, 3000, 3000).fill({ color: 0xf4f4f4 });
    flash.alpha = 0.6;
    world.addChild(flash);

    // Pixel slash X
    const slash = new Graphics();
    slash.moveTo(-12, -12).lineTo(12, 12).stroke({ width: 4, color: 0xff1e56 });
    slash.moveTo(12, -12).lineTo(-12, 12).stroke({ width: 4, color: 0xff1e56 });
    slash.x = monsterPos.x;
    slash.y = monsterPos.y;
    world.addChild(slash);

    let elapsed = 0;
    const duration = 350;
    let hitTriggered = false;

    const tween: ActiveTween = {
      update(deltaMs: number) {
        elapsed += deltaMs;
        const progress = Math.min(1, elapsed / duration);

        // Trigger hit at midpoint
        if (!hitTriggered && progress >= 0.3) {
          hitTriggered = true;
          if (onHit) onHit();
        }

        flash.alpha = 0.6 * (1 - progress);
        slash.alpha = 1 - progress;

        if (progress >= 1) {
          world.removeChild(flash);
          world.removeChild(slash);
          flash.destroy();
          slash.destroy();
          if (onComplete) onComplete();
          return true;
        }
        return false;
      },
    };

    activeTweens.add(tween);
  },
};
