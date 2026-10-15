package main

// The dashboard is self-contained: no remote scripts, credentials or full-page
// refresh. The existing token charts are refreshed separately from the form.
const dashboardCSS = `
[hidden]{display:none!important}
.dashboard{margin-bottom:14px}.dash-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,310px),1fr));gap:24px;margin-top:20px}
.dash-panel{margin-bottom:14px}.dash-panel h2{font-size:15px;color:var(--tx);margin-bottom:12px}
.dash-actions{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
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
.quick-toolbar{display:flex;justify-content:flex-end;flex-wrap:wrap;gap:8px;margin:4px 0 24px}.quick-toolbar .dash-message{flex-basis:100%;text-align:right}
.disclosure{margin-bottom:12px;scroll-margin-top:24px}.disclosure>summary{display:flex;align-items:center;gap:12px;list-style:none;min-height:26px}.disclosure>summary::-webkit-details-marker{display:none}.disclosure>summary:after{content:'+';font-size:21px;font-weight:400;color:var(--sub);margin-left:auto}.disclosure[open]>summary:after{content:'−'}.summary-note{font-size:12px;font-weight:400;color:var(--sub)}
.dash-help{margin-top:12px;font-size:12px;color:var(--sub)}.dash-help>summary{font-size:12px;font-weight:400}.dash-help[open]>summary{margin-bottom:4px}
.dash-field{display:flex;flex-direction:column;gap:5px;margin-bottom:12px}.dash-check{display:inline-flex;gap:7px;align-items:center;margin:4px 14px 4px 0}.dash-check input{accent-color:#0f766e;width:16px;height:16px}
.dash-form-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:8px 20px}
.dash-fieldset{min-width:0;border:1px solid var(--line);border-radius:9px;padding:10px 12px;margin:0 0 14px}.dash-fieldset legend{color:var(--sub);padding:0 5px}
.dash-rules{overflow:auto}.dash-rules th,.dash-rules td{padding:8px 9px;white-space:nowrap}.dash-rules input[type=number]{width:80px}.dash-rules input[type=checkbox]{width:16px;height:16px;accent-color:#0f766e}
details>summary{cursor:pointer;font-weight:700;font-size:15px}details[open]>summary{margin-bottom:16px}.dash-status-row{padding:10px 0;border-bottom:1px solid var(--line)}.dash-status-row:last-child{border:0}.dash-status-title{display:flex;justify-content:space-between;gap:10px;align-items:center}.dash-status-title .chip{color:var(--sub)}.dash-status-row p{margin:4px 0;font-size:12px;color:var(--sub);overflow-wrap:anywhere}
.dash-history-chart{width:100%;height:230px;display:block}.dash-chart-wrap{overflow:hidden;margin:14px 0 6px}.dash-chart-labels{display:flex;justify-content:space-between;gap:12px;font-size:12px;color:var(--sub)}.dash-chart-point:focus{outline:none;stroke:#202938;stroke-width:3}
.dash-forecast{border-left:3px solid #0f766e;padding:10px 14px;background:#f2f8f6;border-radius:8px;font-size:13px}.dash-forecast.risk{border-color:var(--bad);background:#fff5f5}.dash-forecast p{margin:4px 0}.dash-release{color:#0f766e}.dash-update-notes{white-space:pre-wrap;max-height:160px;overflow:auto;color:var(--sub);font-size:12px}
body.summary-view{max-width:640px;padding:18px}body.summary-view #token-snapshot,body.summary-view .dashboard-full{display:none}body.summary-view h1{font-size:21px}body.summary-view .cl{font-size:26px}
@media(max-width:620px){.dash-grid{grid-template-columns:1fr}.dash-form-grid{grid-template-columns:1fr}.dash-actions select{max-width:100%}.summary-note{display:none}.quick-toolbar{justify-content:flex-start}.quick-toolbar .dash-message{text-align:left}}
`

