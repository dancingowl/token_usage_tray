package main

// The dashboard is self-contained: no remote scripts, credentials or full-page
// refresh. The existing token charts are refreshed separately from the form.
const dashboardCSS = `
[hidden]{display:none!important}
.dashboard{margin-bottom:14px}.dash-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,310px),1fr));gap:24px;margin-top:20px}
body:not([data-page="usage"]) #history-details,body:not([data-page="usage"]) #usage-plan,body:not([data-page="settings"]) #settings,body:not([data-page="settings"]) #system-details{display:none}


.page-section{margin-bottom:16px}.page-section>h2{font-size:17px;color:var(--tx);margin-bottom:20px}.page-section>h2 .summary-note{margin-left:10px}
#usage-plan-title{flex-wrap:wrap}#usage-plan-context{display:inline}
.dash-panel{margin-bottom:14px}.dash-panel h2{font-size:15px;color:var(--tx);margin-bottom:12px}
.dash-actions{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
.history-controls{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;max-width:650px}.history-controls label{display:flex;align-items:center;gap:8px;white-space:nowrap}.history-controls select{flex:1;width:0}
select,input[type=number]{min-height:36px;min-width:0;max-width:100%;font:inherit;color:var(--tx);background:#fff;border:1px solid #cbd3dd;border-radius:8px;padding:7px 10px}
button,.button{display:inline-flex;align-items:center;justify-content:center;gap:8px;min-height:36px;padding:8px 14px;box-sizing:border-box;font:inherit;font-weight:600;line-height:1.2;color:#202938;background:#f3f5f8;border:1px solid #d7dfe8;border-radius:8px;text-decoration:none;cursor:pointer;transition:background-color .14s,border-color .14s}
button:hover:not(:disabled),.button:hover{background:#e8edf4;border-color:#aab8c8}button:active:not(:disabled),.button:active{background:#dde5f0;border-color:#9aa8b6}
button:disabled{color:#758291;background:#f5f6f8;border-color:#e1e5ea;opacity:.7;cursor:default}
button.primary{background:#0f766e;border-color:#258b83;color:#fff}button.primary:hover:not(:disabled){background:#115e59;border-color:#35a298}button.primary:active:not(:disabled){background:#134e4a}button.primary:disabled{background:#29433f;border-color:#3c5752;color:#b1bfba}
.button.subtle{background:transparent;border-color:transparent;color:#526171}.button.subtle:hover{background:#e8edf4;border-color:#d7dfe8;color:#202938}
.button-icon{width:16px;height:16px;flex:none;fill:none;stroke:currentColor;stroke-width:1.75;stroke-linecap:round;stroke-linejoin:round}
button:focus-visible,input:focus-visible,select:focus-visible,summary:focus-visible,a:focus-visible{outline:2px solid #0f766e;outline-offset:3px}
@media(prefers-reduced-motion:reduce){button,.button{transition:none}}
.dash-note{font-size:12px;color:var(--sub);margin:8px 0;overflow-wrap:anywhere}.dash-message{color:#0f766e;font-size:13px;margin:8px 0 0;overflow-wrap:anywhere}.dash-message:empty{display:none}.dash-message.error{color:var(--bad)}
.quick-toolbar{margin:0}.header-message{margin:0 0 12px;text-align:right}
.app-update-banner{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px 16px;margin-bottom:16px;background:var(--panel);border:1px solid var(--line);border-radius:10px;font-size:13px}.app-update-banner p{margin:0}
.disclosure{margin-bottom:12px;scroll-margin-top:24px}.disclosure>summary{display:flex;align-items:center;gap:12px;list-style:none;min-height:26px}.disclosure>summary::-webkit-details-marker{display:none}.disclosure>summary:after{content:'+';font-size:21px;font-weight:400;color:var(--sub);margin-left:auto}.disclosure[open]>summary:after{content:'−'}.summary-note{font-size:12px;font-weight:400;color:var(--sub)}
.history-heading{display:flex;align-items:center;gap:12px;position:relative;margin-bottom:20px}.history-heading h2{margin:0;font-size:17px;color:var(--tx)}.history-heading .summary-note{margin-left:10px}
.history-help{margin-left:auto}.history-help>summary{display:flex;align-items:center;justify-content:center;list-style:none;width:25px;height:25px;border:1px solid #cbd3dd;border-radius:50%;color:var(--sub);font-size:13px;font-weight:600;cursor:pointer}.history-help>summary::-webkit-details-marker{display:none}.history-help[open]>summary{margin:0;background:#e1eeea;color:#0c695d}.history-help-content{position:absolute;right:0;top:34px;z-index:3;width:340px;max-width:100%;padding:12px 14px;border:1px solid var(--line);border-radius:10px;background:#fff;box-shadow:0 4px 16px #20293820;color:var(--sub);font-size:12px}.history-help-content p{margin:0 0 8px}.history-help-content p:last-child{margin:0}
.hover-help{padding-bottom:9px;margin-bottom:-9px}
.history-range{font-size:11px;color:var(--sub);margin:4px 0 0;overflow-wrap:anywhere}
.history-view-controls{display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap;margin-top:14px}.history-view-controls span{font-size:12px;color:var(--sub)}#history-zoom[aria-pressed="true"]{color:#0c695d;background:#e1eeea;border-color:#a9d4c9}.history-overview{margin-top:12px;padding:10px 12px;border:1px solid var(--line);border-radius:9px;background:var(--panel2)}.history-overview svg{display:block;width:100%;height:76px}.history-overview-labels{display:flex;justify-content:space-between;gap:10px;font-size:11px;color:var(--sub)}
.settings-fields{border:0;padding:0;margin:0;min-width:0}.settings-section{padding:22px 0;border-bottom:1px solid var(--line)}.settings-section:first-child{padding-top:0}.settings-section:last-child{border-bottom:0;padding-bottom:0}.settings-section>h2{font-size:16px;color:var(--tx);margin:0 0 12px}.settings-row{display:grid;grid-template-columns:minmax(0,1fr) minmax(180px,auto);align-items:center;gap:10px 24px;padding:12px 0}.settings-label{font-size:13px;color:var(--tx);min-width:0}.settings-control{display:flex;align-items:center;justify-content:flex-end;gap:8px;flex-wrap:wrap;min-width:0}.settings-control select{width:230px;max-width:100%}.settings-row .dash-note{grid-column:1/-1;margin:0}.settings-switch{appearance:none;position:relative;flex:none;width:36px;height:22px;border:1px solid #b6c1cd;border-radius:12px;background:#dbe1e8;cursor:pointer;margin:0;vertical-align:middle}.settings-switch:before{content:'';position:absolute;left:2px;top:2px;width:16px;height:16px;border-radius:50%;background:#fff;box-shadow:0 1px 2px #20293825}.settings-switch:checked{background:#0f766e;border-color:#0f766e}.settings-switch:checked:before{left:16px}.settings-switch:disabled{opacity:.45;cursor:default}.settings-fields:disabled .settings-label{color:var(--sub)}
.service-options{display:flex;justify-content:flex-end;flex-wrap:wrap;gap:8px}.service-option{display:inline-flex;gap:6px;align-items:center;font-size:12px;padding:7px 9px;border:1px solid var(--line);border-radius:7px;cursor:pointer}.service-option input{margin:0;accent-color:#0f766e}.icon-options{display:flex;gap:6px}.icon-option{position:relative;cursor:pointer}.icon-option input{position:absolute;opacity:0;width:1px;height:1px}.icon-option-face{display:flex;align-items:center;gap:8px;min-height:38px;padding:0 12px;border:1px solid var(--line);border-radius:8px;font-size:13px;background:#fff}.icon-option input:checked+.icon-option-face{border-color:#75b4a9;color:#0c695d;background:#edf6f3}.icon-option input:focus-visible+.icon-option-face{outline:2px solid #0f766e;outline-offset:3px}.icon-option input:disabled+.icon-option-face{opacity:.5}.icon-preview{display:inline-flex;align-items:center;justify-content:center;width:22px;height:22px;box-sizing:border-box}.preview-donut{border:3px solid #0f766e;border-right-color:#dce5e1;border-radius:50%;transform:rotate(-35deg)}.preview-number{font-size:17px;letter-spacing:-.06em;font-weight:750;color:#0f766e}
.alert-service{margin-top:16px}.alert-service h3{display:flex;align-items:center;gap:10px;margin:0;padding:8px 0;font-size:13px;color:var(--tx)}.alert-service-state{font-weight:400;font-size:11px;color:var(--sub)}.alert-rule{display:grid;grid-template-columns:76px minmax(0,1fr) 36px;align-items:center;gap:10px 20px;border-top:1px solid var(--line);padding:12px 0}.alert-rule-label{font-size:13px}.alert-rule-controls{display:flex;align-items:center;gap:10px 20px;flex-wrap:wrap;font-size:12px;color:var(--sub)}.alert-threshold{display:inline-flex;align-items:center;gap:6px}.alert-threshold input{width:66px;text-align:center}.alert-recovery{display:inline-flex;align-items:center;gap:6px;cursor:pointer}.alert-recovery input{margin:0;accent-color:#0f766e}.alert-rule.rule-off .alert-rule-controls{opacity:.65}.settings-row input[name=poll_minutes]{width:92px}.settings-compact{padding:5px 8px;min-height:30px;font-size:12px;background:transparent}
.settings-feedback{display:flex;align-items:center;gap:10px 20px;flex-wrap:wrap;margin-top:16px}.settings-feedback .dash-message{margin:0;font-size:12px;flex:1;min-width:140px}.settings-info .dash-grid{margin-top:16px}
@media(max-width:620px){.alert-rule{grid-template-columns:minmax(0,1fr) 36px;gap:10px}.alert-rule-controls{grid-column:1/-1;grid-row:2}.alert-rule>.settings-switch{grid-column:2;grid-row:1}.settings-row{grid-template-columns:1fr;gap:10px}.settings-row>.settings-switch{justify-self:start}.settings-control,.service-options{justify-content:flex-start}.settings-control select{width:100%}.settings-row .dash-note{grid-column:auto}.settings-feedback .dash-message{flex-basis:100%}}
details>summary{cursor:pointer;font-weight:700;font-size:15px}details[open]>summary{margin-bottom:16px}.dash-status-row{padding:10px 0;border-bottom:1px solid var(--line)}.dash-status-row:last-child{border:0}.dash-status-title{display:flex;justify-content:space-between;gap:10px;align-items:center}.dash-status-title .chip{color:var(--sub)}.dash-status-row p{margin:4px 0;font-size:12px;color:var(--sub);overflow-wrap:anywhere}
.dash-history-chart{width:100%;height:250px;display:block}.dash-chart-wrap{position:relative;overflow:hidden;margin:14px 0 6px}.dash-chart-point{fill:transparent;stroke:none}.dash-chart-point:focus{outline:none}
.history-tooltip{position:absolute;z-index:2;width:280px;max-width:calc(100% - 16px);padding:12px 14px;border:1px solid #cbded8;border-radius:10px;background:#fff;box-shadow:0 4px 16px #20293820;pointer-events:none;color:var(--tx);font-size:12px;line-height:1.55;overflow-wrap:anywhere}.history-tooltip strong{display:block;font-size:19px;color:#0f766e;margin:2px 0 5px}.history-tooltip p{margin:2px 0}.history-tooltip .tooltip-time{color:var(--sub)}
.dash-forecast{font-size:13px}.dash-forecast p{margin:4px 0}
.pace-groups{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:24px}.pace-group{min-width:0}.pace-group h3{font-size:14px;font-weight:650;margin:0 0 16px}.pace-plan{border-left:1px solid var(--line);padding-left:24px}.pace-metrics,.pace-observation-metrics{display:grid;gap:12px;margin:0}.pace-metric{min-width:0}.pace-metric dt{font-size:12px;color:var(--sub);margin-bottom:7px}.pace-metric dd{margin:0;font-size:26px;line-height:1.3;font-weight:650;letter-spacing:-.03em;font-variant-numeric:tabular-nums}.pace-metric dd span{font-size:13px;font-weight:400;letter-spacing:0;margin-left:4px;color:var(--sub)}.pace-note{font-size:12px;color:var(--sub);overflow-wrap:anywhere}.pace-metric .pace-note{margin-top:6px}.pace-plan .pace-metric{display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:baseline;gap:4px 12px}.pace-plan .pace-metric dt{margin:0;word-break:keep-all}.pace-plan .pace-metric dd{font-size:24px;white-space:nowrap}.pace-plan .pace-metric .pace-note{grid-column:1/-1}.pace-plan>.pace-reduction{margin-top:12px;word-break:keep-all}.pace-forecast{border-bottom:1px solid var(--line);padding-bottom:18px;margin-bottom:20px}.pace-forecast-label{display:block;color:var(--sub);font-size:12px;margin-bottom:7px}.pace-estimate{display:flex;align-items:baseline;gap:6px 10px;flex-wrap:wrap;min-width:0}.pace-date{font-size:30px;font-weight:750;line-height:1.25;letter-spacing:-.03em;font-variant-numeric:tabular-nums}.pace-status{color:var(--sub);font-size:14px;font-weight:500}.risk .pace-status{color:#a33c3c}.pace-assumption{color:var(--sub);font-size:12px;padding-top:6px}.pace-empty{color:var(--sub);font-size:12px}
body.summary-view{max-width:640px;padding:18px}body.summary-view #token-snapshot,body.summary-view .dashboard-full{display:none}body.summary-view h1{font-size:21px}body.summary-view .cl{font-size:26px}
@media(max-width:620px){.dash-grid{grid-template-columns:1fr}.dash-form-grid{grid-template-columns:1fr}.dash-actions select{max-width:100%}.summary-note{display:none}.quick-toolbar{justify-content:flex-start}.quick-toolbar .dash-message{text-align:left}}
@media(max-width:620px){.history-controls{grid-template-columns:1fr}.history-controls label{display:grid;grid-template-columns:60px minmax(0,1fr)}.history-controls select{width:100%}.dash-chart-labels{font-size:11px}}
@media(max-width:620px){.pace-groups{grid-template-columns:1fr;gap:18px}.pace-plan{border-left:0;border-top:1px solid var(--line);padding:18px 0 0}}
`

