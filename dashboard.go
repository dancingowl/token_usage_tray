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
.disclosure{margin-bottom:12px;scroll-margin-top:24px}.disclosure>summary{display:flex;align-items:center;gap:12px;list-style:none;min-height:26px}.disclosure>summary::-webkit-details-marker{display:none}.disclosure>summary:after{content:'+';font-size:21px;font-weight:400;color:var(--sub);margin-left:auto}.disclosure[open]>summary:after{content:'−'}.summary-note{font-size:12px;font-weight:400;color:var(--sub)}
.history-heading{display:flex;align-items:center;gap:12px;position:relative;margin-bottom:20px}.history-heading h2{margin:0;font-size:17px;color:var(--tx)}.history-heading .summary-note{margin-left:10px}
.history-help{margin-left:auto}.history-help>summary{display:flex;align-items:center;justify-content:center;list-style:none;width:25px;height:25px;border:1px solid #cbd3dd;border-radius:50%;color:var(--sub);font-size:13px;font-weight:600;cursor:pointer}.history-help>summary::-webkit-details-marker{display:none}.history-help[open]>summary{margin:0;background:#e1eeea;color:#0c695d}.history-help-content{position:absolute;right:0;top:34px;z-index:3;width:340px;max-width:100%;padding:12px 14px;border:1px solid var(--line);border-radius:10px;background:#fff;box-shadow:0 4px 16px #20293820;color:var(--sub);font-size:12px}.history-help-content p{margin:0 0 8px}.history-help-content p:last-child{margin:0}
.hover-help{padding-bottom:9px;margin-bottom:-9px}
.history-range{font-size:11px;color:var(--sub);margin:4px 0 0;overflow-wrap:anywhere}
.history-view-controls{display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap;margin-top:14px}.history-view-controls span{font-size:12px;color:var(--sub)}#history-zoom[aria-pressed="true"]{color:#0c695d;background:#e1eeea;border-color:#a9d4c9}.history-overview{margin-top:12px;padding:10px 12px;border:1px solid var(--line);border-radius:9px;background:var(--panel2)}.history-overview svg{display:block;width:100%;height:76px}.history-overview-labels{display:flex;justify-content:space-between;gap:10px;font-size:11px;color:var(--sub)}
.dash-field{display:flex;flex-direction:column;gap:5px;margin-bottom:12px}.dash-check{display:inline-flex;gap:7px;align-items:center;margin:4px 14px 4px 0}.dash-check input{accent-color:#0f766e;width:16px;height:16px}
.dash-form-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:8px 20px}
.dash-fieldset{min-width:0;border:1px solid var(--line);border-radius:9px;padding:10px 12px;margin:0 0 14px}.dash-fieldset legend{color:var(--sub);padding:0 5px}
.dash-rules{overflow:auto}.dash-rules th,.dash-rules td{padding:8px 9px;white-space:nowrap}.dash-rules input[type=number]{width:80px}.dash-rules input[type=checkbox]{width:16px;height:16px;accent-color:#0f766e}
details>summary{cursor:pointer;font-weight:700;font-size:15px}details[open]>summary{margin-bottom:16px}.dash-status-row{padding:10px 0;border-bottom:1px solid var(--line)}.dash-status-row:last-child{border:0}.dash-status-title{display:flex;justify-content:space-between;gap:10px;align-items:center}.dash-status-title .chip{color:var(--sub)}.dash-status-row p{margin:4px 0;font-size:12px;color:var(--sub);overflow-wrap:anywhere}
.dash-history-chart{width:100%;height:250px;display:block}.dash-chart-wrap{position:relative;overflow:hidden;margin:14px 0 6px}.dash-chart-point{fill:transparent;stroke:none}.dash-chart-point:focus{outline:none}
.history-tooltip{position:absolute;z-index:2;width:280px;max-width:calc(100% - 16px);padding:12px 14px;border:1px solid #cbded8;border-radius:10px;background:#fff;box-shadow:0 4px 16px #20293820;pointer-events:none;color:var(--tx);font-size:12px;line-height:1.55;overflow-wrap:anywhere}.history-tooltip strong{display:block;font-size:19px;color:#0f766e;margin:2px 0 5px}.history-tooltip p{margin:2px 0}.history-tooltip .tooltip-time{color:var(--sub)}
.dash-forecast{font-size:13px}.dash-forecast p{margin:4px 0}.dash-release{color:#0f766e}.dash-update-notes{white-space:pre-wrap;max-height:160px;overflow:auto;color:var(--sub);font-size:12px}
.pace-metrics{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;margin:0}.pace-metric dt{font-size:12px;color:var(--sub);margin-bottom:7px}.pace-metric dd{margin:0;font-size:26px;line-height:1.3;font-weight:650;letter-spacing:-.03em;font-variant-numeric:tabular-nums}.pace-metric dd span{font-size:13px;font-weight:400;letter-spacing:0;margin-left:4px;color:var(--sub)}.pace-metric .pace-note{margin-top:6px;font-size:12px;color:var(--sub)}.pace-forecast{border-bottom:1px solid var(--line);padding-bottom:18px;margin-bottom:20px}.pace-forecast-label{display:block;color:var(--sub);font-size:12px;margin-bottom:7px}.pace-estimate{display:flex;align-items:baseline;gap:6px 10px;flex-wrap:wrap;min-width:0}.pace-date{font-size:30px;font-weight:750;line-height:1.25;letter-spacing:-.03em;font-variant-numeric:tabular-nums}.pace-status{color:var(--sub);font-size:14px;font-weight:500}.risk .pace-status{color:#a33c3c}.pace-empty{color:var(--sub);font-size:12px}
body.summary-view{max-width:640px;padding:18px}body.summary-view #token-snapshot,body.summary-view .dashboard-full{display:none}body.summary-view h1{font-size:21px}body.summary-view .cl{font-size:26px}
@media(max-width:620px){.dash-grid{grid-template-columns:1fr}.dash-form-grid{grid-template-columns:1fr}.dash-actions select{max-width:100%}.summary-note{display:none}.quick-toolbar{justify-content:flex-start}.quick-toolbar .dash-message{text-align:left}}
@media(max-width:620px){.history-controls{grid-template-columns:1fr}.history-controls label{display:grid;grid-template-columns:60px minmax(0,1fr)}.history-controls select{width:100%}.dash-chart-labels{font-size:11px}}
`

const headerActionsHTML = `<div class="dash-actions quick-toolbar"><button type="button" data-action="refresh"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20 7v5h-5M4 17v-5h5M6 8a7 7 0 0 1 11-2l3 3M18 16a7 7 0 0 1-11 2l-3-3"/></svg>새로고침</button><button type="button" data-action="detail" id="detail-action"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M14 4h6v6M20 4l-9 9M10 5H5v14h14v-5"/></svg>상세 보기</button><a class="button subtle" href="/settings"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M4 7h16M4 17h16M8 4v6M16 14v6"/></svg>설정</a></div>`

const dashboardHTML = `
<div class="dashboard">
 <section id="history-details" class="panel page-section dashboard-full" aria-labelledby="history-title"><div class="history-heading"><h2 id="history-title">잔여량 추이<span class="summary-note">이력 · 예상 소진</span></h2><details id="history-help" class="history-help"><summary aria-label="그래프 도움말" title="그래프 도움말">?</summary><div class="history-help-content"><p>기록 구간 확대를 켜면 위에는 선택 기간 전체와 확대 위치를, 아래에는 기록이 있는 구간을 크게 보여줍니다. 버튼을 다시 누르면 전체 기간 보기로 돌아갑니다.</p><p>각 시간 구간의 마지막 관측 잔량을 표시하며 확대 범위에 맞춰 간격을 조절합니다. 원본 기록은 보존하고 초기화 전후와 빈 구간은 잇지 않습니다. 호버·터치·탭 키로 상세를 확인하고 Esc로 닫습니다.</p><p>소진 예상은 원본 기록의 최근 소비 속도에 따른 추정입니다.</p></div></details></div>
  <div class="history-controls"><label for="history-agent">에이전트<select id="history-agent"><option value="claude">Claude</option><option value="codex">Codex</option><option value="antigravity">Antigravity</option></select></label><label for="history-window">한도<select id="history-window"><option value="short">단기 · 5시간</option><option value="week">주간</option></select></label><label for="history-days">기간<select id="history-days"><option value="1">최근 1일</option><option value="7" selected>최근 7일</option><option value="30">최근 30일</option></select></label></div>
  <div class="history-view-controls"><span id="history-view-label">전체 기간</span><button type="button" id="history-zoom" aria-pressed="true" aria-controls="history-chart history-overview" disabled>기록 구간 확대</button></div>
  <div id="history-overview" class="history-overview" hidden></div>
  <p id="history-message" class="dash-message" role="status" aria-live="polite">기록을 불러오고 있습니다.</p><div id="history-chart" class="dash-chart-wrap"></div>
 </section>
 <section id="usage-plan" class="panel page-section dashboard-full" aria-labelledby="usage-plan-title"><div class="history-heading"><h2 id="usage-plan-title">사용 계획<span id="usage-plan-context" class="summary-note"></span></h2><details id="usage-plan-help" class="history-help hover-help"><summary aria-label="사용 계획 도움말">?</summary><div class="history-help-content"><p>하루 사용 가능량은 남은 한도를 초기화까지 균등하게 나눈 값입니다. 최근 사용량은 현재 한도 주기의 관측 기록으로 계산한 하루 기준 속도입니다.</p><p>%p는 전체 한도 기준의 변화량입니다. 잔량이 80%에서 70%가 되면 10%p를 사용한 것입니다. 초기화까지 하루 미만이면 시간당 값도 표시합니다.</p><p>소진 시점은 최근 사용 속도가 유지된다는 가정의 추정값입니다.</p></div></details></div>
  <div id="forecast-state" class="dash-forecast">예측에 필요한 기록을 확인하고 있습니다.</div>
 </section>
 <section id="settings" class="dashboard-full"><div id="settings-details" class="panel page-section"><h2>설정<span class="summary-note">표시 · 알림 · 자동 실행</span></h2>
  <form id="settings-form"><fieldset id="settings-fields" class="dash-fieldset" disabled><legend>트레이와 수집</legend>
   <div class="dash-form-grid"><label class="dash-field">아이콘 표시<select name="icon_style"><option value="donut">도넛 · 남은 비율</option><option value="number">숫자 · 남은 퍼센트</option></select></label><label class="dash-field">아이콘에 표시할 한도<select name="display_window"><option value="week">주간 한도</option><option value="short">단기 한도</option><option value="urgent">잔여 비율이 가장 낮은 한도</option></select></label><label class="dash-field">수집 간격 (초)<input name="poll_seconds" type="number" min="60" max="3600" step="1" required><span class="dash-note">60~3,600초 · 변경 후 다음 수집부터 적용</span></label></div>
   <p class="dash-note">API 요청 보호를 위해 Claude는 최소 3분, Codex는 최소 2분 간격으로 조회합니다. 더 짧은 수집 간격을 선택해도 이 제한은 유지됩니다.</p>
   <div><span>트레이에 표시할 에이전트</span><br><label class="dash-check"><input name="visible_claude" type="checkbox">Claude</label><label class="dash-check"><input name="visible_codex" type="checkbox">Codex</label><label class="dash-check"><input name="visible_antigravity" type="checkbox">Antigravity</label></div>
   <p class="dash-note">저장하면 트레이에 바로 반영됩니다. 상세 화면과 이력에서는 모든 에이전트를 확인할 수 있으며 수집도 계속됩니다.</p>
   <label class="dash-check"><input name="alerts_enabled" type="checkbox">잔여량 알림 켜기</label><label class="dash-check"><input name="notify_limits" type="checkbox">사용 한도 도달 알림 켜기</label>
   <div class="dash-rules"><table><thead><tr><th scope="col">알림 대상</th><th scope="col">사용</th><th scope="col">남은량 기준 (%)</th><th scope="col">회복 알림</th></tr></thead><tbody id="alert-rules"></tbody></table></div><p class="dash-note">잔여량이 기준 미만이 되면 알립니다. 기준은 1~99%입니다.</p>
   <div class="dash-actions"><button type="submit" class="primary" id="save-settings">설정 저장</button><button type="button" id="reset-settings">저장한 설정 다시 불러오기</button></div>
  </fieldset></form><p id="settings-message" class="dash-message" role="status" aria-live="polite">설정을 불러오고 있습니다.</p>
  <fieldset class="dash-fieldset"><legend>로그인 시 자동 실행</legend><p id="startup-state" class="dash-note">자동 실행 지원 여부를 확인하고 있습니다.</p><label class="dash-check"><input id="startup-enabled" type="checkbox" disabled>로그인할 때 Usage Tray 실행</label><button type="button" id="save-startup" disabled>자동 실행 설정 적용</button><p class="dash-note">선택만으로 적용되지 않습니다. 위 적용 버튼을 눌러 등록 또는 해제합니다.</p><p id="startup-message" class="dash-message" role="status" aria-live="polite"></p></fieldset>
 </div></section>
 <section id="system-details" class="panel page-section dashboard-full" aria-labelledby="system-title"><h2 id="system-title">앱 상태<span class="summary-note">연결 · 업데이트</span></h2><div class="dash-grid">
  <section aria-labelledby="status-title"><h2 id="status-title">연결 상태</h2><div id="agent-status"><p class="none">상태를 확인하고 있습니다.</p></div><p id="activity-state" class="dash-note"></p><p id="status-message" class="dash-message" role="status"></p></section>
  <section aria-labelledby="update-title"><h2 id="update-title">업데이트</h2><p id="update-state" class="dash-note">버전 정보를 확인하고 있습니다.</p><div class="dash-actions"><button type="button" id="check-update">업데이트 확인</button><a class="dash-release" id="release-link" target="_blank" rel="noopener noreferrer" hidden>릴리스 열기</a></div><pre id="update-notes" class="dash-update-notes" hidden></pre><p id="update-message" class="dash-message" role="status" aria-live="polite"></p></section>
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
let savedSettings = null, settingsBusy = false, historyData = null, historyRequest = 0, historySelectionInitialized = false, historySelectionTouched = false, polling = false, mutationCount = 0, historyZoom = true;
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
let settingsDirty=false, startupDirty=false;
window.addEventListener('beforeunload',event=>{
 if(settingsDirty || startupDirty) { event.preventDefault();event.returnValue=''; }
});
function message(id, text, error) { const e=$(id); e.textContent=text; e.classList.toggle('error',!!error); }
function textNode(tag, text, className) { const e=document.createElement(tag); e.textContent=text; if(className)e.className=className; return e; }
function timeLabel(value) { const d=new Date(value); return value && !isNaN(d) ? d.toLocaleString('ko-KR') : '아직 성공한 기록 없음'; }
async function api(path, method, body) {
 const options={cache:'no-store',headers:{'Accept':'application/json'}};
 if(method) { options.method=method; options.headers['X-Usage-Tray']='1'; options.headers['Content-Type']='application/json'; if(body!==undefined)options.body=JSON.stringify(body); }
 const res=await fetch(path,options);
 if(!res.ok) throw new Error(res.status===400 ? '입력값을 확인해 주세요.' : '요청을 완료하지 못했습니다. 잠시 후 다시 시도해 주세요.');
 return res.json();
}
function field(name) { return form.elements.namedItem(name); }
function setChecked(name,value) { field(name).checked=!!value; }
function drawRules(settings) {
 const tbody=$('alert-rules'); tbody.replaceChildren();
 for(const [key,label] of Object.entries(ruleNames)) {
  const rule=settings.alert_rules[key] || {enabled:false,threshold:20,recovery:false};
  const tr=document.createElement('tr'); const title=textNode('th',label); title.scope='row'; tr.append(title);
  for(const type of ['enabled','threshold','recovery']) {
   const td=document.createElement('td'), input=document.createElement('input'); input.name=key+'_'+type; input.setAttribute('aria-label',label+' '+({enabled:'알림 사용',threshold:'남은량 기준 퍼센트',recovery:'회복 알림'}[type]));
   if(type==='threshold') { input.type='number';input.min='1';input.max='99';input.step='1';input.required=true;input.value=rule.threshold; }
   else { input.type='checkbox';input.checked=!!rule[type]; } td.append(input); tr.append(td);
  } tbody.append(tr);
 }
}
function fillSettings(settings) {
 settingsDirty=false;
 savedSettings=settings; field('icon_style').value=settings.icon_style; field('display_window').value=settings.display_window; field('poll_seconds').value=settings.poll_seconds;
 for(const key of Object.keys(names))setChecked('visible_'+key,settings.visible_agents[key]);
 setChecked('alerts_enabled',settings.alerts_enabled);setChecked('notify_limits',settings.notify_limits);drawRules(settings);
}
function setSettingsBusy(busy) {
 settingsBusy=busy;$('settings-fields').disabled=busy || !savedSettings;form.setAttribute('aria-busy',String(busy));
}
async function loadSettings() {
 if(settingsBusy)return;setSettingsBusy(true);message('settings-message','설정을 불러오고 있습니다.');
 try { fillSettings(await api('/api/settings'));message('settings-message','저장한 설정을 불러왔습니다.'); }
 catch(e) { message('settings-message','설정을 불러오지 못했습니다. 페이지를 다시 열어 주세요.',true); }
 finally { setSettingsBusy(false); }
}
form.addEventListener('input',()=>{settingsDirty=true;message('settings-message','아직 저장하지 않은 변경사항이 있습니다.');});
form.addEventListener('submit',async e=>{
 e.preventDefault(); if(settingsBusy || !savedSettings || !form.reportValidity())return;
 const settings={icon_style:field('icon_style').value,display_window:field('display_window').value,poll_seconds:Number(field('poll_seconds').value),visible_agents:{},alerts_enabled:field('alerts_enabled').checked,notify_limits:field('notify_limits').checked,alert_rules:{}};
 for(const key of Object.keys(names))settings.visible_agents[key]=field('visible_'+key).checked;
 for(const key of Object.keys(ruleNames))settings.alert_rules[key]={enabled:field(key+'_enabled').checked,threshold:Number(field(key+'_threshold').value),recovery:field(key+'_recovery').checked};
 setSettingsBusy(true);mutationCount++;message('settings-message','설정을 저장하고 있습니다.');
 try { const result=await api('/api/settings','PUT',settings);fillSettings(result.icon_style ? result : settings);message('settings-message','설정을 저장했습니다.'); }
 catch(err) { message('settings-message',err.message,true); }
 finally { setSettingsBusy(false);mutationCount--; }
});
$('reset-settings').addEventListener('click',loadSettings);
function renderStartup(data) {
 startupDirty=false;
 $('startup-enabled').checked=!!data.enabled;$('startup-enabled').disabled=!data.supported;$('save-startup').disabled=!data.supported;
 $('startup-state').textContent=data.supported ? (data.enabled?'로그인 시 자동 실행이 등록되어 있습니다.':'로그인 시 자동 실행이 꺼져 있습니다.') : '이 운영체제에서는 자동 실행 설정을 지원하지 않습니다.';
}
async function loadStartup() { try { renderStartup(await api('/api/startup')); } catch(e) { message('startup-message','자동 실행 상태를 확인하지 못했습니다.',true); } }
$('startup-enabled').addEventListener('change',()=>{startupDirty=true;message('startup-message','적용 버튼을 누르면 자동 실행 설정을 변경합니다.');});
$('save-startup').addEventListener('click',async()=>{
 const enabled=$('startup-enabled').checked;$('save-startup').disabled=true;$('startup-enabled').disabled=true;mutationCount++;
 try { renderStartup(await api('/api/startup','POST',{enabled}));message('startup-message',enabled?'자동 실행을 등록했습니다.':'자동 실행을 해제했습니다.'); }
 catch(e) { message('startup-message',e.message,true);$('save-startup').disabled=false;$('startup-enabled').disabled=false; }
 finally { mutationCount--; }
});
async function loadStatus() {
 try {
  const data=await api('/api/status'), container=$('agent-status');container.replaceChildren();
  for(const agent of data.agents || []) {
   const row=document.createElement('div');row.className='dash-status-row';row.dataset.agent=agent.name.toLowerCase();
   const title=document.createElement('div');title.className='dash-status-title';title.append(textNode('strong',agent.name),textNode('span',agent.label || '상태 확인 중','chip'+(['error','auth_error'].includes(agent.status)?' bad':'')));row.append(title);
   row.append(textNode('p','갱신 '+timeLabel(agent.last_success)+(agent.last_success && agent.age_minutes>=0?' · '+agent.age_minutes+'분 전':'')+(agent.source?' · '+agent.source:'')));
   if(agent.error)row.append(textNode('p',agent.error));
   container.append(row);
  }
  if(!container.children.length)container.append(textNode('p','연결된 에이전트가 없습니다.','none'));
  $('activity-state').textContent=[data.refreshing?'한도 갱신 중':'',data.scanning?'로컬 토큰 집계 중':''].filter(Boolean).join(' · ');message('status-message','');
 } catch(e) { message('status-message','연결 상태를 확인하지 못했습니다. 표시된 정보는 직전 조회 결과입니다.',true); }
}
document.querySelectorAll('[data-action]').forEach(button=>button.addEventListener('click',async()=>{
 const action=button.dataset.action;button.disabled=true;mutationCount++;
 try { await api('/api/actions','POST',{action});message('action-message',action==='refresh'?'새로고침을 요청했습니다. 수집이 끝나면 화면에 반영됩니다.':'상세 보기를 열었습니다.');if(action==='refresh')await loadStatus(); }
 catch(e) { message('action-message',e.message,true); }
 finally { button.disabled=false;mutationCount--; }
}));
function validReleaseURL(value) { try { const u=new URL(value);return u.protocol==='https:' && u.hostname==='github.com' && !u.username && !u.password && !u.port && u.pathname.startsWith('/dancingowl/token_usage_tray/releases/'); } catch(e) { return false; } }
function renderUpdate(data) {
 const version=data.current_version || 'dev';
 const available=version!=='dev' && data.available;
 const checked=data.checked_at && !isNaN(new Date(data.checked_at)) && new Date(data.checked_at).getUTCFullYear()>1;
 const state=data.error?' · 최근 확인 실패':available?' · 새 버전 '+data.latest_version+' 사용 가능':data.latest_version?' · 최신 릴리스 '+data.latest_version+(version==='dev'?' · 개발 빌드는 자동 버전 비교를 지원하지 않습니다.':''):' · 아직 확인하지 않았습니다.';
 $('update-state').textContent='현재 '+version+state+(checked?' · 확인 '+timeLabel(data.checked_at):'');
 const link=$('release-link');link.hidden=!data.latest_version || !validReleaseURL(data.release_url);if(!link.hidden)link.href=data.release_url;else link.removeAttribute('href');
 $('update-notes').hidden=!data.notes;$('update-notes').textContent=data.notes || '';
 message('update-message',data.error || (available?'릴리스 내용을 확인한 뒤 다운로드해 주세요.':checked?'확인이 끝났습니다.':''),!!data.error);
}
async function loadUpdate(check) {
 if(check){$('check-update').disabled=true;message('update-message','새 버전을 확인하고 있습니다.');}
 try { renderUpdate(await api('/api/update',check?'POST':undefined,check?{}:undefined)); }
 catch(e) { message('update-message','업데이트 정보를 확인하지 못했습니다. 잠시 후 다시 시도해 주세요.',true); }
 finally { $('check-update').disabled=false; }
}
$('check-update').addEventListener('click',()=>loadUpdate(true));
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
 const metrics=textNode('dl','','pace-metrics');
 function metric(label,value,className){
  const group=textNode('div','','pace-metric '+className);group.append(textNode('dt',label));
  const amount=textNode('dd',value===null?'—':quotaPaceLabel(value).replace('%p',''));
  if(value!==null)amount.append(textNode('span','%p / 일'));
  group.append(amount);metrics.append(group);return group;
 }
 const target=metric('하루 사용 가능량',budget,'pace-target');
 metric('최근 사용량',ready?forecast.rate_per_day:null,'pace-recent');
 if(budget===0)target.append(textNode('p','남은 한도 없음','pace-note'));
 else if(pace && pace.reduction>0)target.append(textNode('p','최근보다 '+(pace.reduction<1?'1% 미만':Math.min(99,Math.round(pace.reduction))+'%')+' 적게','pace-note pace-reduction'));
 if(budget>0 && forecast.reset_at*1000-historyData.rangeEnd<86400000 && forecast.reset_at*1000>historyData.rangeEnd)target.append(textNode('p','시간당 '+quotaPaceLabel(budget/24),'pace-note'));
 box.append(metrics);
 if(!ready){box.append(textNode('p',forecast && forecast.reason || '예측에 필요한 기록이 부족합니다.','pace-empty'));return;}
 box.classList.toggle('risk',!!forecast.exhausts_before_reset);
 if(forecast.estimated_exhaustion){
  const date=new Date(forecast.estimated_exhaustion),line=textNode('div','','pace-forecast');
  line.append(textNode('span','예상 소진','pace-forecast-label'));
  const display=isNaN(date)?'확인 중':date.toLocaleString('ko-KR',{...(date.getFullYear()!==new Date(historyData.rangeEnd).getFullYear()?{year:'numeric'}:{}),month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit',hour12:false});
  const estimate=textNode('span','','pace-estimate'),stamp=textNode('time',display,'pace-date');if(!isNaN(date))stamp.dateTime=date.toISOString();estimate.append(stamp);
  const status=forecastResetLead(forecast).replace('초기화보다 약 ','초기화 ').replace(' 먼저 소진 예상',' 전') || (forecast.exhausts_before_reset?'초기화 전 소진 예상':'초기화 전 소진 위험 낮음');
  estimate.append(textNode('span','('+status+')','pace-status'));line.append(estimate);box.prepend(line);
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
$('history-agent').addEventListener('change',()=>{historySelectionTouched=true;drawHistory();});$('history-window').addEventListener('change',()=>{historySelectionTouched=true;drawHistory();});$('history-days').addEventListener('change',()=>{historySelectionTouched=true;loadHistory();});
$('history-zoom').addEventListener('click',()=>{historyZoom=!historyZoom;drawHistory();});
let chartResizeFrame=0;
window.addEventListener('resize',()=>{if(page!=='usage' || compact)return;cancelAnimationFrame(chartResizeFrame);chartResizeFrame=requestAnimationFrame(drawHistory);});
async function refreshSnapshots() {
 const res=await fetch(location.pathname,{cache:'no-store'});if(!res.ok)throw new Error('snapshot');const doc=new DOMParser().parseFromString(await res.text(),'text/html');
 const ids=page==='usage'?['usage-snapshot','snapshot-time']:page==='tokens'?['token-snapshot','snapshot-time']:['snapshot-time'];
 for(const id of ids) { const source=doc.getElementById(id), target=$(id);if(source && target)target.replaceChildren(...Array.from(source.childNodes)); }
}
async function poll() {
 if(polling || document.hidden || mutationCount)return;polling=true;
 try { await Promise.all([page==='settings'?loadStatus():Promise.resolve(),page==='usage' && !compact?loadHistory():Promise.resolve(),refreshSnapshots()]); }
 catch(e) { message(page==='settings'?'status-message':'action-message','화면 자동 갱신을 완료하지 못했습니다. 앱 연결을 확인해 주세요.',true); }
 finally { polling=false; }
}
if(page==='settings'){loadSettings();loadStartup();loadStatus();loadUpdate(false);}else if(page==='usage' && !compact){loadHistory();}setInterval(poll,30000);
document.addEventListener('visibilitychange',()=>{if(!document.hidden)poll();});
})();
`
