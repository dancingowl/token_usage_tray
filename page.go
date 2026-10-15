package main

// 상세 화면 — 127.0.0.1 의 작은 HTTP 서버가 내는 HTML 한 장.
//
// 왜 네이티브 창이 아닌가: 트레이 라이브러리(systray)는 창을 만들지 못하고, 창을 쓰려면
// Windows 는 WinForms, macOS 는 Cocoa 로 따로 만들어야 한다(= 두 벌 유지). HTML 한 장이면
// 양쪽이 같고, 차트·표를 그리기도 훨씬 쉽다. 브라우저 탭으로 열린다.
//
// 화면별 주소를 나누고, 갱신 가능한 통계를 분리해 설정 입력을 유지한다.

import (
	"encoding/base64"
	"fmt"
	"html"
	"image/color"
	"strings"
	"time"
)

func fmtTokens(n int64) string {
	f := float64(n)
	switch {
	case f >= 1e9:
		return trimZero(fmt.Sprintf("%.1f", f/1e9)) + "B"
	case f >= 1e6:
		return trimZero(fmt.Sprintf("%.1f", f/1e6)) + "M"
	case f >= 1e3:
		return trimZero(fmt.Sprintf("%.1f", f/1e3)) + "k"
	}
	return fmt.Sprintf("%d", n)
}

func trimZero(s string) string { return strings.TrimSuffix(s, ".0") }

var weekdayKo = [...]string{"일", "월", "화", "수", "목", "금", "토"}

// 리셋 시각을 사람이 바로 읽게. 오늘/내일은 그렇게 쓰고, 그 밖은 9/25(금) 08:00.
func fmtResetHuman(unix int64) string { return fmtResetKo(unix, true) }

// withWeekday=false 면 요일을 뺀다 — 트레이 툴팁이 63자밖에 못 담아서 쓰는 축약판.
func fmtResetKo(unix int64, withWeekday bool) string {
	if unix <= 0 {
		return "?"
	}
	t := time.Unix(unix, 0).Local()
	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(t.Sub(today).Hours() / 24)
	switch days {
	case 0:
		return "오늘 " + t.Format("15:04")
	case 1:
		return "내일 " + t.Format("15:04")
	}
	if !withWeekday {
		return fmt.Sprintf("%d/%d %s", int(t.Month()), t.Day(), t.Format("15:04"))
	}
	return fmt.Sprintf("%d/%d(%s) %s", int(t.Month()), t.Day(), weekdayKo[int(t.Weekday())], t.Format("15:04"))
}

// 시각 없이 날짜만 — 오늘 · 내일 · 9/25. 툴팁 자리가 모자랄 때 쓴다.
func fmtResetDay(unix int64) string {
	if unix <= 0 {
		return "?"
	}
	t := time.Unix(unix, 0).Local()
	now := time.Now().Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch int(t.Sub(today).Hours() / 24) {
	case 0:
		return "오늘"
	case 1:
		return "내일"
	}
	return fmt.Sprintf("%d/%d", int(t.Month()), t.Day())
}

// 얼마나 남았나 — 5시간 창은 분까지, 주간 창은 일까지.
func fmtUntil(unix int64) string {
	if unix <= 0 {
		return ""
	}
	d := time.Until(time.Unix(unix, 0))
	if d <= 0 {
		return "곧"
	}
	switch {
	case d >= 48*time.Hour:
		return fmt.Sprintf("%d일 뒤", int(d.Hours()/24))
	case d >= time.Hour:
		return fmt.Sprintf("%d시간 뒤", int(d.Hours()))
	default:
		return fmt.Sprintf("%d분 뒤", int(d.Minutes()))
	}
}

func fmtAge(min int) string {
	if min < 0 {
		return ""
	}
	if min >= 60 {
		return fmt.Sprintf("%dh%02dm", min/60, min%60)
	}
	return fmt.Sprintf("%dm", min)
}

// 플랜 문자열을 사람이 읽게 — API 값 self_serve_business_prolite 를 Business Prolite 로.
func prettyPlan(p string) string {
	p = strings.ReplaceAll(strings.TrimPrefix(p, "self_serve_"), "_", " ")
	words := strings.Fields(p)
	for i, w := range words {
		r := []rune(w)
		words[i] = strings.ToUpper(string(r[:1])) + string(r[1:])
	}
	return strings.Join(words, " ")
}

// 에이전트 마크 — 이름과 브랜드 색. 표 · 제목 · 요약이 모두 이걸 쓴다.
type agentMark struct {
	name  string
	brand color.NRGBA
}