const headerActionsHTML = `<div class="dash-actions quick-toolbar"><button type="button" data-action="refresh"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20 7v5h-5M4 17v-5h5M6 8a7 7 0 0 1 11-2l3 3M18 16a7 7 0 0 1-11 2l-3-3"/></svg>새로고침</button>` + themeControlHTML + `<button type="button" data-action="detail" id="detail-action"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M14 4h6v6M20 4l-9 9M10 5H5v14h14v-5"/></svg>상세 보기</button><a class="button subtle" href="/settings"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M4 7h16M4 17h16M8 4v6M16 14v6"/></svg>설정</a></div>`

const dashboardHTML = `
<div class="dashboard">
 <div id="app-update-banner" class="app-update-banner" hidden role="status"><p id="app-update-message"></p><button type="button" id="reload-app">화면 갱신</button></div>
 <section id="history-details" class="panel page-section dashboard-full" aria-labelledby="history-title"><div class="history-heading"><h2 id="history-title">잔여량 추이<span class="summary-note">이력 · 예상 소진</span></h2><details id="history-help" class="history-help"><summary aria-label="그래프 도움말" title="그래프 도움말">?</summary><div class="history-help-content"><p>기록 구간 확대를 켜면 위에는 선택 기간 전체와 확대 위치를, 아래에는 기록이 있는 구간을 크게 보여줍니다. 버튼을 다시 누르면 전체 기간 보기로 돌아갑니다.</p><p>각 시간 구간의 마지막 관측 잔량을 표시하며 확대 범위에 맞춰 간격을 조절합니다. 원본 기록은 보존하고 초기화 전후와 빈 구간은 잇지 않습니다. 호버·터치·탭 키로 상세를 확인하고 Esc로 닫습니다.</p><p>소진 예상은 원본 기록의 최근 소비 속도에 따른 추정입니다.</p></div></details></div>
  <div class="history-controls"><label for="history-agent">에이전트<select id="history-agent"><option value="claude">Claude</option><option value="codex">Codex</option><option value="antigravity">Antigravity</option></select></label><label for="history-window">한도<select id="history-window"><option value="short">단기 · 5시간</option><option value="week">주간</option></select></label><label for="history-days">기간<select id="history-days"><option value="1">최근 1일</option><option value="7" selected>최근 7일</option><option value="30">최근 30일</option></select></label></div>
  <div class="history-view-controls"><span id="history-view-label">전체 기간</span><button type="button" id="history-zoom" aria-pressed="true" aria-controls="history-chart history-overview" disabled>기록 구간 확대</button></div>
  <div id="history-overview" class="history-overview" hidden></div>
  <p id="history-message" class="dash-message" role="status" aria-live="polite">기록을 불러오고 있습니다.</p><div id="history-chart" class="dash-chart-wrap"></div>
 </section>
 <section id="usage-plan" class="panel page-section dashboard-full" aria-labelledby="usage-plan-title"><div class="history-heading"><h2 id="usage-plan-title">사용 전망<span id="usage-plan-context" class="summary-note"></span></h2><details id="usage-plan-help" class="history-help hover-help"><summary aria-label="사용 전망 도움말">?</summary><div class="history-help-content"><p>관측 사용량은 표시된 시작·종료 시각 사이의 잔여 한도 감소량입니다. 정수로 반올림해 저장한 잔량의 차이이며, 실제 토큰 수가 아닙니다.</p><p>현재 환산 속도는 관측 사용량을 24시간 기준으로 환산한 값이며, 지난 하루의 실제 사용량과 다릅니다. 초기화까지의 하루 예산은 남은 한도를 초기화까지 균등하게 나눠 쓰기 위한 하루 목표입니다.</p><p>%p는 전체 한도 기준의 변화량입니다. 잔량이 80%에서 70%가 되면 10%p를 사용한 것입니다. 초기화까지 하루 미만이면 시간당 예산도 표시합니다.</p><p>관측 구간은 초기화·출처 변화·잔량 회복 이후부터 계산하며 그래프 기간 필터와 독립적입니다. 소진 시점은 이 구간의 평균 사용 속도가 유지된다는 가정의 추정값입니다.</p></div></details></div>
  <div id="forecast-state" class="dash-forecast">예측에 필요한 기록을 확인하고 있습니다.</div>
 </section>
 <section id="settings" class="dashboard-full"><div id="settings-details" class="panel page-section">
  <form id="settings-form" novalidate><fieldset id="settings-fields" class="settings-fields" aria-label="앱 설정" disabled>
   <section class="settings-section" aria-labelledby="tray-settings-title"><h2 id="tray-settings-title">트레이</h2>
    <div class="settings-row"><span class="settings-label" id="visible-agents-label">표시할 서비스</span><div class="service-options" role="group" aria-labelledby="visible-agents-label"><label class="service-option"><input name="visible_claude" type="checkbox">Claude</label><label class="service-option"><input name="visible_codex" type="checkbox">Codex</label><label class="service-option"><input name="visible_antigravity" type="checkbox">Antigravity</label></div></div>
    <div class="settings-row"><span class="settings-label" id="icon-style-label">아이콘 모양</span><div class="settings-control"><div class="icon-options" role="radiogroup" aria-labelledby="icon-style-label"><label class="icon-option"><input name="icon_style" type="radio" value="donut"><span class="icon-option-face"><span class="icon-preview preview-donut" aria-hidden="true"></span>도넛</span></label><label class="icon-option"><input name="icon_style" type="radio" value="number"><span class="icon-option-face"><span class="icon-preview preview-number" aria-hidden="true">75</span>숫자</span></label></div></div></div>
    <div class="settings-row"><label class="settings-label" for="display-window">표시할 한도</label><div class="settings-control"><select name="display_window" id="display-window"><option value="week">주간 한도</option><option value="short">단기 한도</option><option value="urgent">잔량이 가장 적은 한도</option></select></div></div>
   </section>
   <section class="settings-section" aria-labelledby="alert-settings-title"><h2 id="alert-settings-title">알림</h2>
    <div class="settings-row"><label class="settings-label" for="alerts-enabled">잔여량 알림</label><div class="settings-control"><input id="alerts-enabled" name="alerts_enabled" class="settings-switch" type="checkbox" role="switch" aria-controls="notification-options"></div></div>
    <div class="settings-row"><label class="settings-label" for="alert-pause">알림 일시 중지</label><div class="settings-control"><select id="alert-pause" name="alert_pause"><option value="none">중지 안 함</option><option value="hour">1시간 동안</option><option value="today">오늘까지</option><option value="until" hidden>현재 중지 유지</option></select></div><p id="alert-pause-state" class="dash-note" hidden></p></div>
    <div id="notification-options"><div id="alert-rules"></div>
     <div class="settings-row"><label class="settings-label" for="notify-limits">한도 도달 시 알림</label><div class="settings-control"><input id="notify-limits" name="notify_limits" class="settings-switch" type="checkbox" role="switch"></div></div>
    </div>
   </section>
   <section class="settings-section" aria-labelledby="general-settings-title"><h2 id="general-settings-title">일반</h2>

    <div class="settings-row"><label class="settings-label" for="startup-enabled">로그인 시 실행</label><div class="settings-control"><button type="button" id="retry-startup" class="settings-compact" hidden>다시 확인</button><input id="startup-enabled" class="settings-switch" type="checkbox" role="switch" aria-describedby="startup-state" disabled></div><p id="startup-state" class="dash-note" role="status">자동 실행 상태를 확인하고 있습니다.</p></div>
    <div class="settings-row"><label class="settings-label" for="poll-minutes">수집 간격</label><div class="settings-control"><input id="poll-minutes" name="poll_minutes" type="number" min="1" max="60" step="any" inputmode="decimal" required aria-describedby="poll-help">분</div><p class="dash-note" id="poll-help">1~60분 · 최소 조회 간격: Claude 3분, Codex·Antigravity 2분</p></div>
   </section>
  </fieldset><div id="settings-feedback" class="settings-feedback" hidden><p id="settings-message" class="dash-message" role="status" aria-live="polite"></p><button type="button" id="retry-settings" class="settings-compact" hidden>실패한 변경 다시 시도</button></div></form>
 </div></section>
 <section id="system-details" class="panel page-section dashboard-full settings-info" aria-labelledby="system-title"><h2 id="system-title">앱 정보</h2><div class="dash-grid">
  <section aria-labelledby="status-title"><h2 id="status-title">연결 상태</h2><div id="agent-status"><p class="none">상태를 확인하고 있습니다.</p></div><p id="activity-state" class="dash-note"></p><p id="status-message" class="dash-message" role="status"></p></section>
  <section aria-labelledby="update-title"><h2 id="update-title">버전</h2><p id="update-state" class="dash-note">현재 버전 {{APP_VERSION}}</p><p id="auto-update-state" class="dash-note" role="status" aria-live="polite">자동 업데이트 상태를 확인하고 있습니다.</p><div class="dash-actions"><a class="button" id="release-link" href="{{RELEASES_URL}}" target="_blank" rel="noopener noreferrer">릴리스 보기<span aria-hidden="true">↗</span></a></div></section>
 </div></section>
</div>
`

