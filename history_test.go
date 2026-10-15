package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func historySamples(now time.Time) []usageHistorySample {
	var samples []usageHistorySample
	for i, left := range []int{100, 90, 80} {
		samples = append(samples, usageHistorySample{At: now.Add(time.Duration(i-2) * 30 * time.Minute), Agent: "claude", Window: "short",
			Left: left, ResetAt: now.Add(8 * time.Hour).Unix(), WindowS: 12 * 3600, Source: "api"})
	}
	return samples
}

func isolateUsageHistory(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("USAGE_TRAY_DATA_DIR", dir)
	previousOnly := only
	empty := ""
	only = &empty
	mu.Lock()
	previousState := state
	state = &State{}
	mu.Unlock()
	t.Cleanup(func() { only = previousOnly; mu.Lock(); state = previousState; mu.Unlock() })
	return dir
}

func historyStateFromSample(sample usageHistorySample) *State {
	u := emptyAgent()
	u.OK, u.Available, u.Src, u.FetchedAt = true, true, sample.Source, sample.At
	u.Short = Window{Left: sample.Left, ResetAt: sample.ResetAt, WindowS: sample.WindowS}
	s := &State{Updated: sample.At, Claude: emptyAgent(), Codex: emptyAgent(), Antigravity: emptyAgent()}
	switch sample.Agent {
	case "claude":
		s.Claude = u
	case "codex":
		s.Codex = u
	case "antigravity":
		s.Antigravity = u
	}
	return s
}

func TestUsageForecastCalculation(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	forecast := forecastUsageHistory(historySamples(now), now)[0]
	if !forecast.Ready || forecast.Samples != 3 || math.Abs(forecast.RatePerDay-480) > 0.000001 || !forecast.ExhaustsBeforeReset {
		t.Fatalf("unexpected rate: %+v", forecast)
	}
	if forecast.EstimatedExhaustion == nil || !forecast.EstimatedExhaustion.Equal(now.Add(4*time.Hour)) {
		t.Fatalf("wrong exhaustion: %+v", forecast)
	}
	if forecast.ResetAt != historySamples(now)[2].ResetAt {
		t.Fatalf("forecast must expose the same cycle reset used for its comparison: %+v", forecast)
	}
	if forecast.DailyBudget == nil || math.Abs(*forecast.DailyBudget-240) > 0.000001 {
		t.Fatalf("wrong quota percent per day: %+v", forecast)
	}
	// The timestamp is anchored to the observed remaining quota, not to each page read.
	later := forecastUsageHistory(historySamples(now), now.Add(10*time.Minute))[0]
	if !later.EstimatedExhaustion.Equal(*forecast.EstimatedExhaustion) {
		t.Fatal("page read postponed predicted exhaustion")
	}
	if !strings.Contains(forecast.Reason, "선형") {
		t.Fatal("prediction assumption omitted")
	}
}

func TestUsageObservationSeparatesMeasuredDropFromDailyRate(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	samples := historySamples(now)
	for i, left := range []int{100, 96, 93} {
		samples[i].At = now.Add(time.Duration(i-2) * time.Hour)
		samples[i].Left = left
	}
	f := forecastUsageHistory(samples, now)[0]
	o := f.Observation
	if !f.Ready || f.RatePerDay != 84 || o == nil || o.UsedPP != 7 || o.StartLeft != 100 || o.EndLeft != 93 ||
		!o.StartedAt.Equal(samples[0].At) || !o.EndedAt.Equal(samples[2].At) {
		t.Fatalf("measured interval must remain separate from daily extrapolation: %+v, %+v", f, o)
	}
}

