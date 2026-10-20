package main

import (
	"math"
	"testing"
	"time"
)

func TestForecastStabilityDistinguishesChangingAndRoundedUsage(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	for _, tc := range []struct {
		name      string
		left      []int
		stability string
		ready     bool
	}{
		{"steady", []int{100, 98, 96, 94, 92, 90, 88, 86}, "steady", true},
		{"accelerating", []int{100, 100, 100, 100, 100, 99, 95, 85}, "changing", false},
		{"slowing", []int{100, 90, 80, 70, 70, 70, 70, 70}, "changing", false},
		{"rounding", []int{100, 100, 100, 100, 100, 100, 100, 99}, "limited", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var samples []usageHistorySample
			for i, left := range tc.left {
				samples = append(samples, usageHistorySample{At: now.Add(time.Duration(i-len(tc.left)+1) * 30 * time.Minute), Agent: "codex", Window: "week", Left: left, ResetAt: now.Add(5 * 24 * time.Hour).Unix(), WindowS: 7 * 24 * 3600, Source: "live"})
			}
			f := forecastUsageHistory(samples, now)[0]
			if f.Stability != tc.stability || f.Ready != tc.ready {
				t.Fatalf("unexpected forecast: %+v", f)
			}
			if f.RecentRatePerDay == nil || f.RecentMinutes != 60 {
				t.Fatalf("recent independent observation missing: %+v", f)
			}
			if !tc.ready && (f.EstimatedExhaustion != nil || f.ExhaustsBeforeReset || f.DailyBudget == nil || f.Observation == nil) {
				t.Fatalf("changing trend should suppress only exhaustion, retain budget and actual usage: %+v", f)
			}
			if tc.name == "steady" && math.Abs(*f.RecentRatePerDay-f.RatePerDay) > .000001 {
				t.Fatalf("constant slopes differ: %+v", f)
			}
		})
	}
}

func TestShortForecastRemainsAvailableButExplicitlyLimited(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	f := forecastUsageHistory(historySamples(now), now)[0]
	if !f.Ready || f.Stability != "limited" || f.StabilityReason == "" {
		t.Fatalf("short evidence must keep existing estimate with a limitation: %+v", f)
	}
}
