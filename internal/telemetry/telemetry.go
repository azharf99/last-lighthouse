package telemetry

import (
	"context"
	"sync"
	"time"
)

// MatchReport merangkum data statistik satu sesi permainan tanpa data pribadi pemain (Zero-PII).
type MatchReport struct {
	MatchID            string    `json:"matchId"`
	DurationSec        int       `json:"durationSec"`
	Status             string    `json:"status"` // "won", "lost"
	TotalRounds        int       `json:"totalRounds"`
	FinalDarkness      int       `json:"finalDarkness"`
	PlayerCount        int       `json:"playerCount"`
	Characters         []string  `json:"characters"`
	MonstersDefeated   int       `json:"monstersDefeated"`
	ComponentsRepaired int       `json:"componentsRepaired"`
	CreatedAt          time.Time `json:"createdAt"`
}

// GlobalStats menyediakan ringkasan agregat keseimbangan game untuk dasbor telemetri.
type GlobalStats struct {
	TotalMatches       int            `json:"totalMatches"`
	Wins               int            `json:"wins"`
	Losses             int            `json:"losses"`
	WinRatePercent     float64        `json:"winRatePercent"`
	AvgRounds          float64        `json:"avgRounds"`
	AvgDarkness        float64        `json:"avgDarkness"`
	TotalMonstersSlain int            `json:"totalMonstersSlain"`
	RoleDistribution   map[string]int `json:"roleDistribution"`
	LastUpdated        time.Time      `json:"lastUpdated"`
}

// Collector adalah pengumpul telemetri analitik game (M6).
type Collector struct {
	mu      sync.RWMutex
	reports []MatchReport
}

func NewCollector() *Collector {
	c := &Collector{
		reports: make([]MatchReport, 0),
	}
	// Seed sample baseline metrics for initial telemetry dashboard
	c.seedBaselineData()
	return c
}

func (c *Collector) seedBaselineData() {
	now := time.Now()
	sampleReports := []MatchReport{
		{MatchID: "base_001", DurationSec: 720, Status: "won", TotalRounds: 7, FinalDarkness: 6, PlayerCount: 3, Characters: []string{"navigator", "engineer", "hunter"}, MonstersDefeated: 2, ComponentsRepaired: 5, CreatedAt: now.Add(-2 * time.Hour)},
		{MatchID: "base_002", DurationSec: 540, Status: "lost", TotalRounds: 5, FinalDarkness: 8, PlayerCount: 2, Characters: []string{"engineer", "scholar"}, MonstersDefeated: 1, ComponentsRepaired: 3, CreatedAt: now.Add(-1 * time.Hour)},
		{MatchID: "base_003", DurationSec: 890, Status: "won", TotalRounds: 8, FinalDarkness: 7, PlayerCount: 4, Characters: []string{"navigator", "engineer", "hunter", "scholar"}, MonstersDefeated: 3, ComponentsRepaired: 5, CreatedAt: now.Add(-30 * time.Minute)},
	}
	c.reports = append(c.reports, sampleReports...)
}

func (c *Collector) RecordMatch(_ context.Context, r MatchReport) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	c.reports = append(c.reports, r)
}

func (c *Collector) GetStats(_ context.Context) GlobalStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := len(c.reports)
	if total == 0 {
		return GlobalStats{
			RoleDistribution: make(map[string]int),
			LastUpdated:      time.Now(),
		}
	}

	wins := 0
	losses := 0
	sumRounds := 0
	sumDarkness := 0
	sumMonsters := 0
	roles := make(map[string]int)

	for _, r := range c.reports {
		if r.Status == "won" {
			wins++
		} else {
			losses++
		}
		sumRounds += r.TotalRounds
		sumDarkness += r.FinalDarkness
		sumMonsters += r.MonstersDefeated

		for _, ch := range r.Characters {
			roles[ch]++
		}
	}

	winRate := 0.0
	if total > 0 {
		winRate = (float64(wins) / float64(total)) * 100.0
	}

	return GlobalStats{
		TotalMatches:       total,
		Wins:               wins,
		Losses:             losses,
		WinRatePercent:     winRate,
		AvgRounds:          float64(sumRounds) / float64(total),
		AvgDarkness:        float64(sumDarkness) / float64(total),
		TotalMonstersSlain: sumMonsters,
		RoleDistribution:   roles,
		LastUpdated:        time.Now(),
	}
}