func TestUsageObservationIndependentOfForecastReadiness(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	for _, condition := range []string{"single", "two", "short_zero_drop", "unknown_reset", "unknown_duration", "stale", "elapsed_reset", "reset_now", "freshness_boundary"} {
		t.Run(condition, func(t *testing.T) {
			samples := historySamples(now)
			want := true
			switch condition {
			case "single":
				samples, want = samples[2:], false
			case "two":
				samples = samples[1:]
			case "short_zero_drop":
				for i := range samples {
					samples[i].At, samples[i].Left = now.Add(time.Duration(i-2)*time.Minute), 80
				}
			case "unknown_reset", "unknown_duration":
				for i := range samples {
					if condition == "unknown_reset" {
						samples[i].ResetAt = 0
					} else {
						samples[i].WindowS = 0
					}
				}
			case "stale", "freshness_boundary":
				age := historyFreshness
				if condition == "stale" {
					age += time.Nanosecond
					want = false
				}
				for i := range samples {
					samples[i].At = samples[i].At.Add(-age)
				}
			case "elapsed_reset", "reset_now":
				reset := now
				if condition == "elapsed_reset" {
					reset = now.Add(-time.Second)
				}
				for i := range samples {
					samples[i].ResetAt = reset.Unix()
				}
				want = false
			}
			f := forecastUsageHistory(samples, now)[0]
			if (f.Observation != nil) != want {
				t.Fatalf("unexpected observed interval availability: %+v", f)
			}
			if condition != "freshness_boundary" && f.Ready {
				t.Fatalf("insufficient forecast evidence became ready: %+v", f)
			}
			if want {
				o := f.Observation
				if o.UsedPP != samples[0].Left-samples[len(samples)-1].Left || !o.StartedAt.Equal(samples[0].At) || !o.EndedAt.Equal(samples[len(samples)-1].At) {
					t.Fatalf("wrong actual endpoints: %+v", o)
				}
			}
		})
	}
}

func TestUsageForecastMinimumEvidenceAndResetComparison(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	samples := historySamples(now)
	for i := range samples {
		samples[i].At = now.Add(time.Duration(i-2) * 15 * time.Minute)
		samples[i].Left = 100
		samples[i].ResetAt = now.Add(time.Hour).Unix()
	}
	samples[2].Left = 99
	f := forecastUsageHistory(samples, now)[0]
	if !f.Ready || f.RatePerDay != 48 || f.ExhaustsBeforeReset {
		t.Fatalf("minimum evidence or after-reset prediction wrong: %+v", f)
	}
	for i := range samples {
		samples[i].ResetAt = f.EstimatedExhaustion.Unix()
	}
	f = forecastUsageHistory(samples, now)[0]
	if !f.Ready || f.ExhaustsBeforeReset {
		t.Fatal("exhaustion exactly at reset is not before reset")
	}
}

func TestUsageForecastNewCycle(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	for _, change := range []string{"recovery", "reset", "duration", "source"} {
		t.Run(change, func(t *testing.T) {
			samples := historySamples(now.Add(-time.Minute))
			last := samples[len(samples)-1]
			last.At = now
			switch change {
			case "recovery":
				last.Left = 100
			case "reset":
				last.ResetAt += 3600
			case "duration":
				last.WindowS += 3600
			case "source":
				last.Source = "앱 기록"
			}
			f := forecastUsageHistory(append(samples, last), now)[0]
			if f.Ready || f.Samples != 1 || f.EstimatedExhaustion != nil || f.Observation != nil {
				t.Fatalf("mixed quota cycles: %+v", f)
			}
			// A second observation measures only the new cycle, including recovery
			// and source boundaries, without waiting for forecast readiness.
			next := last
			next.At, next.Left = now.Add(time.Minute), last.Left-1
			f = forecastUsageHistory(append(samples, last, next), next.At)[0]
			if f.Ready || f.Observation == nil || f.Observation.UsedPP != 1 || !f.Observation.StartedAt.Equal(last.At) {
				t.Fatalf("observation crossed cycle boundary: %+v", f)
			}
		})
	}
}

func TestUsageForecastInsufficientOrStaleEvidence(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	for _, condition := range []string{"two", "short_span", "no_decrease", "stale", "past_reset", "unknown_reset", "unknown_duration"} {
		t.Run(condition, func(t *testing.T) {
			samples := historySamples(now)
			switch condition {
			case "two":
				samples = samples[1:]
			case "short_span":
				for i := range samples {
					samples[i].At = now.Add(time.Duration(i-2) * 5 * time.Minute)
				}
			case "no_decrease":
				for i := range samples {
					samples[i].Left = 80
				}
			case "stale":
				for i := range samples {
					samples[i].At = samples[i].At.Add(-31 * time.Minute)
				}
			case "past_reset":
				for i := range samples {
					samples[i].ResetAt = now.Add(-time.Second).Unix()
				}
			case "unknown_reset":
				for i := range samples {
					samples[i].ResetAt = 0
				}
			case "unknown_duration":
				for i := range samples {
					samples[i].WindowS = 0
				}
			}
			f := forecastUsageHistory(samples, now)[0]
			if f.Ready || f.EstimatedExhaustion != nil || f.RatePerDay != 0 || f.Reason == "" {
				t.Fatalf("unsupported forecast: %+v", f)
			}
			if condition == "stale" || condition == "past_reset" || condition == "unknown_reset" {
				if f.DailyBudget != nil {
					t.Fatalf("unsupported budget: %+v", f)
				}
			} else if f.DailyBudget == nil {
				t.Fatal("budget unnecessarily depends on consumption forecast")
			}
		})
	}
}

