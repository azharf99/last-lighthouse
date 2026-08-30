import { useState, useEffect } from 'react';
import { achievementManager } from '../achievements/achievements';
import type { Achievement } from '../achievements/types';
import { sfx } from '../audio/sfx';
import { i18n } from '../i18n';

export function AchievementToast() {
  const [current, setCurrent] = useState<Achievement | null>(null);

  useEffect(() => {
    const unsubscribe = achievementManager.subscribe((ach) => {
      sfx.playAchievement();
      setCurrent(ach);

      const timer = setTimeout(() => {
        setCurrent((prev) => (prev?.id === ach.id ? null : prev));
      }, 4500);

      return () => clearTimeout(timer);
    });

    return unsubscribe;
  }, []);

  if (!current) return null;

  const tierColors: Record<string, string> = {
    bronze: '#cd7f32',
    silver: '#c0c0c0',
    gold: '#ffd700',
    platinum: '#00ffff',
  };

  const color = tierColors[current.tier] || '#ffd700';

  return (
    <div
      className="achievement-toast"
      onClick={() => setCurrent(null)}
      role="status"
      aria-live="polite"
      style={{ borderColor: color }}
    >
      <div className="achievement-toast__icon">{current.icon}</div>
      <div className="achievement-toast__content">
        <div className="achievement-toast__badge" style={{ color }}>
          🏆 {i18n.t('ach.unlocked_badge')} · {current.tier.toUpperCase()}
        </div>
        <div className="achievement-toast__title">{i18n.t(current.titleKey)}</div>
        <div className="achievement-toast__desc">{i18n.t(current.descKey)}</div>
      </div>
    </div>
  );
}
