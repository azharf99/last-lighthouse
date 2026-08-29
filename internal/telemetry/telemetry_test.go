package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestTelemetryCollector(t *testing.T) {
	ctx := context.Background()
	c := &Collector{reports: make([]MatchReport, 0)}

	// Initial empty stats
	stats := c.GetStats(ctx)
	if stats.TotalMatches != 0 || stats.WinRatePercent != 0 {
		t.Fatalf("expected 0 matches, got %d", stats.TotalMatches)
	}

	// Record a victory match
	c.RecordMatch(ctx, MatchReport{
		MatchID:            "m_won",
		DurationSec:        600,
		Status:             "won",
		TotalRounds:        6,
		FinalDarkness:      5,
		PlayerCount:        3,
		Characters:         []string{"navigator", "engineer", "hunter"},
		MonstersDefeated:   2,
		ComponentsRepaired: 5,
		CreatedAt:          time.Now(),
	})

	// Record a defeat match
	c.RecordMatch(ctx, MatchReport{
		MatchID:            "m_lost",
		DurationSec:        400,
		Status:             "lost",
		TotalRounds:        4,
		FinalDarkness:      8,
		PlayerCount:        2,
		Characters:         []string{"scholar", "engineer"},
		MonstersDefeated:   1,
		ComponentsRepaired: 2,
		CreatedAt:          time.Now(),
	})

	stats = c.GetStats(ctx)
	if stats.TotalMatches != 2 {
		t.Fatalf("expected 2 matches, got %d", stats.TotalMatches)
	}
	if stats.Wins != 1 || stats.Losses != 1 {
		t.Fatalf("expected 1 win 1 loss, got %d wins %d losses", stats.Wins, stats.Losses)
	}
	if stats.WinRatePercent != 50.0 {
		t.Fatalf("expected 50.0%% win rate, got %f", stats.WinRatePercent)
	}
	if stats.AvgRounds != 5.0 {
		t.Fatalf("expected avg 5.0 rounds, got %f", stats.AvgRounds)
	}
	if stats.AvgDarkness != 6.5 {
		t.Fatalf("expected avg 6.5 darkness, got %f", stats.AvgDarkness)
	}
	if stats.TotalMonstersSlain != 3 {
		t.Fatalf("expected 3 total monsters slain, got %d", stats.TotalMonstersSlain)
	}
	if stats.RoleDistribution["engineer"] != 2 || stats.RoleDistribution["navigator"] != 1 {
		t.Fatalf("unexpected role distribution: %+v", stats.RoleDistribution)
	}
}
