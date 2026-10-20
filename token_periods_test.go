package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTokenViewOptionsAndURLRoundTrip(t *testing.T) {
	for _, days := range []int{7, 14, 30} {
		want := tokenViewOptions{Days: days, Model: `model+<&"`, Project: `프로젝트 &?#/`, All: true}
		u, err := url.Parse(tokenViewURL(want))
		if err != nil {
			t.Fatal(err)
		}
		got, err := parseTokenViewOptions(u.Query())
		if err != nil || got != want {
			t.Fatalf("round trip %v: %v, %v", want, got, err)
		}
	}
	got, err := parseTokenViewOptions(url.Values{})
	if err != nil || got.Days != 14 {
		t.Fatalf("default: %+v %v", got, err)
	}
	for _, raw := range []string{"days=0", "days=31", "days=14&days=7", "days=014", "all=true", "all=1&all=0", "model=x%00", "project=%FF"} {
		q, _ := url.ParseQuery(raw)
		_, err := parseTokenViewOptions(q)

		if err == nil {
			t.Errorf("accepted malformed query %q", raw)
		}
	}
}

func TestTokenPeriodsUseCalendarBoundsAndIntersectFilters(t *testing.T) {
	now := time.Date(2026, 11, 2, 13, 0, 0, 0, time.Local)
	agg := map[string]*Row{}
	for i := 0; i < 61; i++ {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		agg[day+"|m|p"] = &Row{Input: 10, Requests: 1}
	}
	agg[now.AddDate(0, 0, 1).Format("2006-01-02")+"|m|p"] = &Row{Input: 9999, Requests: 999}
	agg[now.Format("2006-01-02")+"|other|p"] = &Row{Input: 7, Requests: 1}
	agg[now.Format("2006-01-02")+"|m|other"] = &Row{Input: 8, Requests: 1}
	agg["garbage|m|p"] = &Row{Input: 9999}
	agg["2026-11-02|nil|p"] = nil
	for _, days := range []int{7, 14, 30} {
		got := tokenPeriodView(agg, tokenViewOptions{Days: days, Model: "m", Project: "p"}, now, false)
		if len(got.Days14) != days || got.Total7d.Input != int64(days*10) || got.Total7d.Requests != int64(days) || got.Today.Input != 10 {
			t.Fatalf("%d-day view: %+v", days, got)
		}
		if got.Days14[0].Day != now.AddDate(0, 0, 1-days).Format("2006-01-02") || got.Days14[days-1].Day != now.Format("2006-01-02") {
			t.Fatal("calendar boundaries shifted")
		}
		if len(got.Models7d) != 1 || got.Models7d[0].Name != "m" || len(got.Projects7d) != 1 || got.Projects7d[0].Name != "p" {
			t.Fatal("filters must intersect")
		}
	}
	if len(agg) != 66 {
		t.Fatalf("view mutated raw cache: %d keys", len(agg))
	}
}

func TestTokenTopSixKeepsCompleteTotalsAndMergedProjectCandidates(t *testing.T) {
	now := time.Now()
	day := now.Format("2006-01-02")
	claude, codex := map[string]*Row{}, map[string]*Row{}
	for i := 0; i < 8; i++ {
		claude[fmt.Sprintf("%s|c%d|c%d", day, i, i)] = &Row{Input: 100, Requests: 1}
		codex[fmt.Sprintf("%s|x%d|x%d", day, i, i)] = &Row{Input: 100, Requests: 1}
	}
	// Seventh per service, but first when the shared project is merged.
	claude[day+"|shared|shared"] = &Row{Input: 90, Requests: 1}
	codex[day+"|shared|shared"] = &Row{Input: 90, CacheRead: 80, Output: 1, Requests: 1}
	tok := &TokenStats{Claude: tokenPeriodView(claude, tokenViewOptions{}, now, false), Codex: tokenPeriodView(codex, tokenViewOptions{}, now, true)}
	s := &State{}
	s.Claude.Available = true
	s.Codex.Available = true
	top := mergeProjectsView(s, tok, false)
	all := mergeProjectsView(s, tok, true)
	if len(top) != 6 || len(all) != 17 || top[0].Name != "shared" || top[0].total() != 181 {
		t.Fatalf("merged projects/top limit wrong: top %+v, all %d", top, len(all))
	}
	if len(mergeModelsView(s, tok, false)) != 6 || len(mergeModelsView(s, tok, true)) != 18 {
		t.Fatal("all mode must expand models")
	}
	if !reflect.DeepEqual(agentTotals(s, tok, top), agentTotals(s, tok, all)) {
		t.Fatal("total must not depend on top-six expansion")
	}
	totals := agentTotals(s, tok, top)
	if totals[len(totals)-1].total() != 1781 {
		t.Fatalf("Codex cached input counted twice: %+v", totals)
	}
	if tok.Codex.Total7d.hitPct() != 8 {
		t.Fatalf("Codex cache denominator should use inclusive input: %+v", tok.Codex.Total7d)
	}
}

