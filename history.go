package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Percentages are quota percentages, not token estimates. Only actual provider
// observations enter the history; serving the same cached observation adds nothing.
type usageHistorySample struct {
	At      time.Time `json:"at"`
	Agent   string    `json:"agent"`
	Window  string    `json:"window"`
	Left    int       `json:"left"`
	ResetAt int64     `json:"reset_at"`
	WindowS int64     `json:"window_seconds"`
	Source  string    `json:"source"`
}

type usageForecast struct {
	Agent               string     `json:"agent"`
	Window              string     `json:"window"`
	Ready               bool       `json:"ready"`
	Reason              string     `json:"reason"`
	RatePerDay          float64    `json:"rate_per_day"`
	EstimatedExhaustion *time.Time `json:"estimated_exhaustion,omitempty"`
	ExhaustsBeforeReset bool       `json:"exhausts_before_reset"`
	DailyBudget         *float64   `json:"daily_budget,omitempty"`
	Samples             int        `json:"samples"`
}

const (
	historyRetention  = 30 * 24 * time.Hour
	historyFreshness  = 30 * time.Minute
	historyMaxSamples = 300000 // More than all six streams sampled every minute for 30 days.
	historyMaxBytes   = 64 << 20
)

var historyMu sync.Mutex

func historyPath() string { return filepath.Join(dataDir(), "history"+suffix()+".json") }

func freshHistorySource(src string) bool {
	switch src {
	case "api", "live", "local", "앱 기록":
		return true
	default:
		return false
	}
}

func historyAgentUsage(s *State, agent string) AgentUsage {
	switch agent {
	case "claude":
		return s.Claude
	case "codex":
		return s.Codex
	default:
		return s.Antigravity
	}
}

func validHistorySample(s usageHistorySample, now time.Time) bool {
	return (s.Agent == "claude" || s.Agent == "codex" || s.Agent == "antigravity") &&
		(s.Window == "short" || s.Window == "week") && s.Left >= 0 && s.Left <= 100 &&
		!s.At.IsZero() && !s.At.After(now) && !s.At.Before(now.Add(-historyRetention)) &&
		s.ResetAt >= 0 && s.WindowS >= 0 && freshHistorySource(s.Source)
}

func normalizeHistory(samples []usageHistorySample, now time.Time) []usageHistorySample {
	// Sort before deduplication so merge order never changes time ordering.
	sort.SliceStable(samples, func(i, j int) bool {
		if !samples[i].At.Equal(samples[j].At) {
			return samples[i].At.Before(samples[j].At)
		}
		if samples[i].Agent != samples[j].Agent {
			return samples[i].Agent < samples[j].Agent
		}
		return samples[i].Window < samples[j].Window
	})
	type sampleKey struct {
		agent, window string
		at            int64
	}
	seen := make(map[sampleKey]bool, len(samples))
	out := make([]usageHistorySample, 0, len(samples))
	for _, s := range samples {
		if !validHistorySample(s, now) {
			continue
		}
		key := sampleKey{s.Agent, s.Window, s.At.UnixNano()}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, s)
	}
	if len(out) > historyMaxSamples {
		out = out[len(out)-historyMaxSamples:]
	}
	return out
}

func readUsageHistory(path string) ([]usageHistorySample, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, historyMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > historyMaxBytes {
		return nil, fmt.Errorf("사용량 이력 파일이 너무 큽니다: %s", path)
	}
	var samples []usageHistorySample
	if err := json.Unmarshal(data, &samples); err != nil {
		return nil, fmt.Errorf("사용량 이력 읽기 %s: %w", path, err)
	}
	return samples, nil
}

func writeUsageHistory(path string, samples []usageHistorySample) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".usage-history-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if err := json.NewEncoder(f).Encode(samples); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func recordUsageHistory(s *State) error {
	if s == nil {
		return nil
	}
	historyMu.Lock()
	defer historyMu.Unlock()
	now := time.Now()
	path := historyPath()
	samples, err := readUsageHistory(path)
	if err != nil {
		return err
	} // Preserve a corrupt file for diagnosis rather than replacing it.
	for _, agent := range []string{"claude", "codex", "antigravity"} {
		if tag() != "" && tag() != agent {
			continue
		}
		u := historyAgentUsage(s, agent)
		if !u.OK || !freshHistorySource(u.Src) || u.FetchedAt.IsZero() || now.Sub(u.FetchedAt) > historyFreshness {
			continue
		}
		for _, w := range []struct {
			name  string
			value Window
		}{{"short", u.Short}, {"week", u.Week}} {
			samples = append(samples, usageHistorySample{At: u.FetchedAt, Agent: agent, Window: w.name,
				Left: w.value.Left, ResetAt: w.value.ResetAt, WindowS: w.value.WindowS, Source: u.Src})
		}
	}
	samples = normalizeHistory(samples, now)
	return writeUsageHistory(path, samples)
}