var (
	markAntigravity = agentMark{"Antigravity", colAntigravity}
	markClaude      = agentMark{"Claude", colClaude}
	markCodex       = agentMark{"Codex", colCodex}
)

// 두 에이전트의 마크를 인라인 SVG 로 그린다. 파일을 읽지 않으므로 단일 실행파일 그대로다.
//
// Claude Code — 스타버스트(긴 살 4 + 짧은 살 4).
// Codex — 겹친 타원 3개(꽃 모양).
// 실제 배포 로고 파일이 있으면 이 함수만 바꿔 끼우면 된다.
func agentSVG(name string) string {
	if name == "Antigravity" {
		return `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 19L12 4l9 15M7 14h10" fill="none" stroke="currentColor" stroke-width="2.5"/></svg>`
	}
	if name == "Codex" {
		return `<svg viewBox="0 0 24 24" aria-hidden="true">` +
			`<g transform="translate(12 12)" fill="none" stroke="currentColor" stroke-width="1.7">` +
			`<ellipse rx="3.7" ry="8.6"/><ellipse rx="3.7" ry="8.6" transform="rotate(60)"/>` +
			`<ellipse rx="3.7" ry="8.6" transform="rotate(120)"/></g></svg>`
	}
	// 몸통 · 좌우 귀 · 다리 둘을 한 path 로 잇고, 눈은 같은 path 의 부분경로로 둔다.
	// evenodd 라 몸통 안에 든 눈만 구멍이 되고(서로 겹치지 않게 좌표를 잡았다) 배지의
	// 반투명 배경이 비쳐 원본 이미지처럼 어두운 눈이 된다.
	return `<svg viewBox="0 0 24 24" aria-hidden="true">` +
		`<path fill="currentColor" fill-rule="evenodd" d="` +
		`M4 4h16v12.5H4Z` + // 몸통
		`M2 8h2v4H2Z` + // 왼쪽 귀
		`M20 8h2v4h-2Z` + // 오른쪽 귀
		`M5.5 16.5h2v3h-2Z` + // 왼쪽 다리
		`M15.5 16.5h2v3h-2Z` + // 오른쪽 다리
		`M6.5 6.5h1.6v3.6H6.5Z` + // 왼쪽 눈(구멍)
		`M15.9 6.5h1.6v3.6h-1.6Z` + // 오른쪽 눈(구멍)
		`"/></svg>`
}

// cls: "lg" 표의 큰 배지 · "sm" 제목·요약의 작은 마크.
func agentIcon(m agentMark, cls string) string {
	return fmt.Sprintf(`<span class="ic %s" style="--c:%d,%d,%d" title="%s">%s</span>`,
		cls, m.brand.R, m.brand.G, m.brand.B, html.EscapeString(m.name), agentSVG(m.name))
}

// 카드에 붙는 작은 꼬리표. Kind 는 "" 보통 · "bad" 경고 · "live" 실시간.
type chip struct {
	Text string
	Kind string
}

type gaugeCard struct {
	Title   string
	Left    int
	Brand   color.NRGBA
	ResetAt int64
	Chips   []chip
	Mark    agentMark
}