func writeTokenPeriodCache(t *testing.T, dir, name string, cache tokenCache, at time.Time) {
	t.Helper()
	b, err := json.Marshal(cache)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err = os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Chtimes(p, at, at); err != nil {
		t.Fatal(err)
	}
}

func TestTokenViewLoadsCombinedAndSuffixCachesWithoutDoubleCounting(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	day := now.Format("2006-01-02")
	combined := tokenCache{Version: 1, Claude: map[string]*Row{day + "|c|combined": {Input: 10, Requests: 1}}, Codex: map[string]*Row{day + "|x|combined": {Input: 20, CacheRead: 15, Requests: 1}}}
	writeTokenPeriodCache(t, dir, "tokens-cache.json", combined, now.Add(-time.Hour))
	// Dedicated Claude wins for Claude; its stale copied Codex section must not leak.
	dedicated := tokenCache{Version: 1, Claude: map[string]*Row{day + "|new|dedicated": {Input: 30, Requests: 1}}, Codex: map[string]*Row{day + "|wrong|wrong": {Input: 99999}}}
	writeTokenPeriodCache(t, dir, "tokens-cache-claude.json", dedicated, now)
	if err := os.WriteFile(filepath.Join(dir, "tokens-cache-codex.json"), []byte(`{"version":1`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tokens-cache-extra.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	got := tokenViewFromCacheDir(dir, tokenViewOptions{Days: 30, All: true}, now)
	if got == nil || got.Claude.Total7d.Input != 30 || got.Codex.Total7d.Input != 20 || got.Codex.Total7d.total() != 20 || len(got.Claude.Days14) != 30 {
		t.Fatalf("cache selection: %+v", got)
	}
	if !reflect.DeepEqual(got.ModelNames, []string{"new", "x"}) || !reflect.DeepEqual(got.ProjectNames, []string{"combined", "dedicated"}) {
		t.Fatalf("choice lists: %+v", got)
	}
	filtered := tokenViewFromCacheDir(dir, tokenViewOptions{Days: 7, Model: "new", Project: "dedicated"}, now)
	if filtered.Claude.Total7d.Input != 30 || filtered.Codex.Total7d.total() != 0 || len(filtered.ModelNames) != 2 {
		t.Fatal("drilldown must filter data but retain alternative choices")
	}
	if got := tokenViewFromCacheDir(t.TempDir(), tokenViewOptions{}, now); got != nil {
		t.Fatal("missing raw cache should not invent historical data")
	}
}

func TestTokenControlsAndDrilldownEscapeAndPreserveOptions(t *testing.T) {
	name := `<img src=x onerror="bad"> & +?`
	o := tokenViewOptions{Days: 30, Model: name, Project: name, All: true}
	tok := &TokenStats{Updated: time.Now(), ModelNames: []string{name}, ProjectNames: []string{name}, Claude: AgentTokens{Models7d: []NamedRow{{Name: name, Row: Row{Input: 10, Requests: 1}}}, Projects7d: []NamedRow{{Name: name, Row: Row{Input: 10, Requests: 1}}}}}
	s := &State{}
	s.Claude.Available = true
	page := renderPageViewWithTokenOptions(s, tok, "tokens", o)
	controls, snapshot := strings.Index(page, `id="token-controls"`), strings.Index(page, `id="token-snapshot"`)
	if controls < 0 || snapshot < controls {
		t.Fatal("controls must remain outside replaceable snapshot")
	}
	for _, want := range []string{`id="token-filter-form" method="get" action="/tokens"`, `id="token-model"`, `id="token-project"`, `id="token-all"`, `value="30" selected`, `모델별 · 30일`, `프로젝트별 · 30일`, `Claude · 30일`, `href="` + html.EscapeString(tokenViewURL(o)) + `"`} {
		if !strings.Contains(page, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(page, `<img src=x`) || strings.Contains(page, `onerror="bad"`) {
		t.Fatal("untrusted filter name escaped out of markup")
	}
}