func TestUsageHistoryRecordsFreshObservationsOnce(t *testing.T) {
	isolateUsageHistory(t)
	now := time.Now().Add(-time.Minute)
	s := historyStateFromSample(historySamples(now)[2])
	if err := recordUsageHistory(s); err != nil {
		t.Fatal(err)
	}
	if err := recordUsageHistory(s); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"cache", "캐시 20s", "직전값", "rollout", ""} {
		s.Claude.Src, s.Claude.FetchedAt = source, now.Add(time.Second)
		if err := recordUsageHistory(s); err != nil {
			t.Fatal(err)
		}
	}
	s.Claude.Src, s.Claude.FetchedAt = "api", time.Time{}
	if err := recordUsageHistory(s); err != nil {
		t.Fatal(err)
	}
	s.Claude.FetchedAt = now.Add(-31 * time.Minute)
	if err := recordUsageHistory(s); err != nil {
		t.Fatal(err)
	}
	s.Claude.FetchedAt, s.Claude.OK = now.Add(2*time.Second), false
	if err := recordUsageHistory(s); err != nil {
		t.Fatal(err)
	}
	samples, err := readUsageHistory(historyPath())
	if err != nil || len(samples) != 1 || !samples[0].At.Equal(now) {
		t.Fatalf("invented observations: %+v, %v", samples, err)
	}
}

func TestUsageHistoryProcessIsolationAndMerge(t *testing.T) {
	dir := isolateUsageHistory(t)
	now := time.Now().Truncate(time.Second)
	claude := "claude"
	only = &claude
	samples := historySamples(now)
	s := historyStateFromSample(samples[2])
	s.Codex = s.Claude
	if err := recordUsageHistory(s); err != nil {
		t.Fatal(err)
	}
	part, err := readUsageHistory(filepath.Join(dir, "history-claude.json"))
	if err != nil || len(part) != 1 || part[0].Agent != "claude" {
		t.Fatalf("process wrote another agent: %v %v", part, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "history.json")); !os.IsNotExist(err) {
		t.Fatal("wrong process path")
	}
	if err := writeUsageHistory(filepath.Join(dir, "history-claude.json"), samples); err != nil {
		t.Fatal(err)
	}
	if err := writeUsageHistory(filepath.Join(dir, "history.json"), samples); err != nil {
		t.Fatal(err)
	}
	codex := append([]usageHistorySample(nil), samples...)
	for i := range codex {
		codex[i].Agent, codex[i].Source = "codex", "live"
	}
	if err := writeUsageHistory(filepath.Join(dir, "history-codex.json"), codex); err != nil {
		t.Fatal(err)
	}
	empty := ""
	only = &empty
	s.Codex = historyStateFromSample(codex[2]).Codex
	mu.Lock()
	state = s
	mu.Unlock()
	merged, forecasts, err := usageHistorySnapshot(now, 7)
	if err != nil || len(merged) != 6 || len(forecasts) != 2 || !forecasts[0].Ready || !forecasts[1].Ready {
		t.Fatalf("wrong merge: %d %+v %v", len(merged), forecasts, err)
	}
	mu.Lock()
	state.Claude.Src = "직전값"
	mu.Unlock()
	_, forecasts, err = usageHistorySnapshot(now, 7)
	if err != nil || forecasts[0].Ready || forecasts[0].DailyBudget != nil || !forecasts[1].Ready {
		t.Fatalf("cache should suppress only its own prediction: %+v %v", forecasts, err)
	}
	mu.Lock()
	state.Claude.Src = "api"
	state.Claude.Short.ResetAt++
	mu.Unlock()
	_, forecasts, err = usageHistorySnapshot(now, 7)
	if err != nil || forecasts[0].Ready || forecasts[0].DailyBudget != nil {
		t.Fatal("unrecorded new cycle reused old forecast")
	}
}