// 14일 막대 차트 — HTML/CSS. 텍스트가 축소되지 않는 것이 SVG 대신 쓰는 이유다.
func barChartHTML(days []DayRow, brand color.NRGBA, valueOf func(DayRow) int64) string {
	var max int64 = 1
	vals := make([]int64, len(days))
	for i, d := range days {
		vals[i] = valueOf(d)
		if vals[i] > max {
			max = vals[i]
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<div class="chart" style="--brand:rgb(%d,%d,%d)">`, brand.R, brand.G, brand.B)

	// 눈금 4줄 — 위에서 아래로. 라벨은 왼쪽 여백에 오른쪽 정렬로 앉힌다.
	b.WriteString(`<div class="ch-scale" aria-hidden="true">`)
	for k := 3; k >= 0; k-- {
		fmt.Fprintf(&b, `<div class="ch-line"><span>%s</span></div>`, fmtTokens(max*int64(k)/3))
	}
	b.WriteString(`</div><div class="ch-bars">`)

	for i, v := range vals {
		h := 0.0
		if v > 0 {
			h = float64(v) / float64(max) * 100
			if h < 1.2 {
				h = 1.2
			}
		}
		cls := "ch-bar"
		if i == len(vals)-1 {
			cls += " is-today"
		}
		// 날짜는 14일을 빠짐없이 적는다. 붙어 보이지 않게 앞의 0 을 떼어 9/18 꼴로 쓰고,
		// 칸이 정말 좁은 화면(휴대폰 폭)에서만 CSS 가 .sec 을 숨긴다.
		md := days[i].Day[5:] // "09-18"
		label := strings.TrimLeft(md[:2], "0") + "/" + md[3:]
		xcls := "ch-x"
		if i%2 == 0 && i != len(vals)-1 {
			xcls += " sec"
		}
		fmt.Fprintf(&b, `<div class="ch-col" title="%s · %s"><div class="ch-track">`+
			`<div class="%s" style="height:%.1f%%"></div></div><div class="%s">%s</div></div>`,
			html.EscapeString(days[i].Day), fmtTokens(v), cls, h, xcls, label)
	}

	b.WriteString(`</div></div>`)
	return b.String()
}

const pageCSS = `
:root{
  color-scheme:light;
  --bg:#f4f6f8; --panel:#fff; --panel2:#f7f9fa; --line:#e1e6ec;
  --tx:#202938; --sub:#626b78; --dim:#626b78;
  --accent:#157f71; --warn:#a66b08; --bad:#c64242;
}
*{box-sizing:border-box}
html{scroll-behavior:smooth;scroll-padding-top:20px}
body{margin:0 auto;max-width:1040px;background:var(--bg);color:var(--tx);
  font:14px/1.55 "Segoe UI","Malgun Gothic","Apple SD Gothic Neo",system-ui,sans-serif;padding:28px 24px 34px}
.num,em,.cl,.ch-line span,.ch-x{font-variant-numeric:tabular-nums}
header{margin-bottom:22px}
.header-top{display:flex;align-items:center;justify-content:space-between;gap:16px 24px;flex-wrap:wrap;margin-bottom:20px}
.header-status{display:flex;align-items:center;justify-content:flex-end;gap:14px;flex-wrap:wrap;margin-left:auto}
.brand{display:flex;align-items:center;gap:10px;min-width:0}
.app-icon{width:34px;height:34px;flex:none;border-radius:9px}
h1{font-size:21px;line-height:1.2;letter-spacing:-.03em;margin:0;font-weight:750}
.meta{color:var(--dim);font-size:12px}
.dot{display:inline-block;width:5px;height:5px;border-radius:50%;background:#39a68e;margin-right:5px;vertical-align:2px}
.page-nav{display:flex;align-items:center;gap:28px;border-bottom:1px solid var(--line)}
.page-nav a{color:var(--sub);font-size:14px;text-decoration:none;padding:10px 0 12px;border-bottom:3px solid transparent;margin-bottom:-1px;white-space:nowrap}
.page-nav a:hover{color:var(--accent);border-bottom-color:#a9d4c9}
.page-nav a[aria-current="page"]{border-bottom-color:var(--accent);color:#0c695d;font-weight:700}
body:not([data-page="usage"]) #usage-snapshot,body:not([data-page="tokens"]) #token-details{display:none}
a:focus-visible,summary:focus-visible{outline:2px solid var(--accent);outline-offset:3px}
.row{display:grid;gap:12px;margin-bottom:14px;min-width:0}
.row.cards,.row.two{grid-template-columns:repeat(2,minmax(0,1fr))}
.row.cards>.card:only-child,.row.two>.panel:only-child,.stats>.stat:only-child{grid-column:1/-1}
.card,.panel,.stat{background:var(--panel);border:1px solid var(--line);border-radius:13px;min-width:0}
.card{padding:17px 19px;box-shadow:0 2px 5px #20293803}
.cbody{min-width:0}
.card-top{display:flex;align-items:center;justify-content:space-between;gap:12px}
.ct{font-weight:650;font-size:14px;display:flex;align-items:center;gap:8px;min-width:0}
.ct-label{overflow-wrap:anywhere}
.cl{font-size:29px;font-weight:750;line-height:1.2;white-space:nowrap;letter-spacing:-.03em}
.cl span{font-size:12px;color:var(--sub);font-weight:500;margin-left:3px;letter-spacing:0}
.cl.unknown{font-size:17px;color:var(--sub);font-weight:600;letter-spacing:0}
.cl.warn{color:var(--warn)}.cl.bad{color:var(--bad)}
.cs{display:flex;align-items:baseline;justify-content:space-between;gap:8px 16px;color:var(--sub);font-size:13px;margin-top:12px;flex-wrap:wrap}
.cs strong{font-size:14px;color:var(--tx);font-weight:750}
.cs time{white-space:nowrap;font-variant-numeric:tabular-nums}
.card-notes{color:var(--dim);font-size:11px;margin-top:5px;overflow-wrap:anywhere}
.card-note.bad{color:var(--bad)}.card-note.live{color:var(--accent)}
.chip{background:var(--panel2);border:1px solid var(--line);color:var(--dim);font-size:11px;line-height:1.3;padding:3px 7px;border-radius:6px}
.panel{padding:17px 19px}
.panel h2{font-size:13px;color:var(--sub);margin:0 0 14px;font-weight:650;display:flex;align-items:center;gap:8px}
.token-details{margin-top:2px}
.token-details>h2{color:var(--tx);font-size:17px;margin-bottom:18px}
.token-details>h2 .summary-note{margin-left:12px}
.token-details #token-snapshot>.row:last-of-type{margin-bottom:0}
.token-details .panel,.token-details .stat{background:var(--panel2)}
.chart{position:relative;padding-left:48px}
.ch-scale{position:absolute;left:48px;right:0;top:0;height:152px;display:flex;flex-direction:column;justify-content:space-between;pointer-events:none}
.ch-line{position:relative;border-top:1px solid var(--line)}
.ch-line:not(:first-child){border-top-style:dashed}
.ch-line span{position:absolute;left:-48px;top:-9px;width:41px;text-align:right;font-size:11px;color:var(--dim)}
.ch-bars{position:relative;display:flex;gap:5px;height:176px}
.ch-col{flex:1;display:flex;flex-direction:column;min-width:0}
.ch-track{flex:1;display:flex;align-items:flex-end}
.ch-bar{width:100%;background:var(--brand);opacity:.45;border-radius:3px 3px 0 0;min-height:2px}
.ch-bar.is-today{opacity:.9}.ch-col:hover .ch-bar{opacity:.8}
.ch-x{height:24px;line-height:24px;text-align:center;font-size:10px;color:var(--dim);letter-spacing:-.3px;white-space:nowrap}
.stats{display:grid;gap:12px;grid-template-columns:repeat(2,minmax(0,1fr));margin-bottom:14px}
.stat{padding:12px 14px;display:flex;flex-wrap:wrap;align-items:center;gap:6px 14px}
.stat b{font-size:12px;display:flex;align-items:center;gap:7px}
.stat i{font-style:normal;color:var(--dim);font-size:11px}
.stat em{font-style:normal;color:var(--tx);font-weight:700;font-size:12px;margin-left:5px}
.table-scroll{overflow-x:auto;max-width:100%;scrollbar-width:thin}
table{width:100%;border-collapse:collapse;font-size:12px}
.table-scroll table{min-width:350px;table-layout:fixed}
.table-scroll th:first-child{width:36%}
.projects-table th:nth-child(2){width:18%}
th{text-align:left;color:var(--dim);font-weight:500;font-size:11px;padding:0 0 8px;border-bottom:1px solid var(--line)}
td{padding:9px 0;border-bottom:1px solid var(--line)}
tr:last-child td{border-bottom:0}
th.num,td.num{text-align:right;padding-left:10px;white-space:nowrap}
td.num{color:var(--sub)}
td.name,td.plain{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
td.who{width:1%;white-space:nowrap}
.mrow{display:flex;align-items:center;min-width:0}
.mt{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.ic{display:inline-flex;align-items:center;justify-content:center;flex:none;color:rgb(var(--c))}
.ic svg{width:100%;height:100%;display:block}
.ic.lg{width:25px;height:25px;padding:4px;margin-right:8px;border-radius:6px;background:rgba(var(--c),.08)}
.ic.sm{width:21px;height:21px}
.marks{display:inline-flex;gap:4px;align-items:center}
tfoot td{border-top:1px solid var(--line);border-bottom:0;padding-top:10px;font-weight:700;color:var(--tx)}
tfoot tr+tr td{border-top:0;padding-top:2px}tfoot td.num{color:var(--tx)}
tfoot .sum{color:var(--sub);font-weight:500;font-size:11px}
footer{color:var(--dim);font-size:11px;margin-top:16px;overflow-wrap:anywhere}
.none{color:var(--sub);font-size:12px}
@media(max-width:760px){
  .row.cards,.row.two,.stats{grid-template-columns:1fr}
  .ch-x.sec{visibility:hidden}
}
@media(max-width:480px){
  body{padding:20px 14px 26px}header{margin-bottom:16px}
  .header-top{gap:14px;margin-bottom:12px}.header-status{width:100%;margin-left:0;justify-content:space-between;gap:8px}.meta{font-size:11px}
  .page-nav{gap:24px}.page-nav a{font-size:13px;padding:9px 0 11px}
  .card,.panel{padding:15px}.cl{font-size:27px}.ct{font-size:13px;gap:6px}
  .cl span{font-size:11px}.card-top{gap:8px}.token-details .panel{padding:12px}
}
@media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}}
`

func renderPage(s *State, tok *TokenStats) string {
	return renderPageView(s, tok, "usage")
}

func renderPageView(s *State, tok *TokenStats, view string) string {
	pageTitle := "한도 현황"
	switch view {
	case "tokens":
		pageTitle = "토큰 통계"
	case "settings":
		pageTitle = "설정"
	default:
		view = "usage"
	}
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="ko"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light">
<link rel="icon" href="data:image/png;base64,` + base64.StdEncoding.EncodeToString(faviconPNG) + `">
<title>Usage Tray · ` + pageTitle + `</title><style>` + pageCSS + dashboardCSS + `</style></head><body data-page="` + view + `">`)

	fmt.Fprintf(&b, `<header><div class="header-top"><div class="brand"><img class="app-icon" src="data:image/png;base64,%s" alt="">`+
		`<h1>Usage Tray</h1></div><div class="header-status"><div class="meta" id="snapshot-time"><span class="dot"></span>%s 갱신 · 자동 갱신</div>`,
		base64.StdEncoding.EncodeToString(faviconPNG), s.Updated.Local().Format("15:04:05"))
	b.WriteString(headerActionsHTML + `</div></div><p id="action-message" class="dash-message header-message" role="status" aria-live="polite"></p><nav class="page-nav" aria-label="화면 이동">`)
	for _, item := range []struct{ view, path, label string }{
		{"usage", "/", "한도 현황"}, {"tokens", "/tokens", "토큰 통계"}, {"settings", "/settings", "설정"},
	} {
		active := ""
		if item.view == view {
			active = ` aria-current="page"`
		}
		fmt.Fprintf(&b, `<a href="%s"%s>%s</a>`, item.path, active, item.label)
	}
	b.WriteString(`</nav></header>`)

	b.WriteString(`<section id="usage-snapshot"><div class="row cards">`)
	for _, c := range gaugeCards(s) {
		b.WriteString(cardHTML(c))
	}
	b.WriteString(`</div>`)

	for _, a := range []struct {
		name  string
		usage AgentUsage
	}{{"Claude", s.Claude}, {"Codex", s.Codex}, {"Antigravity", s.Antigravity}} {
		if a.usage.Available && a.usage.Error != "" {
			fmt.Fprintf(&b, `<div class="panel" style="margin-bottom:14px"><strong>%s</strong> · %s</div>`, html.EscapeString(a.name), html.EscapeString(a.usage.Error))
		}
	}
	if s.Antigravity.Available {
		b.WriteString(`<p class="none">Antigravity는 모델별 잔여 한도와 리셋 시각을 표시합니다. 토큰 수와 과거 사용량은 제공되지 않습니다.</p>`)
	}
	b.WriteString(`</section>` + dashboardHTML + `<section id="token-details" class="token-details panel dashboard-full" aria-labelledby="token-title"><h2 id="token-title">토큰 통계<span class="summary-note">일별 · 모델별 · 프로젝트별</span></h2><section id="token-snapshot">`)
	if tok == nil {
		b.WriteString(`<div class="panel none">토큰 집계가 아직 없습니다. 첫 스캔이 끝나면 채워집니다.</div></section></section>` + dashboardScriptTag() + `</body></html>`)
		return b.String()
	}

	b.WriteString(`<div class="row two">`)
	if s.Claude.Available {
		fmt.Fprintf(&b, `<div class="panel"><h2>%sClaude · 14일 토큰 (입력+캐시+출력)</h2>%s</div>`,
			agentIcon(markClaude, "sm"),
			barChartHTML(tok.Claude.Days14, colClaude, func(d DayRow) int64 { return d.total() }))
	}
	if s.Codex.Available {
		fmt.Fprintf(&b, `<div class="panel"><h2>%sCodex · 14일 토큰 (입력+출력)</h2>%s</div>`,
			agentIcon(markCodex, "sm"),
			barChartHTML(tok.Codex.Days14, colCodex, func(d DayRow) int64 { return d.Input + d.Output }))
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="stats">`)
	if s.Claude.Available {
		t := tok.Claude.Today
		b.WriteString(statHTML("Claude 오늘", markClaude, [][2]string{
			{"입력", fmtTokens(t.Input + t.CacheRead + t.CacheCreate)},
			{"출력", fmtTokens(t.Output)},
			{"요청", fmt.Sprintf("%d", t.Requests)},
			{"캐시 적중", fmt.Sprintf("%d%%", t.hitPct())},
		}))
	}
	if s.Codex.Available {
		t := tok.Codex.Today
		b.WriteString(statHTML("Codex 오늘", markCodex, [][2]string{
			{"입력", fmtTokens(t.Input)},
			{"출력", fmtTokens(t.Output)},
			{"요청", fmt.Sprintf("%d", t.Requests)},
			{"캐시", fmt.Sprintf("%d%%", t.hitPct())},
		}))
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="row two">`)

	models := mergeModels(s, tok)
	b.WriteString(`<div class="panel"><h2>모델별 · 7일</h2><div class="table-scroll"><table>` +
		`<tr><th>모델</th><th class="num">요청</th><th class="num">입력</th>` +
		`<th class="num">출력</th><th class="num">캐시</th></tr>`)
	for _, m := range models {
		fmt.Fprintf(&b, `<tr><td class="name">%s</td><td class="num">%d</td>`+
			`<td class="num">%s</td><td class="num">%s</td><td class="num">%d%%</td></tr>`,
			modelCell(m), m.Requests,
			fmtTokens(m.Input+m.CacheCreate), fmtTokens(m.Output), m.hitPct())
	}
	b.WriteString(`<tfoot>`)
	for _, t := range agentTotals(s, tok, models) {
		fmt.Fprintf(&b, `<tr><td class="sum">%s</td><td class="num">%d</td>`+
			`<td class="num">%s</td><td class="num">%s</td><td class="num">%d%%</td></tr>`,
			html.EscapeString(t.label), t.Requests,
			fmtTokens(t.Input+t.CacheCreate), fmtTokens(t.Output), t.hitPct())
	}
	b.WriteString(`</tfoot></table></div></div>`)

	projects := mergeProjects(s, tok)
	b.WriteString(`<div class="panel"><h2>프로젝트별 · 7일</h2><div class="table-scroll"><table class="projects-table">` +
		`<tr><th>프로젝트</th><th></th><th class="num">요청</th><th class="num">토큰</th></tr>`)
	for _, p := range projects {
		fmt.Fprintf(&b, `<tr><td class="plain" title="%s">%s</td><td class="who">%s</td>`+
			`<td class="num">%d</td><td class="num">%s</td></tr>`,
			html.EscapeString(p.Name), html.EscapeString(p.Name),
			marksHTML(p.marks), p.Requests, fmtTokens(p.total()))
	}
	b.WriteString(`<tfoot>`)
	for _, t := range agentTotals(s, tok, projects) {
		fmt.Fprintf(&b, `<tr><td class="sum" colspan="2">%s</td><td class="num">%d</td>`+
			`<td class="num">%s</td></tr>`,
			html.EscapeString(t.label), t.Requests, fmtTokens(t.total()))
	}
	b.WriteString(`</tfoot></table></div></div></div>`)

	fmt.Fprintf(&b, `<footer>집계 %s · %dms · 출처 Claude %s / Codex %s</footer>`,
		tok.Updated.Local().Format("2006-01-02 15:04:05"), tok.ScanMs,
		html.EscapeString(orDash(s.Claude.Src)), html.EscapeString(orDash(s.Codex.Src)))

	b.WriteString(`</section></section>` + dashboardScriptTag() + `</body></html>`)
	return b.String()
}

// Keep service-wide metadata explicit so a limit is not mistaken for one quota's exhaustion.
func quotaNotes(name string, u AgentUsage) []chip {
	var notes []chip
	if p := prettyPlan(u.Plan); p != "" {
		notes = append(notes, chip{Text: p})
	}
	if u.Src != "" {
		text := "출처 " + u.Src
		kind := ""
		if u.Src == "api" || u.Src == "live" {
			text = "실시간"
			kind = "live"
		} else {
			if u.Src == "cache" {
				text = "캐시"
			}
			if u.AgeMin >= 0 {
				text += fmt.Sprintf(" · %d분 전", u.AgeMin)
			}
		}
		notes = append(notes, chip{Text: text, Kind: kind})
	} else if u.AgeMin >= 0 {
		notes = append(notes, chip{Text: "기록 " + fmtAge(u.AgeMin) + " 전"})
	}
	if u.Extra != "" {
		notes = append(notes, chip{Text: u.Extra})
	}
	if u.Limit != "" {
		notes = append(notes, chip{Text: name + " 이용 제한: " + u.Limit, Kind: "bad"})
	}
	return notes
}

func gaugeCards(s *State) []gaugeCard {
	var cards []gaugeCard
	for _, a := range []struct {
		usage AgentUsage
		mark  agentMark
	}{{s.Claude, markClaude}, {s.Codex, markCodex}} {
		if !a.usage.Available {
			continue
		}
		if w := a.usage.Short; w.Left >= 0 || w.ResetAt > 0 {
			cards = append(cards, gaugeCard{a.mark.name + " · " + windowName(w, "5시간") + " 한도",
				w.Left, a.mark.brand, w.ResetAt, nil, a.mark})
		}
		w := a.usage.Week
		cards = append(cards, gaugeCard{a.mark.name + " · " + windowName(w, "주간") + " 한도",
			w.Left, a.mark.brand, w.ResetAt, quotaNotes(a.mark.name, a.usage), a.mark})
	}
	if s.Antigravity.Available {
		notes := quotaNotes("Antigravity", s.Antigravity)
		if len(s.Antigravity.Models) == 0 {
			cards = append(cards, gaugeCard{"Antigravity · 모델 한도", -1, colAntigravity, 0, notes, markAntigravity})
		}
		for _, m := range s.Antigravity.Models {
			cards = append(cards, gaugeCard{"Antigravity · " + m.Label + " 한도", m.Left, colAntigravity, m.ResetAt, notes, markAntigravity})
		}
	}
	return cards
}

func cardHTML(c gaugeCard) string {
	return cardHTMLAt(c, time.Now())
}

func cardHTMLAt(c gaugeCard, now time.Time) string {
	cls := ""
	switch {
	case c.Left < 0:
		cls = " unknown"
	case c.Left <= 10:
		cls = " bad"
	case c.Left <= 20:
		cls = " warn"
	}
	left := `확인 중`
	if c.Left >= 0 {
		left = fmt.Sprintf(`%d%%<span>남음</span>`, c.Left)
	}
	// Share native tooltip formatting, then put its countdown before the date.
	reset := clickResetDetail(Window{Left: c.Left, ResetAt: c.ResetAt}, now)
	resetHTML := `<strong>` + html.EscapeString(reset) + `</strong>`
	if date, remaining, ok := strings.Cut(reset, " 초기화 · "); ok {
		if strings.HasSuffix(remaining, " 후") {
			remaining += " 초기화"
		}
		resetHTML = fmt.Sprintf(`<strong>%s</strong><time datetime="%s">%s</time>`,
			html.EscapeString(remaining), time.Unix(c.ResetAt, 0).In(now.Location()).Format(time.RFC3339), html.EscapeString(date))
	}
	var notes strings.Builder
	for _, note := range c.Chips {
		if note.Text == "" {
			continue
		}
		if notes.Len() > 0 {
			notes.WriteString(` · `)
		}
		kind := ""
		if note.Kind != "" {
			kind = " " + html.EscapeString(note.Kind)
		}
		fmt.Fprintf(&notes, `<span class="card-note%s">%s</span>`, kind, html.EscapeString(note.Text))
	}
	info := ""
	if notes.Len() > 0 {
		info = `<div class="card-notes">` + notes.String() + `</div>`
	}
	return fmt.Sprintf(`<div class="card" data-agent="%s"><div class="cbody"><div class="card-top">`+
		`<div class="ct">%s<span class="ct-label">%s</span></div><div class="cl%s">%s</div></div>`+
		`<div class="cs">%s</div>%s</div></div>`,
		strings.ToLower(c.Mark.name), agentIcon(c.Mark, "sm"), html.EscapeString(c.Title),
		cls, left, resetHTML, info)
}

func statHTML(title string, m agentMark, pairs [][2]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="stat"><b>%s%s</b>`, agentIcon(m, "sm"), html.EscapeString(title))
	for _, p := range pairs {
		fmt.Fprintf(&b, `<i>%s<em>%s</em></i>`, html.EscapeString(p[0]), html.EscapeString(p[1]))
	}
	b.WriteString(`</div>`)
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// 표의 한 줄 — 어느 에이전트에서 왔는지 마크로 들고 다닌다. 프로젝트 표는 이름으로 병합하므로
// 마크가 둘일 수 있다(예: skmp 를 Claude 와 Codex 가 함께 쓴다). 모델 표는 항상 하나다.
type projRow struct {
	NamedRow
	marks []agentMark
}

// 합계 행. label 은 "Claude 합계" · "전체 합계".
type totalRow struct {
	Row
	label string
}

func mergeModels(s *State, tok *TokenStats) []projRow {
	var out []projRow
	if s.Claude.Available {
		for _, m := range tok.Claude.Models7d {
			out = append(out, projRow{m, []agentMark{markClaude}})
		}
	}
	if s.Codex.Available {
		for _, m := range tok.Codex.Models7d {
			out = append(out, projRow{m, []agentMark{markCodex}})
		}
	}
	sortRows(out)
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

// 모델 한 줄 — 그 모델을 돌린 에이전트 마크 + 모델명. 마크가 곧 분류라 별도 태그 열을 뺐다.
func modelCell(m projRow) string {
	icon := ""
	if len(m.marks) > 0 {
		icon = agentIcon(m.marks[0], "lg")
	}
	return fmt.Sprintf(`<div class="mrow" title="%s">%s<span class="mt">%s</span></div>`,
		html.EscapeString(m.Name), icon, html.EscapeString(m.Name))
}

func marksHTML(marks []agentMark) string {
	if len(marks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<span class="marks">`)
	for _, m := range marks {
		b.WriteString(agentIcon(m, "sm"))
	}
	b.WriteString(`</span>`)
	return b.String()
}

// 합계 — 표에 보이는 상위 6개가 아니라 7일 창의 전체 합이다(AgentTokens.Total7d).
// 옛 tokens.json 캐시에는 그 값이 없으므로, 0 이면 보이는 줄의 합으로 메운다.
func agentTotals(s *State, tok *TokenStats, shown []projRow) []totalRow {
	var out []totalRow
	var all Row
	add := func(name string, r Row) {
		if r.Requests == 0 {
			r = sumShown(shown, name)
		}
		if r.Requests == 0 && r.total() == 0 {
			return
		}
		all.add(r)
		out = append(out, totalRow{r, name + " 합계"})
	}
	if s.Claude.Available {
		add("Claude", tok.Claude.Total7d)
	}
	if s.Codex.Available {
		add("Codex", tok.Codex.Total7d)
	}
	switch len(out) {
	case 0:
		return nil
	case 1:
		out[0].label = "합계"
	default:
		out = append(out, totalRow{all, "전체 합계"})
	}
	return out
}

// 옛 캐시 대비용 — 그 에이전트의 마크가 붙은 줄만 더한다. 병합된 줄은 양쪽에 걸리므로
// 합이 살짝 커질 수 있으나, Total7d 가 있으면 이 경로는 타지 않는다.
func sumShown(shown []projRow, who string) Row {
	var r Row
	for _, x := range shown {
		for _, m := range x.marks {
			if m.name == who {
				r.add(x.Row)
				break
			}
		}
	}
	return r
}

// 프로젝트는 **이름으로 병합**한다 — 같은 저장소를 Claude 와 Codex 로 같이 쓰면 두 줄로
// 갈려 보여서 한눈에 "이 프로젝트에 얼마 썼나" 가 안 나왔다. 합쳐서 한 줄로 내고 어느
// 에이전트가 쓰였는지는 마크로 보인다.
func mergeProjects(s *State, tok *TokenStats) []projRow {
	byName := map[string]*projRow{}
	var order []string
	take := func(rows []NamedRow, mark agentMark) {
		for _, p := range rows {
			cur := byName[p.Name]
			if cur == nil {
				cur = &projRow{NamedRow: NamedRow{Name: p.Name}}
				byName[p.Name] = cur
				order = append(order, p.Name)
			}
			cur.NamedRow.Row.add(p.Row)
			cur.marks = append(cur.marks, mark)
		}
	}
	if s.Claude.Available {
		take(tok.Claude.Projects7d, markClaude)
	}
	if s.Codex.Available {
		take(tok.Codex.Projects7d, markCodex)
	}

	var out []projRow
	for _, name := range order {
		out = append(out, *byName[name])
	}
	sortRows(out)
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func sortRows(rows []projRow) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j].total() > rows[j-1].total(); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}