func dashboardScriptTag() string { return "<script>" + dashboardJS + "</script>" }

const dashboardJS = `
(function () {
'use strict';
const $ = id => document.getElementById(id);
const names = {claude:'Claude', codex:'Codex', antigravity:'Antigravity'};
const ruleNames = {claude5h:'Claude · 5시간',claude7d:'Claude · 주간',codex5h:'Codex · 단기',codex7d:'Codex · 주간',antigravity:'Antigravity'};
const form = $('settings-form');
let savedSettings = null, savedStartup = null, startupLoading = false, settingsAgentState = null, settingsBusy = false, historyData = null, historyRequest = 0, historySelectionInitialized = false, historySelectionTouched = false, polling = false, mutationCount = 0, historyZoom = true;
const page = document.body.dataset.page || 'usage';
const compact = page==='usage' && new URLSearchParams(location.search).get('view') === 'summary';
document.body.classList.toggle('summary-view', compact);
if (!compact) { $('detail-action').hidden = true;document.querySelector('.quick-toolbar a[href="/settings"]').hidden=true; }
function migrateLegacyLink() {
 if(location.pathname!=='/')return;
 const target=({'#settings':'/settings','#token-details':'/tokens','#system-details':'/settings#system-details','#updates':'/settings#system-details'})[location.hash];
 if(target)location.replace(target);
}
migrateLegacyLink();window.addEventListener('hashchange',migrateLegacyLink);
const settingsQueue=new Map(),failedSettings=new Map(),numberTimers=new Map(),settingVersions=new Map(),settingSignatures=new Map();
let settingsSaving=false;
let pausedUntil=0,metadataBusy=false,reloadPending=false,refreshWatcher=0;
let themePreference=window.getUsageTrayTheme?window.getUsageTrayTheme():'system',themeBusy=false,themeRequest=0;
const initialInstance=document.querySelector('meta[name="usage-tray-instance"]')?.content || '';
const initialUI=document.querySelector('meta[name="usage-tray-ui"]')?.content || '';
let knownInstance=initialInstance,knownUI=initialUI;
function storageRead(key,fallback=null){try{return JSON.parse(localStorage.getItem(key)) ?? fallback;}catch(e){return fallback;}}
function storageWrite(key,value){try{localStorage.setItem(key,JSON.stringify(value));}catch(e){/* Preferences remain usable when browser storage is unavailable. */}}
function saveHistoryView(){storageWrite('usage-tray-history-view',{agent:$('history-agent').value,window:$('history-window').value,days:Number($('history-days').value),zoom:historyZoom});}
function restoreHistoryView(){
 const saved=storageRead('usage-tray-history-view');if(!saved || typeof saved!=='object')return;
 if(Object.prototype.hasOwnProperty.call(names,saved.agent) && ['short','week'].includes(saved.window)){$('history-agent').value=saved.agent;$('history-window').value=saved.window;historySelectionTouched=true;historySelectionInitialized=true;}
 if([1,7,30].includes(saved.days))$('history-days').value=String(saved.days);if(typeof saved.zoom==='boolean')historyZoom=saved.zoom;
}
restoreHistoryView();
function showReloadNotice(text){$('app-update-message').textContent=text;$('app-update-banner').hidden=false;$('reload-app').disabled=hasPendingSettings() || settingsBusy || mutationCount>0;}
async function checkAppMetadata(){
 if(metadataBusy || document.hidden)return;metadataBusy=true;const themeAtStart=themeRequest;
 try{
  const data=await api('/api/meta');
  if(!data || typeof data.instance_id!=='string' || typeof data.ui_revision!=='string')return;
  if(!knownInstance){knownInstance=data.instance_id;knownUI=data.ui_revision;}
  if(data.instance_id!==knownInstance || data.ui_revision!==knownUI){reloadPending=true;if(hasPendingSettings() || settingsBusy || mutationCount)showReloadNotice('앱이 업데이트되었습니다. 설정 반영이 끝나면 화면을 갱신해 주세요.');else location.reload();}
  if(!themeBusy && themeAtStart===themeRequest)adoptTheme(data.theme || 'system');
 }catch(e){/* A restart temporarily closes the local server; retry on the next poll. */}
 finally{metadataBusy=false;}
}
$('reload-app').addEventListener('click',()=>{if(!hasPendingSettings() && !settingsBusy && !mutationCount)location.reload();});
function message(id, text, error) { const e=$(id); e.textContent=text; e.classList.toggle('error',!!error); if(id==='settings-message')$('settings-feedback').hidden=!error; }
function textNode(tag, text, className) { const e=document.createElement(tag); e.textContent=text; if(className)e.className=className; return e; }
const tokenTooltip=textNode('div','','token-tooltip');tokenTooltip.id='token-tooltip';tokenTooltip.setAttribute('role','tooltip');tokenTooltip.hidden=true;document.body.append(tokenTooltip);
let inspectedTokenBar=null,pinnedTokenBar=null;
function dismissTokenDetails(){if(inspectedTokenBar){inspectedTokenBar.classList.remove('is-inspected');inspectedTokenBar.removeAttribute('aria-describedby');}inspectedTokenBar=null;pinnedTokenBar=null;tokenTooltip.hidden=true;}
function showTokenDetails(bar){
 if(inspectedTokenBar!==bar)dismissTokenDetails();inspectedTokenBar=bar;
 const data=bar.dataset,exact=value=>BigInt(value || '0').toLocaleString('ko-KR');
 tokenTooltip.replaceChildren(textNode('p',data.day+' · '+data.agent,'token-tooltip-title'));
 const total=textNode('strong',exact(data.total),'token-tooltip-total');total.append(textNode('span','토큰'));tokenTooltip.append(total);
 const details=document.createElement('dl'),add=(label,value)=>details.append(textNode('dt',label),textNode('dd',exact(value)));
 if(data.agent==='Codex'){add('입력 (캐시 포함)',data.input);add('출력',data.output);add('입력 중 캐시 읽기',data.cacheRead);}
 else {add('일반 입력',data.input);add('캐시 읽기',data.cacheRead);add('캐시 쓰기',data.cacheCreate);add('출력',data.output);}
 add('요청',data.requests);tokenTooltip.append(details);tokenTooltip.hidden=false;
 bar.classList.add('is-inspected');bar.setAttribute('aria-describedby',tokenTooltip.id);
 positionTokenDetails(bar);
}
function positionTokenDetails(bar){
 const rect=bar.getBoundingClientRect(),width=tokenTooltip.offsetWidth,height=tokenTooltip.offsetHeight;
 tokenTooltip.style.left=Math.max(12,Math.min(window.innerWidth-width-12,rect.left+rect.width/2-width/2))+'px';
 const above=rect.top-height-10;tokenTooltip.style.top=Math.max(12,above>=12?above:Math.min(window.innerHeight-height-12,rect.bottom+10))+'px';
}
function tokenBar(target){return target instanceof Element?target.closest('#token-snapshot .ch-col'):null;}
document.addEventListener('pointerover',event=>{const bar=tokenBar(event.target);if(bar && event.pointerType!=='touch' && !bar.contains(event.relatedTarget))showTokenDetails(bar);});
document.addEventListener('pointerout',event=>{const bar=tokenBar(event.target);if(bar && !bar.contains(event.relatedTarget) && inspectedTokenBar===bar && pinnedTokenBar!==bar && !bar.contains(document.activeElement))dismissTokenDetails();});
document.addEventListener('focusin',event=>{const bar=tokenBar(event.target);if(bar)showTokenDetails(bar);});
document.addEventListener('focusout',event=>{const bar=tokenBar(event.target);if(bar && !bar.contains(event.relatedTarget) && pinnedTokenBar!==bar)dismissTokenDetails();});
document.addEventListener('pointerdown',event=>{
 const bar=tokenBar(event.target);if(!bar){dismissTokenDetails();return;}
 if(event.pointerType==='touch'){event.preventDefault();if(pinnedTokenBar===bar){dismissTokenDetails();return;}showTokenDetails(bar);pinnedTokenBar=bar;}
});
document.addEventListener('pointercancel',dismissTokenDetails);
document.addEventListener('keydown',event=>{if(event.key==='Escape')dismissTokenDetails();});
document.addEventListener('scroll',()=>{
 const bar=inspectedTokenBar;
 if(bar && !pinnedTokenBar && bar.contains(document.activeElement)){
  const rect=bar.getBoundingClientRect();
  if(rect.bottom>0 && rect.top<window.innerHeight)positionTokenDetails(bar);else dismissTokenDetails();
 }else dismissTokenDetails();
},true);window.addEventListener('resize',dismissTokenDetails);
function timeLabel(value) { const d=new Date(value); return value && !isNaN(d) ? d.toLocaleString('ko-KR') : '아직 성공한 기록 없음'; }
const themeLabels={system:'시스템 설정',light:'밝게',dark:'어둡게'};
const themeChoices=Array.from(document.querySelectorAll('[data-theme-choice]'));
function renderThemeControl(){
 const selected=themeChoices.find(button=>button.dataset.themeChoice===themePreference),trigger=$('theme-toggle');
 trigger.replaceChildren(selected.querySelector('svg').cloneNode(true));trigger.setAttribute('aria-label','테마: '+themeLabels[themePreference]);trigger.title='테마: '+themeLabels[themePreference];trigger.disabled=themeBusy;
 for(const choice of themeChoices){choice.setAttribute('aria-checked',String(choice.dataset.themeChoice===themePreference));choice.disabled=themeBusy;}
}
function adoptTheme(value){themePreference=Object.prototype.hasOwnProperty.call(themeLabels,value)?value:'system';if(window.applyUsageTrayTheme)window.applyUsageTrayTheme(themePreference);renderThemeControl();}
function closeThemeMenu(restoreFocus=false){$('theme-menu').hidden=true;$('theme-toggle').setAttribute('aria-expanded','false');if(restoreFocus)$('theme-toggle').focus();}
function openThemeMenu(last=false){if(themeBusy)return;$('theme-menu').hidden=false;$('theme-toggle').setAttribute('aria-expanded','true');(last?themeChoices[themeChoices.length-1]:themeChoices.find(button=>button.dataset.themeChoice===themePreference)).focus();}
async function saveTheme(value){
 if(themeBusy)return;closeThemeMenu(true);if(value===themePreference)return;
 themeBusy=true;const request=++themeRequest;mutationCount++;renderThemeControl();
 try{
  const result=await api('/api/theme','PUT',{theme:value});
  if(request===themeRequest)adoptTheme(result.theme);
  else{const beforeRead=themeRequest,latest=await api('/api/theme');if(beforeRead===themeRequest)adoptTheme(latest.theme);}
  message('action-message','');
 }
 catch(e){message('action-message','테마를 저장하지 못했습니다. 다시 선택해 주세요.',true);}
 finally{themeBusy=false;mutationCount--;renderThemeControl();if(document.activeElement===document.body || document.activeElement===$('theme-toggle'))$('theme-toggle').focus();}
}
$('theme-toggle').addEventListener('click',()=>{if($('theme-menu').hidden)openThemeMenu();else closeThemeMenu();});
$('theme-toggle').addEventListener('keydown',event=>{if(event.key==='ArrowDown' || event.key==='ArrowUp'){event.preventDefault();openThemeMenu(event.key==='ArrowUp');}});
for(const choice of themeChoices)choice.addEventListener('click',()=>saveTheme(choice.dataset.themeChoice));
$('theme-menu').addEventListener('keydown',event=>{
 const index=themeChoices.indexOf(document.activeElement);
 if(event.key==='Escape'){event.preventDefault();closeThemeMenu(true);}
 else if(['ArrowDown','ArrowUp','Home','End'].includes(event.key)){event.preventDefault();const next=event.key==='Home'?0:event.key==='End'?themeChoices.length-1:(index+(event.key==='ArrowDown'?1:-1)+themeChoices.length)%themeChoices.length;themeChoices[next].focus();}
 else if(event.key==='Tab')setTimeout(()=>closeThemeMenu(),0);
});
document.addEventListener('pointerdown',event=>{if(!$('theme-control').contains(event.target))closeThemeMenu();});
window.addEventListener('usage-tray-theme-change',()=>{themeRequest++;themePreference=window.getUsageTrayTheme();renderThemeControl();});
renderThemeControl();
async function api(path, method, body) {
 const options={cache:'no-store',headers:{'Accept':'application/json'}};
 if(method) { options.method=method; options.keepalive=true; options.headers['X-Usage-Tray']='1'; options.headers['Content-Type']='application/json'; if(body!==undefined)options.body=JSON.stringify(body); }
 const res=await fetch(path,options);
 if(!res.ok) throw new Error(res.status===400 ? '입력값을 확인해 주세요.' : '요청을 완료하지 못했습니다. 잠시 후 다시 시도해 주세요.');
 return res.json();
}
function field(name) { return form.elements.namedItem(name); }
function setChecked(name,value) { field(name).checked=!!value; }
function drawRules(settings) {
 $('alert-rules').replaceChildren();
 for(const [agent,name] of Object.entries(names)){
  const group=textNode('section','','alert-service');group.dataset.agent=agent;const heading=textNode('h3',name);heading.id='alert-service-'+agent;heading.append(textNode('span','','alert-service-state'));group.setAttribute('aria-labelledby',heading.id);group.append(heading);$('alert-rules').append(group);
 }
 for(const [key,label] of Object.entries(ruleNames)) {
  const rule=settings.alert_rules[key] || {enabled:false,threshold:20,recovery:false};
  const row=textNode('div','','alert-rule');row.dataset.rule=key;
  const toggle=document.createElement('input');toggle.type='checkbox';toggle.name=key+'_enabled';toggle.id='rule-'+key;toggle.checked=!!rule.enabled;toggle.className='settings-switch';toggle.setAttribute('role','switch');toggle.setAttribute('aria-label',label+' 알림');
  const title=textNode('label',label.includes('·')?label.split('·')[1].trim():'모델 한도','alert-rule-label');title.htmlFor=toggle.id;row.append(title);
  const controls=textNode('div','','alert-rule-controls'),threshold=textNode('label','잔량','alert-threshold'),input=document.createElement('input');
  input.name=key+'_threshold';input.type='number';input.min='1';input.max='99';input.step='1';input.required=true;input.value=rule.threshold;input.setAttribute('aria-label',label+' 남은량 기준 퍼센트');threshold.append(input,document.createTextNode('% 미만'));
  const recovery=textNode('label','','alert-recovery'),check=document.createElement('input');check.name=key+'_recovery';check.type='checkbox';check.checked=!!rule.recovery;check.setAttribute('aria-label',label+' 회복 알림');recovery.append(check,document.createTextNode('회복 시 알림'));controls.append(threshold,recovery);row.append(controls,toggle);const agent=key.startsWith('claude')?'claude':key.startsWith('codex')?'codex':'antigravity';$('alert-rules').querySelector('[data-agent="'+agent+'"]').append(row);
 }
 arrangeAlertRules();
}
function arrangeAlertRules() {
 const groups=Array.from($('alert-rules').querySelectorAll('.alert-service'));
 for(const group of groups){const usage=settingsAgentState && settingsAgentState[group.dataset.agent],status=group.querySelector('.alert-service-state');status.textContent=usage && !usage.available?'연결 안 됨':'';}
 if(settingsAgentState){
  groups.sort((a,b)=>Number(!!settingsAgentState[b.dataset.agent]?.available)-Number(!!settingsAgentState[a.dataset.agent]?.available));
  for(const group of groups)$('alert-rules').append(group);
 }
}
async function loadSettingsContext(){try{settingsAgentState=await api('/state.json');arrangeAlertRules();}catch(e){/* Keep all editable rules available when connection status is unknown. */}}
function fillSettings(settings) {
 savedSettings=settings;for(const radio of form.querySelectorAll('[name="icon_style"]'))radio.checked=radio.value===settings.icon_style;
 field('display_window').value=settings.display_window;field('poll_minutes').value=settings.poll_seconds/60;
 pausedUntil=settings.alerts_paused_until || 0;const paused=pausedUntil>Date.now()/1000;field('alert_pause').querySelector('[value="until"]').hidden=!paused;field('alert_pause').value=paused?'until':'none';renderPauseState();
 for(const key of Object.keys(names))setChecked('visible_'+key,settings.visible_agents[key]);
 setChecked('alerts_enabled',settings.alerts_enabled);setChecked('notify_limits',settings.notify_limits);drawRules(settings);
}
function hasPendingSettings(){return settingsSaving || settingsQueue.size>0 || numberTimers.size>0;}
function updateSettingsControls(){
 $('settings-fields').disabled=!savedSettings;form.setAttribute('aria-busy',String(hasPendingSettings()));
 $('startup-enabled').disabled=startupLoading || !savedStartup || !savedStartup.supported;
 $('retry-startup').disabled=startupLoading;
 $('notification-options').hidden=!field('alerts_enabled').checked && !form.querySelector('#notification-options :invalid');
 for(const row of form.querySelectorAll('.alert-rule'))row.classList.toggle('rule-off',!field(row.dataset.rule+'_enabled').checked);
 $('retry-settings').hidden=failedSettings.size===0;
 if(reloadPending)showReloadNotice(hasPendingSettings()?'앱이 업데이트되었습니다. 설정 반영이 끝나면 화면을 갱신해 주세요.':'앱이 업데이트되었습니다. 화면을 갱신해 주세요.');
}
function settingsStatus(){
 const invalid=form.querySelector('input:invalid');
 if(failedSettings.size)message('settings-message','일부 변경을 반영하지 못했습니다. 실패한 변경을 다시 시도해 주세요.',true);
 else if(invalid)message('settings-message','입력 범위를 확인해 주세요. 올바른 숫자를 입력하면 자동으로 반영합니다.',true);
 else message('settings-message','');
 updateSettingsControls();
}
function renderPauseState(){const paused=pausedUntil>Date.now()/1000;$('alert-pause-state').hidden=!paused;$('alert-pause-state').textContent=paused?new Date(pausedUntil*1000).toLocaleString('ko-KR')+'까지 중지한 뒤 자동으로 알림을 재개합니다.':'';}
function settingPatch(input){
 const name=input.name;
 if(input.id==='startup-enabled')return {key:'startup',patch:{enabled:input.checked}};
 if(name==='poll_minutes')return {key:'poll_seconds',patch:{poll_seconds:Math.round(Number(input.value)*60)}};
 if(name.startsWith('visible_')){const agent=name.slice(8);return {key:'visible_agents.'+agent,patch:{visible_agents:{[agent]:input.checked}}};}
 for(const key of Object.keys(ruleNames))if(name.startsWith(key+'_')){const prop=name.slice(key.length+1);return {key:'alert_rules.'+key+'.'+prop,patch:{alert_rules:{[key]:{[prop]:prop==='threshold'?Number(input.value):input.checked}}}};}
 if(name==='alert_pause'){
  if(input.value==='none')pausedUntil=0;else if(input.value==='hour')pausedUntil=Math.floor(Date.now()/1000)+3600;else if(input.value==='today'){const end=new Date();end.setHours(24,0,0,0);pausedUntil=Math.floor(end.getTime()/1000);}
  renderPauseState();return {key:'alerts_paused_until',patch:{alerts_paused_until:pausedUntil}};
 }
 if(['icon_style','display_window','alerts_enabled','notify_limits'].includes(name))return {key:name,patch:{[name]:input.type==='checkbox'?input.checked:input.value}};
 return null;
}
function queueSetting(action){
 const signature=JSON.stringify(action.patch);if(settingSignatures.get(action.key)===signature && !failedSettings.has(action.key)){settingsStatus();return;}settingSignatures.set(action.key,signature);
 action.version=(settingVersions.get(action.key)||0)+1;settingVersions.set(action.key,action.version);
 failedSettings.delete(action.key);settingsQueue.set(action.key,action);settingsStatus();drainSettings();
}
async function drainSettings(){
 if(settingsSaving)return;settingsSaving=true;mutationCount++;settingsStatus();
 try{while(settingsQueue.size){
  const [key,action]=settingsQueue.entries().next().value;settingsQueue.delete(key);
  try{
   if(key==='startup'){
    const result=await api('/api/startup','POST',action.patch);
    if(!result || !result.supported || result.enabled!==action.patch.enabled)throw new Error('startup state mismatch');
    savedStartup={enabled:result.enabled,supported:result.supported};$('startup-state').hidden=true;
   }else savedSettings=await api('/api/settings','PATCH',action.patch);
   if(settingVersions.get(key)===action.version)failedSettings.delete(key);
  }catch(e){if(settingVersions.get(key)===action.version)failedSettings.set(key,action);}
  settingsStatus();
 }}finally{settingsSaving=false;mutationCount--;settingsStatus();}
}
function settingsInput(event){
 const input=event.target;if(!savedSettings || !(input instanceof HTMLInputElement || input instanceof HTMLSelectElement))return;
 if(input.type==='number'){
  clearTimeout(numberTimers.get(input.name));numberTimers.delete(input.name);
  const action=settingPatch(input);if(!action)return;
  // An invalid edit supersedes a queued earlier number, but never reaches the API.
  if(!input.checkValidity() || !input.value.trim()){
   settingsQueue.delete(action.key);failedSettings.delete(action.key);settingSignatures.delete(action.key);settingVersions.set(action.key,(settingVersions.get(action.key)||0)+1);
   input.setAttribute('aria-invalid','true');settingsStatus();return;
  }
  input.removeAttribute('aria-invalid');
  if(event.type==='input'){numberTimers.set(input.name,setTimeout(()=>{numberTimers.delete(input.name);queueSetting(settingPatch(input));},350));settingsStatus();}
  else queueSetting(action);
 }else if(event.type==='change'){const action=settingPatch(input);if(action)queueSetting(action);}
}
form.addEventListener('input',settingsInput);form.addEventListener('change',settingsInput);
form.addEventListener('focusout',event=>{if(event.target.type==='number')settingsInput({target:event.target,type:'change'});});
form.addEventListener('submit',event=>event.preventDefault());
function flushNumberChanges(){for(const name of Array.from(numberTimers.keys()))settingsInput({target:field(name),type:'change'});}
document.addEventListener('click',async event=>{
 const link=event.target.closest && event.target.closest('a[href]');
 if(!link || event.defaultPrevented || event.button!==0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || link.target==='_blank')return;
 const url=new URL(link.href,location.href);if(url.origin!==location.origin)return;
 flushNumberChanges();if(!hasPendingSettings())return;event.preventDefault();
 while(hasPendingSettings())await new Promise(resolve=>setTimeout(resolve,20));
 location.assign(url.href);
});
document.addEventListener('visibilitychange',()=>{if(document.hidden)flushNumberChanges();});
// Teardown flushes valid numbers through the same queue. Each started request
// uses keepalive; never bypass serialization with a second closing request.
function flushClosingSettings(){flushNumberChanges();}
window.addEventListener('pagehide',flushClosingSettings);
$('retry-settings').addEventListener('click',()=>{for(const action of Array.from(failedSettings.values()))queueSetting({...action});});
async function loadSettings(){
 if(settingsBusy)return;settingsBusy=true;message('settings-message','설정을 불러오고 있습니다.');
 try{fillSettings(await api('/api/settings'));settingsStatus();}
 catch(e){message('settings-message','설정을 불러오지 못했습니다. 페이지를 다시 열어 주세요.',true);}
 finally{settingsBusy=false;updateSettingsControls();}
}
function renderStartup(data){
 if(typeof data.enabled!=='boolean' || typeof data.supported!=='boolean')throw new Error('invalid startup state');
 savedStartup={enabled:data.enabled,supported:data.supported};$('startup-enabled').checked=data.enabled;
 $('startup-state').textContent=data.supported?'':'이 운영체제에서는 자동 실행 설정을 지원하지 않습니다.';$('startup-state').hidden=!!data.supported;$('retry-startup').hidden=true;
}
async function loadStartup(){if(startupLoading)return;startupLoading=true;updateSettingsControls();try{renderStartup(await api('/api/startup'));}catch(e){$('startup-state').hidden=false;$('startup-state').textContent='자동 실행 상태를 불러오지 못했습니다.';$('retry-startup').hidden=false;}finally{startupLoading=false;updateSettingsControls();}}
$('retry-startup').addEventListener('click',loadStartup);
async function loadAutomaticUpdateStatus(){
 try{
  const data=await api('/api/auto-update'),labels={idle:'자동 업데이트 대기 중',checking:'새 버전 확인 중',downloading:'업데이트 다운로드 중',installing:'업데이트 적용 중',error:'자동 업데이트 실패',disabled:'자동 업데이트 사용 안 함'};
  const checked=data.checked_at && !data.checked_at.startsWith('0001-'),next=data.next_check_at && !data.next_check_at.startsWith('0001-');
  const text=!data.supported?'이 실행 환경에서는 자동 업데이트를 지원하지 않습니다. 릴리스에서 최신 버전을 확인해 주세요.':(labels[data.state] || '자동 업데이트 상태 확인 중')+(data.version?' · '+data.version:'')+(checked?' · 최근 확인 '+timeLabel(data.checked_at):'')+(data.error?' · '+data.error:'')+(next?' · 다음 확인 '+timeLabel(data.next_check_at):'');
  message('auto-update-state',text,data.state==='error');
 }catch(e){message('auto-update-state','자동 업데이트 상태를 확인하지 못했습니다. 잠시 후 다시 확인합니다.',true);}
}
async function loadStatus() {
 try {
  const data=await api('/api/status'), container=$('agent-status');container.replaceChildren();
  for(const agent of data.agents || []) {
   const row=document.createElement('div');row.className='dash-status-row';row.dataset.agent=agent.name.toLowerCase();
   const title=document.createElement('div');title.className='dash-status-title';title.append(textNode('strong',agent.name),textNode('span',agent.label || '상태 확인 중','chip'+(['error','auth_error'].includes(agent.status)?' bad':'')));row.append(title);
   row.append(textNode('p','갱신 '+timeLabel(agent.last_success)+(agent.last_success && agent.age_minutes>=0?' · '+agent.age_minutes+'분 전':'')+(agent.source?' · '+agent.source:'')));
   if(agent.error)row.append(textNode('p',agent.error));
   if(agent.next_query_at && Date.parse(agent.next_query_at)>Date.now())row.append(textNode('p','다음 조회 '+timeLabel(agent.next_query_at)));
   container.append(row);
  }
  if(!container.children.length)container.append(textNode('p','연결된 에이전트가 없습니다.','none'));
  $('activity-state').textContent=[data.refreshing?'한도 갱신 중':'',data.scanning?'로컬 토큰 집계 중':''].filter(Boolean).join(' · ');message('status-message','');
  return data;
 } catch(e) { message('status-message','연결 상태를 확인하지 못했습니다. 표시된 정보는 직전 조회 결과입니다.',true); }
}
function refreshBusy(busy){document.querySelectorAll('[data-action="refresh"]').forEach(button=>button.disabled=busy);}
async function watchRefresh(expectedID){
 const watcher=++refreshWatcher,start=Date.now();refreshBusy(true);
 try{
  while(watcher===refreshWatcher && Date.now()-start<120000){
   const data=await loadStatus();if(watcher!==refreshWatcher)return;if(!data)throw new Error('수집 상태를 확인할 수 없습니다. 앱 연결을 확인해 주세요.');
   const job=data && data.refresh;
   if(job && (!expectedID || job.id>=expectedID)){
    const waiting=job.phase==='waiting',failed=job.phase==='failed';
    const next=waiting && job.next_at?' · 다음 조회 '+timeLabel(job.next_at):'';
    message('action-message',(job.message || (job.phase==='running'?'사용량을 갱신하고 있습니다.':waiting?'다음 조회를 기다리고 있습니다.':'갱신을 마쳤습니다.'))+next,failed);
    if(['completed','waiting','failed'].includes(job.phase) && !data.scanning){await refreshSnapshots();if(page==='usage' && !compact)await loadHistory();return;}
   }
   await new Promise(resolve=>setTimeout(resolve,1000));
  }
  if(watcher===refreshWatcher)message('action-message','수집이 계속 진행 중입니다. 마지막 갱신 시각을 확인해 주세요.');
 }finally{if(watcher===refreshWatcher)refreshBusy(false);}
}
document.querySelectorAll('[data-action]').forEach(button=>button.addEventListener('click',async()=>{
 const action=button.dataset.action;button.disabled=true;mutationCount++;
 try { const result=await api('/api/actions','POST',{action});message('action-message',action==='refresh'?'갱신 요청을 접수했습니다.':'상세 보기를 열었습니다.');if(action==='refresh')await watchRefresh(result.refresh && result.refresh.id); }
 catch(e) { message('action-message',e.message,true); }
 finally { button.disabled=false;mutationCount--; }
}));
function svgElement(tag,attrs) { const e=document.createElementNS('http://www.w3.org/2000/svg',tag);for(const [k,v] of Object.entries(attrs))e.setAttribute(k,String(v));return e; }
let dismissHistoryDetails=()=>{};
const planHelp=$('usage-plan-help'),planHelpTrigger=planHelp.querySelector('summary');
planHelp.addEventListener('pointerenter',event=>{if(event.pointerType!=='touch')planHelp.open=true;});
planHelp.addEventListener('pointerleave',()=>{planHelp.open=false;});
planHelpTrigger.addEventListener('click',event=>event.preventDefault());
planHelpTrigger.addEventListener('focus',()=>{if(planHelpTrigger.matches(':focus-visible'))planHelp.open=true;});
planHelpTrigger.addEventListener('blur',()=>{if(!planHelp.matches(':hover'))planHelp.open=false;});
document.addEventListener('keydown',event=>{if(event.key==='Escape'){dismissHistoryDetails();$('history-help').open=false;planHelp.open=false;}});
document.addEventListener('pointerdown',event=>{if(!$('history-chart').contains(event.target))dismissHistoryDetails();if(!$('history-help').contains(event.target))$('history-help').open=false;});
// Pure display aggregation: never changes stored samples or forecast inputs.
function historyInterval(span,target=48) {
 const steps=[1,2,5,10,15,30,60,120,240,360,720,1440,2880,10080,20160,43200].map(minutes=>minutes*60000);
 return steps.find(step=>step>=span/target) || steps[steps.length-1];
}
function historyRange(days,end) {
 const duration=([1,7,30].includes(Number(days))?Number(days):7)*86400000;
 return {start:end-duration,end,interval:historyInterval(duration)};
}
function historyZoomRange(samples,full) {
 if(samples.length<2)return full;
 const first=new Date(samples[0].at).getTime(),last=new Date(samples[samples.length-1].at).getTime();
 if(last<=first)return full;
 const padding=Math.max(60000,(last-first)*.04);
 const start=Math.max(full.start,first-padding),end=Math.min(full.end,last+padding);
 return {start,end,interval:historyInterval(end-start)};
}
function historyCycleChanged(a,b) {return a.reset_at!==b.reset_at || a.window_seconds!==b.window_seconds;}
function historyBins(samples,interval) {
 const bins=[];let previous=null,segment=0;
 for(const sample of samples){
  if(previous && historyCycleChanged(previous,sample))segment++;
  const bucket=Math.floor(new Date(sample.at).getTime()/interval),last=bins[bins.length-1];
  if(last && last.bucket===bucket && last.segment===segment){last.s=sample;last.count++;}
  else bins.push({s:sample,bucket,segment,count:1});
  previous=sample;
 }
 return bins;
}
function historyIntervalLabel(interval) {
 const minutes=interval/60000;return minutes<60?minutes+'분':minutes<1440?minutes/60+'시간':minutes/1440+'일';
}
function historyShortDate(ms,includeDate=true) {
 const d=new Date(ms),clock=String(d.getHours()).padStart(2,'0')+':'+String(d.getMinutes()).padStart(2,'0');
 return (includeDate?(d.getMonth()+1)+'/'+d.getDate()+' ':'')+clock;
}
function forecastResetLead(forecast) {
 const gap=forecast.reset_at*1000-Date.parse(forecast.estimated_exhaustion);
 if(!forecast.exhausts_before_reset || !Number.isFinite(gap) || gap<=0)return '';
 const minutes=Math.floor(gap/60000),days=Math.floor(minutes/1440),hours=Math.floor(minutes%1440/60);
 const duration=days?days+'일'+(hours?' '+hours+'시간':''):hours?hours+'시간'+(minutes%60?' '+minutes%60+'분':''):minutes?minutes+'분':'1분 미만';
 return '초기화보다 약 '+duration+' 먼저 소진 예상';
}
function usagePaceAdvice(forecast) {
 if(!forecast || !forecast.ready || !forecast.exhausts_before_reset || !Number.isFinite(forecast.daily_budget) || forecast.daily_budget<0 || !Number.isFinite(forecast.rate_per_day) || forecast.rate_per_day<=0)return null;
 return {budget:forecast.daily_budget,rate:forecast.rate_per_day,reduction:Math.max(0,(1-forecast.daily_budget/forecast.rate_per_day)*100)};
}
function quotaPaceLabel(value) {return value>0 && value<.1?'0.1%p 미만':value.toFixed(1)+'%p';}
function attachHistoryDetails(chart,svg,points,dots,range) {
 const tooltip=textNode('div','','history-tooltip');tooltip.id='history-tooltip';tooltip.setAttribute('role','tooltip');tooltip.hidden=true;chart.append(tooltip);
 const guide=svgElement('line',{y1:20,y2:180,stroke:'#80b5a9','stroke-dasharray':'4 4','pointer-events':'none'});
 const marker=svgElement('circle',{r:5,fill:'#fff',stroke:'#0f766e','stroke-width':2,'pointer-events':'none',class:'history-selected-point'});
 svg.append(guide,marker);
 let focused=null;
 function hide(){tooltip.hidden=true;guide.style.display='none';marker.style.display='none';}
 dismissHistoryDetails=hide;hide();
 function show(p){
  const matrix=svg.getScreenCTM();if(!matrix)return;
  const s=p.s,source=({live:'실시간',api:'실시간',cache:'캐시',local:'로컬 기록'})[s.source] || s.source || '미확인';
  tooltip.replaceChildren(textNode('p','마지막 관측 · '+timeLabel(s.at),'tooltip-time'),textNode('strong',s.left+'% 남음'),textNode('p','출처 · '+source),textNode('p','초기화 · '+(s.reset_at>0?timeLabel(s.reset_at*1000):'미확인')));
  tooltip.hidden=false;guide.style.display='';marker.style.display='';guide.setAttribute('x1',p.x);guide.setAttribute('x2',p.x);marker.setAttribute('cx',p.x);marker.setAttribute('cy',p.y);
  const position=new DOMPoint(p.x,p.y).matrixTransform(matrix),bounds=chart.getBoundingClientRect();
  const x=position.x-bounds.left,y=position.y-bounds.top;
  tooltip.style.left=Math.max(8,Math.min(x+14,chart.clientWidth-tooltip.offsetWidth-8))+'px';
  const above=y-tooltip.offsetHeight-14;
  tooltip.style.top=Math.max(8,Math.min(above>=8?above:y+14,chart.clientHeight-tooltip.offsetHeight-8))+'px';
 }
 function inspect(event){
  const matrix=svg.getScreenCTM();if(!matrix)return;
  const pos=new DOMPoint(event.clientX,event.clientY).matrixTransform(matrix.inverse());
  if(pos.x<48 || pos.x>range.width-18 || pos.y<20 || pos.y>180){hide();return;}
  const at=range.start+(pos.x-48)/(range.width-66)*(range.end-range.start),bucket=Math.floor(at/range.interval);
  const candidates=points.filter(p=>p.bucket===bucket);
  if(!candidates.length){hide();return;}
  const nearest=candidates.reduce((best,p)=>Math.abs(p.x-pos.x)<Math.abs(best.x-pos.x)?p:best,candidates[0]);
  show(nearest);
 }
 svg.addEventListener('pointermove',event=>{if(event.pointerType!=='touch')inspect(event);});
 svg.addEventListener('pointerdown',inspect);
 svg.addEventListener('pointerleave',event=>{if(event.pointerType==='touch')return;if(focused)show(focused);else hide();});
 for(const {dot,point} of dots){
  dot.setAttribute('aria-describedby',tooltip.id);
  dot.addEventListener('focus',()=>{focused=point;show(point);});
  dot.addEventListener('blur',()=>{focused=null;hide();});
 }
}
function drawHistoryOverview(samples,full,zoom,days) {
 const container=$('history-overview');container.hidden=false;container.replaceChildren();
 const width=Math.max(200,container.clientWidth-26),height=76;
 const x=at=>4+(at-full.start)/(full.end-full.start)*(width-8);
 const svg=svgElement('svg',{viewBox:'0 0 '+width+' '+height,role:'img','aria-label':'최근 '+days+'일 전체 기간. 색칠된 영역이 아래 확대 구간입니다.','data-start':full.start,'data-end':full.end});
 svg.append(svgElement('rect',{x:4,y:4,width:width-8,height:68,rx:4,fill:'#edf1f4'}));
 const selected=svgElement('rect',{x:x(zoom.start),y:4,width:Math.max(.5,x(zoom.end)-x(zoom.start)),height:68,fill:'#d2ebe3',stroke:'#69a995','stroke-width':1,class:'history-zoom-window','data-start':zoom.start,'data-end':zoom.end});svg.append(selected);
 const points=historyBins(samples,full.interval).map(bin=>({...bin,x:x(new Date(bin.s.at).getTime()),y:68-bin.s.left*.6}));let segment=[];
 function line(){if(segment.length>1)svg.append(svgElement('polyline',{points:segment.map(p=>p.x+','+p.y).join(' '),fill:'none',stroke:'#0f766e','stroke-width':1.2}));segment=[];}
 points.forEach((p,i)=>{if(i&&(p.segment!==points[i-1].segment||p.bucket-points[i-1].bucket>1))line();segment.push(p);});line();
 const labels=textNode('div','','history-overview-labels');labels.append(textNode('span',historyShortDate(full.start)),textNode('span',historyShortDate(full.end)));
 container.append(textNode('div','최근 '+days+'일 전체 · 색칠한 구간 확대','history-range'),svg,labels);
}
function drawHistory() {
 if(!historyData)return;
 const agent=$('history-agent').value, windowName=$('history-window').value;
 $('usage-plan-context').textContent=names[agent]+' · '+(windowName==='week'?'주간 한도':'단기 한도');
 const full=historyRange(historyData.displayDays,historyData.rangeEnd);
 const samples=(historyData.samples || []).filter(s=>s.agent===agent && s.window===windowName && Number.isFinite(s.left) && s.left>=0 && s.left<=100 && new Date(s.at).getTime()>=full.start && new Date(s.at).getTime()<=full.end).sort((a,b)=>new Date(a.at)-new Date(b.at));
 const zoom=historyZoomRange(samples,full),canZoom=zoom.start>full.start || zoom.end<full.end,zoomed=historyZoom && canZoom;
 const {start,end,interval}=zoomed?zoom:full;
 $('history-zoom').disabled=!canZoom;$('history-zoom').setAttribute('aria-pressed',String(zoomed));
 $('history-view-label').textContent=zoomed?'기록 구간 확대':'최근 '+historyData.displayDays+'일 전체';
 $('history-overview').hidden=true;$('history-overview').replaceChildren();
 if(zoomed)drawHistoryOverview(samples,full,zoom,historyData.displayDays);
 const chart=$('history-chart'), focusedAt=chart.contains(document.activeElement)?document.activeElement.dataset.at:null;
 dismissHistoryDetails();dismissHistoryDetails=()=>{};chart.replaceChildren();
 if(samples.length<2)message('history-message',samples.length===1?'기록이 1개입니다. 추이를 보려면 2개 이상의 기록이 필요합니다.':'이 기간에 수집한 기록이 없습니다. 앱이 실행되는 동안 기록이 쌓입니다.');
 else message('history-message',samples.length+'개 기록 · '+names[agent]+' · '+(windowName==='week'?'주간 한도':'단기 한도'));
 {
  const width=Math.max(240,chart.clientWidth), plotWidth=width-66;
  const svg=svgElement('svg',{viewBox:'0 0 '+width+' 230',class:'dash-history-chart',role:'group','aria-label':names[agent]+(zoomed?' 기록 구간 확대':' 최근 '+historyData.displayDays+'일')+' 잔여 한도 이력','data-start':start,'data-end':end,'data-interval':interval});
  for(const pct of [0,25,50,75,100]) { const y=180-pct*1.6;svg.append(svgElement('line',{x1:48,y1:y,x2:width-18,y2:y,stroke:'#dce3eb','stroke-dasharray':'3 4'})); const label=svgElement('text',{x:4,y:y+4,fill:'#626b78','font-size':12});label.textContent=pct+'%';svg.append(label); }
  const bins=historyBins(samples,interval);
  if(samples.length>1)message('history-message',historyIntervalLabel(interval)+' 간격 · 구간별 마지막 잔량 · '+bins.length+'개 표시 / '+samples.length+'개 기록');
  const points=bins.map(bin=>({...bin,x:end===start?48+plotWidth/2:48+(new Date(bin.s.at).getTime()-start)/(end-start)*plotWidth,y:180-bin.s.left*1.6})), dots=[];let segment=[];
  function line() { if(segment.length>1)svg.append(svgElement('polyline',{points:segment.map(p=>p.x+','+p.y).join(' '),fill:'none',stroke:'#0f766e','stroke-width':1.5,'vector-effect':'non-scaling-stroke'}));segment=[]; }
  points.forEach((p,index)=>{if(index && (p.segment!==points[index-1].segment || p.bucket-points[index-1].bucket>1))line();segment.push(p);});line();
  for(const p of points) { const label='구간 마지막 관측 · '+timeLabel(p.s.at)+' · '+p.s.left+'% 남음 · 출처 '+(p.s.source || '미확인')+' · 초기화 '+(p.s.reset_at>0?timeLabel(p.s.reset_at*1000):'미확인');const dot=svgElement('circle',{cx:p.x,cy:p.y,r:4,tabindex:0,class:'dash-chart-point','aria-label':label,'data-at':p.s.at});svg.append(dot);dots.push({dot,point:p}); }
  const tickCount=Math.max(2,Math.min(6,Math.floor(plotWidth/90)));
  for(let i=0;i<=tickCount;i++){
   const at=start+(end-start)*i/tickCount,x=48+plotWidth*i/tickCount,d=new Date(at);
   const tick=svgElement('text',{x,y:202,fill:'#626b78','font-size':11,'text-anchor':i===0?'start':i===tickCount?'end':'middle',class:'history-tick'});
   const date=svgElement('tspan',{x});date.textContent=(d.getMonth()+1)+'/'+d.getDate();tick.append(date);
   if(end-start<=2*86400000){const clock=svgElement('tspan',{x,dy:15});clock.textContent=historyShortDate(at,false);tick.append(clock);}
   svg.append(tick);
  }
  if(!points.length){const empty=svgElement('text',{x:48+plotWidth/2,y:102,'text-anchor':'middle',fill:'#626b78','font-size':12});empty.textContent='기록 없음';svg.append(empty);}
  chart.append(svg);
  if(points.length)attachHistoryDetails(chart,svg,points,dots,{start,end,interval,width});
  if(focusedAt){const target=dots.reduce((best,item)=>Math.abs(new Date(item.point.s.at)-new Date(focusedAt))<Math.abs(new Date(best.point.s.at)-new Date(focusedAt))?item:best,dots[0]);if(target)target.dot.focus({preventScroll:true});}
 }
 const forecast=(historyData.forecasts || []).find(f=>f.agent===agent && f.window===windowName), box=$('forecast-state');box.replaceChildren();box.classList.remove('risk');
 const pace=usagePaceAdvice(forecast);
 const ready=!!(forecast && forecast.ready && Number.isFinite(forecast.rate_per_day) && forecast.rate_per_day>0);
 const budget=forecast && Number.isFinite(forecast.daily_budget) && forecast.daily_budget>=0 ? forecast.daily_budget:null;
 const observation=forecast && forecast.observation,observedStart=Date.parse(observation && observation.started_at),observedEnd=Date.parse(observation && observation.ended_at);
 const observed=!!(observation && Number.isFinite(observedStart) && Number.isFinite(observedEnd) && observedEnd>observedStart && [observation.start_left,observation.end_left,observation.used_pp].every(value=>Number.isInteger(value) && value>=0 && value<=100) && observation.start_left-observation.end_left===observation.used_pp);
 const groups=textNode('div','','pace-groups'),records=textNode('section','','pace-group pace-records'),plan=textNode('section','','pace-group pace-plan');
 const recordsHeading=textNode('h3','관측 기록'),planHeading=textNode('h3','사용 예산');
 recordsHeading.id='pace-records-title';planHeading.id='pace-plan-title';records.setAttribute('aria-labelledby',recordsHeading.id);plan.setAttribute('aria-labelledby',planHeading.id);
 const observedMetrics=textNode('dl','','pace-observation-metrics'),metrics=textNode('dl','','pace-metrics');
 records.append(recordsHeading,observedMetrics);plan.append(planHeading,metrics);groups.append(records,plan);
 function metric(container,label,value,className,unit='%p / 일'){
  const group=textNode('div','','pace-metric '+className);group.append(textNode('dt',label));
  const amount=textNode('dd',value===null?'—':unit==='%p'?String(value):quotaPaceLabel(value).replace('%p',''));
  if(value!==null)amount.append(textNode('span',unit));
  group.append(amount);container.append(group);return group;
 }
 const actual=metric(observedMetrics,'사용량',observed?observation.used_pp:null,'pace-observed','%p');
 if(observed){
  const minutes=Math.floor((observedEnd-observedStart)/60000),days=Math.floor(minutes/1440),hours=Math.floor(minutes%1440/60),mins=minutes%60;
  const duration=[days?days+'일':'',hours?hours+'시간':'',mins?mins+'분':''].filter(Boolean).join(' ') || '1분 미만';
  actual.append(textNode('p','잔량 '+observation.start_left+'% → '+observation.end_left+'%','pace-note pace-observation-change'));
  const firstDate=new Date(observedStart),lastDate=new Date(observedEnd),sameDay=firstDate.toDateString()===lastDate.toDateString();
  const label=(ms,includeDate)=>{const date=new Date(ms),year=includeDate && date.getFullYear()!==new Date(historyData.rangeEnd).getFullYear()?date.getFullYear()+'/':'';return year+historyShortDate(ms,includeDate);};
  const range=label(observedStart,true)+' ~ '+label(observedEnd,!sameDay);
  actual.append(textNode('p','약 '+duration+' 관측 ('+range+')','pace-note pace-observation-range'));
 }else actual.append(textNode('p','비교할 관측 기록이 부족합니다.','pace-note'));
 metric(observedMetrics,'전체 관측 환산 속도',observed && Number.isFinite(forecast.rate_per_day) && forecast.rate_per_day>0?forecast.rate_per_day:null,'pace-recent');
 if(forecast && Number.isFinite(forecast.recent_rate_per_day))metric(observedMetrics,'최근 '+forecast.recent_minutes+'분 환산 속도',forecast.recent_rate_per_day,'pace-recent-window');
 const target=metric(metrics,'초기화까지의 하루 예산',budget,'pace-target');
 if(budget===0)target.append(textNode('p','남은 한도 없음','pace-note'));
 else if(pace && pace.reduction>0)plan.append(textNode('p','현재 환산 속도보다 '+(pace.reduction<1?'1% 미만':Math.min(99,Math.round(pace.reduction))+'%')+' 적게 사용','pace-note pace-reduction'));
 if(budget>0 && forecast.reset_at*1000-historyData.rangeEnd<86400000 && forecast.reset_at*1000>historyData.rangeEnd)target.append(textNode('p','시간당 '+quotaPaceLabel(budget/24),'pace-note'));
 box.append(groups);
 if(forecast && forecast.stability_reason)box.append(textNode('p',forecast.stability_reason,'pace-note pace-stability'));
 if(!ready){box.append(textNode('p',forecast && forecast.reason || '예측에 필요한 기록이 부족합니다.','pace-empty'));return;}
 box.classList.toggle('risk',!!forecast.exhausts_before_reset);
 if(forecast.estimated_exhaustion){
  const date=new Date(forecast.estimated_exhaustion),line=textNode('div','','pace-forecast');
  line.append(textNode('span','예상 소진','pace-forecast-label'));
  const display=isNaN(date)?'확인 중':date.toLocaleString('ko-KR',{...(date.getFullYear()!==new Date(historyData.rangeEnd).getFullYear()?{year:'numeric'}:{}),month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit',hour12:false});
  const estimate=textNode('span','','pace-estimate'),stamp=textNode('time',display,'pace-date');if(!isNaN(date))stamp.dateTime=date.toISOString();estimate.append(stamp);
  const status=forecastResetLead(forecast).replace('초기화보다 약 ','초기화 ').replace(' 먼저 소진 예상',' 전') || (forecast.exhausts_before_reset?'초기화 전 소진 예상':'초기화 전 소진 위험 낮음');
  estimate.append(textNode('span','('+status+')','pace-status'));line.append(estimate);
  line.append(textNode('p','관측 구간의 평균 사용 속도가 계속 유지될 경우의 추정입니다.','pace-assumption'));box.prepend(line);
 }
}
async function loadHistory() {
 const request=++historyRequest,days=Number($('history-days').value);
 if(historyData && historyData.displayDays!==days){historyData=null;dismissHistoryDetails();$('history-chart').replaceChildren();$('history-overview').hidden=true;$('history-overview').replaceChildren();$('history-zoom').disabled=true;$('history-zoom').setAttribute('aria-pressed','false');$('history-view-label').textContent='최근 '+days+'일 전체';message('history-message','기록을 불러오고 있습니다.');$('forecast-state').replaceChildren(textNode('p','예측 정보를 불러오고 있습니다.'));}
 try {
  const data=await api('/api/history?days='+days);if(request!==historyRequest)return;
  const serverEnd=Date.parse(data.range_end);historyData={...data,displayDays:days,rangeEnd:Number.isFinite(serverEnd)?serverEnd:Date.now()};
  if(!historySelectionInitialized) {
   historySelectionInitialized=true;
   if(!historySelectionTouched) {
    const latest=(data.samples || []).filter(s=>Object.prototype.hasOwnProperty.call(names,s.agent) && ['short','week'].includes(s.window) && Number.isFinite(s.left) && s.left>=0 && s.left<=100 && !isNaN(new Date(s.at))).reduce((last,s)=>!last || new Date(s.at)>new Date(last.at)?s:last,null);
    if(latest) { $('history-agent').value=latest.agent;$('history-window').value=latest.window; }
   }
  }
  drawHistory();
 }
 catch(e) { if(request===historyRequest)message('history-message',historyData?'기록을 불러오지 못했습니다. 표시된 그래프는 직전 조회 결과입니다.':'기록을 불러오지 못했습니다. 기간을 다시 선택해 주세요.',true); }
}
$('history-agent').addEventListener('change',()=>{historySelectionTouched=true;saveHistoryView();drawHistory();});$('history-window').addEventListener('change',()=>{historySelectionTouched=true;saveHistoryView();drawHistory();});$('history-days').addEventListener('change',()=>{historySelectionTouched=true;saveHistoryView();loadHistory();});
$('history-zoom').addEventListener('click',()=>{historyZoom=!historyZoom;saveHistoryView();drawHistory();});
let chartResizeFrame=0;
window.addEventListener('resize',()=>{if(page!=='usage' || compact)return;cancelAnimationFrame(chartResizeFrame);chartResizeFrame=requestAnimationFrame(drawHistory);});
async function refreshSnapshots() {
 const res=await fetch(location.pathname+location.search,{cache:'no-store'});if(!res.ok)throw new Error('snapshot');const doc=new DOMParser().parseFromString(await res.text(),'text/html');
 const ids=page==='usage'?['usage-snapshot','snapshot-time']:page==='tokens'?['token-snapshot','snapshot-time']:['snapshot-time'];
 for(const id of ids) { const source=doc.getElementById(id), target=$(id);if(source && target){if(id==='token-snapshot')dismissTokenDetails();target.replaceChildren(...Array.from(source.childNodes));} }
}
async function poll() {
 if(polling || document.hidden || mutationCount)return;polling=true;
 try { await Promise.all([page==='settings'?Promise.all([loadStatus(),loadAutomaticUpdateStatus()]):Promise.resolve(),page==='usage' && !compact?loadHistory():Promise.resolve(),refreshSnapshots()]); }
 catch(e) { message(page==='settings'?'status-message':'action-message','화면 자동 갱신을 완료하지 못했습니다. 앱 연결을 확인해 주세요.',true); }
 finally { polling=false; }
}
const tokenForm=$('token-filter-form');
if(page==='tokens' && tokenForm){
 const readTokenView=()=>({days:$('token-days').value,model:$('token-model').value,project:$('token-project').value});
 const options=readTokenView();
 const query=new URLSearchParams(location.search),hasExplicit=['days','model','project','all'].some(key=>query.has(key)),saved=storageRead('usage-tray-token-view');
 if(!hasExplicit && saved && [7,14,30].includes(Number(saved.days))){
  const params=new URLSearchParams({days:String(saved.days)});if(typeof saved.model==='string' && saved.model)params.set('model',saved.model);if(typeof saved.project==='string' && saved.project)params.set('project',saved.project);
  if(params.toString()!==new URLSearchParams({days:options.days}).toString() || options.days!==String(saved.days))location.replace('/tokens?'+params.toString());
 }
 storageWrite('usage-tray-token-view',options);
 tokenForm.addEventListener('change',()=>{storageWrite('usage-tray-token-view',readTokenView());tokenForm.requestSubmit();});
 $('token-reset').addEventListener('click',()=>{
  $('token-days').value='14';$('token-model').value='';$('token-project').value='';
  storageWrite('usage-tray-token-view',readTokenView());location.assign('/tokens?days=14');
 });
}
if(page==='settings'){loadSettings();loadStartup();loadSettingsContext();loadStatus();loadAutomaticUpdateStatus();}else if(page==='usage' && !compact){loadHistory();}setInterval(poll,30000);checkAppMetadata();setInterval(checkAppMetadata,10000);setInterval(renderPauseState,30000);
document.addEventListener('visibilitychange',()=>{if(!document.hidden){checkAppMetadata();poll();}});
})();
`
