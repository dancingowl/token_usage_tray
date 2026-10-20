package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const tokenControlsCSS = `
#token-controls{padding:4px 0 18px;border-bottom:1px solid var(--line);margin-bottom:20px}
#token-filter-form{display:flex;gap:8px 10px;align-items:center;flex-wrap:wrap}
#token-filter-form>label{font-weight:600;color:var(--sub);font-size:12px}
#token-filter-form select{max-width:230px;min-width:100px;padding:7px 26px 7px 9px;border:1px solid var(--line);border-radius:7px;background:var(--panel);color:var(--tx);font:inherit;font-size:12px}
#token-filter-form .token-all-label{display:flex;align-items:center;gap:5px;font-weight:500}
#token-filter-form button{padding:7px 15px;border:1px solid var(--line);border-radius:7px;background:var(--panel2);color:var(--tx);font:inherit;font-size:12px;cursor:pointer}
#token-controls p{margin:10px 0 0;font-size:11px}
.token-details td a{color:inherit;text-decoration:none}
.token-details td a:hover,.token-details td a:focus-visible{text-decoration:underline}
@media(max-width:600px){#token-filter-form select{max-width:calc(100% - 65px)}#token-filter-form>label:not(.token-all-label){flex-basis:48px}}
`

type tokenViewOptions struct {
	Days    int
	Model   string
	Project string
	All     bool
}

func defaultTokenViewOptions(o tokenViewOptions) tokenViewOptions {
	if o.Days != 7 && o.Days != 14 && o.Days != 30 {
		o.Days = 14
	}
	return o
}

func parseTokenViewOptions(q url.Values) (tokenViewOptions, error) {
	o := tokenViewOptions{Days: 14}
	for _, key := range []string{"days", "model", "project", "all"} {
		if len(q[key]) > 1 {
			return o, fmt.Errorf("%s must occur once", key)
		}
	}
	if values, ok := q["days"]; ok {
		if len(values) != 1 {
			return o, fmt.Errorf("invalid days")
		}
		d, err := strconv.Atoi(values[0])
		if err != nil || (values[0] != "7" && values[0] != "14" && values[0] != "30") {
			return o, fmt.Errorf("days must be 7, 14 or 30")
		}
		o.Days = d
	}
	for _, key := range []string{"model", "project"} {
		value := q.Get(key)
		if !utf8.ValidString(value) || len(value) > 512 || strings.IndexFunc(value, unicode.IsControl) >= 0 {
			return o, fmt.Errorf("invalid %s", key)
		}
		if key == "model" {
			o.Model = value
		} else {
			o.Project = value
		}
	}
	if values, ok := q["all"]; ok {
		if len(values) != 1 || (values[0] != "0" && values[0] != "1") {
			return o, fmt.Errorf("all must be 0 or 1")
		}
		o.All = values[0] == "1"
	}
	return o, nil
}

func tokenViewURL(o tokenViewOptions) string {
	o = defaultTokenViewOptions(o)
	q := url.Values{"days": {strconv.Itoa(o.Days)}}
	if o.Model != "" {
		q.Set("model", o.Model)
	}
	if o.Project != "" {
		q.Set("project", o.Project)
	}
	if o.All {
		q.Set("all", "1")
	}
	return "/tokens?" + q.Encode()
}

// Days14/Models7d/Total7d retain their wire names for older consumers, but all
// three contain the chosen window. Raw cache data stays at its 60-day horizon.
func tokenPeriodView(agg map[string]*Row, options tokenViewOptions, now time.Time, codex bool) AgentTokens {
	o := defaultTokenViewOptions(options)
	now = now.Local()
	today := now.Format("2006-01-02")
	since := now.AddDate(0, 0, 1-o.Days).Format("2006-01-02")
	var out AgentTokens
	days, models, projects := map[string]*Row{}, map[string]*Row{}, map[string]*Row{}
	for key, raw := range agg {
		parts := strings.SplitN(key, "|", 3)
		if len(parts) != 3 || raw == nil {
			continue
		}
		day, model, project := parts[0], parts[1], parts[2]
		if _, err := time.Parse("2006-01-02", day); err != nil {
			continue
		}
		if day < since || day > today || (o.Model != "" && o.Model != model) || (o.Project != "" && o.Project != project) {
			continue
		}
		row := *raw
		if codex {
			row.IncludedCache = row.CacheRead + row.CacheCreate
		}
		addTo(days, day, row)
		addTo(models, model, row)
		addTo(projects, project, row)
		out.Total7d.add(row)
		if day == today {
			out.Today.add(row)
		}
	}
	for i := o.Days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		row := Row{}
		if got := days[day]; got != nil {
			row = *got
		}
		out.Days14 = append(out.Days14, DayRow{Day: day, Row: row})
	}
	// Keep complete lists until both services have been merged. Only the HTML
	// number tables truncate, so top-six mode cannot lose cross-agent projects.
	out.Models7d = topRows(models, 0)
	out.Projects7d = topRows(projects, 0)
	return out
}