// Forecast each stream independently. A recovery, reset, changed duration or
// observation source starts a new cycle; the old samples remain graphable.
func forecastUsageHistory(samples []usageHistorySample, now time.Time) []usageForecast {
	samples = normalizeHistory(samples, now)
	var out []usageForecast
	for _, agent := range []string{"claude", "codex", "antigravity"} {
		for _, window := range []string{"short", "week"} {
			var cycle []usageHistorySample
			for _, s := range samples {
				if s.Agent != agent || s.Window != window {
					continue
				}
				if len(cycle) > 0 {
					prev := cycle[len(cycle)-1]
					if s.ResetAt != prev.ResetAt || s.WindowS != prev.WindowS || s.Left > prev.Left || s.Source != prev.Source {
						cycle = nil
					}
				}
				cycle = append(cycle, s)
			}
			if len(cycle) == 0 {
				continue
			}
			first, last := cycle[0], cycle[len(cycle)-1]
			f := usageForecast{Agent: agent, Window: window, Samples: len(cycle)}
			if now.Sub(last.At) <= historyFreshness && time.Unix(last.ResetAt, 0).After(now) {
				budget := float64(last.Left) / time.Unix(last.ResetAt, 0).Sub(now).Hours() * 24
				f.DailyBudget = &budget
			}
			switch {
			case now.Sub(last.At) > historyFreshness:
				f.Reason = "최근 관측이 30분보다 오래되어 예상을 숨겼습니다."
			case last.ResetAt <= 0 || last.WindowS <= 0:
				f.Reason = "한도 주기 또는 초기화 시간이 없어 예상할 수 없습니다."
			case !time.Unix(last.ResetAt, 0).After(now):
				f.Reason = "초기화 이후 새 관측을 기다립니다."
			default:
				switch {
				case len(cycle) < 3:
					f.Reason = "현재 한도 주기에 실제 관측이 3개 이상 필요합니다."
				case last.At.Sub(first.At) < 30*time.Minute:
					f.Reason = "30분 이상의 관측이 쌓이면 예상합니다."
				case first.Left-last.Left < 1:
					f.Reason = "관측 중 1% 이상의 감소가 없어 예상할 수 없습니다."
				default:
					// Endpoint slope is transparent and conservatively follows only the
					// current cycle. The exhaustion instant is anchored to the last reading.
					f.RatePerDay = float64(first.Left-last.Left) / last.At.Sub(first.At).Hours() * 24
					estimated := last.At.Add(time.Duration(float64(last.Left) / f.RatePerDay * float64(24*time.Hour)))
					f.EstimatedExhaustion = &estimated
					f.ExhaustsBeforeReset = estimated.Before(time.Unix(last.ResetAt, 0))
					f.Ready = true
					f.Reason = "현재 한도 주기의 첫 관측부터 최근 관측까지 감소 속도가 계속된다는 선형 예상입니다."
				}
			}
			out = append(out, f)
		}
	}
	return out
}

func usageHistorySnapshot(now time.Time, days int) ([]usageHistorySample, []usageForecast, error) {
	if days < 1 {
		days = 1
	}
	if days > 30 {
		days = 30
	}
	historyMu.Lock()
	var samples []usageHistorySample
	var errs []error
	// Enumerate known names only; unrelated files cannot grow the merge unbounded.
	for _, suffix := range []string{"", "-claude", "-codex", "-antigravity"} {
		part, err := readUsageHistory(filepath.Join(dataDir(), "history"+suffix+".json"))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		samples = append(samples, part...)
	}
	samples = normalizeHistory(samples, now)
	historyMu.Unlock()
	forecasts := forecastUsageHistory(samples, now)
	current := mergedState()
	for i := range forecasts {
		f := &forecasts[i]
		u := historyAgentUsage(&current, f.Agent)
		w := u.Short
		if f.Window == "week" {
			w = u.Week
		}
		var latest usageHistorySample
		for j := len(samples) - 1; j >= 0; j-- {
			if samples[j].Agent == f.Agent && samples[j].Window == f.Window {
				latest = samples[j]
				break
			}
		}
		if !u.OK || !freshHistorySource(u.Src) || u.FetchedAt.IsZero() || u.FetchedAt.After(now) || now.Sub(u.FetchedAt) > historyFreshness ||
			!u.FetchedAt.Equal(latest.At) || u.Src != latest.Source || w.ResetAt != latest.ResetAt || w.WindowS != latest.WindowS || w.Left != latest.Left {
			f.Ready, f.ExhaustsBeforeReset, f.RatePerDay = false, false, 0
			f.EstimatedExhaustion, f.DailyBudget = nil, nil
			f.Reason = "현재 조회가 실제 최신 관측이 아니어서 예상을 숨겼습니다."
		}
	}
	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	start := sort.Search(len(samples), func(i int) bool { return !samples[i].At.Before(cutoff) })
	return samples[start:], forecasts, errors.Join(errs...)
}