func TestUsageHistoryCorruptFileKeepsOtherHistory(t *testing.T) {
	dir := isolateUsageHistory(t)
	now := time.Now().Truncate(time.Second)
	bad := []byte("{incomplete")
	if err := os.WriteFile(historyPath(), bad, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recordUsageHistory(historyStateFromSample(historySamples(now)[2])); err == nil {
		t.Fatal("corrupt history silently overwritten")
	}
	got, err := os.ReadFile(historyPath())
	if err != nil || string(got) != string(bad) {
		t.Fatal("lost corrupt history")
	}
	if err := writeUsageHistory(filepath.Join(dir, "history-claude.json"), historySamples(now)); err != nil {
		t.Fatal(err)
	}
	samples, _, err := usageHistorySnapshot(now, 1)
	if err == nil || len(samples) != 3 {
		t.Fatalf("should return valid partial history and diagnostic: %v %v", samples, err)
	}
}

func TestUsageHistorySnapshotSuppressesObservationForInvalidCurrentState(t *testing.T) {
	for _, condition := range []string{"not_ok", "cached", "missing_time", "future", "stale", "different_time", "different_source", "reset", "duration", "left"} {
		t.Run(condition, func(t *testing.T) {
			isolateUsageHistory(t)
			now := time.Now().Truncate(time.Second)
			samples := historySamples(now)
			if err := writeUsageHistory(historyPath(), samples); err != nil {
				t.Fatal(err)
			}
			s := historyStateFromSample(samples[2])
			mu.Lock()
			state = s
			mu.Unlock()
			_, forecasts, err := usageHistorySnapshot(now, 7)
			if err != nil || len(forecasts) != 1 || forecasts[0].Observation == nil {
				t.Fatalf("valid current state must expose observation: %+v, %v", forecasts, err)
			}
			switch condition {
			case "not_ok":
				s.Claude.OK = false
			case "cached":
				s.Claude.Src = "직전값"
			case "missing_time":
				s.Claude.FetchedAt = time.Time{}
			case "future":
				s.Claude.FetchedAt = now.Add(time.Second)
			case "stale":
				s.Claude.FetchedAt = now.Add(-historyFreshness - time.Second)
			case "different_time":
				s.Claude.FetchedAt = now.Add(-time.Second)
			case "different_source":
				s.Claude.Src = "앱 기록"
			case "reset":
				s.Claude.Short.ResetAt++
			case "duration":
				s.Claude.Short.WindowS++
			case "left":
				s.Claude.Short.Left--
			}
			_, forecasts, err = usageHistorySnapshot(now, 7)
			if err != nil || len(forecasts) != 1 {
				t.Fatalf("snapshot failed: %+v, %v", forecasts, err)
			}
			f := forecasts[0]
			if f.Observation != nil || f.Ready || f.RatePerDay != 0 || f.EstimatedExhaustion != nil || f.DailyBudget != nil {
				t.Fatalf("current mismatch retained derived information: %+v", f)
			}
		})
	}
}

func TestUsageHistoryRetentionAndGraphRange(t *testing.T) {
	isolateUsageHistory(t)
	now := time.Now().Truncate(time.Second)
	samples := historySamples(now)
	for _, ago := range []time.Duration{2 * 24 * time.Hour, 31 * 24 * time.Hour} {
		older := samples[0]
		older.At = now.Add(-ago)
		samples = append(samples, older)
	}
	invalid := samples[0]
	invalid.Left = 101
	invalid.At = now.Add(-2 * time.Hour)
	samples = append(samples, invalid)
	future := samples[0]
	future.At = now.Add(time.Minute)
	samples = append(samples, future)
	if err := writeUsageHistory(historyPath(), samples); err != nil {
		t.Fatal(err)
	}
	s := historyStateFromSample(historySamples(now)[2])
	if err := recordUsageHistory(s); err != nil {
		t.Fatal(err)
	}
	stored, err := readUsageHistory(historyPath())
	if err != nil || len(stored) != 4 {
		t.Fatalf("invalid or expired samples retained: %d %v", len(stored), err)
	}
	oneDay, _, err := usageHistorySnapshot(now, 1)
	if err != nil || len(oneDay) != 3 {
		t.Fatalf("one day graph: %d %v", len(oneDay), err)
	}
	sevenDays, _, err := usageHistorySnapshot(now, 7)
	if err != nil || len(sevenDays) != 4 {
		t.Fatalf("seven day graph: %d %v", len(sevenDays), err)
	}
	data, err := os.ReadFile(historyPath())
	if err != nil || !json.Valid(data) {
		t.Fatal("history is not complete JSON")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(historyPath()), ".usage-history-*.tmp"))
	if len(leftovers) != 0 {
		t.Fatalf("temporary files leaked: %v", leftovers)
	}
}