// Read exactly one cache per service. A combined process and agent-specific
// processes can describe identical source logs; adding files would double count.
func mergedTokenView(options tokenViewOptions) *TokenStats {
	return tokenViewFromCacheDir(dataDir(), options, time.Now())
}

func tokenViewFromCacheDir(dir string, options tokenViewOptions, now time.Time) *TokenStats {
	type source struct {
		rows     map[string]*Row
		updated  time.Time
		found    bool
		priority int
	}
	var claude, codex source
	for _, filename := range []string{"tokens-cache.json", "tokens-cache-claude.json", "tokens-cache-codex.json"} {
		path := filepath.Join(dir, filename)
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cache tokenCache
		if json.Unmarshal(b, &cache) != nil || cache.Version != 1 {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		choose := func(dst *source, rows map[string]*Row, priority int) {
			if rows == nil {
				return
			}
			if !dst.found || info.ModTime().After(dst.updated) || (info.ModTime().Equal(dst.updated) && priority > dst.priority) {
				*dst = source{rows, info.ModTime(), true, priority}
			}
		}
		if filename != "tokens-cache-codex.json" {
			p := 0
			if filename == "tokens-cache-claude.json" {
				p = 1
			}
			choose(&claude, cache.Claude, p)
		}
		if filename != "tokens-cache-claude.json" {
			p := 0
			if filename == "tokens-cache-codex.json" {
				p = 1
			}
			choose(&codex, cache.Codex, p)
		}
	}
	if !claude.found && !codex.found {
		return nil
	}
	o := defaultTokenViewOptions(options)
	out := &TokenStats{Updated: claude.updated, ViewDays: o.Days, AllRows: o.All}
	if codex.updated.After(out.Updated) {
		out.Updated = codex.updated
	}
	out.Claude = tokenPeriodView(claude.rows, o, now, false)
	out.Codex = tokenPeriodView(codex.rows, o, now, true)
	// Choice lists use the period without either drilldown, so a selection never
	// prevents switching to another model or project.
	models, projects := map[string]bool{}, map[string]bool{}
	for _, agent := range []AgentTokens{tokenPeriodView(claude.rows, tokenViewOptions{Days: o.Days}, now, false), tokenPeriodView(codex.rows, tokenViewOptions{Days: o.Days}, now, true)} {
		for _, row := range agent.Models7d {
			models[row.Name] = true
		}
		for _, row := range agent.Projects7d {
			projects[row.Name] = true
		}
	}
	for name := range models {
		out.ModelNames = append(out.ModelNames, name)
	}
	for name := range projects {
		out.ProjectNames = append(out.ProjectNames, name)
	}
	sort.Strings(out.ModelNames)
	sort.Strings(out.ProjectNames)
	return out
}

func tokenControlsHTML(tok *TokenStats, options tokenViewOptions) string {
	o := defaultTokenViewOptions(options)
	var b strings.Builder
	b.WriteString(`<section id="token-controls"><form id="token-filter-form" method="get" action="/tokens"><label for="token-days">기간</label><select id="token-days" name="days">`)
	for _, days := range []int{7, 14, 30} {
		selected := ""
		if days == o.Days {
			selected = ` selected`
		}
		fmt.Fprintf(&b, `<option value="%d"%s>최근 %d일</option>`, days, selected, days)
	}
	b.WriteString(`</select>`)
	selectHTML := func(id, name, label, current string, names []string) {
		fmt.Fprintf(&b, `<label for="%s">%s</label><select id="%s" name="%s"><option value="">전체 %s</option>`, id, label, id, name, label)
		seen := false
		for _, value := range names {
			selected := ""
			if value == current {
				selected = ` selected`
				seen = true
			}
			fmt.Fprintf(&b, `<option value="%s"%s>%s</option>`, html.EscapeString(value), selected, html.EscapeString(value))
		}
		if current != "" && !seen {
			fmt.Fprintf(&b, `<option value="%s" selected>%s</option>`, html.EscapeString(current), html.EscapeString(current))
		}
		b.WriteString(`</select>`)
	}
	var models, projects []string
	if tok != nil {
		models, projects = tok.ModelNames, tok.ProjectNames
	}
	selectHTML("token-model", "model", "모델", o.Model, models)
	selectHTML("token-project", "project", "프로젝트", o.Project, projects)
	checked := ""
	if o.All {
		checked = ` checked`
	}
	fmt.Fprintf(&b, `<label class="token-all-label"><input id="token-all" type="checkbox" name="all" value="1"%s>모델·프로젝트 전체 보기</label><button type="submit">적용</button></form><p class="summary-note">선택한 기간과 필터를 모든 토큰 통계에 적용합니다. 전체 보기는 표의 상위 6개 제한을 해제합니다.</p></section>`, checked)
	return b.String()
}
