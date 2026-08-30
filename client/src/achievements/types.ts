// Definisi tipe data sistem Pencapaian (Achievements) dan Statistik Karir Pemain

export type AchievementTier = 'bronze' | 'silver' | 'gold' | 'platinum';

export type AchievementCategory = 'victory' | 'combat' | 'exploration' | 'team' | 'mastery';

export interface Achievement {
  id: string;
  titleKey: string;
  descKey: string;
  icon: string;
  tier: AchievementTier;
  category: AchievementCategory;
  secret?: boolean;
}

export interface UnlockedAchievement {
  id: string;
  unlockedAt: string; // ISO date string
}

export interface PlayerCareerStats {
  totalMatches: number;
  wins: number;
  losses: number;
  highestVP: number;
  totalMonstersSlain: number;
  totalRepairs: number;
  charactersWon: string[];
  unlockedAchievements: Record<string, string>; // achievementId -> unlockedAt
}