const dashboardHTML = `
<div class="dashboard">
 <section class="quick-toolbar" aria-label="빠른 작업">
  <div class="dash-actions"><button type="button" data-action="refresh"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20 7v5h-5M4 17v-5h5M6 8a7 7 0 0 1 11-2l3 3M18 16a7 7 0 0 1-11 2l-3-3"/></svg>새로고침</button><button type="button" data-action="detail" id="detail-action"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M14 4h6v6M20 4l-9 9M10 5H5v14h14v-5"/></svg>상세 보기</button><a class="button subtle" href="/#settings"><svg class="button-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M4 7h16M4 17h16M8 4v6M16 14v6"/></svg>설정</a></div>
  <p id="action-message" class="dash-message" role="status" aria-live="polite"></p>
 </section>
 <details id="history-details" class="panel disclosure dashboard-full"><summary id="history-title">잔여량 추이<span class="summary-note">이력 · 예상 소진</span></summary>
  <div class="dash-actions"><label for="history-agent">에이전트</label><select id="history-agent"><option value="claude">Claude</option><option value="codex">Codex</option><option value="antigravity">Antigravity</option></select><label for="history-window">한도</label><select id="history-window"><option value="short">단기 · 5시간</option><option value="week">주간</option></select><label for="history-days">기간</label><select id="history-days"><option value="1">최근 1일</option><option value="7" selected>최근 7일</option><option value="30">최근 30일</option></select></div>
  <p id="history-message" class="dash-message" role="status" aria-live="polite">기록을 불러오고 있습니다.</p><div id="history-chart" class="dash-chart-wrap"></div>
  <div id="forecast-state" class="dash-forecast">예측에 필요한 기록을 확인하고 있습니다.</div><details class="dash-help"><summary>그래프와 예상치 안내</summary><p>점을 가리키거나 탭 키로 선택하면 시각·잔여량·출처·초기화를 확인합니다. 초기화 구간은 선을 잇지 않습니다. 예상치는 최근 소비 속도에 따른 추정입니다.</p></details>
 </details>
 <details id="system-details" class="panel disclosure dashboard-full"><summary>앱 상태<span class="summary-note">연결 · 업데이트</span></summary><div class="dash-grid">
  <section aria-labelledby="status-title"><h2 id="status-title">연결 상태</h2><div id="agent-status"><p class="none">상태를 확인하고 있습니다.</p></div><p id="activity-state" class="dash-note"></p><p id="status-message" class="dash-message" role="status"></p></section>
  <section aria-labelledby="update-title"><h2 id="update-title">업데이트</h2><p id="update-state" class="dash-note">버전 정보를 확인하고 있습니다.</p><div class="dash-actions"><button type="button" id="check-update">업데이트 확인</button><a class="dash-release" id="release-link" target="_blank" rel="noopener noreferrer" hidden>릴리스 열기</a></div><pre id="update-notes" class="dash-update-notes" hidden></pre><p id="update-message" class="dash-message" role="status" aria-live="polite"></p></section>
 </div></details>
 <section id="settings" class="dashboard-full"><details id="settings-details" class="panel disclosure"><summary>설정<span class="summary-note">표시 · 알림 · 자동 실행</span></summary>
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
 </details></section>
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
let savedSettings = null, settingsBusy = false, historyData = null, historyRequest = 0, historySelectionInitialized = false, historySelectionTouched = false, polling = false, mutationCount = 0;
const compact = new URLSearchParams(location.search).get('view') === 'summary';
document.body.classList.toggle('summary-view', compact);
if (!compact) { $('detail-action').hidden = true;document.querySelector('.quick-toolbar a[href="/#settings"]').hidden=true; }
function openSettingsAnchor() {
 if(compact)return;
 const target=({'#settings':'settings-details','#token-details':'token-details','#history-details':'history-details','#system-details':'system-details'})[location.hash];
 if(target && $(target)) { $(target).open=true;$(target).scrollIntoView({block:'start'}); }
}
openSettingsAnchor();window.addEventListener('hashchange',openSettingsAnchor);
document.addEventListener('click',event=>{
 const anchor=event.target.closest('a[href]');if(!anchor || event.defaultPrevented || event.button!==0 || event.ctrlKey || event.metaKey || event.shiftKey || event.altKey || (anchor.target && anchor.target!=='_self'))return;
 const target=new URL(anchor.href,location.href);
 if(target.origin!==location.origin || target.pathname!==location.pathname || target.search!==location.search)return;
 if(!['#settings','#token-details','#history-details','#system-details'].includes(target.hash))return;
 event.preventDefault();if(location.hash===target.hash)openSettingsAnchor();else location.hash=target.hash;
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
form.addEventListener('input',()=>message('settings-message','아직 저장하지 않은 변경사항이 있습니다.'));
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
 $('startup-enabled').checked=!!data.enabled;$('startup-enabled').disabled=!data.supported;$('save-startup').disabled=!data.supported;
 $('startup-state').textContent=data.supported ? (data.enabled?'로그인 시 자동 실행이 등록되어 있습니다.':'로그인 시 자동 실행이 꺼져 있습니다.') : '이 운영체제에서는 자동 실행 설정을 지원하지 않습니다.';
}
async function loadStartup() { try { renderStartup(await api('/api/startup')); } catch(e) { message('startup-message','자동 실행 상태를 확인하지 못했습니다.',true); } }
$('startup-enabled').addEventListener('change',()=>message('startup-message','적용 버튼을 누르면 자동 실행 설정을 변경합니다.'));
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
function drawHistory() {
 if(!historyData)return;
 const agent=$('history-agent').value, windowName=$('history-window').value;
 const samples=(historyData.samples || []).filter(s=>s.agent===agent && s.window===windowName && Number.isFinite(s.left) && s.left>=0 && s.left<=100 && !isNaN(new Date(s.at))).sort((a,b)=>new Date(a.at)-new Date(b.at));
 const chart=$('history-chart');chart.replaceChildren();
 if(samples.length<2)message('history-message',samples.length===1?'기록이 1개입니다. 추이를 보려면 2개 이상의 기록이 필요합니다.':'이 기간에 수집한 기록이 없습니다. 앱이 실행되는 동안 기록이 쌓입니다.');
 else message('history-message',samples.length+'개 기록 · '+names[agent]+' · '+(windowName==='week'?'주간 한도':'단기 한도'));
 if(samples.length) {
  const svg=svgElement('svg',{viewBox:'0 0 640 210',class:'dash-history-chart',role:'img','aria-label':names[agent]+' 잔여 한도 이력'}); const start=new Date(samples[0].at).getTime(), end=new Date(samples[samples.length-1].at).getTime();
  for(const pct of [0,25,50,75,100]) { const y=180-pct*1.6;svg.append(svgElement('line',{x1:48,y1:y,x2:622,y2:y,stroke:'#dce3eb','stroke-dasharray':'3 4'})); const label=svgElement('text',{x:4,y:y+4,fill:'#626b78','font-size':12});label.textContent=pct+'%';svg.append(label); }
  // Keep long-running histories responsive while retaining reset boundaries.
  const stride=Math.max(1,Math.ceil(samples.length/600));
  const displayed=samples.filter((s,i)=>i%stride===0 || i===samples.length-1 || (i>0 && s.reset_at!==samples[i-1].reset_at) || (i+1<samples.length && s.reset_at!==samples[i+1].reset_at));
  if(displayed.length<samples.length)message('history-message',samples.length+'개 기록 중 '+displayed.length+'개 대표 지점 · '+names[agent]+' · '+(windowName==='week'?'주간 한도':'단기 한도'));
  const points=displayed.map(s=>({s,x:end===start?335:48+(new Date(s.at).getTime()-start)/(end-start)*574,y:180-s.left*1.6}));let segment=[];
  function line() { if(segment.length>1)svg.append(svgElement('polyline',{points:segment.map(p=>p.x+','+p.y).join(' '),fill:'none',stroke:'#0f766e','stroke-width':2}));segment=[]; }
  points.forEach((p,index)=>{if(index && p.s.reset_at!==points[index-1].s.reset_at)line();segment.push(p);});line();
  for(const p of points) { const label=timeLabel(p.s.at)+' · '+p.s.left+'% 남음 · 출처 '+(p.s.source || '미확인')+' · 초기화 '+(p.s.reset_at>0?timeLabel(p.s.reset_at*1000):'미확인');const dot=svgElement('circle',{cx:p.x,cy:p.y,r:4,fill:'#0f766e',tabindex:0,class:'dash-chart-point','aria-label':label});const title=svgElement('title',{});title.textContent=label;dot.append(title);svg.append(dot); }
  chart.append(svg);const labels=document.createElement('div');labels.className='dash-chart-labels';labels.append(textNode('span',timeLabel(samples[0].at)),textNode('span',timeLabel(samples[samples.length-1].at)));chart.append(labels);
 }
 const forecast=(historyData.forecasts || []).find(f=>f.agent===agent && f.window===windowName), box=$('forecast-state');box.replaceChildren();box.classList.remove('risk');
 if(forecast && Number.isFinite(forecast.daily_budget))box.append(textNode('p','리셋까지 하루 권장 사용량: '+forecast.daily_budget.toFixed(1)+'%p'));
 if(!forecast || !forecast.ready || !Number.isFinite(forecast.rate_per_day) || forecast.rate_per_day<=0) {box.append(textNode('p',forecast && forecast.reason || '소비 속도를 계산할 기록이 부족합니다. 더 많은 기록이 쌓이면 예상치를 표시합니다.'));return;}
 box.classList.toggle('risk',!!forecast.exhausts_before_reset);box.append(textNode('p','최근 소비 속도: 하루 '+forecast.rate_per_day.toFixed(1)+'%p · '+forecast.samples+'개 기록'));
 if(forecast.estimated_exhaustion)box.append(textNode('p','예상 소진: '+timeLabel(forecast.estimated_exhaustion)+(forecast.exhausts_before_reset?' · 리셋 전에 소진될 가능성이 있습니다.':' · 현재 속도에서는 리셋 전 소진 위험이 낮습니다.')));
}
async function loadHistory() {
 const request=++historyRequest;
 try {
  const data=await api('/api/history?days='+$('history-days').value);if(request!==historyRequest)return;historyData=data;
  if(!historySelectionInitialized) {
   historySelectionInitialized=true;
   if(!historySelectionTouched) {
    const latest=(data.samples || []).filter(s=>Object.prototype.hasOwnProperty.call(names,s.agent) && ['short','week'].includes(s.window) && Number.isFinite(s.left) && s.left>=0 && s.left<=100 && !isNaN(new Date(s.at))).reduce((last,s)=>!last || new Date(s.at)>new Date(last.at)?s:last,null);
    if(latest) { $('history-agent').value=latest.agent;$('history-window').value=latest.window; }
   }
  }
  drawHistory();
 }
 catch(e) { if(request===historyRequest)message('history-message','기록을 불러오지 못했습니다. 표시된 그래프는 직전 조회 결과입니다.',true); }
}
$('history-agent').addEventListener('change',()=>{historySelectionTouched=true;drawHistory();});$('history-window').addEventListener('change',()=>{historySelectionTouched=true;drawHistory();});$('history-days').addEventListener('change',()=>{historySelectionTouched=true;loadHistory();});
async function refreshSnapshots() {
 const res=await fetch('/',{cache:'no-store'});if(!res.ok)throw new Error('snapshot');const doc=new DOMParser().parseFromString(await res.text(),'text/html');
 for(const id of ['usage-snapshot','token-snapshot','snapshot-time']) { const source=doc.getElementById(id), target=$(id);if(source && target)target.replaceChildren(...Array.from(source.childNodes)); }
}
async function poll() {
 if(polling || document.hidden || mutationCount)return;polling=true;
 try { await Promise.all([loadStatus(),compact?Promise.resolve():loadHistory(),refreshSnapshots()]); }
 catch(e) { message('action-message','화면 자동 갱신을 완료하지 못했습니다. 앱 연결을 확인해 주세요.',true); }
 finally { polling=false; }
}
loadSettings();loadStartup();loadStatus();if(!compact){loadHistory();loadUpdate(false);}setInterval(poll,30000);
document.addEventListener('visibilitychange',()=>{if(!document.hidden)poll();});
})();
`
